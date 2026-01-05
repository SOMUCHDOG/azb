# TUI Controller Pattern Architecture

## 1. Overview & Philosophy

### Goals
The Controller Pattern provides a clean, extensible architecture for managing user actions in the TUI, separating concerns between:
- **Views** (Tabs): Render state and handle display logic
- **Controllers**: Manage action lifecycle, keybinds, and command execution
- **Configuration**: User-customizable keybindings and preferences

### Design Principles
1. **Separation of Concerns**: Tabs focus on display, Controllers handle actions
2. **Configurability**: All keybindings configurable via YAML dotfile
3. **Safety**: Destructive operations require confirmation
4. **Discoverability**: Context-aware help system ('?' key)
5. **State Awareness**: Commands disabled during input/filtering modes
6. **Extensibility**: Easy to add new actions without modifying core logic

### Architecture Overview
```
┌──────────────────────────────────────────────────────────────┐
│                         Dashboard                             │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              ActionController                          │  │
│  │  - Manages action lifecycle (input → confirm → exec)  │  │
│  │  - Routes messages to appropriate handlers            │  │
│  │  - Tracks pending actions and state transitions       │  │
│  └────────────────────────────────────────────────────────┘  │
│  ┌────────────────┐  ┌──────────────────┐  ┌─────────────┐  │
│  │ KeybindController│  │  HelpController  │  │Notification │  │
│  │ - Load config  │  │  - Per-tab help  │  │  - Success  │  │
│  │ - Match keys   │  │  - Keybind list  │  │  - Errors   │  │
│  └────────────────┘  └──────────────────┘  └─────────────┘  │
└──────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌──────────────────┐  ┌─────────────────┐
│   QueriesTab    │  │  WorkItemsTab    │  │  TemplatesTab   │
│  ActionHandler  │  │  ActionHandler   │  │  ActionHandler  │
│  - ExecuteQuery │  │  - Download      │  │  - Copy         │
│  - RefreshList  │  │  - Edit          │  │  - Create       │
│                 │  │  - Delete        │  │  - Delete       │
│                 │  │  - Create        │  │                 │
│                 │  │  - Update        │  │                 │
└─────────────────┘  └──────────────────┘  └─────────────────┘
```

---

## 2. Architecture Components

### 2.1 KeybindController

Manages all keybindings, loaded from `~/.azure-boards-cli/keybinds.yaml`.

```go
// internal/tui/keybinds.go

package tui

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"gopkg.in/yaml.v3"
)

// KeybindController manages keybindings for all tabs
type KeybindController struct {
	global    map[string]key.Binding  // Global actions (quit, help, etc.)
	queries   map[string]key.Binding  // Queries tab actions
	workitems map[string]key.Binding  // Work items tab actions
	templates map[string]key.Binding  // Templates tab actions
	config    *KeybindConfig          // Loaded configuration
}

// KeybindConfig represents the YAML configuration structure
type KeybindConfig struct {
	Global struct {
		Quit       []string `yaml:"quit"`
		Help       []string `yaml:"help"`
		NextTab    []string `yaml:"next_tab"`
		PrevTab    []string `yaml:"prev_tab"`
		Refresh    []string `yaml:"refresh"`
	} `yaml:"global"`

	Queries struct {
		Execute    []string `yaml:"execute"`
		ExpandAll  []string `yaml:"expand_all"`
		CollapseAll[]string `yaml:"collapse_all"`
	} `yaml:"queries"`

	WorkItems struct {
		Details    []string `yaml:"details"`
		Download   []string `yaml:"download"`
		Edit       []string `yaml:"edit"`
		Delete     []string `yaml:"delete"`
		Create     []string `yaml:"create"`
		ChangeState[]string `yaml:"change_state"`
		Assign     []string `yaml:"assign"`
		AddTags    []string `yaml:"add_tags"`
	} `yaml:"work_items"`

	Templates struct {
		Copy       []string `yaml:"copy"`
		NewTemplate[]string `yaml:"new_template"`
		NewFolder  []string `yaml:"new_folder"`
		Edit       []string `yaml:"edit"`
		Delete     []string `yaml:"delete"`
	} `yaml:"templates"`
}

// NewKeybindController creates a new keybind controller
func NewKeybindController() *KeybindController {
	kc := &KeybindController{
		global:    make(map[string]key.Binding),
		queries:   make(map[string]key.Binding),
		workitems: make(map[string]key.Binding),
		templates: make(map[string]key.Binding),
	}

	// Load configuration or use defaults
	if err := kc.LoadConfig(); err != nil {
		logger.Printf("Failed to load keybinds config: %v, using defaults", err)
		kc.LoadDefaults()
	}

	return kc
}

// LoadConfig loads keybindings from ~/.azure-boards-cli/keybinds.yaml
func (kc *KeybindController) LoadConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".azure-boards-cli", "keybinds.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			return kc.CreateDefaultConfig(configPath)
		}
		return err
	}

	var config KeybindConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	kc.config = &config
	kc.buildBindings()
	return nil
}

// LoadDefaults loads hardcoded default keybindings
func (kc *KeybindController) LoadDefaults() {
	// Global bindings
	kc.global["quit"] = key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	)
	kc.global["help"] = key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	)
	kc.global["next_tab"] = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	)
	kc.global["prev_tab"] = key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev tab"),
	)
	kc.global["refresh"] = key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	)

	// Work Items bindings
	kc.workitems["details"] = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "toggle details"),
	)
	kc.workitems["download"] = key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "download as template"),
	)
	kc.workitems["edit"] = key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit work item"),
	)
	kc.workitems["delete"] = key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete work item"),
	)
	kc.workitems["create"] = key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new work item"),
	)
	kc.workitems["change_state"] = key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "change state"),
	)
	kc.workitems["assign"] = key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "assign to user"),
	)
	kc.workitems["add_tags"] = key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "add tags"),
	)

	// Queries bindings
	kc.queries["execute"] = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "execute query"),
	)
	kc.queries["expand_all"] = key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "expand all folders"),
	)
	kc.queries["collapse_all"] = key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "collapse all folders"),
	)

	// Templates bindings
	kc.templates["copy"] = key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy template"),
	)
	kc.templates["new_template"] = key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new template"),
	)
	kc.templates["new_folder"] = key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "new folder"),
	)
	kc.templates["edit"] = key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit template"),
	)
	kc.templates["delete"] = key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete template"),
	)
}

// buildBindings converts config to key.Binding objects
func (kc *KeybindController) buildBindings() {
	// Implementation: Convert config.Global.Quit string array to key.Binding, etc.
	// ... (similar to LoadDefaults but using config values)
}

// CreateDefaultConfig creates a default keybinds.yaml file
func (kc *KeybindController) CreateDefaultConfig(path string) error {
	defaultConfig := `# Azure Boards CLI Keybindings
