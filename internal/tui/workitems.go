package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
	"gopkg.in/yaml.v3"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/templates"
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
	//nolint:unused // Reserved for future feature: async relationship loading
	loadingRelations bool
	initialized      bool
	err              error
}

// relationshipInfo stores formatted relationship data for a work item
type relationshipInfo struct {
	//nolint:unused // Reserved for future feature: display parent work item
	parent string
	//nolint:unused // Reserved for future feature: display child work items
	children []string
	//nolint:unused // Reserved for future feature: display related pull requests
	prs []string
	//nolint:unused // Reserved for future feature: display related deployments
	deployments []string
	//nolint:unused // Reserved for future feature: track relationship loading state
	loaded bool
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

	case WorkItemUpdatedMsg:
		if msg.Error != nil {
			return t, func() tea.Msg {
				return NotificationMsg{Message: fmt.Sprintf("Update failed: %v", msg.Error), IsError: true}
			}
		}
		// Trigger a refresh to get the latest work item data
		t.loading = true
		return t, tea.Batch(
			t.fetchWorkItems(),
			func() tea.Msg {
				return NotificationMsg{
					Message: fmt.Sprintf("Work item #%d updated successfully", *msg.WorkItem.Id),
					IsError: false,
				}
			},
		)

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

	// Store previous selection to detect changes
	var previousID int
	if t.selectedItem != nil && t.selectedItem.Id != nil {
		previousID = *t.selectedItem.Id
	}

	t.list, cmd = t.list.Update(msg)
	cmds = append(cmds, cmd)

	// If details are showing, check if selection changed and update viewport
	if t.showDetails && len(t.list.Items()) > 0 {
		selectedItem := t.list.SelectedItem()
		if item, ok := selectedItem.(workItemItem); ok {
			currentID := item.ID
			// Only update if selection actually changed
			if currentID != previousID {
				t.selectedItem = &item.workItem
				t.viewport.SetContent(t.formatWorkItemDetails(item.workItem))
				t.viewport.GotoTop() // Reset scroll position for new item
			}
		}
	}

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
		listView := t.list.View()
		detailsPane := RenderDetailsPane("Work Item Details", t.viewport.View())
		combined := lipgloss.JoinVertical(lipgloss.Left, listView, detailsPane)

		// Ensure total height doesn't exceed ContentHeight()
		maxHeight := t.ContentHeight()
		return lipgloss.NewStyle().MaxHeight(maxHeight).Render(combined)
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
		// Account for BoxStyle border (2) + padding (2) + details header (1) = 5 lines
		t.viewport.Height = detailsHeight - 5
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

	// Relationships - display detailed relationship information
	if wi.Relations != nil && len(*wi.Relations) > 0 {
		details += fmt.Sprintf("Relations (%d):\n", len(*wi.Relations))

		// Group relationships by type
		var parents []string
		var children []string
		var others []string

		for _, rel := range *wi.Relations {
			if rel.Rel == nil || rel.Url == nil {
				continue
			}

			relType := *rel.Rel
			relID := extractWorkItemIDFromURL(*rel.Url)

			// Fetch work item title if we have an ID
			var relTitle string
			if relID > 0 {
				// Check cache first
				if cachedWI, ok := t.workItemCache[relID]; ok {
					relTitle = getStringField(cachedWI, "System.Title")
				} else {
					// Fetch from API
					relWI, err := t.client.GetWorkItem(relID)
					if err == nil && relWI != nil {
						relTitle = getStringField(relWI, "System.Title")
						t.workItemCache[relID] = relWI
					}
				}
			}

			switch relType {
			case "System.LinkTypes.Hierarchy-Reverse":
				if relTitle != "" {
					parents = append(parents, fmt.Sprintf("  Parent: #%d - %s", relID, relTitle))
				} else {
					parents = append(parents, fmt.Sprintf("  Parent: #%d", relID))
				}
			case "System.LinkTypes.Hierarchy-Forward":
				if relTitle != "" {
					children = append(children, fmt.Sprintf("  Child: #%d - %s", relID, relTitle))
				} else {
					children = append(children, fmt.Sprintf("  Child: #%d", relID))
				}
			default:
				// Other relationship types (PRs, related work items, etc.)
				relTypeName := relType
				if idx := strings.LastIndex(relType, "-"); idx > 0 {
					relTypeName = relType[idx+1:]
				}
				if relID > 0 {
					if relTitle != "" {
						others = append(others, fmt.Sprintf("  %s: #%d - %s", relTypeName, relID, relTitle))
					} else {
						others = append(others, fmt.Sprintf("  %s: #%d", relTypeName, relID))
					}
				} else {
					others = append(others, fmt.Sprintf("  %s: %s", relTypeName, *rel.Url))
				}
			}
		}

		// Display grouped relationships
		for _, p := range parents {
			details += p + "\n"
		}
		for _, c := range children {
			details += c + "\n"
		}
		for _, o := range others {
			details += o + "\n"
		}
		details += "\n"
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

// GetHelpEntries returns the list of available actions for the Work Items tab
func (t *WorkItemsTab) GetHelpEntries() []HelpEntry {
	return []HelpEntry{
		{Action: "details", Description: "Toggle details view"},
		{Action: "download", Description: "Download as YAML template"},
		{Action: "edit", Description: "Edit work item in $EDITOR"},
		{Action: "delete", Description: "Delete work item and children"},
		{Action: "create", Description: "Create new work item"},
		{Action: "change_state", Description: "Change work item state"},
		{Action: "assign", Description: "Assign to user"},
		{Action: "add_tags", Description: "Add tags"},
		{Action: "refresh", Description: "Refresh work items list"},
	}
}

// handleDownloadAction initiates the download work item action
func (t *WorkItemsTab) handleDownloadAction() tea.Cmd {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		return downloadWorkItem(t.client, item.workItem)
	}
	return nil
}

// downloadWorkItem downloads a work item as a YAML template
func downloadWorkItem(client *api.Client, wi workitemtracking.WorkItem) tea.Cmd {
	return func() tea.Msg {
		// Get work item ID
		id := 0
		if wi.Id != nil {
			id = *wi.Id
		}

		logger.Printf("Downloading work item #%d as template", id)

		// Fetch full work item details (with relations)
		fullWI, err := client.GetWorkItem(id)
		if err != nil {
			logger.Printf("Failed to fetch work item #%d: %v", id, err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to fetch work item #%d: %v", id, err),
				IsError: true,
			}
		}

		// Convert to template format
		template := convertWorkItemToTemplate(client, fullWI)

		// Serialize to YAML
		yamlData, err := yaml.Marshal(template)
		if err != nil {
			logger.Printf("Failed to serialize work item: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to serialize work item: %v", err),
				IsError: true,
			}
		}

		// Save to templates directory with name format: workitem-{id}-{sanitized-title}.yaml
		homeDir, _ := os.UserHomeDir()
		templatesDir := filepath.Join(homeDir, ".azure-boards-cli", "templates")
		os.MkdirAll(templatesDir, 0755)

		title := getStringField(fullWI, "System.Title")
		sanitized := sanitizeFilename(title)
		filename := fmt.Sprintf("workitem-%d-%s.yaml", id, sanitized)
		filePath := filepath.Join(templatesDir, filename)

		if err := os.WriteFile(filePath, yamlData, 0600); err != nil {
			logger.Printf("Failed to save template: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to save template: %v", err),
				IsError: true,
			}
		}

		logger.Printf("Downloaded work item #%d as template: %s", id, filename)
		return NotificationMsg{
			Message: fmt.Sprintf("Downloaded work item #%d as template: %s", id, filename),
			IsError: false,
		}
	}
}

