package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/templates"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

var (
	logger  *log.Logger
	logFile *os.File
)

func init() {
	// Set up logging to file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, disable logging
		logger = log.New(io.Discard, "", 0)
		return
	}

	logDir := filepath.Join(homeDir, ".azure-boards-cli")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If we can't create directory, disable logging
		logger = log.New(io.Discard, "", 0)
		return
	}

	logFile, err = os.OpenFile(filepath.Join(logDir, "tui.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't open log file, disable logging
		logger = log.New(io.Discard, "", 0)
		return
	}

	logger = log.New(logFile, "[TUI] ", log.LstdFlags)
}

// KeyMap defines keyboard shortcuts
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Refresh  key.Binding
	New      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Help     key.Binding
	Quit     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Search   key.Binding
	State    key.Binding
	Assign   key.Binding
	Tags     key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new work item"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		State: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "change state"),
		),
		Assign: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "assign"),
		),
		Tags: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tags"),
		),
	}
}

// Model is the main TUI model
type Model struct {
	client           *api.Client
	workItems        []workitemtracking.WorkItem
	workItemCache    map[int]*workitemtracking.WorkItem // Cache for fetched work items
	relationshipData map[int]*relationshipInfo           // Cache for formatted relationship data
	list             list.Model
	viewport         viewport.Model
	keys             KeyMap
	width            int
	height           int
	selectedItem     *workitemtracking.WorkItem
	showDetails      bool
	loading          bool
	loadingRelations bool
	err              error

	// Tab navigation
	currentTab int      // 0=queries, 1=workitems, 2=templates, 3=pipelines, 4=agents
	tabs       []string // Tab names

	// Queries tab
	queries         []workitemtracking.QueryHierarchyItem
	queryList       list.Model
	loadingQueries  bool
	expandedFolders map[string]bool // Track which folders are expanded by their path

	// Templates tab
	templates              []*templates.TemplateNode
	templateList           list.Model
	loadingTemplates       bool
	expandedTemplateFolders map[string]bool // Track which template folders are expanded
}

// relationshipInfo stores formatted relationship data for a work item
type relationshipInfo struct {
	parent      string
	children    []string
	prs         []string
	deployments []string
	loaded      bool
}

// queryDelegate implements list.ItemDelegate for query items
type queryDelegate struct {
	expandedFolders map[string]bool
}

func (d queryDelegate) Height() int                             { return 1 }
func (d queryDelegate) Spacing() int                            { return 0 }
func (d queryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d queryDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	queryItem, ok := item.(queryListItem)
	if !ok {
		return
	}

	var (
		normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
		folderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
		queryStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	)

	indent := strings.Repeat("  ", queryItem.Depth)
	icon := ""
	nameStyle := normalStyle

	if queryItem.IsFolder {
		// Check if folder is expanded
		expanded := d.expandedFolders[queryItem.Path]
		if expanded {
			icon = "▼ "
		} else {
			icon = "▶ "
		}
		nameStyle = folderStyle
	} else {
		icon = "  🔍 "
		nameStyle = queryStyle
	}

	name := queryItem.Name
	if len(name) > 60 {
		name = name[:57] + "..."
	}

	var output string
	if index == m.Index() {
		output = selectedStyle.Render(fmt.Sprintf("> %s%s%s", indent, icon, name))
	} else {
		output = nameStyle.Render(fmt.Sprintf("  %s%s%s", indent, icon, name))
	}

	fmt.Fprint(w, output)
}

// workItemDelegate implements list.ItemDelegate
type workItemDelegate struct{}