# Customize keybindings for the TUI dashboard
# Multiple keys can be bound to the same action

global:
  quit: ["q", "ctrl+c"]
  help: ["?"]
  next_tab: ["tab"]
  prev_tab: ["shift+tab"]
  refresh: ["r"]

queries:
  execute: ["enter"]
  expand_all: ["E"]
  collapse_all: ["C"]

work_items:
  details: ["enter"]
  download: ["w"]          # Download work item as YAML template
  edit: ["e"]              # Edit work item in $EDITOR
  delete: ["d"]            # Delete work item (with confirmation)
  create: ["n"]            # Create new work item
  change_state: ["s"]      # Change work item state
  assign: ["a"]            # Assign to user
  add_tags: ["t"]          # Add tags

templates:
  copy: ["c"]              # Copy template
  new_template: ["n"]      # Create new template
  new_folder: ["f"]        # Create new folder
  edit: ["e"]              # Edit template in $EDITOR
  delete: ["d"]            # Delete template (with confirmation)
`

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(defaultConfig), 0644)
}

// Matches checks if a key message matches an action
func (kc *KeybindController) Matches(msg tea.KeyMsg, scope string, action string) bool {
	var bindings map[string]key.Binding

	switch scope {
	case "global":
		bindings = kc.global
	case "queries":
		bindings = kc.queries
	case "workitems":
		bindings = kc.workitems
	case "templates":
		bindings = kc.templates
	default:
		return false
	}

	binding, exists := bindings[action]
	if !exists {
		return false
	}

	return key.Matches(msg, binding)
}

// GetBinding returns the key binding for an action
func (kc *KeybindController) GetBinding(scope string, action string) (key.Binding, bool) {
	var bindings map[string]key.Binding

	switch scope {
	case "global":
		bindings = kc.global
	case "queries":
		bindings = kc.queries
	case "workitems":
		bindings = kc.workitems
	case "templates":
		bindings = kc.templates
	default:
		return key.Binding{}, false
	}

	binding, exists := bindings[action]
	return binding, exists
}

// GetAllBindings returns all bindings for a scope (for help display)
func (kc *KeybindController) GetAllBindings(scope string) map[string]key.Binding {
	switch scope {
	case "global":
		return kc.global
	case "queries":
		return kc.queries
	case "workitems":
		return kc.workitems
	case "templates":
		return kc.templates
	default:
		return make(map[string]key.Binding)
	}
}
```

### 2.2 ActionController

Manages the lifecycle of user actions, including multi-step flows (input → confirmation → execution).

