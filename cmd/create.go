package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/auth"
	"github.com/casey/azure-boards-cli/internal/config"
	"github.com/casey/azure-boards-cli/internal/templates"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	createTypeFlag        string
	createTitleFlag       string
	createDescriptionFlag string
	createAssignedToFlag  string
	createAreaPathFlag    string
	createIterationFlag   string
	createPriorityFlag    int
	createTagsFlag        string
	createFieldsFlag      []string
	createTemplateFlag    string
	createParentIDFlag    int

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new work item",
		Long:  `Create a new work item in Azure Boards. Run without flags for interactive mode.`,
		RunE:  runCreate,
	}
)

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVar(&createTypeFlag, "type", "", "Work item type (Bug, Task, User Story, etc.)")
	createCmd.Flags().StringVar(&createTitleFlag, "title", "", "Work item title")
	createCmd.Flags().StringVar(&createDescriptionFlag, "description", "", "Work item description")
	createCmd.Flags().StringVar(&createAssignedToFlag, "assigned-to", "", "Assign to user (@me for current user)")
	createCmd.Flags().StringVar(&createAreaPathFlag, "area-path", "", "Area path")
	createCmd.Flags().StringVar(&createIterationFlag, "iteration", "", "Iteration path")
	createCmd.Flags().IntVar(&createPriorityFlag, "priority", 0, "Priority (1-4)")
	createCmd.Flags().StringVar(&createTagsFlag, "tags", "", "Tags (comma-separated)")
	createCmd.Flags().StringArrayVar(&createFieldsFlag, "field", []string{}, "Custom field in format 'FieldName=value' (can be repeated)")
	createCmd.Flags().StringVarP(&createTemplateFlag, "template", "t", "", "Use a template")
	createCmd.Flags().IntVar(&createParentIDFlag, "parent-id", 0, "Parent work item ID (to create as a child)")
}