func (d workItemDelegate) Height() int                             { return 1 }
func (d workItemDelegate) Spacing() int                            { return 0 }
func (d workItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d workItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	workItem, ok := item.(workItemItem)
	if !ok {
		return
	}

	var (
		title        = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		selected     = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
		dimmed       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		stateActive  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		stateNew     = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
		stateClosed  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		stateBlocked = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	)

	id := fmt.Sprintf("%-8d", workItem.ID)
	titleStr := workItem.Title
	if len(titleStr) > 40 {
		titleStr = titleStr[:37] + "..."
	}
	titleStr = fmt.Sprintf("%-40s", titleStr)

	state := workItem.State
	stateStyle := dimmed
	switch state {
	case "Active":
		stateStyle = stateActive
	case "New":
		stateStyle = stateNew
	case "Closed", "Resolved":
		stateStyle = stateClosed
	case "Blocked":
		stateStyle = stateBlocked
	}
	stateStr := stateStyle.Render(fmt.Sprintf("%-12s", state))

	assignee := workItem.AssignedTo
	if len(assignee) > 20 {
		assignee = assignee[:17] + "..."
	}
	assigneeStr := fmt.Sprintf("%-20s", assignee)

	var output string
	if index == m.Index() {
		output = selected.Render(fmt.Sprintf("> %s │ %s │ %s │ %s", id, titleStr, stateStr, assigneeStr))
	} else {
		output = title.Render(fmt.Sprintf("  %s │ %s │ %s │ %s", id, titleStr, stateStr, assigneeStr))
	}

	fmt.Fprint(w, output)
}

// queryListItem wraps a query for the list
type queryListItem struct {
	Name     string
	Path     string
	IsFolder bool
	Depth    int
	query    workitemtracking.QueryHierarchyItem
}

func (i queryListItem) FilterValue() string { return i.Name }

// workItemItem wraps a work item for the list
type workItemItem struct {
	ID         int
	Title      string
	State      string
	AssignedTo string
	workItem   workitemtracking.WorkItem
}

func (i workItemItem) FilterValue() string { return i.Title }

// templateListItem wraps a template node for the list
type templateListItem struct {
	Name     string
	Path     string
	IsDir    bool
	Depth    int
	Template *templates.Template
	node     *templates.TemplateNode
}

func (i templateListItem) FilterValue() string { return i.Name }

// templateDelegate implements list.ItemDelegate for template items
type templateDelegate struct {
	expandedFolders map[string]bool
}

func (d templateDelegate) Height() int                             { return 1 }
func (d templateDelegate) Spacing() int                            { return 0 }
func (d templateDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d templateDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	templateItem, ok := item.(templateListItem)
	if !ok {
		return
	}

	var (
		normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
		folderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
		templateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	)

	indent := strings.Repeat("  ", templateItem.Depth)
	icon := ""
	nameStyle := normalStyle

	if templateItem.IsDir {
		// Check if folder is expanded
		expanded := d.expandedFolders[templateItem.Path]
		if expanded {
			icon = "▼ "
		} else {
			icon = "▶ "
		}
		nameStyle = folderStyle
	} else {
		icon = "  📄 "
		nameStyle = templateStyle
	}

	name := templateItem.Name
	if len(name) > 60 {
		name = name[:57] + "..."
	}

	var output string
	if index == m.Index() {
		output = selectedStyle.Render(fmt.Sprintf("> %s%s%s", indent, icon, name))
	} else {
		output = nameStyle.Render(fmt.Sprintf("  %s%s%s", indent, icon, name))
	}

	fmt.Fprint(w, output)
}

