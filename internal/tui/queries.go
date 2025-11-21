package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// QueriesTab displays saved queries in a tree view
type QueriesTab struct {
	TabBase
	client          *api.Client
	queries         []workitemtracking.QueryHierarchyItem
	list            list.Model
	expandedFolders map[string]bool
	loading         bool
	err             error
}

// NewQueriesTab creates a new queries tab
func NewQueriesTab(client *api.Client, width, height int) *QueriesTab {
	tab := &QueriesTab{
		TabBase:         NewTabBase(width, height),
		client:          client,
		expandedFolders: make(map[string]bool),
		loading:         true,
	}

	// Initialize list with empty delegate for now
	tab.list = list.New([]list.Item{}, queryDelegate{expandedFolders: tab.expandedFolders}, width, tab.ContentHeight())
	tab.list.Title = "Saved Queries"
	tab.list.SetShowStatusBar(true)
	tab.list.SetFilteringEnabled(true)
	tab.list.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color(ColorSecondary)).
		Foreground(lipgloss.Color(ColorYellow)).
		Padding(0, 1)

	return tab
}

// Name returns the tab name
func (t *QueriesTab) Name() string {
	return "Queries"
}

// Init initializes the tab
func (t *QueriesTab) Init(width, height int) tea.Cmd {
	t.SetSize(width, height)
	return t.fetchQueries()
}

// Update handles messages
func (t *QueriesTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case QueriesLoadedMsg:
		t.loading = false
		if msg.Error != nil {
			t.err = msg.Error
			return t, nil
		}
		t.queries = msg.Queries
		t.rebuildList()
		return t, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			return t.handleEnter()
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			t.loading = true
			return t, t.fetchQueries()
		default:
			t.list, cmd = t.list.Update(msg)
			return t, cmd
		}
	}

	t.list, cmd = t.list.Update(msg)
	return t, cmd
}

// View renders the tab
func (t *QueriesTab) View() string {
	if t.loading {
		return RenderLoading("Loading queries...")
	}

	if t.err != nil {
		return RenderErrorWithRetry(t.err)
	}

	return t.list.View()
}

// SetSize updates the tab dimensions
func (t *QueriesTab) SetSize(width, height int) {
	t.TabBase.SetSize(width, height)
	t.list.SetSize(width, t.ContentHeight())
}

// handleEnter toggles folders or executes queries
func (t *QueriesTab) handleEnter() (Tab, tea.Cmd) {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(queryListItem); ok {
		if item.IsFolder {
			// Toggle folder expand/collapse
			t.expandedFolders[item.Path] = !t.expandedFolders[item.Path]
			t.rebuildList()
			return t, nil
		} else {
			// Execute query and switch to Work Items tab
			return t, tea.Batch(
				t.executeQuery(item.query),
				func() tea.Msg {
					return SwitchToTabMsg{TabIndex: 1}
				},
			)
		}
	}
	return t, nil
}

// rebuildList rebuilds the list with current expanded state
func (t *QueriesTab) rebuildList() {
	items := t.flattenQueries(t.queries, 0)
	delegate := queryDelegate{expandedFolders: t.expandedFolders}
	t.list.SetDelegate(delegate)
	t.list.SetItems(items)
}

// flattenQueries recursively flattens the query hierarchy
func (t *QueriesTab) flattenQueries(queries []workitemtracking.QueryHierarchyItem, depth int) []list.Item {
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
			if t.expandedFolders[path] {
				childItems := t.flattenQueries(*q.Children, depth+1)
				items = append(items, childItems...)
			}
		}
	}

	return items
}

// fetchQueries loads queries from the API
func (t *QueriesTab) fetchQueries() tea.Cmd {
	return func() tea.Msg {
		queriesPtr, err := t.client.ListQueries("", 2)
		if err != nil {
			return QueriesLoadedMsg{Error: err}
		}

		var queries []workitemtracking.QueryHierarchyItem
		if queriesPtr != nil {
			queries = *queriesPtr
		}

		return QueriesLoadedMsg{Queries: queries}
	}
}

// executeQuery executes a saved query
func (t *QueriesTab) executeQuery(query workitemtracking.QueryHierarchyItem) tea.Cmd {
	return func() tea.Msg {
		queryID := ""
		if query.Id != nil {
			queryID = query.Id.String()
		}

		workItemsPtr, err := t.client.ExecuteQuery(queryID, 100)
		if err != nil {
			return QueryExecutedMsg{Error: err}
		}

		var workItems []workitemtracking.WorkItem
		if workItemsPtr != nil {
			workItems = *workItemsPtr
		}

		return QueryExecutedMsg{WorkItems: workItems}
	}
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

	indent := strings.Repeat("  ", queryItem.Depth)
	icon := ""
	var nameStyle lipgloss.Style

	if queryItem.IsFolder {
		// Check if folder is expanded
		expanded := d.expandedFolders[queryItem.Path]
		if expanded {
			icon = "▼ "
		} else {
			icon = "▶ "
		}
		nameStyle = FolderStyle
	} else {
		icon = "  🔍 "
		nameStyle = FileStyle
	}

	name := queryItem.Name
	if len(name) > 60 {
		name = name[:57] + "..."
	}

	var output string
	if index == m.Index() {
		output = SelectedStyle.Render(fmt.Sprintf("> %s%s%s", indent, icon, name))
	} else {
		output = nameStyle.Render(fmt.Sprintf("  %s%s%s", indent, icon, name))
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

// GetHelpEntries returns the list of available actions for the Queries tab
func (t *QueriesTab) GetHelpEntries() []HelpEntry {
	return []HelpEntry{
		{Action: "execute", Description: "Execute query or expand folder"},
		{Action: "expand_all", Description: "Expand all folders"},
		{Action: "collapse_all", Description: "Collapse all folders"},
		{Action: "refresh", Description: "Refresh queries list"},
	}
}
