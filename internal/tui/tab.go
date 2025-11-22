package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Tab represents a single tab in the dashboard
// Each tab is responsible for its own state, updates, and rendering
type Tab interface {
	// Name returns the display name of the tab
	Name() string

	// Init initializes the tab and returns any initial commands
	Init(width, height int) tea.Cmd

	// Update handles messages and updates the tab state
	Update(msg tea.Msg) (Tab, tea.Cmd)

	// View renders the tab's content
	View() string

	// SetSize updates the tab's dimensions
	SetSize(width, height int)

	// GetHelpEntries returns the list of actions available for this tab
	// Used by the help system to display context-aware help
	GetHelpEntries() []HelpEntry
}

// HelpEntry describes a single action for help display
type HelpEntry struct {
	Action      string // Action identifier (e.g., "download", "edit")
	Description string // Human-readable description
}

// TabBase provides common functionality for tabs
// Embed this in your tab implementations to reduce boilerplate
type TabBase struct {
	width  int
	height int
}

// NewTabBase creates a new TabBase
func NewTabBase(width, height int) TabBase {
	return TabBase{
		width:  width,
		height: height,
	}
}

// SetSize updates the tab dimensions
func (t *TabBase) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// Width returns the tab width
func (t *TabBase) Width() int {
	return t.width
}

// Height returns the tab height
func (t *TabBase) Height() int {
	return t.height
}

// ContentHeight returns the height available for content
// (accounting for header and tab bar)
func (t *TabBase) ContentHeight() int {
	headerHeight := 3
	tabBarHeight := 1
	margins := headerHeight + tabBarHeight
	return t.height - margins
}
