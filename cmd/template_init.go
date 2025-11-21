package cmd

import (
	"fmt"

	"github.com/casey/azure-boards-cli/internal/templates"
	"github.com/spf13/cobra"
)

var (
	templateInitCmd = &cobra.Command{
		Use:   "init <template-name> <work-item-type>",
		Short: "Create a new template file with example fields",
		Long:  `Create a new template file with all common fields as examples that you can customize.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runTemplateInit,
	}
)

func init() {
	templateCmd.AddCommand(templateInitCmd)
}

func runTemplateInit(cmd *cobra.Command, args []string) error {
	name := args[0]
	workItemType := args[1]

	// Check if template already exists
	exists, err := templates.Exists(name)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("template '%s' already exists. Use 'azb template show %s' to view it", name, name)
	}

	// Create example template based on work item type
	template := createExampleTemplate(name, workItemType)

	// Save the template
	if err := templates.Save(template); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	// Get the template path to show the user
	path, _ := templates.GetTemplatePath(name)

	fmt.Printf("✓ Template '%s' created\n", name)
	fmt.Printf("\nTemplate file: %s\n", path)
	fmt.Println("\nEdit this file to customize the template with your default values.")
	fmt.Println("Fields can be removed if you don't need them.")
	fmt.Printf("\nExample usage:\n")
	fmt.Printf("  azb create --template %s --title \"Your work item title\"\n", name)

	return nil
}

func createExampleTemplate(name, workItemType string) *templates.Template {
	fields := make(map[string]interface{})

	// Common fields for all work item types
	fields["System.Title"] = "Example Title - Edit Me"
	fields["System.Description"] = "Example description - Edit or remove this field"
	fields["System.AssignedTo"] = "@me"
	fields["System.Tags"] = "example,template"

	// Add type-specific fields
	switch workItemType {
	case "User Story":
		fields["System.State"] = "New"
		fields["Microsoft.VSTS.Common.Priority"] = 2
		fields["Microsoft.VSTS.Common.ValueArea"] = "Business"
		fields["Microsoft.VSTS.Scheduling.StoryPoints"] = 0
		fields["Microsoft.VSTS.Common.AcceptanceCriteria"] = "- [ ] Acceptance criteria 1\n- [ ] Acceptance criteria 2"
		fields["Microsoft.VSTS.Common.Activity"] = "Development"
		// Add placeholder for custom fields users might need
		fields["Custom.ApplicationName"] = "YourApp"

	case "Bug":
		fields["System.State"] = "New"
		fields["Microsoft.VSTS.Common.Priority"] = 2
		fields["Microsoft.VSTS.Common.Severity"] = "3 - Medium"
		fields["Microsoft.VSTS.TCM.ReproSteps"] = "1. Step one\n2. Step two\n3. Observe the issue"
		fields["Microsoft.VSTS.TCM.SystemInfo"] = "Browser/OS information"

	case "Task":
		fields["System.State"] = "New"
		fields["Microsoft.VSTS.Common.Priority"] = 2
		fields["Microsoft.VSTS.Common.Activity"] = "Development"
		fields["Microsoft.VSTS.Scheduling.RemainingWork"] = 0

	case "Feature":
		fields["System.State"] = "New"
		fields["Microsoft.VSTS.Common.Priority"] = 2
		fields["Microsoft.VSTS.Common.ValueArea"] = "Business"
		fields["Microsoft.VSTS.Scheduling.TargetDate"] = ""

	case "Epic":
		fields["System.State"] = "New"
		fields["Microsoft.VSTS.Common.Priority"] = 2
		fields["Microsoft.VSTS.Common.ValueArea"] = "Business"
		fields["Microsoft.VSTS.Scheduling.TargetDate"] = ""
	}

	return &templates.Template{
		Name:        name,
		Description: fmt.Sprintf("Template for %s work items - edit me!", workItemType),
		Type:        workItemType,
		Fields:      fields,
	}
}
