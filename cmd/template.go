package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/SOMUCHDOG/azb/internal/api"
	"github.com/SOMUCHDOG/azb/internal/auth"
	"github.com/SOMUCHDOG/azb/internal/config"
	"github.com/SOMUCHDOG/azb/internal/templates"
	"github.com/SOMUCHDOG/azb/internal/workitemtpl"
)

var (
	templateFormatFlag string
	templateIDFlag     int

	templateCmd = &cobra.Command{
		Use:   "template",
		Short: "Manage work item templates",
		Long:  `List, show, and manage work item templates.`,
		RunE:  runTemplate,
	}

	templateListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all templates",
		Long:  `List all available work item templates.`,
		RunE:  runTemplateList,
	}

	templateShowCmd = &cobra.Command{
		Use:   "show <template-name>",
		Short: "Show template details",
		Long:  `Show the contents of a work item template.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runTemplateShow,
	}

	templateDeleteCmd = &cobra.Command{
		Use:   "delete <template-name>",
		Short: "Delete a template",
		Long:  `Delete a work item template.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runTemplateDelete,
	}

	templateSaveCmd = &cobra.Command{
		Use:   "save <template-name>",
		Short: "Save current create flags as a template",
		Long:  `Save the current work item configuration as a template for reuse.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runTemplateSave,
	}
)

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateDeleteCmd)
	templateCmd.AddCommand(templateSaveCmd)
	templateCmd.Flags().IntVar(&templateIDFlag, "id", 0, "Download a work item as a template by ID")

	templateShowCmd.Flags().StringVarP(&templateFormatFlag, "format", "f", "yaml", "Output format (yaml, json)")

	// Add template save flags (reuse create flags)
	templateSaveCmd.Flags().StringVar(&createTypeFlag, "type", "", "Work item type")
	templateSaveCmd.Flags().StringVar(&createDescriptionFlag, "description", "", "Template description")
	templateSaveCmd.Flags().StringVar(&createTitleFlag, "title", "", "Default title")
	templateSaveCmd.Flags().StringVar(&createAssignedToFlag, "assigned-to", "", "Default assigned to")
	templateSaveCmd.Flags().StringVar(&createAreaPathFlag, "area-path", "", "Default area path")
	templateSaveCmd.Flags().StringVar(&createIterationFlag, "iteration", "", "Default iteration")
	templateSaveCmd.Flags().IntVar(&createPriorityFlag, "priority", 0, "Default priority")
	templateSaveCmd.Flags().StringVar(&createTagsFlag, "tags", "", "Default tags")
	templateSaveCmd.Flags().StringArrayVar(&createFieldsFlag, "field", []string{}, "Custom field in format 'FieldName=value'")

	//nolint:errcheck // Flag requirement error is non-critical at init time
	templateSaveCmd.MarkFlagRequired("type")
}

func runTemplate(cmd *cobra.Command, args []string) error {
	if templateIDFlag > 0 {
		return runTemplateDownloadByID(templateIDFlag)
	}
	return cmd.Help()
}

func runTemplateDownloadByID(workItemID int) error {
	token, err := auth.GetToken()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	org := viper.GetString("organization")
	if org == "" {
		org = cfg.Organization
	}
	if org == "" {
		return fmt.Errorf("organization not configured. Run 'azb config organization <org>'")
	}

	project := viper.GetString("project")
	if project == "" {
		project = cfg.Project
	}
	if project == "" {
		return fmt.Errorf("project not configured. Run 'azb config project <project>'")
	}

	client, err := api.NewClient(api.NormalizeOrganizationURL(org), project, token)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	workItem, err := client.GetWorkItem(workItemID)
	if err != nil {
		return fmt.Errorf("failed to get work item %d: %w", workItemID, err)
	}

	template := workitemtpl.ConvertToTemplate(client, workItem)
	yamlData, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to serialize template: %w", err)
	}

	templatesDir, err := templates.GetTemplatesDir()
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	filename := fmt.Sprintf("workitem-%d-%s.yaml", workItemID, sanitizeTemplateFilename(template.Name))
	filePath := filepath.Join(templatesDir, filename)
	if err := os.WriteFile(filePath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	fmt.Printf("✓ Downloaded work item #%d template to %s\n", workItemID, filePath)
	return nil
}

func sanitizeTemplateFilename(s string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := s
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "-")
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return strings.ToLower(strings.TrimSpace(result))
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	// Get templates directory
	templatesDir, err := templates.GetTemplatesDir()
	if err != nil {
		return fmt.Errorf("failed to get templates directory: %w", err)
	}

	templatesList, err := templates.List()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	fmt.Printf("Templates directory: %s\n\n", templatesDir)

	if len(templatesList) == 0 {
		fmt.Println("No templates found")
		fmt.Println("\nCreate a template with: azb template save <name> --type <type> [options]")
		return nil
	}

	fmt.Println("Available Templates:")
	fmt.Println()

	for _, tmpl := range templatesList {
		fmt.Printf("  %s\n", tmpl.Name)
		if tmpl.Description != "" {
			fmt.Printf("    Description: %s\n", tmpl.Description)
		}
		fmt.Printf("    Type: %s\n", tmpl.Type)
		if len(tmpl.Fields) > 0 {
			fmt.Printf("    Fields: %d configured\n", len(tmpl.Fields))
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d templates\n", len(templatesList))
	fmt.Println("\nUse 'azb template show <name>' to view template details")
	fmt.Println("Use 'azb create --template <name>' to create a work item from a template")

	return nil
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	template, err := templates.Load(name)
	if err != nil {
		return err
	}

	switch templateFormatFlag {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(template)
	case "yaml":
		fallthrough
	default:
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(template)
	}
}

func runTemplateDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if template exists
	exists, err := templates.Exists(name)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("template '%s' not found", name)
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete template '%s'? (y/N): ", name)
	var response string
	//nolint:errcheck // User input error is acceptable - will default to 'no'
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	if err := templates.Delete(name); err != nil {
		return err
	}

	fmt.Printf("✓ Template '%s' deleted\n", name)

	return nil
}

func runTemplateSave(cmd *cobra.Command, args []string) error {
	name := args[0]

	if createTypeFlag == "" {
		return fmt.Errorf("--type is required")
	}

	// Build fields map
	fields := make(map[string]interface{})

	if createTitleFlag != "" {
		fields["System.Title"] = createTitleFlag
	}

	if createDescriptionFlag != "" {
		fields["System.Description"] = createDescriptionFlag
	}

	if createAssignedToFlag != "" {
		fields["System.AssignedTo"] = createAssignedToFlag
	}

	if createAreaPathFlag != "" {
		fields["System.AreaPath"] = createAreaPathFlag
	}

	if createIterationFlag != "" {
		fields["System.IterationPath"] = createIterationFlag
	}

	if createPriorityFlag > 0 {
		fields["Microsoft.VSTS.Common.Priority"] = createPriorityFlag
	}

	if createTagsFlag != "" {
		fields["System.Tags"] = createTagsFlag
	}

	// Parse custom fields
	for _, fieldArg := range createFieldsFlag {
		parts := strings.SplitN(fieldArg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid field format '%s', expected 'FieldName=value'", fieldArg)
		}
		fields[parts[0]] = parts[1]
	}

	// Create template
	template := &templates.Template{
		Name:        name,
		Description: cmd.Flag("description").Value.String(),
		Type:        createTypeFlag,
		Fields:      fields,
	}

	// Save template
	if err := templates.Save(template); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	fmt.Printf("✓ Template '%s' saved\n", name)
	fmt.Printf("\nUse 'azb create --template %s' to create a work item from this template\n", name)

	return nil
}