// NewModel creates a new dashboard model
func NewModel(client *api.Client) Model {
	keys := DefaultKeyMap()

	// Create work items list
	items := []list.Item{}
	delegate := workItemDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.Title = "Work Items"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	// Create queries list with delegate that tracks expanded folders
	expandedFolders := make(map[string]bool)
	queryItems := []list.Item{}
	queryDel := queryDelegate{expandedFolders: expandedFolders}
	ql := list.New(queryItems, queryDel, 0, 0)
	ql.Title = "Saved Queries"
	ql.SetShowStatusBar(true)
	ql.SetFilteringEnabled(true)
	ql.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	// Create templates list with delegate that tracks expanded folders
	expandedTemplateFolders := make(map[string]bool)
	templateItems := []list.Item{}
	templateDel := templateDelegate{expandedFolders: expandedTemplateFolders}
	tl := list.New(templateItems, templateDel, 0, 0)
	tl.Title = "Templates"
	tl.SetShowStatusBar(true)
	tl.SetFilteringEnabled(true)
	tl.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	// Create viewport for details
	vp := viewport.New(0, 0)

	return Model{
		client:           client,
		workItems:        []workitemtracking.WorkItem{},
		workItemCache:    make(map[int]*workitemtracking.WorkItem),
		relationshipData: make(map[int]*relationshipInfo),
		list:             l,
		viewport:         vp,
		keys:             keys,
		showDetails:      false,
		loading:          true,
		loadingRelations: false,

		// Initialize tabs
		currentTab: 1, // Start on "Work Items" tab (index 1)
		tabs:       []string{"Queries", "Work Items", "Templates", "Pipelines", "Agents"},

		// Initialize queries
		queries:         []workitemtracking.QueryHierarchyItem{},
		queryList:       ql,
		loadingQueries:  true,
		expandedFolders: make(map[string]bool),

		// Initialize templates
		templates:               []*templates.TemplateNode{},
		templateList:            tl,
		loadingTemplates:        true,
		expandedTemplateFolders: expandedTemplateFolders,
	}
}

