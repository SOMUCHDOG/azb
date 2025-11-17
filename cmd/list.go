package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/casey/azure-boards-cli/internal/api"
	"github.com/casey/azure-boards-cli/internal/auth"
	"github.com/casey/azure-boards-cli/internal/config"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	stateFlag      string
	assignedToFlag string
	typeFlag       string
	sprintFlag     string
	areaPathFlag   string
	tagsFlag       string
	formatFlag     string
	limitFlag      int

	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List work items",
		Long:  `List work items with optional filters.`,
		RunE:  runList,
	}
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&stateFlag, "state", "", "Filter by state (e.g., Active, Resolved, Closed)")
	listCmd.Flags().StringVar(&assignedToFlag, "assigned-to", "", "Filter by assigned to (@me for current user)")
	listCmd.Flags().StringVar(&typeFlag, "type", "", "Filter by work item type (e.g., Bug, Task, User Story)")
	listCmd.Flags().StringVar(&sprintFlag, "sprint", "", "Filter by sprint/iteration")
	listCmd.Flags().StringVar(&areaPathFlag, "area-path", "", "Filter by area path")
	listCmd.Flags().StringVar(&tagsFlag, "tags", "", "Filter by tags (comma-separated)")
	listCmd.Flags().StringVarP(&formatFlag, "format", "f", "table", "Output format (table, json, csv, ids)")
	listCmd.Flags().IntVarP(&limitFlag, "limit", "l", 50, "Maximum number of results")
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Override with flags if provided
	org := viper.GetString("organization")
	if org == "" {
		org = cfg.Organization
	}
	if org == "" {
		return fmt.Errorf("organization not configured. Run 'ab config set organization <org>'")
	}

	project := viper.GetString("project")
	if project == "" {
		project = cfg.Project
	}
	if project == "" {
		return fmt.Errorf("project not configured. Run 'ab config set project <project>'")
	}

	// Build organization URL
	orgURL := api.NormalizeOrganizationURL(org)

	// Create API client
	client, err := api.NewClient(orgURL, project, token)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Build WIQL query
	wiql := buildWIQLQuery(project)

	// Debug output
	if os.Getenv("DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "Organization URL: %s\n", orgURL)
		fmt.Fprintf(os.Stderr, "Project: %s\n", project)
		fmt.Fprintf(os.Stderr, "WIQL Query: %s\n", wiql)
		fmt.Fprintf(os.Stderr, "Limit: %d\n", limitFlag)
	}

	// Execute query
	workItems, err := client.ListWorkItems(wiql, limitFlag)
	if err != nil {
		return fmt.Errorf("failed to list work items: %w", err)
	}

	if workItems == nil || len(*workItems) == 0 {
		fmt.Println("No work items found")
		return nil
	}

	// Output results based on format
	return outputWorkItems(*workItems, formatFlag)
}

func buildWIQLQuery(project string) string {
	var conditions []string

	// Base query (limit is handled via API parameter, not in WIQL)
	query := fmt.Sprintf("SELECT [System.Id], [System.Title], [System.State], [System.AssignedTo], [System.WorkItemType] FROM WorkItems WHERE [System.TeamProject] = '%s'", project)

	// Add filters
	if typeFlag != "" {
		conditions = append(conditions, fmt.Sprintf("[System.WorkItemType] = '%s'", typeFlag))
	}

	if stateFlag != "" {
		conditions = append(conditions, fmt.Sprintf("[System.State] = '%s'", stateFlag))
	}

	if assignedToFlag != "" {
		if assignedToFlag == "@me" {
			conditions = append(conditions, "[System.AssignedTo] = @me")
		} else {
			conditions = append(conditions, fmt.Sprintf("[System.AssignedTo] = '%s'", assignedToFlag))
		}
	}

	if sprintFlag != "" {
		if sprintFlag == "current" || sprintFlag == "@current" {
			conditions = append(conditions, "[System.IterationPath] = @currentIteration")
		} else {
			conditions = append(conditions, fmt.Sprintf("[System.IterationPath] = '%s'", sprintFlag))
		}
	}

	if areaPathFlag != "" {
		conditions = append(conditions, fmt.Sprintf("[System.AreaPath] = '%s'", areaPathFlag))
	}

	if tagsFlag != "" {
		tags := strings.Split(tagsFlag, ",")
		for _, tag := range tags {
			conditions = append(conditions, fmt.Sprintf("[System.Tags] CONTAINS '%s'", strings.TrimSpace(tag)))
		}
	}

	// Add conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add order by
	query += " ORDER BY [System.ChangedDate] DESC"

	return query
}

func outputWorkItems(workItems interface{}, format string) error {
	switch format {
	case "json":
		return outputJSON(workItems)
	case "csv":
		return outputCSV(workItems)
	case "ids":
		return outputIDs(workItems)
	case "table":
		fallthrough
	default:
		return outputTable(workItems)
	}
}

func outputJSON(workItems interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(workItems)
}

func outputCSV(workItems interface{}) error {
	items, ok := workItems.([]workitemtracking.WorkItem)
	if !ok {
		return fmt.Errorf("invalid work items type")
	}

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Title", "Type", "State", "Assigned To"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, item := range items {
		id := ""
		if item.Id != nil {
			id = fmt.Sprintf("%d", *item.Id)
		}

		title := getFieldValue(item.Fields, "System.Title")
		workItemType := getFieldValue(item.Fields, "System.WorkItemType")
		state := getFieldValue(item.Fields, "System.State")
		assignedTo := getFieldValue(item.Fields, "System.AssignedTo")

		row := []string{id, title, workItemType, state, assignedTo}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func outputIDs(workItems interface{}) error {
	items, ok := workItems.([]workitemtracking.WorkItem)
	if !ok {
		return fmt.Errorf("invalid work items type")
	}

	for _, item := range items {
		if item.Id != nil {
			fmt.Println(*item.Id)
		}
	}
	return nil
}

func outputTable(workItems interface{}) error {
	items, ok := workItems.([]workitemtracking.WorkItem)
	if !ok {
		return fmt.Errorf("invalid work items type")
	}

	if len(items) == 0 {
		fmt.Println("No work items found")
		return nil
	}

	// Print header
	fmt.Printf("%-8s %-50s %-15s %-15s %-30s\n", "ID", "Title", "Type", "State", "Assigned To")
	fmt.Println(strings.Repeat("-", 120))

	// Print rows
	for _, item := range items {
		id := ""
		if item.Id != nil {
			id = fmt.Sprintf("%d", *item.Id)
		}

		title := getFieldValue(item.Fields, "System.Title")
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		workItemType := getFieldValue(item.Fields, "System.WorkItemType")
		state := getFieldValue(item.Fields, "System.State")
		assignedTo := getFieldValue(item.Fields, "System.AssignedTo")
		if len(assignedTo) > 30 {
			assignedTo = assignedTo[:27] + "..."
		}

		fmt.Printf("%-8s %-50s %-15s %-15s %-30s\n", id, title, workItemType, state, assignedTo)
	}

	fmt.Printf("\nTotal: %d work items\n", len(items))

	return nil
}

func getFieldValue(fields *map[string]interface{}, fieldName string) string {
	if fields == nil {
		return ""
	}

	value, ok := (*fields)[fieldName]
	if !ok {
		return ""
	}

	// Handle different field types
	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}:
		// For user fields like AssignedTo, extract display name
		if displayName, ok := v["displayName"].(string); ok {
			return displayName
		}
		if uniqueName, ok := v["uniqueName"].(string); ok {
			return uniqueName
		}
	}

	return fmt.Sprintf("%v", value)
}
