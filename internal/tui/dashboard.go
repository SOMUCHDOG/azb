package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	logger  *log.Logger
	logFile *os.File
)

func init() {
	// Set up logging to file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger = log.New(io.Discard, "", 0)
		return
	}

	logDir := filepath.Join(homeDir, ".azure-boards-cli")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger = log.New(io.Discard, "", 0)
		return
	}

	logFile, err = os.OpenFile(filepath.Join(logDir, "tui.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger = log.New(io.Discard, "", 0)
		return
	}

	logger = log.New(logFile, "[TUI] ", log.LstdFlags)
}

// Dashboard is the main TUI model that coordinates tabs
type Dashboard struct {
	client       *api.Client
	tabs         []Tab
	currentTab   int
	width        int
	height       int
	notification *Notification
	inputPrompt  *InputPrompt
	confirmation *ConfirmationDialog
	err          error
}

// NewDashboard creates a new dashboard
func NewDashboard(client *api.Client) *Dashboard {
	dashboard := &Dashboard{
		client:       client,
		notification: NewNotification("", false),
		inputPrompt:  NewInputPrompt(),
		confirmation: NewConfirmationDialog(),
	}

	// Initialize tabs
	dashboard.tabs = []Tab{
		NewQueriesTab(client, 0, 0),
		NewWorkItemsTab(client, 0, 0),
		NewTemplatesTab(0, 0),
		NewPipelinesTab(0, 0),
		NewAgentsTab(0, 0),
	}

	return dashboard
}

// Init initializes the dashboard
func (d *Dashboard) Init() tea.Cmd {
	// Initialize all tabs
	var cmds []tea.Cmd
	for i, tab := range d.tabs {
		if cmd := tab.Init(d.width, d.height); cmd != nil {
			cmds = append(cmds, cmd)
		}
		logger.Printf("Initialized tab %d: %s", i, tab.Name())
	}
	return tea.Batch(cmds...)
}

// Update handles messages and routes them to appropriate tabs
func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

		// Propagate size to all tabs and forward the message for initialization
		for i, tab := range d.tabs {
			tab.SetSize(d.width, d.height)
			// Forward WindowSizeMsg to each tab so they can trigger initial fetch
			updatedTab, tabCmd := tab.Update(msg)
			d.tabs[i] = updatedTab
			if tabCmd != nil {
				cmds = append(cmds, tabCmd)
			}
		}

		logger.Printf("Window resized to %dx%d", d.width, d.height)
		return d, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Handle global input prompt
		if d.inputPrompt.Active {
			switch msg.Type {
			case tea.KeyEnter:
				value := d.inputPrompt.Value()
				d.inputPrompt.Hide()
				// TODO: Handle input submission based on action
				logger.Printf("Input submitted: %s (action: %s)", value, d.inputPrompt.Action)
				return d, nil
			case tea.KeyEsc:
				d.inputPrompt.Hide()
				return d, nil
			default:
				cmd = d.inputPrompt.Update(msg)
				return d, cmd
			}
		}

		// Handle global confirmation dialog
		if d.confirmation.Active {
			switch msg.String() {
			case "y", "Y":
				d.confirmation.Hide()
				// TODO: Handle confirmation based on action
				logger.Printf("Confirmed action: %s", d.confirmation.Action)
				return d, nil
			case "n", "N", "esc":
				d.confirmation.Hide()
				return d, nil
			}
			return d, nil
		}

		// Handle quit
		if key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))) {
			return d, tea.Quit
		}

		// Handle tab switching
		if msg.String() == "tab" {
			d.currentTab = (d.currentTab + 1) % len(d.tabs)
			logger.Printf("Switched to tab %d: %s", d.currentTab, d.tabs[d.currentTab].Name())
			return d, nil
		}
		if msg.String() == "shift+tab" {
			d.currentTab = (d.currentTab - 1 + len(d.tabs)) % len(d.tabs)
			logger.Printf("Switched to tab %d: %s", d.currentTab, d.tabs[d.currentTab].Name())
			return d, nil
		}

		// Route message to active tab
		tab, cmd := d.tabs[d.currentTab].Update(msg)
		d.tabs[d.currentTab] = tab
		return d, cmd

	case NotificationMsg:
		d.notification.Show(msg.Message, msg.IsError)
		logger.Printf("Notification: %s (error: %v)", msg.Message, msg.IsError)
		return d, nil

	case ClearNotificationMsg:
		d.notification.Clear()
		return d, nil

	case SwitchToTabMsg:
		if msg.TabIndex >= 0 && msg.TabIndex < len(d.tabs) {
			d.currentTab = msg.TabIndex
			logger.Printf("Switched to tab %d: %s", d.currentTab, d.tabs[d.currentTab].Name())
		}
		return d, nil

	case WorkItemsLoadedMsg, QueryExecutedMsg, WorkItemDeletedMsg:
		// Route work item messages to Work Items tab (index 1)
		logger.Printf("Routing work item message to Work Items tab")
		if len(d.tabs) > 1 {
			tab, cmd := d.tabs[1].Update(msg)
			d.tabs[1] = tab
			cmds = append(cmds, cmd)
		}
		return d, tea.Batch(cmds...)

	case QueriesLoadedMsg:
		// Route queries messages to Queries tab (index 0)
		logger.Printf("Routing queries message to Queries tab")
		if len(d.tabs) > 0 {
			tab, cmd := d.tabs[0].Update(msg)
			d.tabs[0] = tab
			cmds = append(cmds, cmd)
		}
		return d, tea.Batch(cmds...)

	case TemplatesLoadedMsg:
		// Route templates messages to Templates tab (index 2)
		logger.Printf("Routing templates message to Templates tab")
		if len(d.tabs) > 2 {
			tab, cmd := d.tabs[2].Update(msg)
			d.tabs[2] = tab
			cmds = append(cmds, cmd)
		}
		return d, tea.Batch(cmds...)

	default:
		// Route all other messages to the active tab
		tab, cmd := d.tabs[d.currentTab].Update(msg)
		d.tabs[d.currentTab] = tab
		cmds = append(cmds, cmd)
	}

	return d, tea.Batch(cmds...)
}

// View renders the dashboard
func (d *Dashboard) View() string {
	if d.err != nil {
		return RenderErrorWithRetry(d.err)
	}

	// Build tab names
	tabNames := make([]string, len(d.tabs))
	for i, tab := range d.tabs {
		tabNames[i] = tab.Name()
	}

	// Render components
	parts := []string{
		RenderHeader(),
		RenderTabBar(tabNames, d.currentTab),
		d.tabs[d.currentTab].View(),
	}

	// Add overlays
	if d.notification.Visible {
		parts = append(parts, d.notification.View())
	}

	if d.inputPrompt.Active {
		parts = append(parts, d.inputPrompt.View())
	}

	if d.confirmation.Active {
		parts = append(parts, d.confirmation.View())
	}

	// Add footer
	parts = append(parts, RenderFooter(""))

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// Run starts the dashboard TUI
func Run(client *api.Client) error {
	dashboard := NewDashboard(client)

	p := tea.NewProgram(dashboard, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running dashboard: %w", err)
	}

	return nil
}
