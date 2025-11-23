package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
const (
	ColorPrimary   = "6"   // Cyan
	ColorSecondary = "62"  // Purple
	ColorAccent    = "170" // Pink
	ColorSuccess   = "10"  // Green
	ColorWarning   = "11"  // Yellow
	ColorError     = "9"   // Red
	ColorInfo      = "12"  // Blue
	ColorMuted     = "8"   // Gray
	ColorNormal    = "7"   // White
	ColorYellow    = "230" // Light yellow
)

// Common styles
var (
	// Text styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary))

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary))

	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorNormal))

	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo))

	// State-specific styles for work items
	StateActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSuccess))

	StateNewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo))

	StateClosedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorMuted))

	StateBlockedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorError))

	// Folder/tree styles
	FolderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo))

	FileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess))

	// Container styles
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorSecondary)).
			Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorSecondary)).
			Padding(1)

	// Tab styles
	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			Underline(true).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorMuted)).
				Padding(0, 2)

	// Notification styles
	NotificationSuccessStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color(ColorSuccess)).
					Bold(true).
					Padding(0, 1).
					Margin(0, 2)

	NotificationErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorError)).
				Bold(true).
				Padding(0, 1).
				Margin(0, 2)

	// Dialog styles
	DialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorWarning)).
			Padding(1, 2).
			Width(60)

	DialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(ColorWarning))

	InputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPrimary)).
			Padding(1, 2).
			Width(60)

	InputTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary))

	// Selection dialog styles
	SelectedOptionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(ColorPrimary))
)

// Helper functions for common styling patterns

// RenderTitle renders a title with consistent styling
func RenderTitle(text string) string {
	return TitleStyle.Render(text)
}

// RenderSelected renders selected text
func RenderSelected(text string) string {
	return SelectedStyle.Render(text)
}

// RenderError renders error text
func RenderError(text string) string {
	return ErrorStyle.Render(text)
}

// RenderSuccess renders success text
func RenderSuccess(text string) string {
	return SuccessStyle.Render(text)
}

// RenderMuted renders muted text
func RenderMuted(text string) string {
	return MutedStyle.Render(text)
}

// RenderInBox renders text in a bordered box
func RenderInBox(content string) string {
	return BoxStyle.Render(content)
}
