package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PipelinesTab is a placeholder for future pipelines functionality
type PipelinesTab struct {
	TabBase
}

// NewPipelinesTab creates a new pipelines tab
func NewPipelinesTab(width, height int) *PipelinesTab {
	return &PipelinesTab{
		TabBase: NewTabBase(width, height),
	}
}

// Name returns the tab name
func (t *PipelinesTab) Name() string {
	return "Pipelines"
}

// Init initializes the tab
func (t *PipelinesTab) Init(width, height int) tea.Cmd {
	t.SetSize(width, height)
	return nil
}

// Update handles messages
func (t *PipelinesTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, nil
}

// View renders the tab
func (t *PipelinesTab) View() string {
	return RenderComingSoon("Pipelines")
}

// SetSize updates the tab dimensions
func (t *PipelinesTab) SetSize(width, height int) {
	t.TabBase.SetSize(width, height)
}