// Init initializes the dashboard
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchWorkItems(m.client),
		fetchQueries(m.client),
		fetchTemplates(),
		tea.EnterAltScreen,
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3
		footerHeight := 3
		verticalMargins := headerHeight + footerHeight

		// Update query list size
		m.queryList.SetSize(m.width, m.height-verticalMargins)

		// Update template list size
		m.templateList.SetSize(m.width, m.height-verticalMargins)

		if m.showDetails {
			// Split view: list on top, details on bottom
			listHeight := (m.height - verticalMargins) / 2
			detailsHeight := (m.height - verticalMargins) - listHeight
			m.list.SetSize(m.width, listHeight)
			m.viewport.Width = m.width - 4
			m.viewport.Height = detailsHeight - 4
		} else {
			// Full list view
			m.list.SetSize(m.width, m.height-verticalMargins)
		}

		// Update viewport content if we have a selected item
		if m.showDetails && m.selectedItem != nil {
			m.viewport.SetContent(m.formatWorkItemDetails(*m.selectedItem))
		}

	case tea.KeyMsg:
		// Handle quit
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		// Handle tab navigation
		if msg.String() == "tab" {
			m.currentTab = (m.currentTab + 1) % len(m.tabs)
			return m, nil
		}
		if msg.String() == "shift+tab" {
			m.currentTab = (m.currentTab - 1 + len(m.tabs)) % len(m.tabs)
			return m, nil
		}

		// Handle refresh
		if key.Matches(msg, m.keys.Refresh) {
			logger.Println("Refresh triggered - clearing cache")
			m.loading = true
			// Clear caches
			m.workItemCache = make(map[int]*workitemtracking.WorkItem)
			m.relationshipData = make(map[int]*relationshipInfo)
			return m, fetchWorkItems(m.client)
		}

		// Handle enter to show/hide details or execute query
		if key.Matches(msg, m.keys.Enter) {
			if m.currentTab == 0 {
				// Toggle folder or execute query
				selectedItem := m.queryList.SelectedItem()
				if item, ok := selectedItem.(queryListItem); ok {
					if item.IsFolder {
						// Toggle folder expand/collapse
						m.expandedFolders[item.Path] = !m.expandedFolders[item.Path]
						logger.Printf("Toggling folder '%s' to expanded=%v", item.Name, m.expandedFolders[item.Path])

						// Rebuild the query list with new expanded state and delegate
						items := m.flattenQueries(m.queries, 0)
						queryDel := queryDelegate{expandedFolders: m.expandedFolders}
						m.queryList = list.New(items, queryDel, m.width, m.height-6)
						m.queryList.Title = "Saved Queries"
						m.queryList.SetShowStatusBar(true)
						m.queryList.SetFilteringEnabled(true)
						m.queryList.Styles.Title = lipgloss.NewStyle().
							Background(lipgloss.Color("62")).
							Foreground(lipgloss.Color("230")).
							Padding(0, 1)
					} else {
						// Execute query
						logger.Printf("Executing query: %s", item.Name)
						m.loading = true
						m.currentTab = 1 // Switch to Work Items tab to show results
						return m, executeQuery(m.client, item.query)
					}
				}
				return m, nil
			} else if m.currentTab == 1 {
				// Work Items tab - toggle details
				m.showDetails = !m.showDetails
				if m.showDetails && len(m.list.Items()) > 0 {
					// Get selected work item
					selectedItem := m.list.SelectedItem()
					if item, ok := selectedItem.(workItemItem); ok {
						m.selectedItem = &item.workItem

						// Resize viewport for details view
						headerHeight := 3
						footerHeight := 3
						verticalMargins := headerHeight + footerHeight
						listHeight := (m.height - verticalMargins) / 2
						detailsHeight := (m.height - verticalMargins) - listHeight
						m.list.SetSize(m.width, listHeight)
						m.viewport.Width = m.width - 4
						m.viewport.Height = detailsHeight - 4

						// Set content
						m.viewport.SetContent(m.formatWorkItemDetails(item.workItem))
						m.viewport.GotoTop()

						// Load relationships if not already loaded
						workItemID := 0
						if item.workItem.Id != nil {
							workItemID = *item.workItem.Id
						}
						if _, exists := m.relationshipData[workItemID]; !exists && workItemID > 0 {
							logger.Printf("Cache MISS for work item #%d - loading...", workItemID)
							m.loadingRelations = true
							return m, loadRelationships(m.client, item.workItem)
						} else if exists {
							logger.Printf("Cache HIT for work item #%d - using cached data", workItemID)
						}
					}
				} else {
					// Resize list for full view
					headerHeight := 3
					footerHeight := 3
					verticalMargins := headerHeight + footerHeight
					m.list.SetSize(m.width, m.height-verticalMargins)
				}
				return m, nil
			} else if m.currentTab == 2 {
				// Templates tab - toggle folder
				selectedItem := m.templateList.SelectedItem()
				if item, ok := selectedItem.(templateListItem); ok {
					if item.IsDir {
						// Toggle folder expand/collapse
						m.expandedTemplateFolders[item.Path] = !m.expandedTemplateFolders[item.Path]
						logger.Printf("Toggling template folder '%s' to expanded=%v", item.Name, m.expandedTemplateFolders[item.Path])

						// Rebuild the template list with new expanded state and delegate
						items := m.flattenTemplates(m.templates, 0)
						templateDel := templateDelegate{expandedFolders: m.expandedTemplateFolders}
						m.templateList = list.New(items, templateDel, m.width, m.height-6)
						m.templateList.Title = "Templates"
						m.templateList.SetShowStatusBar(true)
						m.templateList.SetFilteringEnabled(true)
						m.templateList.Styles.Title = lipgloss.NewStyle().
							Background(lipgloss.Color("62")).
							Foreground(lipgloss.Color("230")).
							Padding(0, 1)
					} else {
						// TODO: Handle template selection/use
						logger.Printf("Selected template: %s", item.Name)
					}
				}
				return m, nil
			}
		}

		// Handle back
		if key.Matches(msg, m.keys.Back) && m.showDetails {
			m.showDetails = false
			return m, nil
		}

	case workItemsMsg:
		m.loading = false
		m.workItems = msg.items
		items := make([]list.Item, len(msg.items))
		for i, wi := range msg.items {
			items[i] = workItemItem{
				ID:         getIntField(&wi, "System.Id"),
				Title:      getStringField(&wi, "System.Title"),
				State:      getStringField(&wi, "System.State"),
				AssignedTo: getStringField(&wi, "System.AssignedTo"),
				workItem:   wi,
			}
		}
		m.list.SetItems(items)

	case queriesMsg:
		m.loadingQueries = false
		m.queries = msg.queries
		items := m.flattenQueries(msg.queries, 0)
		m.queryList.SetItems(items)
		logger.Printf("Query list populated with %d items", len(items))

	case templatesMsg:
		m.loadingTemplates = false
		m.templates = msg.templates
		items := m.flattenTemplates(msg.templates, 0)
		m.templateList.SetItems(items)
		logger.Printf("Template list populated with %d items", len(items))

	case relationshipsLoadedMsg:
		m.loadingRelations = false
		m.relationshipData[msg.workItemID] = msg.relInfo

		// Update viewport if we're viewing this work item
		if m.showDetails && m.selectedItem != nil && m.selectedItem.Id != nil && *m.selectedItem.Id == msg.workItemID {
			m.viewport.SetContent(m.formatWorkItemDetails(*m.selectedItem))
		}

	case errMsg:
		m.loading = false
		m.err = msg.err
	}

	// Update appropriate list based on current tab
	if m.currentTab == 0 {
		// Update query list
		m.queryList, cmd = m.queryList.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.currentTab == 1 {
		// Update work item list
		prevIndex := m.list.Index()
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Update details if selection changed while in details view
		if m.showDetails && m.list.Index() != prevIndex && len(m.list.Items()) > 0 {
			selectedItem := m.list.SelectedItem()
			if item, ok := selectedItem.(workItemItem); ok {
				m.selectedItem = &item.workItem
				m.viewport.SetContent(m.formatWorkItemDetails(item.workItem))
				m.viewport.GotoTop()

				// Load relationships if not already loaded
				workItemID := 0
				if item.workItem.Id != nil {
					workItemID = *item.workItem.Id
				}
				if _, exists := m.relationshipData[workItemID]; !exists && workItemID > 0 {
					logger.Printf("Cache MISS for work item #%d - loading...", workItemID)
					m.loadingRelations = true
					cmds = append(cmds, loadRelationships(m.client, item.workItem))
				} else if exists {
					logger.Printf("Cache HIT for work item #%d - using cached data", workItemID)
				}
			}
		}

		// Update viewport if details are shown
		if m.showDetails {
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else if m.currentTab == 2 {
		// Update template list
		m.templateList, cmd = m.templateList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// flattenQueries recursively flattens the query hierarchy into a list, respecting expanded state
func (m Model) flattenQueries(queries []workitemtracking.QueryHierarchyItem, depth int) []list.Item {
	var items []list.Item

	for _, q := range queries {
		name := ""
		if q.Name != nil {
			name = *q.Name
		}

		path := ""
		if q.Path != nil {
			path = *q.Path
		}

		isFolder := q.IsFolder != nil && *q.IsFolder

		// Always add the current item (folder or query)
		items = append(items, queryListItem{
			Name:     name,
			Path:     path,
			IsFolder: isFolder,
			Depth:    depth,
			query:    q,
		})

		// Only add children if this is a folder AND it's expanded
		if isFolder && q.Children != nil && len(*q.Children) > 0 {
			// Check if folder is expanded
			if m.expandedFolders[path] {
				childItems := m.flattenQueries(*q.Children, depth+1)
				items = append(items, childItems...)
			}
		}
	}

	return items
}

// flattenTemplates recursively flattens the template hierarchy into a list, respecting expanded state
func (m Model) flattenTemplates(templateNodes []*templates.TemplateNode, depth int) []list.Item {
	var items []list.Item

	for _, node := range templateNodes {
		// Always add the current item (folder or template)
		items = append(items, templateListItem{
			Name:     node.Name,
			Path:     node.Path,
			IsDir:    node.IsDir,
			Depth:    depth,
			Template: node.Template,
			node:     node,
		})

		// Only add children if this is a directory AND it's expanded
		if node.IsDir && node.Children != nil && len(node.Children) > 0 {
			// Check if folder is expanded
			if m.expandedTemplateFolders[node.Path] {
				childItems := m.flattenTemplates(node.Children, depth+1)
				items = append(items, childItems...)
			}
		}
	}

	return items
}

// View renders the dashboard
func (m Model) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Padding(1).
			Render("Loading work items...")
	}

	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Padding(1).
			Render(fmt.Sprintf("Error: %v\n\nPress q to quit, r to retry", m.err))
	}

	// Render based on current tab
	switch m.currentTab {
	case 0: // Queries tab
		if m.loadingQueries {
			return lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderHeader(),
				m.renderTabBar(),
				"Loading queries...",
				m.renderFooter(),
			)
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(),
			m.renderTabBar(),
			m.queryList.View(),
			m.renderFooter(),
		)
	case 1: // Work Items tab
		if m.showDetails {
			return lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderHeader(),
				m.renderTabBar(),
				m.list.View(),
				m.renderDetailsHeader(),
				lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("62")).
					Padding(1).
					Render(m.viewport.View()),
				m.renderFooter(),
			)
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(),
			m.renderTabBar(),
			m.list.View(),
			m.renderFooter(),
		)
	case 2: // Templates tab
		if m.loadingTemplates {
			return lipgloss.JoinVertical(
				lipgloss.Left,
				m.renderHeader(),
				m.renderTabBar(),
				"Loading templates...",
				m.renderFooter(),
			)
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(),
			m.renderTabBar(),
			m.templateList.View(),
			m.renderFooter(),
		)
	case 3: // Pipelines tab
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(),
			m.renderTabBar(),
			"Pipelines tab - Coming soon!",
			m.renderFooter(),
		)
	case 4: // Agents tab
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(),
			m.renderTabBar(),
			"Agents tab - Coming soon!",
			m.renderFooter(),
		)
	}

	return "Unknown tab"
}