```go
// internal/tui/actions.go

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ActionController manages action lifecycle and execution
type ActionController struct {
	pendingAction *PendingAction
	keybinds      *KeybindController
}

// PendingAction represents an action in progress
type PendingAction struct {
	Type    ActionType
	Context interface{}
	Step    ActionStep
	TabIndex int  // Which tab initiated this action
}

// ActionType defines the type of action
type ActionType string

const (
	ActionDownloadWorkItem ActionType = "download_work_item"
	ActionEditWorkItem     ActionType = "edit_work_item"
	ActionDeleteWorkItem   ActionType = "delete_work_item"
	ActionCreateWorkItem   ActionType = "create_work_item"
	ActionChangeState      ActionType = "change_state"
	ActionAssign           ActionType = "assign"
	ActionAddTags          ActionType = "add_tags"
	ActionCopyTemplate     ActionType = "copy_template"
	ActionDeleteTemplate   ActionType = "delete_template"
)

// ActionStep defines where in the action lifecycle we are
type ActionStep string

const (
	StepIdle      ActionStep = "idle"
	StepInput     ActionStep = "input"      // Awaiting user input
	StepConfirm   ActionStep = "confirm"    // Awaiting confirmation
	StepExecute   ActionStep = "execute"    // Executing action
	StepComplete  ActionStep = "complete"   // Action completed
)

// NewActionController creates a new action controller
func NewActionController(keybinds *KeybindController) *ActionController {
	return &ActionController{
		keybinds: keybinds,
	}
}

// CanExecuteAction returns true if actions can be executed
// (not during filtering, input mode, etc.)
func (ac *ActionController) CanExecuteAction(tab Tab) bool {
	// Check if there's already a pending action
	if ac.pendingAction != nil {
		return false
	}

	// Check tab-specific conditions
	switch t := tab.(type) {
	case *WorkItemsTab:
		// Can't execute actions while list is filtering
		if t.list.FilterState() == list.Filtering {
			return false
		}
	case *QueriesTab:
		if t.list.FilterState() == list.Filtering {
			return false
		}
	case *TemplatesTab:
		if t.list.FilterState() == list.Filtering {
			return false
		}
	}

	return true
}

// StartAction initiates an action
func (ac *ActionController) StartAction(actionType ActionType, context interface{}, tabIndex int) {
	ac.pendingAction = &PendingAction{
		Type:     actionType,
		Context:  context,
		Step:     StepInput,  // or StepConfirm for actions that don't need input
		TabIndex: tabIndex,
	}
}

// GetPendingAction returns the current pending action
func (ac *ActionController) GetPendingAction() *PendingAction {
	return ac.pendingAction
}

// ClearPendingAction clears the current pending action
func (ac *ActionController) ClearPendingAction() {
	ac.pendingAction = nil
}

// AdvanceStep moves the action to the next step
func (ac *ActionController) AdvanceStep(nextStep ActionStep) {
	if ac.pendingAction != nil {
		ac.pendingAction.Step = nextStep
	}
}
```

### 2.3 HelpController

Manages per-tab help display.

```go
// internal/tui/help.go

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// HelpController manages help overlay display
type HelpController struct {
	visible     bool
	currentTab  string
	keybinds    *KeybindController
}

// NewHelpController creates a new help controller
func NewHelpController(keybinds *KeybindController) *HelpController {
	return &HelpController{
		keybinds: keybinds,
	}
}

// Toggle toggles help visibility
func (hc *HelpController) Toggle() {
	hc.visible = !hc.visible
}

// Show shows the help overlay
func (hc *HelpController) Show(tabName string) {
	hc.visible = true
	hc.currentTab = tabName
}

// Hide hides the help overlay
func (hc *HelpController) Hide() {
	hc.visible = false
}

// IsVisible returns whether help is currently visible
func (hc *HelpController) IsVisible() bool {
	return hc.visible
}

// View renders the help overlay
func (hc *HelpController) View(width, height int) string {
	if !hc.visible {
		return ""
	}

	var scope string
	switch hc.currentTab {
	case "Queries":
		scope = "queries"
	case "Work Items":
		scope = "workitems"
	case "Templates":
		scope = "templates"
	default:
		scope = "global"
	}

	// Get all bindings for this scope
	globalBindings := hc.keybinds.GetAllBindings("global")
	scopeBindings := hc.keybinds.GetAllBindings(scope)

	// Build help content
	var helpLines []string
	helpLines = append(helpLines, TitleStyle.Render(fmt.Sprintf("Help: %s", hc.currentTab)))
	helpLines = append(helpLines, "")

	// Global actions
	helpLines = append(helpLines, lipgloss.NewStyle().Bold(true).Render("Global Actions:"))
	for action, binding := range globalBindings {
		helpLines = append(helpLines, fmt.Sprintf("  %s - %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSecondary)).Render(binding.Help().Key),
			binding.Help().Desc))
	}
	helpLines = append(helpLines, "")

	// Tab-specific actions
	if len(scopeBindings) > 0 {
		helpLines = append(helpLines, lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s Actions:", hc.currentTab)))
		for action, binding := range scopeBindings {
			helpLines = append(helpLines, fmt.Sprintf("  %s - %s",
				lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSecondary)).Render(binding.Help().Key),
				binding.Help().Desc))
		}
	}

	helpLines = append(helpLines, "")
	helpLines = append(helpLines, MutedStyle.Render("Press ? again to close help"))

	content := strings.Join(helpLines, "\n")

	// Create centered overlay
	helpBox := BoxStyle.
		Width(min(width-4, 60)).
		Height(min(height-4, len(helpLines)+4)).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, helpBox)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

### 2.4 ActionHandler Interface

Each tab implements action handlers for their specific domain.

```go
// internal/tui/tab.go (additions to existing file)

// ActionHandler defines methods for handling tab-specific actions
type ActionHandler interface {
	// HandleAction processes an action for this tab
	// Returns (success, notification message, error)
	HandleAction(actionType ActionType, context interface{}) (bool, string, error)

	// GetActionContext returns the context for an action (e.g., selected work item)
	GetActionContext(actionType ActionType) (interface{}, error)

	// CanHandleAction returns true if this tab can handle the action
	CanHandleAction(actionType ActionType) bool
}
```

---

## 3. Action Lifecycle Pattern

### State Machine

```
┌──────┐   User presses key   ┌───────┐   Needs input?   ┌───────┐
│ IDLE │ ───────────────────→ │ INPUT │ ─────Yes───────→ │ AWAIT │
└──────┘                       └───────┘                   │ INPUT │
                                   │                       └───────┘
                                   No                          │
                                   │                           │
                                   ▼                           │
                              ┌─────────┐                      │
                              │ CONFIRM │ ◄────────────────────┘
                              └─────────┘    Input complete
                                   │
                              Destructive?
                                   │
                                   ├──Yes──→ Show confirmation
                                   │         │
                                   │         ▼
                                   │    User confirms? ──No──→ CANCEL
                                   │         │
                                   No        Yes
                                   │         │
                                   ▼         ▼
                              ┌──────────────┐
                              │   EXECUTE    │
                              └──────────────┘
                                   │
                                   ▼
                              ┌──────────┐
                              │ COMPLETE │ ─→ Show notification → IDLE
                              └──────────┘
