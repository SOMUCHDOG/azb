package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template represents a work item template
type Template struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description,omitempty"`
	Type        string                 `yaml:"type"`
	Fields      map[string]interface{} `yaml:"fields"`
	Relations   *Relations             `yaml:"relations,omitempty"`
}

// Relations represents work item relationships
type Relations struct {
	ParentID int              `yaml:"parentId,omitempty"`
	Children []ChildWorkItem  `yaml:"children,omitempty"`
}

// ChildWorkItem represents a child work item to be created
type ChildWorkItem struct {
	Type        string                 `yaml:"type,omitempty"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description,omitempty"`
	AssignedTo  string                 `yaml:"assignedTo,omitempty"`
	Fields      map[string]interface{} `yaml:"fields,omitempty"`
}

// GetTemplatesDir returns the path to the templates directory
func GetTemplatesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	templatesDir := filepath.Join(home, ".azure-boards-cli", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create templates directory: %w", err)
	}

	return templatesDir, nil
}

// GetTemplatePath returns the full path for a template file
func GetTemplatePath(name string) (string, error) {
	templatesDir, err := GetTemplatesDir()
	if err != nil {
		return "", err
	}

	// Ensure .yaml extension
	if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
		name = name + ".yaml"
	}

	return filepath.Join(templatesDir, name), nil
}

// Load loads a template by name
func Load(name string) (*Template, error) {
	path, err := GetTemplatePath(name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("template '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	var template Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &template, nil
}

// Save saves a template
func Save(template *Template) error {
	path, err := GetTemplatePath(template.Name)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// List lists all available templates
func List() ([]*Template, error) {
	templatesDir, err := GetTemplatesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []*Template
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
		template, err := Load(name)
		if err != nil {
			// Skip invalid templates
			continue
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// Delete deletes a template by name
func Delete(name string) error {
	path, err := GetTemplatePath(name)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("template '%s' not found", name)
		}
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// Exists checks if a template exists
func Exists(name string) (bool, error) {
	path, err := GetTemplatePath(name)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