func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true).
		Render("Azure Boards Dashboard")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Render(title)
}

func (m Model) renderTabBar() string {
	var tabs []string

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 2)

	activeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true).
		Underline(true).
		Padding(0, 2)

	for i, tab := range m.tabs {
		if i == m.currentTab {
			tabs = append(tabs, activeTabStyle.Render(tab))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(tab))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("  tab: switch")

	return lipgloss.JoinHorizontal(lipgloss.Top, tabBar, help)
}

func (m Model) renderDetailsHeader() string {
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true).
		Render("Work Item Details")

	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(header)
}

func (m Model) renderFooter() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("↑/↓: navigate • enter: details • r: refresh • ?: help • q: quit")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Render(help)
}

func (m Model) formatWorkItemDetails(wi workitemtracking.WorkItem) string {
	var details string

	id := getIntField(&wi, "System.Id")
	title := getStringField(&wi, "System.Title")
	workItemType := getStringField(&wi, "System.WorkItemType")
	state := getStringField(&wi, "System.State")
	assignedTo := getStringField(&wi, "System.AssignedTo")
	description := getStringField(&wi, "System.Description")
	acceptanceCriteria := getStringField(&wi, "Microsoft.VSTS.Common.AcceptanceCriteria")
	createdDate := getStringField(&wi, "System.CreatedDate")
	changedDate := getStringField(&wi, "System.ChangedDate")
	priority := getStringField(&wi, "Microsoft.VSTS.Common.Priority")
	tags := getStringField(&wi, "System.Tags")

	details += lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("#%d - %s\n\n", id, title))
	details += fmt.Sprintf("Type: %s | State: %s | Priority: %s\n\n", workItemType, state, priority)

	if description != "" {
		details += "Description:\n"
		details += description + "\n\n"
	}

	if acceptanceCriteria != "" {
		details += "Acceptance Criteria:\n"
		details += acceptanceCriteria + "\n\n"
	}

	// Format relationships from cache (parent, children, PRs, deployments)
	workItemID := id
	if relInfo, exists := m.relationshipData[workItemID]; exists && relInfo.loaded {
		// Use cached relationship data
		if relInfo.parent != "" {
			details += "Parent:\n"
			details += relInfo.parent + "\n\n"
		}

		if len(relInfo.children) > 0 {
			details += "Children:\n"
			for _, child := range relInfo.children {
				details += child + "\n"
			}
			details += "\n"
		}

		if len(relInfo.prs) > 0 {
			details += "Pull Requests:\n"
			for _, pr := range relInfo.prs {
				details += pr + "\n"
			}
			details += "\n"
		}

		if len(relInfo.deployments) > 0 {
			details += "Deployments:\n"
			for _, deployment := range relInfo.deployments {
				details += deployment + "\n"
			}
			details += "\n"
		}
	} else if wi.Relations != nil && len(*wi.Relations) > 0 {
		// Show loading indicator if there are relations but not loaded yet
		if m.loadingRelations {
			details += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Loading relationships...\n\n")
		}
	}

	if assignedTo != "" {
		details += fmt.Sprintf("Assigned To: %s\n", assignedTo)
	}

	if tags != "" {
		details += fmt.Sprintf("Tags: %s\n", tags)
	}

	details += fmt.Sprintf("\nCreated: %s | Updated: %s\n", createdDate, changedDate)

	return details
}