func runCreate(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("organization not configured. Run 'azb config set organization <org>'")
	}

	project := viper.GetString("project")
	if project == "" {
		project = cfg.Project
	}
	if project == "" {
		return fmt.Errorf("project not configured. Run 'azb config set project <project>'")
	}

	// Build organization URL
	orgURL := api.NormalizeOrganizationURL(org)

	// Create API client
	client, err := api.NewClient(orgURL, project, token)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Load template if specified
	var template *templates.Template
	if createTemplateFlag != "" {
		template, err = templates.Load(createTemplateFlag)
		if err != nil {
			return fmt.Errorf("failed to load template: %w", err)
		}
		fmt.Printf("Using template: %s\n", template.Name)
		if template.Description != "" {
			fmt.Printf("Description: %s\n", template.Description)
		}
		fmt.Println()
	}

	// Determine if interactive mode or CLI mode
	isInteractive := createTitleFlag == "" && createTypeFlag == "" && createTemplateFlag == ""

	var workItemType, title, description, assignedTo, areaPath, iteration, tags string
	var priority int
	customFields := make(map[string]string)

	// Apply template defaults
	if template != nil {
		workItemType = template.Type
		for fieldName, fieldValue := range template.Fields {
			switch fieldName {
			case "System.Title":
				title = fmt.Sprintf("%v", fieldValue)
			case "System.Description":
				description = fmt.Sprintf("%v", fieldValue)
			case "System.AssignedTo":
				assignedTo = fmt.Sprintf("%v", fieldValue)
			case "System.AreaPath":
				areaPath = fmt.Sprintf("%v", fieldValue)
			case "System.IterationPath":
				iteration = fmt.Sprintf("%v", fieldValue)
			case "System.Tags":
				tags = fmt.Sprintf("%v", fieldValue)
			case "Microsoft.VSTS.Common.Priority":
				if p, ok := fieldValue.(int); ok {
					priority = p
				}
			default:
				customFields[fieldName] = fmt.Sprintf("%v", fieldValue)
			}
		}
	}

	// Get work item type first (skip if from template)
	if workItemType == "" {
		if isInteractive {
			workItemType, err = promptWorkItemType()
			if err != nil {
				return err
			}
		} else {
			if createTypeFlag == "" {
				return fmt.Errorf("--type is required")
			}
			workItemType = createTypeFlag
		}
	}

	// Discover required fields for this work item type
	requiredFields, err := client.GetRequiredFields(workItemType)
	if err != nil {
		// If we can't get required fields, continue with defaults
		fmt.Fprintf(os.Stderr, "Warning: Could not determine required fields: %v\n", err)
		requiredFields = []string{}
	}

	if isInteractive {
		// Interactive mode
		fmt.Println("\nCreate New Work Item")
		fmt.Println("====================")
		fmt.Printf("Type: %s\n\n", workItemType)

		title, err = promptRequired("Title")
		if err != nil {
			return err
		}

		description, err = promptOptional("Description")
		if err != nil {
			return err
		}

		assignedTo, err = promptOptional("Assigned To (leave empty or use @me)")
		if err != nil {
			return err
		}

		areaPath, err = promptOptional(fmt.Sprintf("Area Path (default: %s)", cfg.DefaultAreaPath))
		if err != nil {
			return err
		}
		if areaPath == "" {
			areaPath = cfg.DefaultAreaPath
		}

		iteration, err = promptOptional(fmt.Sprintf("Iteration (default: %s)", cfg.DefaultIteration))
		if err != nil {
			return err
		}
		if iteration == "" {
			iteration = cfg.DefaultIteration
		}

		priorityStr, err := promptOptional("Priority (1-4, default: 2)")
		if err != nil {
			return err
		}
		if priorityStr == "" {
			priority = 2
		} else {
			priority, err = strconv.Atoi(priorityStr)
			if err != nil || priority < 1 || priority > 4 {
				return fmt.Errorf("priority must be between 1 and 4")
			}
		}

		tags, err = promptOptional("Tags (comma-separated)")
		if err != nil {
			return err
		}

		// Prompt for custom required fields
		for _, fieldRef := range requiredFields {
			// Skip fields we already handle
			if isStandardField(fieldRef) {
				continue
			}

			// Get field definition for display name
			fieldDef, err := client.GetFieldDefinition(workItemType, fieldRef)
			displayName := fieldRef
			helpText := ""
			if err == nil && fieldDef.Name != nil {
				displayName = *fieldDef.Name
				if fieldDef.HelpText != nil {
					helpText = fmt.Sprintf(" (%s)", *fieldDef.HelpText)
				}
			}

			value, err := promptRequired(fmt.Sprintf("%s [REQUIRED]%s", displayName, helpText))
			if err != nil {
				return err
			}
			customFields[fieldRef] = value
		}

	} else {
		// CLI mode - override template values only if flags are provided
		// Template values are already loaded above, we only override if flag is non-empty

		if createTitleFlag != "" {
			title = createTitleFlag
		} else if title == "" {
			// No template and no flag
			return fmt.Errorf("--title is required")
		}

		// Override with flags only if provided
		if createDescriptionFlag != "" {
			description = createDescriptionFlag
		}
		if createAssignedToFlag != "" {
			assignedTo = createAssignedToFlag
		}
		if createAreaPathFlag != "" {
			areaPath = createAreaPathFlag
		}
		if createIterationFlag != "" {
			iteration = createIterationFlag
		}
		if createPriorityFlag > 0 {
			priority = createPriorityFlag
		}
		if createTagsFlag != "" {
			tags = createTagsFlag
		}

		// Apply defaults from config if still empty
		if areaPath == "" {
			areaPath = cfg.DefaultAreaPath
		}
		if iteration == "" {
			iteration = cfg.DefaultIteration
		}
		if priority == 0 {
			priority = 2
		}

		// Parse custom fields from --field flags (these override template custom fields)
		for _, fieldArg := range createFieldsFlag {
			parts := strings.SplitN(fieldArg, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format '%s', expected 'FieldName=value'", fieldArg)
			}
			customFields[parts[0]] = parts[1]
		}
	}

	// Build fields map
	fields := make(map[string]interface{})

	fields["System.Title"] = title

	if description != "" {
		fields["System.Description"] = description
	}

	if assignedTo != "" {
		if assignedTo == "@me" {
			// Leave empty to assign to current user
			// Azure DevOps will automatically set it
		} else {
			fields["System.AssignedTo"] = assignedTo
		}
	}

	if areaPath != "" {
		fields["System.AreaPath"] = areaPath
	}

	if iteration != "" {
		fields["System.IterationPath"] = iteration
	}

	if priority > 0 {
		fields["Microsoft.VSTS.Common.Priority"] = priority
	}

	if tags != "" {
		fields["System.Tags"] = tags
	}

	// Add custom fields
	for fieldRef, value := range customFields {
		fields[fieldRef] = value
	}

	// Determine parent ID (flag takes precedence over template)
	parentID := createParentIDFlag
	if parentID == 0 && template != nil && template.Relations != nil {
		parentID = template.Relations.ParentID
	}

	// Create the work item
	workItem, err := client.CreateWorkItem(workItemType, fields, parentID)
	if err != nil {
		return fmt.Errorf("failed to create work item: %w", err)
	}

	// Display result
	fmt.Println("\n✓ Work item created successfully!")
	if workItem.Id != nil {
		fmt.Printf("  ID: %d\n", *workItem.Id)
	}
	fmt.Printf("  Type: %s\n", workItemType)
	fmt.Printf("  Title: %s\n", title)
	if parentID > 0 {
		fmt.Printf("  Parent ID: %d\n", parentID)
	}
	if workItem.Url != nil {
		fmt.Printf("  URL: %s\n", *workItem.Url)
	}

	// Create child work items if specified in template
	if template != nil && template.Relations != nil && len(template.Relations.Children) > 0 && workItem.Id != nil {
		fmt.Printf("\nCreating %d child work items...\n", len(template.Relations.Children))

		for i, child := range template.Relations.Children {
			childFields := make(map[string]interface{})

			// Use child-specific values or defaults
			childType := child.Type
			if childType == "" {
				childType = "Task" // Default child type
			}

			childFields["System.Title"] = child.Title

			if child.Description != "" {
				childFields["System.Description"] = child.Description
			}

			if child.AssignedTo != "" {
				if child.AssignedTo == "@me" {
					// Leave empty for current user
				} else {
					childFields["System.AssignedTo"] = child.AssignedTo
				}
			}

			// Add any additional fields from the child template
			for fieldName, fieldValue := range child.Fields {
				childFields[fieldName] = fieldValue
			}

			// Inherit fields from parent if not specified in child
			// Inherit AreaPath
			if _, hasAreaPath := childFields["System.AreaPath"]; !hasAreaPath && areaPath != "" {
				childFields["System.AreaPath"] = areaPath
			}

			// Inherit IterationPath
			if _, hasIteration := childFields["System.IterationPath"]; !hasIteration && iteration != "" {
				childFields["System.IterationPath"] = iteration
			}

			// Inherit Custom.ApplicationName from parent
			if parentAppName, hasParentAppName := fields["Custom.ApplicationName"]; hasParentAppName {
				if _, hasChildAppName := childFields["Custom.ApplicationName"]; !hasChildAppName {
					childFields["Custom.ApplicationName"] = parentAppName
				}
			}

			// Inherit other custom fields from parent template if present
			for customFieldKey, customFieldValue := range customFields {
				if _, hasField := childFields[customFieldKey]; !hasField {
					childFields[customFieldKey] = customFieldValue
				}
			}

			// Create child work item with parent relationship
			childWorkItem, err := client.CreateWorkItem(childType, childFields, *workItem.Id)
			if err != nil {
				fmt.Printf("  ✗ Failed to create child %d (%s): %v\n", i+1, child.Title, err)
				continue
			}

			fmt.Printf("  ✓ Child %d created: ID %d - %s\n", i+1, *childWorkItem.Id, child.Title)
		}
	}

	return nil
}

