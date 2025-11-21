package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Notification displays temporary success or error messages
type Notification struct {
	Message string
	IsError bool
	Visible bool
}

// NewNotification creates a new notification
func NewNotification(message string, isError bool) *Notification {
	return &Notification{
		Message: message,
		IsError: isError,
		Visible: true,
	}
}

// Show sets the notification message
func (n *Notification) Show(message string, isError bool) {
	n.Message = message
	n.IsError = isError
	n.Visible = true
}

// Clear clears the notification
func (n *Notification) Clear() {
	n.Visible = false
	n.Message = ""
}

// View renders the notification
func (n *Notification) View() string {
	if !n.Visible || n.Message == "" {
		return ""
	}

	var style lipgloss.Style
	var icon string
	if n.IsError {
		style = NotificationErrorStyle
		icon = "✗"
	} else {
		style = NotificationSuccessStyle
		icon = "✓"
	}

	return style.Render(fmt.Sprintf("%s %s", icon, n.Message))
}

// InputPrompt displays an input field for user text entry
type InputPrompt struct {
	Title       string
	Placeholder string
	Input       textinput.Model
	Active      bool
	Action      string      // What action this input is for
	Context     interface{} // Additional context for the action
}

// NewInputPrompt creates a new input prompt
func NewInputPrompt() *InputPrompt {
	ti := textinput.New()
	ti.CharLimit = 100
	ti.Width = 50

	return &InputPrompt{
		Input:  ti,
		Active: false,
	}
}

// Show displays the input prompt
func (i *InputPrompt) Show(title, placeholder, action string, context interface{}) tea.Cmd {
	i.Title = title
	i.Placeholder = placeholder
	i.Action = action
	i.Context = context
	i.Active = true
	i.Input.Placeholder = placeholder
	i.Input.SetValue("")
	i.Input.Focus()
	return nil
}

// Hide hides the input prompt
func (i *InputPrompt) Hide() {
	i.Active = false
	i.Input.Blur()
}

// Update updates the input prompt
func (i *InputPrompt) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	i.Input, cmd = i.Input.Update(msg)
	return cmd
}

// Value returns the current input value
func (i *InputPrompt) Value() string {
	return strings.TrimSpace(i.Input.Value())
}

// View renders the input prompt
func (i *InputPrompt) View() string {
	if !i.Active {
		return ""
	}

	title := InputTitleStyle.Render(i.Title)
	input := i.Input.View()
	help := MutedStyle.Render("(Enter to submit, Esc to cancel)")

	content := fmt.Sprintf("%s\n\n%s\n\n%s", title, input, help)
	return InputBoxStyle.Render(content)
}

// ConfirmationDialog displays a yes/no confirmation prompt
type ConfirmationDialog struct {
	Prompt  string
	Active  bool
	Action  string      // What action to confirm
	Context interface{} // Additional context for the action
}

// NewConfirmationDialog creates a new confirmation dialog
func NewConfirmationDialog() *ConfirmationDialog {
	return &ConfirmationDialog{
		Active: false,
	}
}

// Show displays the confirmation dialog
func (c *ConfirmationDialog) Show(prompt, action string, context interface{}) {
	c.Prompt = prompt
	c.Action = action
	c.Context = context
	c.Active = true
}

// Hide hides the confirmation dialog
func (c *ConfirmationDialog) Hide() {
	c.Active = false
}

// View renders the confirmation dialog
func (c *ConfirmationDialog) View() string {
	if !c.Active {
		return ""
	}

	title := DialogTitleStyle.Render("⚠ Confirmation Required")
	prompt := NormalStyle.Render(c.Prompt)
	help := MutedStyle.Render("(y/n)")

	content := fmt.Sprintf("%s\n\n%s\n\n%s", title, prompt, help)
	return DialogBoxStyle.Render(content)
}

// Header renders the application header
func RenderHeader() string {
	title := TitleStyle.Render("Azure Boards Dashboard")
	return HeaderStyle.Render(title)
}

// RenderFooter renders the application footer with help text
func RenderFooter(helpText string) string {
	if helpText == "" {
		helpText = "q: quit • tab: switch tabs • r: refresh • ?: help"
	}
	return MutedStyle.Render(helpText)
}

// RenderTabBar renders the tab navigation bar
func RenderTabBar(tabs []string, currentTab int) string {
	var renderedTabs []string

	for i, tab := range tabs {
		if i == currentTab {
			renderedTabs = append(renderedTabs, ActiveTabStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, InactiveTabStyle.Render(tab))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	help := MutedStyle.Render("  tab/shift+tab: switch")

	return lipgloss.JoinHorizontal(lipgloss.Top, tabBar, help)
}

// RenderLoading renders a loading message
func RenderLoading(message string) string {
	if message == "" {
		message = "Loading..."
	}
	return lipgloss.NewStyle().Padding(1).Render(message)
}

// RenderError renders an error message with retry hint
func RenderErrorWithRetry(err error) string {
	errorText := ErrorStyle.Render(fmt.Sprintf("Error: %v", err))
	help := MutedStyle.Render("\nPress 'r' to retry, 'q' to quit")
	return lipgloss.NewStyle().Padding(1).Render(errorText + help)
}

// RenderComingSoon renders a "coming soon" message
func RenderComingSoon(feature string) string {
	title := TitleStyle.Render(fmt.Sprintf("%s Tab", feature))
	message := MutedStyle.Render("Coming soon!")
	return lipgloss.NewStyle().Padding(2).Render(title + "\n\n" + message)
}