```

### Message Flow Example: Delete Work Item

```
User presses 'd' key
    │
    ▼
Dashboard.Update() receives tea.KeyMsg
    │
    ▼
KeybindController.Matches(msg, "workitems", "delete") = true
    │
    ▼
ActionController.CanExecuteAction(workitemsTab) = true
    │
    ▼
WorkItemsTab.GetActionContext(ActionDeleteWorkItem) returns selected work item
    │
    ▼
ActionController.StartAction(ActionDeleteWorkItem, workItem, tabIndex=1)
    │
    ▼
Dashboard shows ConfirmationDialog:
    "Delete work item #123: 'Fix login bug' and its 3 child tasks?"
    │
    ▼
User presses 'y'
    │
    ▼
ActionController.AdvanceStep(StepExecute)
    │
    ▼
Return tea.Cmd that:
    1. Fetches child work items
    2. Deletes children first (in order)
    3. Deletes parent work item
    4. Returns WorkItemDeletedMsg
    │
    ▼
Dashboard routes WorkItemDeletedMsg to WorkItemsTab
    │
    ▼
WorkItemsTab updates state, removes item from list
    │
    ▼
Dashboard shows NotificationMsg:
    "Successfully deleted work item #123 and 3 child tasks"
    │
    ▼
ActionController.ClearPendingAction()
```

---

## 4. Specific Action Implementations

### 4.1 Download Work Item as YAML Template

**Keybind**: `w` (customizable via `work_items.download`)

**Flow**: Single-step action (no input/confirmation required)

**Implementation**:
```go
// internal/tui/workitems.go

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

		// Fetch full work item details (not just list fields)
		fullWI, err := client.GetWorkItem(id, nil)
		if err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to fetch work item #%d: %v", id, err),
				IsError: true,
			}
		}

		// Convert to template format
		template := convertWorkItemToTemplate(fullWI)

		// Serialize to YAML
		yamlData, err := yaml.Marshal(template)
		if err != nil {
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
		filepath := filepath.Join(templatesDir, filename)

		if err := os.WriteFile(filepath, yamlData, 0644); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to save template: %v", err),
				IsError: true,
			}
		}

		return NotificationMsg{
			Message: fmt.Sprintf("Downloaded work item #%d as template: %s", id, filename),
			IsError: false,
		}
	}
}

func convertWorkItemToTemplate(wi *workitemtracking.WorkItem) *templates.Template {
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
		"System.Tags",
		"Microsoft.VSTS.Common.Priority",
		"Microsoft.VSTS.Common.AcceptanceCriteria",
		"System.AreaPath",
		"System.IterationPath",
	}

	for _, fieldName := range relevantFields {
		if value, ok := (*wi.Fields)[fieldName]; ok {
			template.Fields[fieldName] = value
		}
	}

	// Handle relationships (children)
	if wi.Relations != nil {
		for _, rel := range *wi.Relations {
			if rel.Rel != nil && *rel.Rel == "System.LinkTypes.Hierarchy-Forward" {
				// This is a child work item
				childID := extractWorkItemIDFromURL(*rel.Url)
				if childID > 0 {
					child := templates.ChildTemplate{
						Title: getStringField(wi, "System.Title"), // Will be updated when creating
						Type:  "Task",
					}
					if template.Relations == nil {
						template.Relations = &templates.Relations{}
					}
					template.Relations.Children = append(template.Relations.Children, child)
				}
			}
		}
	}

	return template
}

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
```

### 4.2 Edit Work Item

**Keybind**: `e` (customizable via `work_items.edit`)

**Flow**: Multi-step action
1. Fetch full work item by ID
2. Convert to YAML template
3. Create temporary file
4. Open in $EDITOR
5. Wait for editor to close
6. Parse YAML changes
7. Update work item via API

**Implementation**:
```go
// internal/tui/workitems.go

func (t *WorkItemsTab) handleEditAction() tea.Cmd {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		return editWorkItem(t.client, item.workItem)
	}
	return nil
}

