package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/auth"
	"github.com/casey/azure-boards-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	updateTitleFlag       string
	updateDescriptionFlag string
	updateStateFlag       string
	updateAssignedToFlag  string
	updateAreaPathFlag    string
	updateIterationFlag   string
	updatePriorityFlag    int
	updateAddTagsFlag     string
	updateRemoveTagsFlag  string
	updateFieldsFlag      []string
	updateInteractiveFlag bool

	updateCmd = &cobra.Command{
		Use:   "update <id> [id2,id3...]",
		Short: "Update work item(s)",
		Long:  `Update one or more work items. Provide a single ID or comma-separated IDs for bulk updates.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runUpdate,
	}
)

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVar(&updateTitleFlag, "title", "", "Update title")
	updateCmd.Flags().StringVar(&updateDescriptionFlag, "description", "", "Update description")
	updateCmd.Flags().StringVar(&updateStateFlag, "state", "", "Update state (e.g., Active, Resolved, Closed)")
	updateCmd.Flags().StringVar(&updateAssignedToFlag, "assigned-to", "", "Update assigned to (@me for current user)")
	updateCmd.Flags().StringVar(&updateAreaPathFlag, "area-path", "", "Update area path")
	updateCmd.Flags().StringVar(&updateIterationFlag, "iteration", "", "Update iteration path")
	updateCmd.Flags().IntVar(&updatePriorityFlag, "priority", 0, "Update priority (1-4)")
	updateCmd.Flags().StringVar(&updateAddTagsFlag, "add-tag", "", "Add tags (comma-separated)")
	updateCmd.Flags().StringVar(&updateRemoveTagsFlag, "remove-tag", "", "Remove tags (comma-separated)")
	updateCmd.Flags().StringArrayVar(&updateFieldsFlag, "field", []string{}, "Update custom field in format 'FieldName=value' (can be repeated)")
	updateCmd.Flags().BoolVarP(&updateInteractiveFlag, "interactive", "i", false, "Interactive edit mode (prompts for each field)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Parse work item IDs (supports single ID or comma-separated list)
	idsStr := args[0]
	idsList := strings.Split(idsStr, ",")

	var ids []int
	for _, idStr := range idsList {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return fmt.Errorf("invalid work item ID: %s", idStr)
		}
		ids = append(ids, id)
	}

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

	// Interactive mode only works with single ID
	if updateInteractiveFlag {
		if len(ids) > 1 {
			return fmt.Errorf("interactive mode only supports a single work item ID")
		}
		return runInteractiveUpdate(client, ids[0])
	}

	// Build fields to update
	fields := make(map[string]interface{})

	if updateTitleFlag != "" {
		fields["System.Title"] = updateTitleFlag
	}

	if updateDescriptionFlag != "" {
		fields["System.Description"] = updateDescriptionFlag
	}

	if updateStateFlag != "" {
		fields["System.State"] = updateStateFlag
	}

	if updateAssignedToFlag != "" {
		if updateAssignedToFlag == "@me" {
			// Clear assigned to, Azure DevOps will set to current user
			fields["System.AssignedTo"] = ""
		} else {
			fields["System.AssignedTo"] = updateAssignedToFlag
		}
	}

	if updateAreaPathFlag != "" {
		fields["System.AreaPath"] = updateAreaPathFlag
	}

	if updateIterationFlag != "" {
		fields["System.IterationPath"] = updateIterationFlag
	}

	if updatePriorityFlag > 0 {
		fields["Microsoft.VSTS.Common.Priority"] = updatePriorityFlag
	}

	// Parse custom fields
	for _, fieldArg := range updateFieldsFlag {
		parts := strings.SplitN(fieldArg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid field format '%s', expected 'FieldName=value'", fieldArg)
		}
		fields[parts[0]] = parts[1]
	}

	// Handle tag operations separately since they require reading current tags
	hasTagOperation := updateAddTagsFlag != "" || updateRemoveTagsFlag != ""

	// Check if any fields to update
	if len(fields) == 0 && !hasTagOperation {
		return fmt.Errorf("no fields to update. Specify at least one --field flag")
	}

	// Update each work item
	var successCount, failCount int
	for _, id := range ids {
		// Handle tag operations
		updateFields := make(map[string]interface{})
		for k, v := range fields {
			updateFields[k] = v
		}

		if hasTagOperation {
			// Get current work item to read tags
			workItem, err := client.GetWorkItem(id)
			if err != nil {
				fmt.Printf("✗ Failed to get work item %d: %v\n", id, err)
				failCount++
				continue
			}

			// Get current tags
			currentTags := ""
			if workItem.Fields != nil {
				if tagsValue, ok := (*workItem.Fields)["System.Tags"]; ok {
					if tagsStr, ok := tagsValue.(string); ok {
						currentTags = tagsStr
					}
				}
			}

			// Process tag updates
			newTags := processTagUpdates(currentTags, updateAddTagsFlag, updateRemoveTagsFlag)
			updateFields["System.Tags"] = newTags
		}

		// Update work item
		_, err := client.UpdateWorkItem(id, updateFields)
		if err != nil {
			fmt.Printf("✗ Failed to update work item %d: %v\n", id, err)
			failCount++
			continue
		}

		fmt.Printf("✓ Updated work item %d\n", id)
		successCount++
	}

	// Summary
	fmt.Printf("\nSummary: %d updated, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("some work items failed to update")
	}

	return nil
}

// processTagUpdates adds and removes tags from the current tag string
func processTagUpdates(currentTags, addTags, removeTags string) string {
	// Parse current tags
	tagMap := make(map[string]bool)
	if currentTags != "" {
		for _, tag := range strings.Split(currentTags, ";") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagMap[tag] = true
			}
		}
	}

	// Remove tags
	if removeTags != "" {
		for _, tag := range strings.Split(removeTags, ",") {
			tag = strings.TrimSpace(tag)
			delete(tagMap, tag)
		}
	}

	// Add tags
	if addTags != "" {
		for _, tag := range strings.Split(addTags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagMap[tag] = true
			}
		}
	}

	// Convert back to string
	var tags []string
	for tag := range tagMap {
		tags = append(tags, tag)
	}

	return strings.Join(tags, "; ")
}

// runInteractiveUpdate prompts the user for each field to update
func runInteractiveUpdate(client *api.Client, id int) error {
	fmt.Printf("Interactive update for work item %d\n", id)
	fmt.Println("Leave blank to keep current value, enter new value to update")
	fmt.Println()

	// Get current work item
	workItem, err := client.GetWorkItem(id)
	if err != nil {
		return fmt.Errorf("failed to get work item: %w", err)
	}

	fields := make(map[string]interface{})

	// Helper to get current field value
	getCurrentValue := func(fieldName string) string {
		if workItem.Fields == nil {
			return ""
		}
		if value, ok := (*workItem.Fields)[fieldName]; ok {
			return fmt.Sprintf("%v", value)
		}
		return ""
	}

	// Title
	currentTitle := getCurrentValue("System.Title")
	fmt.Printf("Title [%s]: ", currentTitle)
	newTitle, _ := promptOptional("") // Error ignored - user input is optional
	if newTitle != "" {
		fields["System.Title"] = newTitle
	}

	// Description
	currentDesc := getCurrentValue("System.Description")
	descPreview := currentDesc
	if len(descPreview) > 50 {
		descPreview = descPreview[:47] + "..."
	}
	fmt.Printf("Description [%s]: ", descPreview)
	newDesc, _ := promptOptional("") // Error ignored - user input is optional
	if newDesc != "" {
		fields["System.Description"] = newDesc
	}

	// State
	currentState := getCurrentValue("System.State")
	fmt.Printf("State [%s]: ", currentState)
	newState, _ := promptOptional("") // Error ignored - user input is optional
	if newState != "" {
		fields["System.State"] = newState
	}

	// Assigned To
	currentAssignedTo := getCurrentValue("System.AssignedTo")
	fmt.Printf("Assigned To [%s]: ", currentAssignedTo)
	newAssignedTo, _ := promptOptional("") // Error ignored - user input is optional
	if newAssignedTo != "" {
		if newAssignedTo == "@me" {
			fields["System.AssignedTo"] = ""
		} else {
			fields["System.AssignedTo"] = newAssignedTo
		}
	}

	// Tags
	currentTags := getCurrentValue("System.Tags")
	fmt.Printf("Tags [%s]: ", currentTags)
	newTags, _ := promptOptional("") // Error ignored - user input is optional
	if newTags != "" {
		fields["System.Tags"] = newTags
	}

	// Priority
	currentPriority := getCurrentValue("Microsoft.VSTS.Common.Priority")
	fmt.Printf("Priority [%s]: ", currentPriority)
	newPriority, _ := promptOptional("") // Error ignored - user input is optional
	if newPriority != "" {
		priority, err := strconv.Atoi(newPriority)
		if err == nil && priority >= 1 && priority <= 4 {
			fields["Microsoft.VSTS.Common.Priority"] = priority
		}
	}

	// Check if any fields to update
	if len(fields) == 0 {
		fmt.Println("\nNo changes made")
		return nil
	}

	// Confirm update
	fmt.Println("\nFields to update:")
	for fieldName, value := range fields {
		fmt.Printf("  %s = %v\n", fieldName, value)
	}

	fmt.Print("\nUpdate work item? (y/N): ")
	confirm, _ := promptOptional("") // Error ignored - user input is optional
	if confirm != "y" && confirm != "Y" {
		fmt.Println("Update cancelled")
		return nil
	}

	// Update work item
	_, err = client.UpdateWorkItem(id, fields)
	if err != nil {
		return fmt.Errorf("failed to update work item: %w", err)
	}

	fmt.Printf("\n✓ Updated work item %d\n", id)

	return nil
}
