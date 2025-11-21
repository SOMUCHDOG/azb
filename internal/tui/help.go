package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpController manages help overlay display
type HelpController struct {
	visible    bool
	currentTab string
	keybinds   *KeybindController
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
	for _, binding := range globalBindings {
		helpLines = append(helpLines, fmt.Sprintf(
			"  %s - %s",
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSecondary)).
				Render(binding.Help().Key),
			binding.Help().Desc,
		))
	}
	helpLines = append(helpLines, "")

	// Tab-specific actions
	if len(scopeBindings) > 0 {
		helpLines = append(helpLines, lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s Actions:", hc.currentTab)))
		for _, binding := range scopeBindings {
			helpLines = append(helpLines, fmt.Sprintf("  %s - %s",
				lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSecondary)).Render(binding.Help().Key),
				binding.Help().Desc))
		}
	}

	helpLines = append(helpLines, "")
	helpLines = append(helpLines, MutedStyle.Render("Press ? again to close help"))

	content := strings.Join(helpLines, "\n")

	// Create centered overlay
	helpWidth := min(width-4, 60)
	helpHeight := min(height-4, len(helpLines)+4)

	helpBox := BoxStyle.
		Width(helpWidth).
		Height(helpHeight).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, helpBox)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
