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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	deleteForceFlag bool

	deleteCmd = &cobra.Command{
		Use:   "delete <id> [id2,id3...]",
		Short: "Delete work item(s)",
		Long:  `Delete one or more work items. Provide a single ID or comma-separated IDs for bulk deletion.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runDelete,
	}
)

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteForceFlag, "force", "f", false, "Skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	// Confirmation prompt unless --force is specified
	if !deleteForceFlag {
		// Show work items to be deleted
		fmt.Println("The following work items will be deleted:")
		for _, id := range ids {
			// Try to get work item details for confirmation
			workItem, err := client.GetWorkItem(id)
			if err != nil {
				fmt.Printf("  - ID %d (unable to fetch details)\n", id)
			} else {
				title := ""
				if workItem.Fields != nil {
					if titleValue, ok := (*workItem.Fields)["System.Title"]; ok {
						title = fmt.Sprintf("%v", titleValue)
					}
				}
				fmt.Printf("  - ID %d: %s\n", id, title)
			}
		}

		fmt.Println()
		fmt.Print("Are you sure you want to delete these work items? This cannot be undone. (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete each work item
	var successCount, failCount int
	for _, id := range ids {
		err := client.DeleteWorkItem(id)
		if err != nil {
			fmt.Printf("✗ Failed to delete work item %d: %v\n", id, err)
			failCount++
			continue
		}

		fmt.Printf("✓ Deleted work item %d\n", id)
		successCount++
	}

	// Summary
	fmt.Printf("\nSummary: %d deleted, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("some work items failed to delete")
	}

	return nil
}