// convertWorkItemToTemplate converts a work item to a template
func convertWorkItemToTemplate(client *api.Client, wi *workitemtracking.WorkItem) *templates.Template {
	template := &templates.Template{
		Name:        getStringField(wi, "System.Title"),
		Type:        getStringField(wi, "System.WorkItemType"),
		Description: fmt.Sprintf("Template created from work item #%d", *wi.Id),
		Fields:      make(map[string]interface{}),
	}

	// Copy relevant fields
	relevantFields := []string{
		"System.Title",
		"System.Description",
		"System.State",
		"System.Tags",
		"Microsoft.VSTS.Common.Priority",
		"Microsoft.VSTS.Common.AcceptanceCriteria",
		"System.AreaPath",
		"System.IterationPath",
		"Custom.ApplicationName",
	}

	if wi.Fields != nil {
		for _, fieldName := range relevantFields {
			if value, ok := (*wi.Fields)[fieldName]; ok {
				template.Fields[fieldName] = value
			}
		}
	}

	// Handle relationships (children and parent)
	if wi.Relations != nil {
		for _, rel := range *wi.Relations {
			if rel.Rel != nil && rel.Url != nil {
				relType := *rel.Rel

				// Check for parent relationship
				if relType == "System.LinkTypes.Hierarchy-Reverse" {
					parentID := extractWorkItemIDFromURL(*rel.Url)
					if parentID > 0 {
						if template.Relations == nil {
							template.Relations = &templates.Relations{}
						}
						template.Relations.ParentID = parentID
					}
				}

				// Check for child relationship
				if relType == "System.LinkTypes.Hierarchy-Forward" {
					// Extract child work item ID from relationship URL
					childID := extractWorkItemIDFromURL(*rel.Url)
					if childID > 0 {
						if template.Relations == nil {
							template.Relations = &templates.Relations{}
						}

						// Fetch child work item details to get title, type, and description
						childTitle := fmt.Sprintf("Child Work Item #%d", childID)
						childType := "Task"
						childDescription := ""
						childWI, err := client.GetWorkItem(childID)
						if err == nil && childWI != nil {
							childTitle = getStringField(childWI, "System.Title")
							childDescription = getStringField(childWI, "System.Description")
							childWorkItemType := getStringField(childWI, "System.WorkItemType")
							if childWorkItemType != "" {
								childType = childWorkItemType
							}
						}

						// Create child entry with actual title, type, and description
						child := templates.ChildWorkItem{
							Title:       childTitle,
							Type:        childType,
							Description: childDescription,
							Fields: map[string]interface{}{
								"System.Id": childID,
							},
						}
						template.Relations.Children = append(template.Relations.Children, child)
					}
				}
			}
		}
	}

	return template
}

