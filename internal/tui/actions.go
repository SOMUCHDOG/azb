package tui

import (
	"github.com/charmbracelet/bubbles/list"
)

// ActionController manages action lifecycle and execution
type ActionController struct {
	pendingAction *PendingAction
	keybinds      *KeybindController
}

// PendingAction represents an action in progress
type PendingAction struct {
	Type     ActionType
	Context  interface{}
	Step     ActionStep
	TabIndex int // Which tab initiated this action
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
	StepIdle     ActionStep = "idle"
	StepInput    ActionStep = "input"    // Awaiting user input
	StepConfirm  ActionStep = "confirm"  // Awaiting confirmation
	StepExecute  ActionStep = "execute"  // Executing action
	StepComplete ActionStep = "complete" // Action completed
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
		Step:     StepInput, // or StepConfirm for actions that don't need input
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
