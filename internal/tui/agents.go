package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// AgentsTab is a placeholder for future agents functionality
type AgentsTab struct {
	TabBase
}

// NewAgentsTab creates a new agents tab
func NewAgentsTab(width, height int) *AgentsTab {
	return &AgentsTab{
		TabBase: NewTabBase(width, height),
	}
}

// Name returns the tab name
func (t *AgentsTab) Name() string {
	return "Agents"
}

// Init initializes the tab
func (t *AgentsTab) Init(width, height int) tea.Cmd {
	t.SetSize(width, height)
	return nil
}

// Update handles messages
func (t *AgentsTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, nil
}

// View renders the tab
func (t *AgentsTab) View() string {
	return RenderComingSoon("Agents")
}

// SetSize updates the tab dimensions
func (t *AgentsTab) SetSize(width, height int) {
	t.TabBase.SetSize(width, height)
}
