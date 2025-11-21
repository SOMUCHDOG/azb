package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/casey/azure-boards-cli/internal/templates"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TemplatesTab displays and manages templates
type TemplatesTab struct {
	TabBase
	templates        []*templates.TemplateNode
	list             list.Model
	preview          viewport.Model
	expandedFolders  map[string]bool
	selectedTemplate *templates.Template
	loading          bool
	err              error
}

// NewTemplatesTab creates a new templates tab
func NewTemplatesTab(width, height int) *TemplatesTab {
	tab := &TemplatesTab{
		TabBase:         NewTabBase(width, height),
		expandedFolders: make(map[string]bool),
		loading:         true,
	}

	// Split view: list on left, preview on right
	listWidth := width / 2
	previewWidth := width - listWidth - 2

	// Initialize list
	tab.list = list.New([]list.Item{}, templateDelegate{expandedFolders: tab.expandedFolders}, listWidth, tab.ContentHeight())
	tab.list.Title = "Templates"
	tab.list.SetShowStatusBar(false)
	tab.list.SetFilteringEnabled(true)
	tab.list.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color(ColorSecondary)).
		Foreground(lipgloss.Color(ColorYellow)).
		Padding(0, 1)

	// Initialize preview
	tab.preview = viewport.New(previewWidth, tab.ContentHeight()-2)

	return tab
}

// Name returns the tab name
func (t *TemplatesTab) Name() string {
	return "Templates"
}

// Init initializes the tab
func (t *TemplatesTab) Init(width, height int) tea.Cmd {
	t.SetSize(width, height)
	return t.FetchTemplates()
}

// Update handles messages
func (t *TemplatesTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TemplatesLoadedMsg:
		t.loading = false
		if msg.Error != nil {
			t.err = msg.Error
			return t, nil
		}
		t.templates = msg.Templates
		t.rebuildList()
		return t, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			return t.handleEnter()
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			t.loading = true
			return t, t.FetchTemplates()
		default:
			// Update list and preview on selection change
			t.list, cmd = t.list.Update(msg)
			cmds = append(cmds, cmd)
			t.updatePreview()
			return t, tea.Batch(cmds...)
		}
	}

	t.list, cmd = t.list.Update(msg)
	cmds = append(cmds, cmd)
	t.preview, cmd = t.preview.Update(msg)
	cmds = append(cmds, cmd)

	return t, tea.Batch(cmds...)
}

// View renders the tab
func (t *TemplatesTab) View() string {
	if t.loading {
		return RenderLoading("Loading templates...")
	}

	if t.err != nil {
		return RenderErrorWithRetry(t.err)
	}

	// Split view: list on left, preview on right
	previewBox := BoxStyle.Render(t.preview.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, t.list.View(), previewBox)
}

// SetSize updates the tab dimensions
func (t *TemplatesTab) SetSize(width, height int) {
	t.TabBase.SetSize(width, height)

	listWidth := width / 2
	previewWidth := width - listWidth - 2

	t.list.SetSize(listWidth, t.ContentHeight())
	t.preview.Width = previewWidth
	t.preview.Height = t.ContentHeight() - 2
}

// handleEnter toggles folder expansion or shows template details
func (t *TemplatesTab) handleEnter() (Tab, tea.Cmd) {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(templateListItem); ok {
		if item.IsDir {
			// Toggle folder expand/collapse
			t.expandedFolders[item.Path] = !t.expandedFolders[item.Path]
			t.rebuildList()
			return t, nil
		}
		// Template selected - preview is already updated
		t.updatePreview()
		return t, nil
	}
	return t, nil
}

// updatePreview updates the preview pane based on selection
func (t *TemplatesTab) updatePreview() {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(templateListItem); ok {
		if item.IsDir {
			t.preview.SetContent(t.formatFolderPreview(item))
		} else if item.Template != nil {
			t.preview.SetContent(t.formatTemplatePreview(item.Template))
		}
	}
}

// rebuildList rebuilds the list with current expanded state
func (t *TemplatesTab) rebuildList() {
	items := t.flattenTemplates(t.templates, 0)
	delegate := templateDelegate{expandedFolders: t.expandedFolders}
	t.list.SetDelegate(delegate)
	t.list.SetItems(items)

	// Update preview for the selected item
	t.updatePreview()
}

// flattenTemplates recursively flattens the template hierarchy
func (t *TemplatesTab) flattenTemplates(templateNodes []*templates.TemplateNode, depth int) []list.Item {
	var items []list.Item

	for _, node := range templateNodes {
		// Always add the current item (folder or template)
		items = append(items, templateListItem{
			Name:     node.Name,
			Path:     node.Path,
			IsDir:    node.IsDir,
			Depth:    depth,
			Template: node.Template,
			node:     node,
		})

		// Only add children if this is a directory AND it's expanded
		if node.IsDir && node.Children != nil && len(node.Children) > 0 {
			if t.expandedFolders[node.Path] {
				childItems := t.flattenTemplates(node.Children, depth+1)
				items = append(items, childItems...)
			}
		}
	}

	return items
}

// FetchTemplates loads templates from the filesystem
func (t *TemplatesTab) FetchTemplates() tea.Cmd {
	return func() tea.Msg {
		templateNodes, err := templates.ListTree()
		if err != nil {
			return TemplatesLoadedMsg{Error: err}
		}

		return TemplatesLoadedMsg{Templates: templateNodes}
	}
}