// extractWorkItemIDFromURL extracts the work item ID from a URL
func extractWorkItemIDFromURL(url string) int {
	// URL format: https://.../workItems/12345
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		idStr := parts[len(parts)-1]
		if id, err := strconv.Atoi(idStr); err == nil {
			return id
		}
	}
	return 0
}

// Helper functions
func getStringField(wi *workitemtracking.WorkItem, fieldName string) string {
	if wi.Fields == nil {
		return ""
	}
	if value, ok := (*wi.Fields)[fieldName]; ok {
		// Handle identity fields (System.AssignedTo, System.CreatedBy, etc.)
		if fieldName == "System.AssignedTo" || fieldName == "System.CreatedBy" || fieldName == "System.ChangedBy" {
			if identityMap, ok := value.(map[string]interface{}); ok {
				if displayName, ok := identityMap["displayName"].(string); ok {
					return displayName
				}
				if uniqueName, ok := identityMap["uniqueName"].(string); ok {
					return uniqueName
				}
			}
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}

func getIntField(wi *workitemtracking.WorkItem, fieldName string) int {
	if wi.Id != nil {
		return *wi.Id
	}
	return 0
}

// cleanAssignedTo extracts just the name or email from the assigned to field
// Input format: "Display Name <email@example.com>" or just "email@example.com"
// Returns: "Display Name" or "email@example.com"
func cleanAssignedTo(assignedTo string) string {
	if assignedTo == "" {
		return ""
	}

	// Check for format: "Name <email>"
	if idx := strings.Index(assignedTo, "<"); idx > 0 {
		// Return just the name part, trimmed
		return strings.TrimSpace(assignedTo[:idx])
	}

	// Otherwise return as-is (likely just email or name)
	return assignedTo
}

// Messages

type workItemsMsg struct {
	items []workitemtracking.WorkItem
}

type queriesMsg struct {
	queries []workitemtracking.QueryHierarchyItem
}

type templatesMsg struct {
	templates []*templates.TemplateNode
}

type relationshipsLoadedMsg struct {
	workItemID int
	relInfo    *relationshipInfo
}

type errMsg struct {
	err error
}

// Commands

func loadRelationships(client *api.Client, wi workitemtracking.WorkItem) tea.Cmd {
	return func() tea.Msg {
		workItemID := 0
		if wi.Id != nil {
			workItemID = *wi.Id
		}

		logger.Printf("Loading relationships for work item #%d", workItemID)

		relInfo := &relationshipInfo{
			loaded: false,
		}

		// Process relationships
		if wi.Relations != nil && len(*wi.Relations) > 0 {
			for _, rel := range *wi.Relations {
				if rel.Rel == nil || rel.Url == nil {
					continue
				}

				relType := *rel.Rel

				// Parent work item
				if relType == "System.LinkTypes.Hierarchy-Reverse" {
					if parentID := extractWorkItemIDFromURL(*rel.Url); parentID > 0 {
						parentWI, err := client.GetWorkItem(parentID)
						if err == nil && parentWI.Fields != nil {
							parentTitle := getStringField(parentWI, "System.Title")
							relInfo.parent = fmt.Sprintf("  #%d - %s", parentID, parentTitle)
						}
					}
				}

				// Child work items
				if relType == "System.LinkTypes.Hierarchy-Forward" {
					if childID := extractWorkItemIDFromURL(*rel.Url); childID > 0 {
						childWI, err := client.GetWorkItem(childID)
						if err == nil && childWI.Fields != nil {
							childTitle := getStringField(childWI, "System.Title")
							childState := getStringField(childWI, "System.State")
							checkbox := "[ ]"
							if childState == "Closed" || childState == "Resolved" {
								checkbox = "[x]"
							}
							relInfo.children = append(relInfo.children, fmt.Sprintf("  %s #%d - %s", checkbox, childID, childTitle))
						}
					}
				}

				// Pull Requests
				if relType == "ArtifactLink" && rel.Attributes != nil {
					if name, ok := (*rel.Attributes)["name"].(string); ok && name == "Pull Request" {
						relInfo.prs = append(relInfo.prs, fmt.Sprintf("  - %s", *rel.Url))
					}
				}

				// Deployments
				if relType == "Hyperlink" && rel.Attributes != nil {
					if comment, ok := (*rel.Attributes)["comment"].(string); ok {
						if strings.Contains(strings.ToLower(comment), "deployment") {
							relInfo.deployments = append(relInfo.deployments, fmt.Sprintf("  - %s", comment))
						}
					}
				}
			}
		}

		relInfo.loaded = true

		return relationshipsLoadedMsg{
			workItemID: workItemID,
			relInfo:    relInfo,
		}
	}
}

func fetchWorkItems(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		// Default query: User Stories assigned to me, excluding closed and removed items
		wiql := `SELECT [System.Id], [System.Title], [System.State], [System.AssignedTo], [System.WorkItemType], [System.Description], [Microsoft.VSTS.Common.AcceptanceCriteria], [System.CreatedDate], [System.ChangedDate], [Microsoft.VSTS.Common.Priority], [System.Tags] FROM WorkItems WHERE [System.AssignedTo] = @me AND [System.WorkItemType] = 'User Story' AND [System.State] <> 'Closed' AND [System.State] <> 'Removed' ORDER BY [System.State] ASC`

		workItemsPtr, err := client.ListWorkItems(wiql, 100)
		if err != nil {
			return errMsg{err: err}
		}

		var workItems []workitemtracking.WorkItem
		if workItemsPtr != nil {
			workItems = *workItemsPtr
		}

		return workItemsMsg{items: workItems}
	}
}

func fetchQueries(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		logger.Println("Fetching saved queries...")

		// Fetch queries with depth 2 to get folder structure
		queriesPtr, err := client.ListQueries("", 2)
		if err != nil {
			logger.Printf("Error fetching queries: %v", err)
			return errMsg{err: err}
		}

		var queries []workitemtracking.QueryHierarchyItem
		if queriesPtr != nil {
			queries = *queriesPtr
		}

		logger.Printf("Loaded %d top-level query items", len(queries))
		return queriesMsg{queries: queries}
	}
}

func executeQuery(client *api.Client, query workitemtracking.QueryHierarchyItem) tea.Cmd {
	return func() tea.Msg {
		queryID := ""
		if query.Id != nil {
			queryID = query.Id.String()
		}

		queryName := ""
		if query.Name != nil {
			queryName = *query.Name
		}

		logger.Printf("Executing query '%s' (ID: %s)", queryName, queryID)

		// Execute the query and get work items
		workItemsPtr, err := client.ExecuteQuery(queryID, 100)
		if err != nil {
			logger.Printf("Error executing query: %v", err)
			return errMsg{err: err}
		}

		var workItems []workitemtracking.WorkItem
		if workItemsPtr != nil {
			workItems = *workItemsPtr
		}

		logger.Printf("Query returned %d work items", len(workItems))
		return workItemsMsg{items: workItems}
	}
}

func fetchTemplates() tea.Cmd {
	return func() tea.Msg {
		logger.Println("Fetching templates...")

		templateNodes, err := templates.ListTree()
		if err != nil {
			logger.Printf("Error fetching templates: %v", err)
			return errMsg{err: err}
		}

		logger.Printf("Loaded %d top-level template items", len(templateNodes))
		return templatesMsg{templates: templateNodes}
	}
}
