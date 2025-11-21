package tui

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

// KeybindController manages keybindings for all tabs
type KeybindController struct {
	global    map[string]key.Binding // Global actions (quit, help, etc.)
	queries   map[string]key.Binding // Queries tab actions
	workitems map[string]key.Binding // Work items tab actions
	templates map[string]key.Binding // Templates tab actions
	config    *KeybindConfig         // Loaded configuration
}

// KeybindConfig represents the YAML configuration structure
type KeybindConfig struct {
	Global struct {
		Quit    []string `yaml:"quit"`
		Help    []string `yaml:"help"`
		NextTab []string `yaml:"next_tab"`
		PrevTab []string `yaml:"prev_tab"`
		Refresh []string `yaml:"refresh"`
	} `yaml:"global"`

	Queries struct {
		Execute     []string `yaml:"execute"`
		ExpandAll   []string `yaml:"expand_all"`
		CollapseAll []string `yaml:"collapse_all"`
	} `yaml:"queries"`

	WorkItems struct {
		Details     []string `yaml:"details"`
		Download    []string `yaml:"download"`
		Edit        []string `yaml:"edit"`
		Delete      []string `yaml:"delete"`
		Create      []string `yaml:"create"`
		ChangeState []string `yaml:"change_state"`
		Assign      []string `yaml:"assign"`
		AddTags     []string `yaml:"add_tags"`
	} `yaml:"work_items"`

	Templates struct {
		Copy        []string `yaml:"copy"`
		NewTemplate []string `yaml:"new_template"`
		NewFolder   []string `yaml:"new_folder"`
		Edit        []string `yaml:"edit"`
		Rename      []string `yaml:"rename"`
		Delete      []string `yaml:"delete"`
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

	// Always load defaults first, then overlay user config
	kc.LoadDefaults()
	if err := kc.LoadConfig(); err != nil {
		logger.Printf("Failed to load keybinds config: %v, using defaults only", err)
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
	kc.templates["rename"] = key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "rename"),
	)
	kc.templates["delete"] = key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete template"),
	)
}