func promptWorkItemType() (string, error) {
	fmt.Println("\nWork Item Type:")
	fmt.Println("  1. Bug")
	fmt.Println("  2. Task")
	fmt.Println("  3. User Story")
	fmt.Println("  4. Feature")
	fmt.Println("  5. Epic")
	fmt.Print("Choice (1-5): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)

	switch input {
	case "1":
		return "Bug", nil
	case "2":
		return "Task", nil
	case "3":
		return "User Story", nil
	case "4":
		return "Feature", nil
	case "5":
		return "Epic", nil
	default:
		return "", fmt.Errorf("invalid choice")
	}
}

func promptRequired(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s: ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		if input != "" {
			return input, nil
		}

		fmt.Println("This field is required. Please enter a value.")
	}
}

func promptOptional(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil
}

// isStandardField checks if a field reference name is a standard field we already handle
func isStandardField(fieldRef string) bool {
	standardFields := []string{
		"System.Title",
		"System.Description",
		"System.AssignedTo",
		"System.AreaPath",
		"System.AreaId",
		"System.IterationPath",
		"System.IterationId",
		"System.State",
		"System.Tags",
		"Microsoft.VSTS.Common.Priority",
		"Microsoft.VSTS.Common.ValueArea", // Has default value, will be auto-set
	}

	for _, standard := range standardFields {
		if fieldRef == standard {
			return true
		}
	}

	return false
}
