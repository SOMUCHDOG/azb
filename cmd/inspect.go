package cmd

import (
	"fmt"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/auth"
	"github.com/casey/azure-boards-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <work-item-type>",
		Short: "Inspect work item type fields",
		Long:  `Show all fields and requirements for a work item type.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runInspect,
	}
)

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	workItemTypeName := args[0]

	// Check authentication
	token, err := auth.GetToken()
	if err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get organization and project
	org := viper.GetString("organization")
	if org == "" {
		org = cfg.Organization
	}
	if org == "" {
		return fmt.Errorf("organization not configured")
	}

	project := viper.GetString("project")
	if project == "" {
		project = cfg.Project
	}
	if project == "" {
		return fmt.Errorf("project not configured")
	}

	// Build organization URL
	orgURL := api.NormalizeOrganizationURL(org)

	// Create API client
	client, err := api.NewClient(orgURL, project, token)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get work item type
	workItemType, err := client.GetWorkItemType(workItemTypeName)
	if err != nil {
		return err
	}

	fmt.Printf("Work Item Type: %s\n", workItemTypeName)
	if workItemType.Description != nil {
		fmt.Printf("Description: %s\n", *workItemType.Description)
	}
	fmt.Println()

	// Get required fields
	requiredFields, err := client.GetRequiredFields(workItemTypeName)
	if err != nil {
		return err
	}

	fmt.Println("Required Fields:")
	if len(requiredFields) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, fieldName := range requiredFields {
			fieldDef, err := client.GetFieldDefinition(workItemTypeName, fieldName)
			if err != nil {
				fmt.Printf("  - %s\n", fieldName)
				continue
			}

			displayName := fieldName
			if fieldDef.Name != nil {
				displayName = *fieldDef.Name
			}

			helpText := ""
			if fieldDef.HelpText != nil {
				helpText = fmt.Sprintf(" (%s)", *fieldDef.HelpText)
			}

			fmt.Printf("  - %s [%s]%s\n", displayName, fieldName, helpText)
		}
	}
	fmt.Println()

	// Show all fields
	fmt.Println("All Fields:")
	if workItemType.Fields != nil {
		for _, field := range *workItemType.Fields {
			if field.Name == nil || field.ReferenceName == nil {
				continue
			}

			required := ""
			if field.AlwaysRequired != nil && *field.AlwaysRequired {
				required = " [REQUIRED]"
			}

			defaultValue := ""
			if field.DefaultValue != nil {
				defaultValue = fmt.Sprintf(" (default: %v)", *field.DefaultValue)
			}

			fmt.Printf("  - %s [%s]%s%s\n", *field.Name, *field.ReferenceName, required, defaultValue)
		}
	}

	return nil
}