// buildBindings converts config to key.Binding objects
func (kc *KeybindController) buildBindings() {
	if kc.config == nil {
		return
	}

	// Build global bindings
	if len(kc.config.Global.Quit) > 0 {
		kc.global["quit"] = key.NewBinding(
			key.WithKeys(kc.config.Global.Quit...),
			key.WithHelp(kc.config.Global.Quit[0], "quit"),
		)
	}
	if len(kc.config.Global.Help) > 0 {
		kc.global["help"] = key.NewBinding(
			key.WithKeys(kc.config.Global.Help...),
			key.WithHelp(kc.config.Global.Help[0], "help"),
		)
	}
	if len(kc.config.Global.NextTab) > 0 {
		kc.global["next_tab"] = key.NewBinding(
			key.WithKeys(kc.config.Global.NextTab...),
			key.WithHelp(kc.config.Global.NextTab[0], "next tab"),
		)
	}
	if len(kc.config.Global.PrevTab) > 0 {
		kc.global["prev_tab"] = key.NewBinding(
			key.WithKeys(kc.config.Global.PrevTab...),
			key.WithHelp(kc.config.Global.PrevTab[0], "prev tab"),
		)
	}
	if len(kc.config.Global.Refresh) > 0 {
		kc.global["refresh"] = key.NewBinding(
			key.WithKeys(kc.config.Global.Refresh...),
			key.WithHelp(kc.config.Global.Refresh[0], "refresh"),
		)
	}

	// Build queries bindings
	if len(kc.config.Queries.Execute) > 0 {
		kc.queries["execute"] = key.NewBinding(
			key.WithKeys(kc.config.Queries.Execute...),
			key.WithHelp(kc.config.Queries.Execute[0], "execute query"),
		)
	}
	if len(kc.config.Queries.ExpandAll) > 0 {
		kc.queries["expand_all"] = key.NewBinding(
			key.WithKeys(kc.config.Queries.ExpandAll...),
			key.WithHelp(kc.config.Queries.ExpandAll[0], "expand all"),
		)
	}
	if len(kc.config.Queries.CollapseAll) > 0 {
		kc.queries["collapse_all"] = key.NewBinding(
			key.WithKeys(kc.config.Queries.CollapseAll...),
			key.WithHelp(kc.config.Queries.CollapseAll[0], "collapse all"),
		)
	}

	// Build work items bindings
	if len(kc.config.WorkItems.Details) > 0 {
		kc.workitems["details"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.Details...),
			key.WithHelp(kc.config.WorkItems.Details[0], "toggle details"),
		)
	}
	if len(kc.config.WorkItems.Download) > 0 {
		kc.workitems["download"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.Download...),
			key.WithHelp(kc.config.WorkItems.Download[0], "download as template"),
		)
	}
	if len(kc.config.WorkItems.Edit) > 0 {
		kc.workitems["edit"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.Edit...),
			key.WithHelp(kc.config.WorkItems.Edit[0], "edit work item"),
		)
	}
	if len(kc.config.WorkItems.Delete) > 0 {
		kc.workitems["delete"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.Delete...),
			key.WithHelp(kc.config.WorkItems.Delete[0], "delete work item"),
		)
	}
	if len(kc.config.WorkItems.Create) > 0 {
		kc.workitems["create"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.Create...),
			key.WithHelp(kc.config.WorkItems.Create[0], "new work item"),
		)
	}
	if len(kc.config.WorkItems.ChangeState) > 0 {
		kc.workitems["change_state"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.ChangeState...),
			key.WithHelp(kc.config.WorkItems.ChangeState[0], "change state"),
		)
	}
	if len(kc.config.WorkItems.Assign) > 0 {
		kc.workitems["assign"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.Assign...),
			key.WithHelp(kc.config.WorkItems.Assign[0], "assign to user"),
		)
	}
	if len(kc.config.WorkItems.AddTags) > 0 {
		kc.workitems["add_tags"] = key.NewBinding(
			key.WithKeys(kc.config.WorkItems.AddTags...),
			key.WithHelp(kc.config.WorkItems.AddTags[0], "add tags"),
		)
	}

	// Build templates bindings
	if len(kc.config.Templates.Copy) > 0 {
		kc.templates["copy"] = key.NewBinding(
			key.WithKeys(kc.config.Templates.Copy...),
			key.WithHelp(kc.config.Templates.Copy[0], "copy template"),
		)
	}
	if len(kc.config.Templates.NewTemplate) > 0 {
		kc.templates["new_template"] = key.NewBinding(
			key.WithKeys(kc.config.Templates.NewTemplate...),
			key.WithHelp(kc.config.Templates.NewTemplate[0], "new template"),
		)
	}
	if len(kc.config.Templates.NewFolder) > 0 {
		kc.templates["new_folder"] = key.NewBinding(
			key.WithKeys(kc.config.Templates.NewFolder...),
			key.WithHelp(kc.config.Templates.NewFolder[0], "new folder"),
		)
	}
	if len(kc.config.Templates.Edit) > 0 {
		kc.templates["edit"] = key.NewBinding(
			key.WithKeys(kc.config.Templates.Edit...),
			key.WithHelp(kc.config.Templates.Edit[0], "edit template"),
		)
	}
	if len(kc.config.Templates.Rename) > 0 {
		kc.templates["rename"] = key.NewBinding(
			key.WithKeys(kc.config.Templates.Rename...),
			key.WithHelp(kc.config.Templates.Rename[0], "rename"),
		)
	}
	if len(kc.config.Templates.Delete) > 0 {
		kc.templates["delete"] = key.NewBinding(
			key.WithKeys(kc.config.Templates.Delete...),
			key.WithHelp(kc.config.Templates.Delete[0], "delete template"),
		)
	}
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
  rename: ["m"]            # Rename template or folder
  delete: ["d"]            # Delete template (with confirmation)
`

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
		return err
	}

	logger.Printf("Created default keybinds config at %s", path)

	// Load the defaults into memory
	kc.LoadDefaults()
	return nil
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
