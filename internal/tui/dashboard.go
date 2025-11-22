package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/casey/azure-boards-cli/internal/api"
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

	// Controllers
	keybinds *KeybindController
	actions  *ActionController
	help     *HelpController
}

// NewDashboard creates a new dashboard
func NewDashboard(client *api.Client) *Dashboard {
	// Initialize keybind controller first
	keybinds := NewKeybindController()

	dashboard := &Dashboard{
		client:       client,
		notification: NewNotification("", false),
		inputPrompt:  NewInputPrompt(),
		confirmation: NewConfirmationDialog(),
		keybinds:     keybinds,
		actions:      NewActionController(keybinds),
		help:         NewHelpController(keybinds),
	}

	// Initialize tabs
	dashboard.tabs = []Tab{
		NewQueriesTab(client, 0, 0),
		NewWorkItemsTab(client, 0, 0),
		NewTemplatesTab(client, 0, 0),
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
				action := d.inputPrompt.Action
				context := d.inputPrompt.Context
				d.inputPrompt.Hide()

				// Handle action based on type
				if action == "rename_template" {
					if oldPath, ok := context.(string); ok {
						logger.Printf("Renaming template '%s' to '%s'", oldPath, value)
						return d, renameTemplate(oldPath, value)
					}
				} else if action == "copy_template" {
					if oldPath, ok := context.(string); ok {
						logger.Printf("Copying template '%s' to '%s'", oldPath, value)
						return d, copyTemplate(oldPath, value)
					}
				} else if action == "new_template" {
					logger.Printf("Creating new template: %s", value)
					return d, tea.Batch(
						createNewTemplate(value),
						func() tea.Msg {
							// Refresh templates after a brief delay
							return RefreshTemplatesMsg{}
						},
					)
				} else if action == "new_folder" {
					logger.Printf("Creating new folder: %s", value)
					return d, tea.Batch(
						createNewFolder(value),
						func() tea.Msg {
							// Refresh templates after a brief delay
							return RefreshTemplatesMsg{}
						},
					)
				}

				logger.Printf("Input submitted: %s (action: %s)", value, action)
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
				action := d.confirmation.Action
				context := d.confirmation.Context
				d.confirmation.Hide()

				// Handle confirmed action
				if action == "delete_work_item" {
					if ctx, ok := context.(ConfirmDeleteWorkItemMsg); ok {
						logger.Printf("Executing delete for work item #%d with %d children", ctx.WorkItemID, len(ctx.ChildIDs))
						return d, deleteWorkItemWithChildren(d.client, ctx.WorkItemID, ctx.ChildIDs)
					}
				} else if action == "delete_template" {
					if ctx, ok := context.(ConfirmDeleteTemplateMsg); ok {
						logger.Printf("Executing delete for template: %s", ctx.Path)
						return d, deleteTemplate(ctx.Path, ctx.IsDir)
					}
				}

				logger.Printf("Confirmed action: %s", action)
				return d, nil
			case "n", "N", "esc":
				d.confirmation.Hide()
				logger.Printf("Cancelled action: %s", d.confirmation.Action)
				return d, nil
			}
			return d, nil
		}

		// Handle help toggle (? key)
		if d.keybinds.Matches(msg, "global", "help") {
			if d.help.IsVisible() {
				d.help.Hide()
				logger.Printf("Help hidden")
			} else {
				currentTabName := d.tabs[d.currentTab].Name()
				d.help.Show(currentTabName)
				logger.Printf("Help shown for tab: %s", currentTabName)
			}
			return d, nil
		}

		// If help is visible, block all other input (except help toggle handled above)
		if d.help.IsVisible() {
			return d, nil
		}

		// Handle quit
		if d.keybinds.Matches(msg, "global", "quit") {
			return d, tea.Quit
		}

		// Handle tab switching
		if d.keybinds.Matches(msg, "global", "next_tab") {
			d.currentTab = (d.currentTab + 1) % len(d.tabs)
			logger.Printf("Switched to tab %d: %s", d.currentTab, d.tabs[d.currentTab].Name())
			return d, nil
		}
		if d.keybinds.Matches(msg, "global", "prev_tab") {
			d.currentTab = (d.currentTab - 1 + len(d.tabs)) % len(d.tabs)
			logger.Printf("Switched to tab %d: %s", d.currentTab, d.tabs[d.currentTab].Name())
			return d, nil
		}

		// Check if actions can be executed (not during filtering, etc.)
		if d.actions.CanExecuteAction(d.tabs[d.currentTab]) {
			// Handle Work Items tab actions
			if d.tabs[d.currentTab].Name() == "Work Items" {
				if workitemsTab, ok := d.tabs[d.currentTab].(*WorkItemsTab); ok {
					// Download work item (w key)
					if d.keybinds.Matches(msg, "workitems", "download") {
						logger.Printf("Download action triggered")
						return d, workitemsTab.handleDownloadAction()
					}
					// Edit work item (e key)
					if d.keybinds.Matches(msg, "workitems", "edit") {
						logger.Printf("Edit action triggered")
						return d, workitemsTab.handleEditAction()
					}
					// Delete work item (d key)
					if d.keybinds.Matches(msg, "workitems", "delete") {
						logger.Printf("Delete action triggered")
						return d, workitemsTab.handleDeleteAction()
					}
				}
			}

			// Handle Templates tab actions
			if d.tabs[d.currentTab].Name() == "Templates" {
				if templatesTab, ok := d.tabs[d.currentTab].(*TemplatesTab); ok {
					// Edit template (e key)
					if d.keybinds.Matches(msg, "templates", "edit") {
						logger.Printf("Edit template action triggered")
						return d, templatesTab.handleEditAction()
					}
					// Rename template (m key)
					if d.keybinds.Matches(msg, "templates", "rename") {
						logger.Printf("Rename action triggered")
						if prompt := templatesTab.handleRenameAction(); prompt != nil {
							d.inputPrompt = prompt
						}
						return d, nil
					}
					// Delete template (d key)
					if d.keybinds.Matches(msg, "templates", "delete") {
						logger.Printf("Delete template action triggered")
						return d, templatesTab.handleDeleteAction()
					}
					// Copy template (c key)
					if d.keybinds.Matches(msg, "templates", "copy") {
						logger.Printf("Copy template action triggered")
						if prompt := templatesTab.handleCopyAction(); prompt != nil {
							d.inputPrompt = prompt
						}
						return d, nil
					}
					// New template (n key)
					if d.keybinds.Matches(msg, "templates", "new_template") {
						logger.Printf("New template action triggered")
						if prompt := templatesTab.handleNewTemplateAction(); prompt != nil {
							d.inputPrompt = prompt
						}
						return d, nil
					}
					// New folder (f key)
					if d.keybinds.Matches(msg, "templates", "new_folder") {
						logger.Printf("New folder action triggered")
						if prompt := templatesTab.handleNewFolderAction(); prompt != nil {
							d.inputPrompt = prompt
						}
						return d, nil
					}
				}
			}
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

	case ConfirmDeleteWorkItemMsg:
		// Show confirmation dialog for work item deletion
		childCount := len(msg.ChildIDs)
		childText := ""
		if childCount > 0 {
			childText = fmt.Sprintf(" and its %d child task(s)", childCount)
		}

		d.confirmation.Show(
			fmt.Sprintf("Delete work item #%d: '%s'%s?", msg.WorkItemID, msg.Title, childText),
			"delete_work_item",
			msg, // Store context for when user confirms
		)
		logger.Printf("Showing delete confirmation for work item #%d with %d children", msg.WorkItemID, childCount)
		return d, nil

	case WorkItemsLoadedMsg, QueryExecutedMsg, WorkItemDeletedMsg, WorkItemCreatedMsg:
		// Route work item messages to Work Items tab (index 1)
		logger.Printf("Routing work item message to Work Items tab")
		if len(d.tabs) > 1 {
			tab, cmd := d.tabs[1].Update(msg)
			d.tabs[1] = tab
			cmds = append(cmds, cmd)
		}

		// Show notification for created work item
		if createdMsg, ok := msg.(WorkItemCreatedMsg); ok {
			if createdMsg.Error == nil && createdMsg.WorkItem != nil {
				workItemID := *createdMsg.WorkItem.Id
				message := fmt.Sprintf("Created work item #%d", workItemID)

				// Switch to Work Items tab (index 1)
				d.currentTab = 1
				logger.Printf("Switched to Work Items tab after creation")

				cmds = append(cmds, func() tea.Msg {
					return NotificationMsg{
						Message: message,
						IsError: false,
					}
				})
			}
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

	case TemplateRenamedMsg:
		// Show notification and refresh templates
		if msg.Error != nil {
			logger.Printf("Failed to rename template: %v", msg.Error)
			return d, func() tea.Msg {
				return NotificationMsg{
					Message: fmt.Sprintf("Failed to rename: %v", msg.Error),
					IsError: true,
				}
			}
		}

		logger.Printf("Template renamed successfully")

		// Show success notification
		cmds = append(cmds, func() tea.Msg {
			return NotificationMsg{
				Message: fmt.Sprintf("Renamed to '%s'", msg.NewPath),
				IsError: false,
			}
		})

		// Refresh templates list
		if len(d.tabs) > 2 {
			if templatesTab, ok := d.tabs[2].(*TemplatesTab); ok {
				cmds = append(cmds, templatesTab.FetchTemplates())
			}
		}

		return d, tea.Batch(cmds...)

	case ConfirmDeleteTemplateMsg:
		// Show confirmation dialog for template deletion
		d.confirmation.Show(
			msg.Prompt,
			"delete_template",
			msg,
		)
		logger.Printf("Showing delete confirmation for template: %s", msg.Path)
		return d, nil

	case TemplateDeletedMsg:
		// Show notification and refresh templates
		if msg.Error != nil {
			logger.Printf("Failed to delete template: %v", msg.Error)
			return d, func() tea.Msg {
				return NotificationMsg{
					Message: fmt.Sprintf("Failed to delete: %v", msg.Error),
					IsError: true,
				}
			}
		}

		logger.Printf("Template deleted successfully")

		// Show success notification
		cmds = append(cmds, func() tea.Msg {
			return NotificationMsg{
				Message: fmt.Sprintf("Deleted '%s'", msg.TemplatePath),
				IsError: false,
			}
		})

		// Refresh templates list
		if len(d.tabs) > 2 {
			if templatesTab, ok := d.tabs[2].(*TemplatesTab); ok {
				cmds = append(cmds, templatesTab.FetchTemplates())
			}
		}

		return d, tea.Batch(cmds...)

	case TemplateCopiedMsg:
		// Show notification and refresh templates
		if msg.Error != nil {
			logger.Printf("Failed to copy template: %v", msg.Error)
			return d, func() tea.Msg {
				return NotificationMsg{
					Message: fmt.Sprintf("Failed to copy: %v", msg.Error),
					IsError: true,
				}
			}
		}

		logger.Printf("Template copied successfully")

		// Show success notification
		cmds = append(cmds, func() tea.Msg {
			return NotificationMsg{
				Message: fmt.Sprintf("Copied to '%s'", msg.NewPath),
				IsError: false,
			}
		})

		// Refresh templates list
		if len(d.tabs) > 2 {
			if templatesTab, ok := d.tabs[2].(*TemplatesTab); ok {
				cmds = append(cmds, templatesTab.FetchTemplates())
			}
		}

		return d, tea.Batch(cmds...)

	case TemplateFolderCreatedMsg:
		// Refresh templates after folder creation (notification already shown)
		if msg.Error == nil {
			if len(d.tabs) > 2 {
				if templatesTab, ok := d.tabs[2].(*TemplatesTab); ok {
					return d, templatesTab.FetchTemplates()
				}
			}
		}
		return d, nil

	case RefreshTemplatesMsg:
		// Refresh templates list
		logger.Printf("Refreshing templates list")
		if len(d.tabs) > 2 {
			if templatesTab, ok := d.tabs[2].(*TemplatesTab); ok {
				return d, templatesTab.FetchTemplates()
			}
		}
		return d, nil

	case OpenEditorForTemplateMsg:
		// Open editor to edit template file
		logger.Printf("Opening editor for template: %s", msg.FilePath)

		// Get editor from environment
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi" // fallback to vi
		}

		// Create command to open editor
		c := exec.Command(editor, msg.FilePath)

		// Return tea.ExecProcess to suspend TUI and run editor
		return d, tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				logger.Printf("Editor error: %v", err)
				return NotificationMsg{
					Message: fmt.Sprintf("Editor error: %v", err),
					IsError: true,
				}
			}
			logger.Printf("Editor closed for template")
			// Refresh templates list after edit
			return RefreshTemplatesMsg{}
		})

	case OpenEditorMsg:
		// Open editor to edit work item
		logger.Printf("Opening editor for work item #%d at %s", msg.WorkItemID, msg.FilePath)

		// Get editor from environment
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi" // fallback to vi
		}

		// Create command to open editor
		c := exec.Command(editor, msg.FilePath)

		// Return tea.ExecProcess to suspend TUI and run editor
		return d, tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				logger.Printf("Editor error: %v", err)
				return NotificationMsg{
					Message: fmt.Sprintf("Editor error: %v", err),
					IsError: true,
				}
			}
			logger.Printf("Editor closed, processing edited work item #%d", msg.WorkItemID)
			return ProcessEditedWorkItemMsg{
				FilePath:   msg.FilePath,
				WorkItemID: msg.WorkItemID,
				Client:     msg.Client,
			}
		})

	case ProcessEditedWorkItemMsg:
		// Process the edited work item after editor closes
		logger.Printf("Processing edited work item #%d", msg.WorkItemID)
		return d, processEditedWorkItem(msg.FilePath, msg.WorkItemID, msg.Client)

	case CreateWorkItemFromTemplateMsg:
		// Create work item from template
		logger.Printf("Creating work item from template: %s", msg.Template.Name)
		return d, executeCreateWorkItemFromTemplate(d.client, msg.Template)

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

	mainView := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Render help overlay on top of everything if visible
	if d.help.IsVisible() {
		return d.help.View(d.width, d.height)
	}

	return mainView
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