// editWorkItem opens a work item in the system editor
func editWorkItem(client *api.Client, wi workitemtracking.WorkItem) tea.Cmd {
	return func() tea.Msg {
		id := 0
		if wi.Id != nil {
			id = *wi.Id
		}

		// Fetch full work item
		fullWI, err := client.GetWorkItem(id, nil)
		if err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to fetch work item #%d: %v", id, err),
				IsError: true,
			}
		}

		// Convert to editable YAML
		template := convertWorkItemToTemplate(fullWI)
		yamlData, err := yaml.Marshal(template)
		if err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to serialize work item: %v", err),
				IsError: true,
			}
		}

		// Create temporary file
		tmpfile, err := os.CreateTemp("", fmt.Sprintf("workitem-%d-*.yaml", id))
		if err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to create temp file: %v", err),
				IsError: true,
			}
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(yamlData); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to write temp file: %v", err),
				IsError: true,
			}
		}
		tmpfile.Close()

		// Open in editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"  // fallback
		}

		cmd := exec.Command(editor, tmpfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Suspend BubbleTea while editor is open
		if err := cmd.Run(); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Editor exited with error: %v", err),
				IsError: true,
			}
		}

		// Read modified YAML
		modifiedData, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to read modified file: %v", err),
				IsError: true,
			}
		}

		// Parse modified template
		var modifiedTemplate templates.Template
		if err := yaml.Unmarshal(modifiedData, &modifiedTemplate); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to parse modified YAML: %v", err),
				IsError: true,
			}
		}

		// Determine what changed and update
		updateDoc := buildUpdateDocument(fullWI, &modifiedTemplate)
		if len(updateDoc) == 0 {
			return NotificationMsg{
				Message: "No changes detected",
				IsError: false,
			}
		}

		// Update work item
		if err := client.UpdateWorkItem(id, updateDoc); err != nil {
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to update work item: %v", err),
				IsError: true,
			}
		}

		// Refresh the work item
		updatedWI, err := client.GetWorkItem(id, nil)
		if err != nil {
			return NotificationMsg{
				Message: "Work item updated, but failed to refresh",
				IsError: false,
			}
		}

		return WorkItemUpdatedMsg{
			WorkItem: updatedWI,
			Error:    nil,
		}
	}
}

// buildUpdateDocument creates a JSON Patch document for API update
func buildUpdateDocument(original *workitemtracking.WorkItem, modified *templates.Template) []interface{} {
	var ops []interface{}

	// Compare fields and build patch operations
	for fieldName, newValue := range modified.Fields {
		oldValue := getFieldValue(original, fieldName)
		if !reflect.DeepEqual(oldValue, newValue) {
			ops = append(ops, map[string]interface{}{
				"op":    "replace",
				"path":  "/fields/" + fieldName,
				"value": newValue,
			})
		}
	}

	return ops
}
```

**Note**: Editing requires suspending the BubbleTea program. Use `tea.ExecProcess` for better integration:

```go
// Better approach using tea.ExecProcess
func (t *WorkItemsTab) handleEditAction() tea.Cmd {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(workItemItem); ok {
		// First fetch and prepare the temp file
		return tea.Sequence(
			prepareEditFile(t.client, item.workItem),
			func() tea.Msg {
				// After file is prepared, open editor
				return OpenEditorMsg{FilePath: /* temp file path */}
			},
		)
	}
	return nil
}

// In Dashboard.Update()
case OpenEditorMsg:
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, msg.FilePath)
	return d, tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorClosedMsg{FilePath: msg.FilePath, Error: err}
	})
```

### 4.3 Delete Work Item (with Children)

**Keybind**: `d` (customizable via `work_items.delete`)

**Flow**: Multi-step action with confirmation
1. Get selected work item
2. Fetch child work items
3. Show confirmation: "Delete work item #123 'Title' and its 3 child tasks?"
4. On confirmation:
   - Delete children first (in dependency order)
   - Delete parent work item
   - Leave parent (if exists) unchanged
5. Refresh work items list

**Implementation**:
```go
// internal/tui/workitems.go

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

		// Fetch full work item with relationships
		expand := "relations"
		fullWI, err := client.GetWorkItem(id, &expand)
		if err != nil {
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
					// This is a child
					childID := extractWorkItemIDFromURL(*rel.Url)
					if childID > 0 {
						childIDs = append(childIDs, childID)
					}
				}
			}
		}

		title := getStringField(fullWI, "System.Title")

		return ConfirmDeleteWorkItemMsg{
			WorkItemID: id,
			Title:      title,
			ChildIDs:   childIDs,
		}
	}
}

// In messages.go, add:
type ConfirmDeleteWorkItemMsg struct {
	WorkItemID int
	Title      string
	ChildIDs   []int
}

// In Dashboard.Update():
case ConfirmDeleteWorkItemMsg:
	// Show confirmation dialog
	childCount := len(msg.ChildIDs)
	childText := ""
	if childCount > 0 {
		childText = fmt.Sprintf(" and its %d child task(s)", childCount)
	}

	d.confirmation.Show(
		fmt.Sprintf("Delete work item #%d: '%s'%s?", msg.WorkItemID, msg.Title, childText),
		"delete_work_item",
		msg,  // Store context
	)
	return d, nil

// When user confirms (presses 'y'):
case tea.KeyMsg:
	if d.confirmation.Active && (msg.String() == "y" || msg.String() == "Y") {
		if d.confirmation.Action == "delete_work_item" {
			d.confirmation.Hide()
			// Execute deletion
			if ctx, ok := d.confirmation.Context.(ConfirmDeleteWorkItemMsg); ok {
				return d, deleteWorkItemWithChildren(d.client, ctx.WorkItemID, ctx.ChildIDs)
			}
		}
	}

// deleteWorkItemWithChildren deletes a work item and all its children
func deleteWorkItemWithChildren(client *api.Client, parentID int, childIDs []int) tea.Cmd {
	return func() tea.Msg {
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
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to delete work item #%d: %v", parentID, err),
				IsError: true,
			}
		}

		logger.Printf("Deleted parent work item #%d", parentID)

		totalDeleted := len(childIDs) + 1
		message := fmt.Sprintf("Successfully deleted work item #%d", parentID)
		if len(childIDs) > 0 {
			message += fmt.Sprintf(" and %d child task(s)", len(childIDs))
		}

		return WorkItemDeletedMsg{
			ID:    parentID,
			Error: nil,
		}
	}
}
```

---

## 5. Help System Integration

### 5.1 Per-Tab Help Registration

Extend the Tab interface:

```go
// internal/tui/tab.go

