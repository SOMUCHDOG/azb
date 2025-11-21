package tui

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// WorkItemsTab displays and manages work items
type WorkItemsTab struct {
	TabBase
	client           *api.Client
	workItems        []workitemtracking.WorkItem
	workItemCache    map[int]*workitemtracking.WorkItem
	relationshipData map[int]*relationshipInfo
	list             list.Model
	viewport         viewport.Model
	selectedItem     *workitemtracking.WorkItem
	showDetails      bool
	loading          bool
	loadingRelations bool
	initialized      bool
	err              error
}

// relationshipInfo stores formatted relationship data for a work item
type relationshipInfo struct {
	parent      string
	children    []string
	prs         []string
	deployments []string
	loaded      bool
}

// NewWorkItemsTab creates a new work items tab
func NewWorkItemsTab(client *api.Client, width, height int) *WorkItemsTab {
	tab := &WorkItemsTab{
		TabBase:          NewTabBase(width, height),
		client:           client,
		workItemCache:    make(map[int]*workitemtracking.WorkItem),
		relationshipData: make(map[int]*relationshipInfo),
		loading:          false, // Don't load until properly initialized
	}

	// Initialize list
	tab.list = list.New([]list.Item{}, workItemDelegate{}, width, tab.ContentHeight())
	tab.list.Title = "Work Items"
	tab.list.SetShowStatusBar(true)
	tab.list.SetFilteringEnabled(true)
	tab.list.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color(ColorSecondary)).
		Foreground(lipgloss.Color(ColorYellow)).
		Padding(0, 1)

	// Initialize viewport
	tab.viewport = viewport.New(width-4, tab.ContentHeight()/2-4)

	return tab
}

// Name returns the tab name
func (t *WorkItemsTab) Name() string {
	return "Work Items"
}

// Init initializes the tab
func (t *WorkItemsTab) Init(width, height int) tea.Cmd {
	t.SetSize(width, height)
	// Only fetch if we have valid dimensions
	if width > 0 && height > 0 {
		t.loading = true
		return t.fetchWorkItems()
	}
	return nil
}

// Update handles messages
func (t *WorkItemsTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Trigger initial fetch when we get proper dimensions for the first time
		if !t.initialized && !t.loading && msg.Width > 0 && msg.Height > 0 {
			logger.Printf("WorkItemsTab: Triggering initial fetch (dimensions: %dx%d)", msg.Width, msg.Height)
			t.initialized = true
			t.loading = true
			return t, t.fetchWorkItems()
		}
		logger.Printf("WorkItemsTab: WindowSizeMsg received (initialized=%v, loading=%v, %dx%d)",
			t.initialized, t.loading, msg.Width, msg.Height)

	case WorkItemsLoadedMsg:
		logger.Printf("WorkItemsTab: Received WorkItemsLoadedMsg with %d items (error: %v)", len(msg.WorkItems), msg.Error)
		t.loading = false
		if msg.Error != nil {
			t.err = msg.Error
			return t, nil
		}
		t.workItems = msg.WorkItems
		t.rebuildList()
		return t, nil

	case QueryExecutedMsg:
		// Handle query results
		t.loading = false
		if msg.Error != nil {
			return t, func() tea.Msg {
				return NotificationMsg{Message: fmt.Sprintf("Query failed: %v", msg.Error), IsError: true}
			}
		}
		t.workItems = msg.WorkItems
		t.rebuildList()
		return t, func() tea.Msg {
			return NotificationMsg{Message: fmt.Sprintf("Loaded %d work items", len(msg.WorkItems)), IsError: false}
		}

	case WorkItemDeletedMsg:
		if msg.Error != nil {
			return t, func() tea.Msg {
				return NotificationMsg{Message: fmt.Sprintf("Delete failed: %v", msg.Error), IsError: true}
			}
		}
		// Remove from list
		t.removeWorkItem(msg.ID)
		t.rebuildList()
		return t, func() tea.Msg {
			return NotificationMsg{Message: fmt.Sprintf("Deleted work item %d", msg.ID), IsError: false}
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Toggle details view
			t.showDetails = !t.showDetails
			if t.showDetails && len(t.list.Items()) > 0 {
				selectedItem := t.list.SelectedItem()
				if item, ok := selectedItem.(workItemItem); ok {
					t.selectedItem = &item.workItem
					t.viewport.SetContent(t.formatWorkItemDetails(item.workItem))
				}
			}
			t.updateSizes()
			return t, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			t.loading = true
			t.workItemCache = make(map[int]*workitemtracking.WorkItem)
			t.relationshipData = make(map[int]*relationshipInfo)
			return t, t.fetchWorkItems()

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if t.showDetails {
				t.showDetails = false
				t.updateSizes()
				return t, nil
			}
		}
	}

	if t.showDetails {
		t.viewport, cmd = t.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	t.list, cmd = t.list.Update(msg)
	cmds = append(cmds, cmd)

	return t, tea.Batch(cmds...)
}

// View renders the tab
func (t *WorkItemsTab) View() string {
	if t.loading {
		return RenderLoading("Loading work items...")
	}

	if t.err != nil {
		return RenderErrorWithRetry(t.err)
	}

	if t.showDetails {
		detailsHeader := TitleStyle.Render("Work Item Details")
		detailsView := BoxStyle.Render(t.viewport.View())
		return lipgloss.JoinVertical(lipgloss.Left, t.list.View(), detailsHeader, detailsView)
	}

	return t.list.View()
}

// SetSize updates the tab dimensions
func (t *WorkItemsTab) SetSize(width, height int) {
	t.TabBase.SetSize(width, height)
	t.updateSizes()
}