// sanitizeFilename removes invalid filename characters
func sanitizeFilename(s string) string {
	// Remove invalid filename characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := s
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "-")
	}
	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}
	return strings.ToLower(strings.TrimSpace(result))
}

// handleDeleteAction initiates the delete work item action
func (t *WorkItemsTab) handleDeleteAction() tea.Cmd {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		return fetchWorkItemForDelete(t.client, item.workItem)
	}
	return nil
}

// fetchWorkItemForDelete fetches work item with relationships for deletion
func fetchWorkItemForDelete(client *api.Client, wi workitemtracking.WorkItem) tea.Cmd {
	return func() tea.Msg {
		id := 0
		if wi.Id != nil {
			id = *wi.Id
		}

		logger.Printf("Fetching work item #%d for deletion", id)

		// Fetch full work item with relationships
		fullWI, err := client.GetWorkItem(id)
		if err != nil {
			logger.Printf("Failed to fetch work item details: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to fetch work item details: %v", err),
				IsError: true,
			}
		}

		// Find all child work items
		childIDs := []int{}
		if fullWI.Relations != nil {
			for _, rel := range *fullWI.Relations {
				if rel.Rel != nil && *rel.Rel == "System.LinkTypes.Hierarchy-Forward" {
					// This is a child - extract ID from URL
					if rel.Url != nil {
						childID := extractWorkItemIDFromURL(*rel.Url)
						if childID > 0 {
							childIDs = append(childIDs, childID)
						}
					}
				}
			}
		}

		title := getStringField(fullWI, "System.Title")

		logger.Printf("Work item #%d has %d child tasks", id, len(childIDs))

		return ConfirmDeleteWorkItemMsg{
			WorkItemID: id,
			Title:      title,
			ChildIDs:   childIDs,
		}
	}
}

