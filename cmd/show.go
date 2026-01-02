package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/SOMUCHDOG/azb/internal/api"
	"github.com/SOMUCHDOG/azb/internal/auth"
	"github.com/SOMUCHDOG/azb/internal/config"
)

var (
	showFormatFlag   string
	showCommentsFlag bool
	showHistoryFlag  bool

	showCmd = &cobra.Command{
		Use:   "show <id>",
		Short: "Show work item details",
		Long:  `Display detailed information about a work item.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runShow,
	}
)

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().StringVarP(&showFormatFlag, "format", "f", "text", "Output format (text, json)")
	showCmd.Flags().BoolVar(&showCommentsFlag, "comments", false, "Show comments")
	showCmd.Flags().BoolVar(&showHistoryFlag, "history", false, "Show history")
}

func runShow(cmd *cobra.Command, args []string) error {
	// Parse work item ID
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid work item ID: %s", args[0])
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

	// Get work item
	workItem, err := client.GetWorkItem(id)
	if err != nil {
		return fmt.Errorf("failed to get work item: %w", err)
	}

	// Output based on format
	switch showFormatFlag {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(workItem)
	case "text":
		fallthrough
	default:
		return displayWorkItem(workItem)
	}
}

func displayWorkItem(workItem interface{}) error {
	// This is a simplified version - in reality, we'd need to properly format
	// the work item fields
	fmt.Println("Work Item Details:")
	fmt.Printf("%+v\n", workItem)
	return nil
}