// updateSizes updates list and viewport sizes based on view mode
func (t *WorkItemsTab) updateSizes() {
	if t.showDetails {
		listHeight := t.ContentHeight() / 2
		detailsHeight := t.ContentHeight() - listHeight
		t.list.SetSize(t.Width(), listHeight)
		t.viewport.Width = t.Width() - 4
		t.viewport.Height = detailsHeight - 4
	} else {
		t.list.SetSize(t.Width(), t.ContentHeight())
	}
}

// rebuildList rebuilds the list with current work items
func (t *WorkItemsTab) rebuildList() {
	items := make([]list.Item, 0, len(t.workItems))
	for _, wi := range t.workItems {
		id := 0
		if wi.Id != nil {
			id = *wi.Id
		}

		items = append(items, workItemItem{
			ID:         id,
			Title:      getStringField(&wi, "System.Title"),
			State:      getStringField(&wi, "System.State"),
			AssignedTo: cleanAssignedTo(getStringField(&wi, "System.AssignedTo")),
			workItem:   wi,
		})
	}
	t.list.SetItems(items)
}

// removeWorkItem removes a work item from the list
func (t *WorkItemsTab) removeWorkItem(id int) {
	for i, wi := range t.workItems {
		if wi.Id != nil && *wi.Id == id {
			t.workItems = append(t.workItems[:i], t.workItems[i+1:]...)
			break
		}
	}
}

// fetchWorkItems loads work items from the API
func (t *WorkItemsTab) fetchWorkItems() tea.Cmd {
	return func() tea.Msg {
		logger.Printf("WorkItemsTab: Starting fetchWorkItems()")
		// Default query: User Stories assigned to me, excluding closed and removed items
		wiql := "SELECT [System.Id], [System.Title], [System.State], [System.AssignedTo], [System.WorkItemType], " +
			"[System.Description], [Microsoft.VSTS.Common.AcceptanceCriteria], [System.CreatedDate], " +
			"[System.ChangedDate], [Microsoft.VSTS.Common.Priority], [System.Tags] " +
			"FROM WorkItems WHERE [System.AssignedTo] = @me AND [System.WorkItemType] = 'User Story' " +
			"AND [System.State] <> 'Closed' AND [System.State] <> 'Removed' " +
			"ORDER BY [System.State] ASC"

		logger.Printf("WorkItemsTab: Executing WIQL query")
		workItemsPtr, err := t.client.ListWorkItems(wiql, 100)
		if err != nil {
			logger.Printf("WorkItemsTab: Error fetching work items: %v", err)
			return WorkItemsLoadedMsg{Error: err}
		}

		var workItems []workitemtracking.WorkItem
		if workItemsPtr != nil {
			workItems = *workItemsPtr
		}

		logger.Printf("WorkItemsTab: Successfully fetched %d work items", len(workItems))
		return WorkItemsLoadedMsg{WorkItems: workItems}
	}
}

// formatWorkItemDetails formats a work item for display
func (t *WorkItemsTab) formatWorkItemDetails(wi workitemtracking.WorkItem) string {
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

	// Relationships (simplified for now)
	if wi.Relations != nil && len(*wi.Relations) > 0 {
		details += fmt.Sprintf("Relations: %d\n\n", len(*wi.Relations))
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

// Helper functions
func getStringField(wi *workitemtracking.WorkItem, fieldName string) string {
	if wi.Fields == nil {
		return ""
	}
	if value, ok := (*wi.Fields)[fieldName]; ok {
		// Handle identity fields
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

func cleanAssignedTo(assignedTo string) string {
	if assignedTo == "" {
		return ""
	}
	if idx := strings.Index(assignedTo, "<"); idx > 0 {
		return strings.TrimSpace(assignedTo[:idx])
	}
	return assignedTo
}

func extractWorkItemIDFromURL(url string) int {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		idStr := parts[len(parts)-1]
		if id, err := strconv.Atoi(idStr); err == nil {
			return id
		}
	}
	return 0
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

	id := fmt.Sprintf("%-8d", workItem.ID)
	titleStr := workItem.Title
	if len(titleStr) > 40 {
		titleStr = titleStr[:37] + "..."
	}
	titleStr = fmt.Sprintf("%-40s", titleStr)

	state := workItem.State
	var stateStyle lipgloss.Style
	switch state {
	case "Active":
		stateStyle = StateActiveStyle
	case "New":
		stateStyle = StateNewStyle
	case "Closed", "Resolved":
		stateStyle = StateClosedStyle
	case "Blocked":
		stateStyle = StateBlockedStyle
	default:
		stateStyle = MutedStyle
	}
	stateStr := stateStyle.Render(fmt.Sprintf("%-12s", state))

	assignee := workItem.AssignedTo
	if len(assignee) > 20 {
		assignee = assignee[:17] + "..."
	}
	assigneeStr := fmt.Sprintf("%-20s", assignee)

	var output string
	if index == m.Index() {
		output = SelectedStyle.Render(fmt.Sprintf("> %s │ %s │ %s │ %s", id, titleStr, stateStr, assigneeStr))
	} else {
		output = NormalStyle.Render(fmt.Sprintf("  %s │ %s │ %s │ %s", id, titleStr, stateStr, assigneeStr))
	}

	fmt.Fprint(w, output)
}

// workItemItem wraps a work item for the list
type workItemItem struct {
	ID         int
	Title      string
	State      string
	AssignedTo string
	workItem   workitemtracking.WorkItem
}

func (i workItemItem) FilterValue() string { return i.Title }