// deleteWorkItemWithChildren deletes a work item and all its children
func deleteWorkItemWithChildren(client *api.Client, parentID int, childIDs []int) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Deleting work item #%d with %d children", parentID, len(childIDs))

		// Delete children first (in reverse order to avoid dependency issues)
		for i := len(childIDs) - 1; i >= 0; i-- {
			childID := childIDs[i]
			if err := client.DeleteWorkItem(childID); err != nil {
				logger.Printf("Failed to delete child work item #%d: %v", childID, err)
				return NotificationMsg{
					Message: fmt.Sprintf("Failed to delete child work item #%d: %v", childID, err),
					IsError: true,
				}
			}
			logger.Printf("Deleted child work item #%d", childID)
		}

		// Delete parent
		if err := client.DeleteWorkItem(parentID); err != nil {
			logger.Printf("Failed to delete work item #%d: %v", parentID, err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to delete work item #%d: %v", parentID, err),
				IsError: true,
			}
		}

		logger.Printf("Deleted parent work item #%d", parentID)
		if len(childIDs) > 0 {
			logger.Printf("Also deleted %d child task(s)", len(childIDs))
		}

		return WorkItemDeletedMsg{
			ID:    parentID,
			Error: nil,
		}
	}
}

// handleEditAction handles the edit work item action (e key)
func (t *WorkItemsTab) handleEditAction() tea.Cmd {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		return prepareEditWorkItem(t.client, item.workItem)
	}
	return nil
}

// prepareEditWorkItem fetches full work item, converts to YAML, and creates temp file
func prepareEditWorkItem(client *api.Client, wi workitemtracking.WorkItem) tea.Cmd {
	return func() tea.Msg {
		id := *wi.Id
		logger.Printf("Preparing to edit work item #%d", id)

		// Fetch full work item with all fields
		fullWI, err := client.GetWorkItem(id)
		if err != nil {
			logger.Printf("Failed to fetch work item #%d: %v", id, err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to fetch work item #%d: %v", id, err),
				IsError: true,
			}
		}

		// Convert to template format for editing
		template := convertWorkItemToTemplate(client, fullWI)

		// Serialize to YAML
		yamlData, err := yaml.Marshal(template)
		if err != nil {
			logger.Printf("Failed to serialize work item #%d to YAML: %v", id, err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to serialize work item: %v", err),
				IsError: true,
			}
		}

		// Create temporary file
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logger.Printf("Failed to get home directory: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to get home directory: %v", err),
				IsError: true,
			}
		}

		tempDir := filepath.Join(homeDir, ".azure-boards-cli", "tmp")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			logger.Printf("Failed to create temp directory: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to create temp directory: %v", err),
				IsError: true,
			}
		}

		tempFile := filepath.Join(tempDir, fmt.Sprintf("edit-workitem-%d.yaml", id))
		if err := os.WriteFile(tempFile, yamlData, 0600); err != nil {
			logger.Printf("Failed to write temp file: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to write temp file: %v", err),
				IsError: true,
			}
		}

		logger.Printf("Created temp file for editing: %s", tempFile)

		return OpenEditorMsg{
			FilePath:   tempFile,
			WorkItemID: id,
			Client:     client,
		}
	}
}

// processEditedWorkItem reads the edited YAML and updates the work item
func processEditedWorkItem(filePath string, workItemID int, client *api.Client) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Processing edited work item #%d from %s", workItemID, filePath)

		// Read edited YAML
		yamlData, err := os.ReadFile(filePath)
		if err != nil {
			logger.Printf("Failed to read edited file: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to read edited file: %v", err),
				IsError: true,
			}
		}

		// Parse YAML
		var template templates.Template
		if err := yaml.Unmarshal(yamlData, &template); err != nil {
			logger.Printf("Failed to parse edited YAML: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to parse edited YAML: %v", err),
				IsError: true,
			}
		}

		// Build update document from template
		updateFields := buildUpdateDocument(&template)

		// Update work item
		_, err = client.UpdateWorkItem(workItemID, updateFields)
		if err != nil {
			logger.Printf("Failed to update work item #%d: %v", workItemID, err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to update work item #%d: %v", workItemID, err),
				IsError: true,
			}
		}

		logger.Printf("Successfully updated work item #%d", workItemID)

		// Clean up temp file
		os.Remove(filePath)

		return NotificationMsg{
			Message: fmt.Sprintf("Successfully updated work item #%d", workItemID),
			IsError: false,
		}
	}
}

