package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/auth"
	"github.com/casey/azure-boards-cli/internal/config"
)

var (
	queryFormatFlag string
	queryLimitFlag  int

	queryCmd = &cobra.Command{
		Use:   "query",
		Short: "Manage and execute saved queries",
		Long:  `List, show, and execute saved Azure Boards queries.`,
	}

	queryListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all saved queries",
		Long:  `List all saved queries in your Azure Boards project.`,
		RunE:  runQueryList,
	}

	queryShowCmd = &cobra.Command{
		Use:   "show <query-name>",
		Short: "Show query details",
		Long:  `Show details of a saved query including its WIQL statement.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runQueryShow,
	}

	queryRunCmd = &cobra.Command{
		Use:   "run <query-name>",
		Short: "Execute a saved query",
		Long:  `Execute a saved query and display the results. Supports both personal and shared queries.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runQueryRun,
	}
)

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.AddCommand(queryListCmd)
	queryCmd.AddCommand(queryShowCmd)
	queryCmd.AddCommand(queryRunCmd)

	// Flags for query list
	queryListCmd.Flags().StringVar(&queryFormatFlag, "format", "table", "Output format (table, json)")

	// Flags for query show
	queryShowCmd.Flags().StringVar(&queryFormatFlag, "format", "text", "Output format (text, json)")

	// Flags for query run
	queryRunCmd.Flags().StringVar(&queryFormatFlag, "format", "table", "Output format (table, json, csv, ids)")
	queryRunCmd.Flags().IntVar(&queryLimitFlag, "limit", 50, "Maximum number of results")
}

func runQueryList(cmd *cobra.Command, args []string) error {
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

	// List queries with depth 2 to get folders and their contents
	queries, err := client.ListQueries("", 2)
	if err != nil {
		return fmt.Errorf("failed to list queries: %w", err)
	}

	// Output based on format
	switch queryFormatFlag {
	case "json":
		return outputQueryJSON(queries)
	case "table":
		return outputQueryTable(queries)
	default:
		return fmt.Errorf("unsupported format: %s", queryFormatFlag)
	}
}

func runQueryShow(cmd *cobra.Command, args []string) error {
	queryName := args[0]

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

	// Find the query
	query, err := findQueryByName(client, queryName)
	if err != nil {
		return err
	}

	// Output based on format
	switch queryFormatFlag {
	case "json":
		data, err := json.MarshalIndent(query, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
	case "text":
		fmt.Printf("Name: %s\n", *query.Name)
		if query.Path != nil {
			fmt.Printf("Path: %s\n", *query.Path)
		}
		if query.Id != nil {
			fmt.Printf("ID: %s\n", query.Id.String())
		}
		if query.IsPublic != nil {
			queryType := "Personal"
			if *query.IsPublic {
				queryType = "Shared"
			}
			fmt.Printf("Type: %s\n", queryType)
		}
		if query.Wiql != nil {
			fmt.Printf("\nWIQL:\n%s\n", *query.Wiql)
		}
	default:
		return fmt.Errorf("unsupported format: %s", queryFormatFlag)
	}

	return nil
}

func runQueryRun(cmd *cobra.Command, args []string) error {
	queryName := args[0]

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

	// Find the query
	query, err := findQueryByName(client, queryName)
	if err != nil {
		return err
	}

	// Execute the query
	if query.Id == nil {
		return fmt.Errorf("query has no ID")
	}

	workItemsPtr, err := client.ExecuteQuery(query.Id.String(), queryLimitFlag)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	// Dereference the pointer for output functions
	var workItems []workitemtracking.WorkItem
	if workItemsPtr != nil {
		workItems = *workItemsPtr
	}

	// Output based on format (reuse output functions from list.go)
	switch queryFormatFlag {
	case "table":
		return outputTable(workItems)
	case "json":
		return outputJSON(workItems)
	case "csv":
		return outputCSV(workItems)
	case "ids":
		return outputIDs(workItems)
	default:
		return fmt.Errorf("unsupported format: %s", queryFormatFlag)
	}
}

// findQueryByName searches for a query by name in all folders
func findQueryByName(client *api.Client, name string) (*workitemtracking.QueryHierarchyItem, error) {
	// List all queries with depth 2 (max allowed by API)
	queries, err := client.ListQueries("", 2)
	if err != nil {
		return nil, fmt.Errorf("failed to list queries: %w", err)
	}

	// Search for the query by name
	var found *workitemtracking.QueryHierarchyItem
	var searchQueries func(items *[]workitemtracking.QueryHierarchyItem)

	searchQueries = func(items *[]workitemtracking.QueryHierarchyItem) {
		if items == nil {
			return
		}

		for i := range *items {
			item := &(*items)[i]

			// Check if this is the query we're looking for
			if item.Name != nil && strings.EqualFold(*item.Name, name) {
				// Only match actual queries, not folders
				// If IsFolder is nil or false, it's a query
				isFolder := item.IsFolder != nil && *item.IsFolder
				if !isFolder {
					found = item
					return
				}
			}

			// Recursively search children
			if item.Children != nil {
				searchQueries(item.Children)
			}

			if found != nil {
				return
			}
		}
	}

	searchQueries(queries)

	if found == nil {
		return nil, fmt.Errorf("query '%s' not found", name)
	}

	// Get the full query details including WIQL
	if found.Id == nil {
		return nil, fmt.Errorf("query has no ID")
	}

	fullQuery, err := client.GetQuery(found.Id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get query details: %w", err)
	}

	return fullQuery, nil
}

// outputQueryTable outputs queries in table format
func outputQueryTable(queries *[]workitemtracking.QueryHierarchyItem) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tTYPE\tPATH")

	var printQueries func(items *[]workitemtracking.QueryHierarchyItem, indent string)
	printQueries = func(items *[]workitemtracking.QueryHierarchyItem, indent string) {
		if items == nil {
			return
		}

		for _, item := range *items {
			itemType := "Query"
			if item.IsFolder != nil && *item.IsFolder {
				itemType = "Folder"
			}

			name := ""
			if item.Name != nil {
				name = indent + *item.Name
			}

			path := ""
			if item.Path != nil {
				path = *item.Path
			}

			fmt.Fprintf(w, "%s\t%s\t%s\n", name, itemType, path)

			// Print children with indentation
			if item.Children != nil {
				printQueries(item.Children, indent+"  ")
			}
		}
	}

	printQueries(queries, "")

	return nil
}

// outputQueryJSON outputs queries in JSON format
func outputQueryJSON(queries *[]workitemtracking.QueryHierarchyItem) error {
	data, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
