package tui

import (
	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/templates"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// NotificationMsg is sent to display a notification
type NotificationMsg struct {
	Message string
	IsError bool
}

// ClearNotificationMsg clears the current notification
type ClearNotificationMsg struct{}

// WorkItemsLoadedMsg is sent when work items are loaded
type WorkItemsLoadedMsg struct {
	WorkItems []workitemtracking.WorkItem
	Error     error
}

// QueriesLoadedMsg is sent when queries are loaded
type QueriesLoadedMsg struct {
	Queries []workitemtracking.QueryHierarchyItem
	Error   error
}

// TemplatesLoadedMsg is sent when templates are loaded
type TemplatesLoadedMsg struct {
	Templates []*templates.TemplateNode
	Error     error
}

// QueryExecutedMsg is sent when a query is executed
type QueryExecutedMsg struct {
	WorkItems []workitemtracking.WorkItem
	Error     error
}

// WorkItemCreatedMsg is sent when a work item is created
type WorkItemCreatedMsg struct {
	WorkItem *workitemtracking.WorkItem
	Error    error
}

// WorkItemUpdatedMsg is sent when a work item is updated
type WorkItemUpdatedMsg struct {
	WorkItem *workitemtracking.WorkItem
	Error    error
}

// WorkItemDeletedMsg is sent when a work item is deleted
type WorkItemDeletedMsg struct {
	ID    int
	Error error
}

// WorkItemDetailsLoadedMsg is sent when work item details (with relationships) are loaded
type WorkItemDetailsLoadedMsg struct {
	WorkItem *workitemtracking.WorkItem
	Error    error
}

// TemplateCopiedMsg is sent when a template is copied
type TemplateCopiedMsg struct {
	OriginalPath string
	NewPath      string
	Error        error
}

// TemplateFolderCreatedMsg is sent when a template folder is created
type TemplateFolderCreatedMsg struct {
	FolderPath string
	Error      error
}

// TemplateDeletedMsg is sent when a template is deleted
type TemplateDeletedMsg struct {
	TemplatePath string
	Error        error
}

// TemplateRenamedMsg is sent when a template is renamed
type TemplateRenamedMsg struct {
	OldPath string
	NewPath string
	Error   error
}

// SwitchToTabMsg is sent to switch to a specific tab
type SwitchToTabMsg struct {
	TabIndex int
}

// ConfirmDeleteWorkItemMsg is sent to request confirmation for deleting a work item
type ConfirmDeleteWorkItemMsg struct {
	WorkItemID int
	Title      string
	ChildIDs   []int
}

// OpenEditorMsg is sent to open an editor for a work item
type OpenEditorMsg struct {
	FilePath   string
	WorkItemID int
	Client     *api.Client
}

// ProcessEditedWorkItemMsg is sent after the editor closes to process changes
type ProcessEditedWorkItemMsg struct {
	FilePath   string
	WorkItemID int
	Client     *api.Client
}

// OpenEditorForTemplateMsg is sent to open an editor for a template
type OpenEditorForTemplateMsg struct {
	FilePath string
}

// ConfirmDeleteTemplateMsg is sent to request confirmation for deleting a template
type ConfirmDeleteTemplateMsg struct {
	Path   string
	Name   string
	IsDir  bool
	Prompt string
}

// RefreshTemplatesMsg is sent to trigger a templates list refresh
type RefreshTemplatesMsg struct{}

// CreateWorkItemFromTemplateMsg is sent to create a work item from a template
type CreateWorkItemFromTemplateMsg struct {
	Template *templates.Template
}