// buildUpdateDocument converts template fields to API update format
func buildUpdateDocument(template *templates.Template) map[string]interface{} {
	fields := make(map[string]interface{})

	// Copy all fields from template
	for fieldName, value := range template.Fields {
		fields[fieldName] = value
	}

	return fields
}

// executeCreateWorkItemFromTemplate creates a work item from a template
func executeCreateWorkItemFromTemplate(client *api.Client, template *templates.Template) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Executing create work item from template: %s", template.Name)

		// Build fields map from template
		fields := make(map[string]interface{})
		for fieldName, value := range template.Fields {
			fields[fieldName] = value
		}

		// Create parent work item
		parentID := 0
		if template.Relations != nil && template.Relations.ParentID > 0 {
			parentID = template.Relations.ParentID
		}

		workItem, err := client.CreateWorkItem(template.Type, fields, parentID)
		if err != nil {
			logger.Printf("Failed to create work item from template: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to create work item: %v", err),
				IsError: true,
			}
		}

		workItemID := *workItem.Id
		logger.Printf("Created work item #%d from template", workItemID)

		// Create child work items if specified
		childCount := 0
		if template.Relations != nil && len(template.Relations.Children) > 0 {
			for _, child := range template.Relations.Children {
				childFields := make(map[string]interface{})
				childFields["System.Title"] = child.Title
				if child.Description != "" {
					childFields["System.Description"] = child.Description
				}
				if child.AssignedTo != "" {
					childFields["System.AssignedTo"] = child.AssignedTo
				}

				// Add any custom fields from child
				for fieldName, value := range child.Fields {
					childFields[fieldName] = value
				}

				// Determine child type
				childType := child.Type
				if childType == "" {
					childType = "Task"
				}

				// Create child work item with parent relationship
				_, err := client.CreateWorkItem(childType, childFields, workItemID)
				if err != nil {
					logger.Printf("Failed to create child work item: %v", err)
					// Continue creating other children even if one fails
					continue
				}
				childCount++
			}
			logger.Printf("Created %d child work items", childCount)
		}

		return WorkItemCreatedMsg{
			WorkItem: workItem,
			Error:    nil,
		}
	}
}

// handleChangeStateAction fetches valid states and shows a selection dialog
func (t *WorkItemsTab) handleChangeStateAction() *SelectionDialog {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		workItemID := *item.workItem.Id
		workItemType := getStringField(&item.workItem, "System.WorkItemType")

		// Fetch valid states for this work item type
		states, err := t.client.GetWorkItemStates(workItemType)
		if err != nil {
			logger.Printf("Failed to fetch states for work item type '%s': %v", workItemType, err)
			// Return nil so no dialog is shown
			return nil
		}

		if len(states) == 0 {
			logger.Printf("No states found for work item type '%s'", workItemType)
			return nil
		}

		// Create and show selection dialog
		dialog := NewSelectionDialog()
		dialog.Show(
			fmt.Sprintf("Change State for Work Item #%d", workItemID),
			states,
			"change_state",
			workItemID,
		)
		logger.Printf("Showing state selection dialog for work item #%d (%d states)", workItemID, len(states))
		return dialog
	}
	return nil
}

// handleAssignAction shows an input prompt for assigning a work item
func (t *WorkItemsTab) handleAssignAction() *InputPrompt {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		workItemID := *item.workItem.Id

		// Create and show input prompt
		prompt := NewInputPrompt()
		prompt.Show(
			fmt.Sprintf("Assign Work Item #%d", workItemID),
			"Enter assignee email or display name",
			"assign_work_item",
			workItemID,
		)
		logger.Printf("Showing assign input prompt for work item #%d", workItemID)
		return prompt
	}
	return nil
}