type Tab interface {
	Name() string
	Init(width, height int) tea.Cmd
	Update(msg tea.Msg) (Tab, tea.Cmd)
	View() string
	SetSize(width, height int)

	// New method for help system
	GetHelpEntries() []HelpEntry
}

// HelpEntry describes a single action for help display
type HelpEntry struct {
	Action      string
	Description string
	Keybind     string  // Will be populated by KeybindController
}
```

### 5.2 Implementation in WorkItemsTab

```go
// internal/tui/workitems.go

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
```

### 5.3 Dashboard Integration

```go
// In Dashboard.Update()

case tea.KeyMsg:
	// Check for help key globally
	if d.keybinds.Matches(msg, "global", "help") {
		if d.help.IsVisible() {
			d.help.Hide()
		} else {
			currentTabName := d.tabs[d.currentTab].Name()
			d.help.Show(currentTabName)
		}
		return d, nil
	}

	// If help is visible, only handle help toggle
	if d.help.IsVisible() {
		return d, nil
	}

// In Dashboard.View()
func (d *Dashboard) View() string {
	// ... existing rendering ...

	// Render help overlay on top of everything
	if d.help.IsVisible() {
		return d.help.View(d.width, d.height)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
```

---

## 6. Integration with Existing TUI

### 6.1 Dashboard Changes

```go
// internal/tui/dashboard.go

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

	// New controllers
	keybinds  *KeybindController
	actions   *ActionController
	help      *HelpController
}

func NewDashboard(client *api.Client) *Dashboard {
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

	// Initialize tabs with keybind controller
	dashboard.tabs = []Tab{
		NewQueriesTab(client, 0, 0),
		NewWorkItemsTab(client, 0, 0),
		NewTemplatesTab(0, 0),
		NewPipelinesTab(0, 0),
		NewAgentsTab(0, 0),
	}

	return dashboard
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Help has highest priority
		if d.keybinds.Matches(msg, "global", "help") {
			d.help.Toggle()
			return d, nil
		}

		// If help is visible, block all other input
		if d.help.IsVisible() {
			return d, nil
		}

		// Handle input prompt
		if d.inputPrompt.Active {
			// ... existing input handling ...
		}

		// Handle confirmation
		if d.confirmation.Active {
			// ... existing confirmation handling ...
		}

		// Check if actions can be executed
		if !d.actions.CanExecuteAction(d.tabs[d.currentTab]) {
			// Route to tab normally (for filtering, etc.)
			tab, cmd := d.tabs[d.currentTab].Update(msg)
			d.tabs[d.currentTab] = tab
			return d, cmd
		}

		// Handle global keybinds
		if d.keybinds.Matches(msg, "global", "quit") {
			return d, tea.Quit
		}

		if d.keybinds.Matches(msg, "global", "next_tab") {
			d.currentTab = (d.currentTab + 1) % len(d.tabs)
			return d, nil
		}

		if d.keybinds.Matches(msg, "global", "prev_tab") {
			d.currentTab = (d.currentTab - 1 + len(d.tabs)) % len(d.tabs)
			return d, nil
		}

		if d.keybinds.Matches(msg, "global", "refresh") {
			tab, cmd := d.tabs[d.currentTab].Update(msg)
			d.tabs[d.currentTab] = tab
			return d, cmd
		}

		// Check tab-specific actions
		currentTab := d.tabs[d.currentTab]
		scope := getTabScope(currentTab.Name())

		// Work Items actions
		if scope == "workitems" {
			if workitemsTab, ok := currentTab.(*WorkItemsTab); ok {
				if d.keybinds.Matches(msg, "workitems", "download") {
					return d, workitemsTab.handleDownloadAction()
				}
				if d.keybinds.Matches(msg, "workitems", "edit") {
					return d, workitemsTab.handleEditAction()
				}
				if d.keybinds.Matches(msg, "workitems", "delete") {
					return d, workitemsTab.handleDeleteAction()
				}
				// ... other actions ...
			}
		}

		// Route to tab for default handling
		tab, cmd := d.tabs[d.currentTab].Update(msg)
		d.tabs[d.currentTab] = tab
		return d, cmd

	// ... rest of existing message handling ...
	}

	return d, tea.Batch(cmds...)
}

