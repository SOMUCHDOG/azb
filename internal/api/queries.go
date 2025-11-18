package api

import (
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// GetQuery retrieves a saved query by path
func (c *Client) GetQuery(queryPath string) (*workitemtracking.QueryHierarchyItem, error) {
	expand := workitemtracking.QueryExpandValues.Wiql

	query, err := c.workItemClient.GetQuery(c.ctx, workitemtracking.GetQueryArgs{
		Project: &c.project,
		Query:   &queryPath,
		Expand:  &expand,
		Depth:   nil,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get query '%s': %w", queryPath, err)
	}

	return query, nil
}

// ListQueries retrieves all queries in a folder
func (c *Client) ListQueries(folderPath string, depth int) (*[]workitemtracking.QueryHierarchyItem, error) {
	queryDepth := depth

	queries, err := c.workItemClient.GetQueries(c.ctx, workitemtracking.GetQueriesArgs{
		Project: &c.project,
		Depth:   &queryDepth,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list queries: %w", err)
	}

	return queries, nil
}

// ExecuteQuery executes a saved query and returns work items
func (c *Client) ExecuteQuery(queryId string, top int) (*[]workitemtracking.WorkItem, error) {
	// Get the query with WIQL
	expand := workitemtracking.QueryExpandValues.Wiql

	query, err := c.workItemClient.GetQuery(c.ctx, workitemtracking.GetQueryArgs{
		Project: &c.project,
		Query:   &queryId,
		Expand:  &expand,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}

	if query.Wiql == nil {
		return nil, fmt.Errorf("query does not have a WIQL statement")
	}

	// Execute the query
	return c.ListWorkItems(*query.Wiql, top)
}