// handleAddTagsAction shows an input prompt for adding tags to a work item
func (t *WorkItemsTab) handleAddTagsAction() *InputPrompt {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		workItemID := *item.workItem.Id

		// Create and show input prompt
		prompt := NewInputPrompt()
		prompt.Show(
			fmt.Sprintf("Add Tags to Work Item #%d", workItemID),
			"Enter tags separated by commas",
			"add_tags",
			workItemID,
		)
		logger.Printf("Showing add tags input prompt for work item #%d", workItemID)
		return prompt
	}
	return nil
}

// changeWorkItemState changes the state of a work item
func changeWorkItemState(client *api.Client, workItemID int, newState string) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Changing state of work item #%d to '%s'", workItemID, newState)

		// Update the work item state
		fields := map[string]interface{}{
			"System.State": newState,
		}

		workItem, err := client.UpdateWorkItem(workItemID, fields)
		if err != nil {
			logger.Printf("Failed to change state of work item #%d: %v", workItemID, err)
			return WorkItemUpdatedMsg{
				WorkItem: nil,
				Error:    err,
			}
		}

		logger.Printf("Successfully changed state of work item #%d to '%s'", workItemID, newState)
		return WorkItemUpdatedMsg{
			WorkItem: workItem,
			Error:    nil,
		}
	}
}

// assignWorkItem assigns a work item to a user
func assignWorkItem(client *api.Client, workItemID int, assignee string) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Assigning work item #%d to '%s'", workItemID, assignee)

		// Update the work item assignee
		fields := map[string]interface{}{
			"System.AssignedTo": assignee,
		}

		workItem, err := client.UpdateWorkItem(workItemID, fields)
		if err != nil {
			logger.Printf("Failed to assign work item #%d: %v", workItemID, err)
			return WorkItemUpdatedMsg{
				WorkItem: nil,
				Error:    err,
			}
		}

		logger.Printf("Successfully assigned work item #%d to '%s'", workItemID, assignee)
		return WorkItemUpdatedMsg{
			WorkItem: workItem,
			Error:    nil,
		}
	}
}

// addWorkItemTags adds tags to a work item
func addWorkItemTags(client *api.Client, workItemID int, tagsInput string) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Adding tags '%s' to work item #%d", tagsInput, workItemID)

		// Fetch the current work item to get existing tags
		workItem, err := client.GetWorkItem(workItemID)
		if err != nil {
			logger.Printf("Failed to fetch work item #%d: %v", workItemID, err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to fetch work item: %v", err),
				IsError: true,
			}
		}

		// Get existing tags
		var existingTags []string
		if workItem.Fields != nil {
			if tagsValue, ok := (*workItem.Fields)["System.Tags"]; ok {
				if tagsStr, ok := tagsValue.(string); ok && tagsStr != "" {
					existingTags = strings.Split(tagsStr, ";")
					// Trim spaces
					for i := range existingTags {
						existingTags[i] = strings.TrimSpace(existingTags[i])
					}
				}
			}
		}

		// Parse new tags from input (comma-separated)
		newTags := strings.Split(tagsInput, ",")
		for i := range newTags {
			newTags[i] = strings.TrimSpace(newTags[i])
		}

		// Merge tags, avoiding duplicates
		tagSet := make(map[string]bool)
		for _, tag := range existingTags {
			if tag != "" {
				tagSet[tag] = true
			}
		}
		for _, tag := range newTags {
			if tag != "" {
				tagSet[tag] = true
			}
		}

		// Convert back to slice
		var allTags []string
		for tag := range tagSet {
			allTags = append(allTags, tag)
		}

		// Join with semicolons (Azure DevOps format)
		tagsStr := strings.Join(allTags, ";")

		// Update the work item tags
		fields := map[string]interface{}{
			"System.Tags": tagsStr,
		}

		updatedWorkItem, err := client.UpdateWorkItem(workItemID, fields)
		if err != nil {
			logger.Printf("Failed to add tags to work item #%d: %v", workItemID, err)
			return WorkItemUpdatedMsg{
				WorkItem: nil,
				Error:    err,
			}
		}

		logger.Printf("Successfully added tags to work item #%d", workItemID)
		return WorkItemUpdatedMsg{
			WorkItem: updatedWorkItem,
			Error:    nil,
		}
	}
}