// formatTemplatePreview formats a template for preview display
func (t *TemplatesTab) formatTemplatePreview(template *templates.Template) string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(template.Name) + "\n\n")

	if template.Description != "" {
		b.WriteString(template.Description + "\n\n")
	}

	b.WriteString(MutedStyle.Render("Type: ") + template.Type + "\n\n")

	if len(template.Fields) > 0 {
		b.WriteString(MutedStyle.Render("Fields:\n"))
		for key, value := range template.Fields {
			// Format field name to be more readable
			fieldName := strings.TrimPrefix(key, "System.")
			fieldName = strings.TrimPrefix(fieldName, "Microsoft.VSTS.Common.")
			fieldName = strings.TrimPrefix(fieldName, "Custom.")

			b.WriteString(fmt.Sprintf("  %s: %v\n", fieldName, value))
		}
		b.WriteString("\n")
	}

	if template.Relations != nil {
		if template.Relations.ParentID > 0 {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("Parent ID: %d\n\n", template.Relations.ParentID)))
		}

		if len(template.Relations.Children) > 0 {
			b.WriteString(MutedStyle.Render(fmt.Sprintf("Children: (%d)\n", len(template.Relations.Children))))
			for i, child := range template.Relations.Children {
				childType := child.Type
				if childType == "" {
					childType = "Task"
				}
				b.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, childType, child.Title))
			}
		}
	}

	return b.String()
}

// formatFolderPreview formats a folder for preview display
func (t *TemplatesTab) formatFolderPreview(item templateListItem) string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("📁 " + item.Name) + "\n\n")

	if item.node != nil && item.node.Children != nil {
		count := len(item.node.Children)
		b.WriteString(fmt.Sprintf("Contains %d item", count))
		if count != 1 {
			b.WriteString("s")
		}
		b.WriteString("\n\n")

		if t.expandedFolders[item.Path] {
			b.WriteString(MutedStyle.Render("Press Enter to collapse"))
		} else {
			b.WriteString(MutedStyle.Render("Press Enter to expand"))
		}
	}

	return b.String()
}

// templateDelegate implements list.ItemDelegate for template items
type templateDelegate struct {
	expandedFolders map[string]bool
}

func (d templateDelegate) Height() int                             { return 1 }
func (d templateDelegate) Spacing() int                            { return 0 }
func (d templateDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d templateDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	templateItem, ok := item.(templateListItem)
	if !ok {
		return
	}

	indent := strings.Repeat("  ", templateItem.Depth)
	icon := ""
	var nameStyle lipgloss.Style

	if templateItem.IsDir {
		// Check if folder is expanded
		expanded := d.expandedFolders[templateItem.Path]
		if expanded {
			icon = "▼ "
		} else {
			icon = "▶ "
		}
		nameStyle = FolderStyle
	} else {
		icon = "  📄 "
		nameStyle = FileStyle
	}

	name := templateItem.Name
	if len(name) > 60 {
		name = name[:57] + "..."
	}

	var output string
	if index == m.Index() {
		output = SelectedStyle.Render(fmt.Sprintf("> %s%s%s", indent, icon, name))
	} else {
		output = nameStyle.Render(fmt.Sprintf("  %s%s%s", indent, icon, name))
	}

	fmt.Fprint(w, output)
}

// templateListItem wraps a template node for the list
type templateListItem struct {
	Name     string
	Path     string
	IsDir    bool
	Depth    int
	Template *templates.Template
	node     *templates.TemplateNode
}

func (i templateListItem) FilterValue() string { return i.Name }

// GetHelpEntries returns the list of available actions for the Templates tab
func (t *TemplatesTab) GetHelpEntries() []HelpEntry {
	return []HelpEntry{
		{Action: "copy", Description: "Copy template"},
		{Action: "new_template", Description: "Create new template"},
		{Action: "new_folder", Description: "Create new folder"},
		{Action: "edit", Description: "Edit template in $EDITOR"},
		{Action: "rename", Description: "Rename template or folder"},
		{Action: "delete", Description: "Delete template"},
		{Action: "refresh", Description: "Refresh templates list"},
	}
}

// handleRenameAction handles the rename template action (m key)
func (t *TemplatesTab) handleRenameAction() *InputPrompt {
	selectedItem := t.list.SelectedItem()
	if item, ok := selectedItem.(templateListItem); ok {
		// Show input prompt for new name
		prompt := NewInputPrompt()
		prompt.Show("Enter new name:", item.Name, "rename_template", item.Path)
		return prompt
	}
	return nil
}

// renameTemplate renames a template or folder
func renameTemplate(oldPath, newName string) tea.Cmd {
	return func() tea.Msg {
		logger.Printf("Renaming '%s' to '%s'", oldPath, newName)

		// Perform rename
		if err := templates.Rename(oldPath, newName); err != nil {
			logger.Printf("Failed to rename: %v", err)
			return NotificationMsg{
				Message: fmt.Sprintf("Failed to rename: %v", err),
				IsError: true,
			}
		}

		logger.Printf("Successfully renamed to '%s'", newName)

		return TemplateRenamedMsg{
			OldPath: oldPath,
			NewPath: newName,
			Error:   nil,
		}
	}
}