func getTabScope(tabName string) string {
	switch tabName {
	case "Queries":
		return "queries"
	case "Work Items":
		return "workitems"
	case "Templates":
		return "templates"
	default:
		return ""
	}
}
```

### 6.2 Tab Interface Update

Add GetHelpEntries method to Tab interface (already shown above in section 5.1).

Update all tabs to implement this method (even if they return empty array for now).

---

## 7. Implementation Checklist

### Phase 1: Foundation (Controllers & Config)
- [ ] Create `internal/tui/keybinds.go`
  - [ ] Define KeybindController struct
  - [ ] Implement config loading from ~/.azure-boards-cli/keybinds.yaml
  - [ ] Implement default keybinds
  - [ ] Implement Matches() and GetBinding() methods
  - [ ] Create default keybinds.yaml template

- [ ] Create `internal/tui/actions.go`
  - [ ] Define ActionController struct
  - [ ] Define ActionType and ActionStep enums
  - [ ] Define PendingAction struct
  - [ ] Implement CanExecuteAction() logic
  - [ ] Implement action lifecycle methods

- [ ] Create `internal/tui/help.go`
  - [ ] Define HelpController struct
  - [ ] Implement Toggle/Show/Hide methods
  - [ ] Implement View() for help overlay rendering

### Phase 2: Tab Interface Extensions
- [ ] Update `internal/tui/tab.go`
  - [ ] Add GetHelpEntries() method to Tab interface
  - [ ] Define HelpEntry struct
  - [ ] Add ActionHandler interface (optional, for now implement directly in tabs)

- [ ] Update all tabs to implement GetHelpEntries()
  - [ ] QueriesTab
  - [ ] WorkItemsTab
  - [ ] TemplatesTab
  - [ ] PipelinesTab (stub)
  - [ ] AgentsTab (stub)

### Phase 3: Dashboard Integration
- [ ] Update `internal/tui/dashboard.go`
  - [ ] Add keybinds, actions, help fields to Dashboard struct
  - [ ] Initialize controllers in NewDashboard()
  - [ ] Add help toggle handling in Update()
  - [ ] Add action execution handling in Update()
  - [ ] Add help overlay rendering in View()
  - [ ] Implement getTabScope() helper

- [ ] Update `internal/tui/messages.go`
  - [ ] Add ConfirmDeleteWorkItemMsg
  - [ ] Add WorkItemUpdatedMsg
  - [ ] Add OpenEditorMsg
  - [ ] Add EditorClosedMsg

### Phase 4: Work Items Actions
- [ ] Update `internal/tui/workitems.go`
  - [ ] Implement handleDownloadAction()
  - [ ] Implement downloadWorkItem() tea.Cmd
  - [ ] Implement convertWorkItemToTemplate()
  - [ ] Implement sanitizeFilename() helper

- [ ] Implement Edit Action
  - [ ] Implement handleEditAction()
  - [ ] Implement editWorkItem() tea.Cmd
  - [ ] Implement buildUpdateDocument()
  - [ ] Handle tea.ExecProcess for editor integration

- [ ] Implement Delete Action
  - [ ] Implement handleDeleteAction()
  - [ ] Implement fetchWorkItemForDelete() tea.Cmd
  - [ ] Implement deleteWorkItemWithChildren() tea.Cmd
  - [ ] Add confirmation flow in Dashboard

### Phase 5: Queries & Templates Actions
- [ ] Update `internal/tui/queries.go`
  - [ ] Implement expand all / collapse all actions
  - [ ] Update keybind handling to use KeybindController

- [ ] Update `internal/tui/templates.go`
  - [ ] Implement copy template action
  - [ ] Implement new template/folder actions
  - [ ] Implement edit template action
  - [ ] Implement delete template action
  - [ ] Update keybind handling to use KeybindController

### Phase 6: Testing & Documentation
- [ ] Test keybind loading and defaults
- [ ] Test each action with success/error paths
- [ ] Test help overlay on each tab
- [ ] Test action disabling during filtering
- [ ] Test confirmation flow for destructive actions
- [ ] Update user documentation with new keybindings
- [ ] Create migration guide from old hardcoded keys

### Phase 7: Polish & UX
- [ ] Add loading indicators for long-running actions
- [ ] Improve notification messages with action context
- [ ] Add undo capability for delete (soft delete?)
- [ ] Add action history/audit log
- [ ] Keyboard shortcut cheatsheet command (`azb keybinds`)

---

## 8. Configuration Example

### Default ~/.azure-boards-cli/keybinds.yaml

```yaml
# Azure Boards CLI Keybindings Configuration
#
# Customize keyboard shortcuts for the TUI dashboard.
# Multiple keys can be bound to the same action (use array format).
#
# Valid key formats:
#   - Single keys: "a", "b", "1", "2", etc.
#   - Modified keys: "ctrl+c", "alt+enter", "shift+tab"
#   - Special keys: "enter", "esc", "space", "backspace", "delete"
#   - Arrow keys: "up", "down", "left", "right"
#   - Function keys: "f1", "f2", etc.

# ============================================
# Global Actions (available on all tabs)
# ============================================
global:
  quit: ["q", "ctrl+c"]
  help: ["?"]
  next_tab: ["tab"]
  prev_tab: ["shift+tab"]
  refresh: ["r"]

# ============================================
# Queries Tab
# ============================================
queries:
  execute: ["enter"]           # Execute selected query or expand folder
  expand_all: ["E"]            # Expand all folders
  collapse_all: ["C"]          # Collapse all folders

# ============================================
# Work Items Tab
# ============================================
work_items:
  details: ["enter"]           # Toggle details panel for selected item
  download: ["w"]              # Download work item as YAML template
  edit: ["e"]                  # Edit work item in $EDITOR
  delete: ["d"]                # Delete work item (requires confirmation)
  create: ["n"]                # Create new work item
  change_state: ["s"]          # Change work item state
  assign: ["a"]                # Assign work item to user
  add_tags: ["t"]              # Add tags to work item

# ============================================
# Templates Tab
# ============================================
templates:
  copy: ["c"]                  # Copy template with new name
  new_template: ["n"]          # Create new template
  new_folder: ["f"]            # Create new folder
  edit: ["e"]                  # Edit template in $EDITOR
  delete: ["d"]                # Delete template (requires confirmation)

# ============================================
# Notes
# ============================================
# - Changes take effect on next launch of azb dashboard
# - Invalid keybinds will be ignored and defaults will be used
# - Keybind conflicts (same key for multiple actions) will log a warning
# - To reset to defaults, delete this file and restart azb
```

---

## 9. Future Enhancements

### 9.1 Action Plugins
Allow users to define custom actions via YAML configuration:

```yaml
# ~/.azure-boards-cli/actions.yaml
custom_actions:
  - name: "export_to_github"
    scope: "workitems"
    keybind: ["ctrl+g"]
    description: "Export work item to GitHub issue"
    command: "azb export github {{.ID}}"
    confirm: false

  - name: "create_pr"
    scope: "workitems"
    keybind: ["ctrl+p"]
    description: "Create PR for this work item"
    command: "gh pr create --title '{{.Title}}' --body '{{.Description}}'"
    confirm: true
```

### 9.2 Action Macros
Allow chaining multiple actions:

```yaml
macros:
  - name: "close_and_archive"
    keybind: ["ctrl+shift+d"]
    steps:
      - action: "change_state"
        value: "Closed"
      - action: "add_tags"
        value: "archived"
      - action: "download"  # Save backup
```

### 9.3 Context-Aware Actions
Actions that are only available based on work item state/type:

```go
func (t *WorkItemsTab) GetAvailableActions() []ActionType {
	selected := t.getSelectedWorkItem()
	if selected == nil {
		return []ActionType{}
	}

	actions := []ActionType{
		ActionDownloadWorkItem,
		ActionEditWorkItem,
		ActionDeleteWorkItem,
	}

	// Only show "change state" if not in terminal state
	state := getStringField(selected, "System.State")
	if state != "Closed" && state != "Removed" {
		actions = append(actions, ActionChangeState)
	}

	// Only show "assign" if unassigned
	assignee := getStringField(selected, "System.AssignedTo")
	if assignee == "" {
		actions = append(actions, ActionAssign)
	}

	return actions
}
```

### 9.4 Undo Support
Implement action history with undo capability:

```go
type ActionHistory struct {
	actions []CompletedAction
	maxSize int
}

type CompletedAction struct {
	Type      ActionType
	Timestamp time.Time
	Context   interface{}
	UndoCmd   tea.Cmd  // Command to undo this action
}

// Keybind: ctrl+z
func (d *Dashboard) handleUndo() tea.Cmd {
	return d.actionHistory.Undo()
}
```

---

## 10. Testing Strategy

### Unit Tests
```go
// internal/tui/keybinds_test.go
func TestKeybindController_Matches(t *testing.T) {
	kc := NewKeybindController()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}

	if !kc.Matches(msg, "workitems", "delete") {
		t.Error("Expected 'd' to match delete action")
	}
}

// internal/tui/actions_test.go
func TestActionController_CanExecuteAction(t *testing.T) {
	ac := NewActionController(nil)
	tab := &WorkItemsTab{/* ... */}

	// Should allow actions when not filtering
	if !ac.CanExecuteAction(tab) {
		t.Error("Expected actions to be allowed")
	}

	// Should block when pending action exists
	ac.StartAction(ActionDeleteWorkItem, nil, 1)
	if ac.CanExecuteAction(tab) {
		t.Error("Expected actions to be blocked with pending action")
	}
}
```

### Integration Tests
```go
func TestWorkItemDownload(t *testing.T) {
	// Setup mock client
	client := &mockAPIClient{}
	client.On("GetWorkItem", 123, nil).Return(mockWorkItem(), nil)

	// Execute download
	cmd := downloadWorkItem(client, mockWorkItem())
	msg := cmd()

	// Verify notification
	if notification, ok := msg.(NotificationMsg); !ok || notification.IsError {
		t.Error("Expected success notification")
	}

	// Verify file was created
	homeDir, _ := os.UserHomeDir()
	path := filepath.Join(homeDir, ".azure-boards-cli", "templates", "workitem-123-*.yaml")
	matches, _ := filepath.Glob(path)
	if len(matches) == 0 {
		t.Error("Expected template file to be created")
	}
}
```

---

## 11. Migration Path

For users with custom keybind expectations, provide gradual migration:

### Step 1: Deprecation Warnings (v0.1.0)
```
⚠ Warning: Hardcoded keybindings are deprecated and will be removed in v0.2.0
  Run 'azb keybinds init' to create a custom keybinds.yaml configuration
```

### Step 2: Dual Support (v0.1.x)
- Both hardcoded and config-based keybinds work
- Config overrides hardcoded
- Warning shown if no config exists

### Step 3: Config Required (v0.2.0+)
- Hardcoded keybinds removed
- Auto-generate default config on first launch
- `azb keybinds` command to manage configuration

---

## Summary

This Controller Pattern provides:

✅ **Separation of Concerns**: Views focus on rendering, controllers handle actions
✅ **Configurability**: All keybindings customizable via YAML
✅ **Safety**: Confirmations for destructive operations
✅ **Discoverability**: Per-tab help system with '?' key
✅ **Extensibility**: Easy to add new actions and tabs
✅ **State Management**: Proper handling of input modes and action lifecycle
✅ **User Experience**: Consistent patterns across all tabs

The architecture builds on the existing Tab interface pattern while adding the necessary infrastructure for sophisticated action handling, making the TUI both powerful and user-friendly.
