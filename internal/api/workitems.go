package api

import (
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// GetWorkItem retrieves a work item by ID
func (c *Client) GetWorkItem(id int) (*workitemtracking.WorkItem, error) {
	workItem, err := c.workItemClient.GetWorkItem(c.ctx, workitemtracking.GetWorkItemArgs{
		Id:     &id,
		Expand: &workitemtracking.WorkItemExpandValues.All,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get work item %d: %w", id, err)
	}

	return workItem, nil
}

// ListWorkItems retrieves a list of work items based on a WIQL query
func (c *Client) ListWorkItems(wiql string, top int) (*[]workitemtracking.WorkItem, error) {
	args := workitemtracking.QueryByWiqlArgs{
		Wiql: &workitemtracking.Wiql{
			Query: &wiql,
		},
		Project: &c.project,
	}

	// Add top parameter if specified
	if top > 0 {
		args.Top = &top
	}

	// Execute WIQL query
	queryResult, err := c.workItemClient.QueryByWiql(c.ctx, args)

	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if queryResult.WorkItems == nil || len(*queryResult.WorkItems) == 0 {
		return &[]workitemtracking.WorkItem{}, nil
	}

	// Extract work item IDs
	var ids []int
	for _, ref := range *queryResult.WorkItems {
		if ref.Id != nil {
			ids = append(ids, *ref.Id)
		}
	}

	if len(ids) == 0 {
		return &[]workitemtracking.WorkItem{}, nil
	}

	// Get full work item details using the batch endpoint
	// Note: We use Expand instead of Fields to get all data
	expand := workitemtracking.WorkItemExpandValues.All

	workItems, err := c.workItemClient.GetWorkItemsBatch(c.ctx, workitemtracking.GetWorkItemsBatchArgs{
		Project: &c.project,
		WorkItemGetRequest: &workitemtracking.WorkItemBatchGetRequest{
			Ids:    &ids,
			Expand: &expand,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get work items: %w", err)
	}

	return workItems, nil
}

// CreateWorkItem creates a new work item
func (c *Client) CreateWorkItem(workItemType string, fields map[string]interface{}) (*workitemtracking.WorkItem, error) {
	// Build JSON patch document
	var patchDocument []webapi.JsonPatchOperation

	for field, value := range fields {
		op := webapi.OperationValues.Add
		path := fmt.Sprintf("/fields/%s", field)
		patchDocument = append(patchDocument, webapi.JsonPatchOperation{
			Op:    &op,
			Path:  &path,
			Value: value,
		})
	}

	validateOnly := false
	// Create work item
	workItem, err := c.workItemClient.CreateWorkItem(c.ctx, workitemtracking.CreateWorkItemArgs{
		Document:     &patchDocument,
		Project:      &c.project,
		Type:         &workItemType,
		ValidateOnly: &validateOnly,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create work item: %w", err)
	}

	return workItem, nil
}

// UpdateWorkItem updates an existing work item
func (c *Client) UpdateWorkItem(id int, fields map[string]interface{}) (*workitemtracking.WorkItem, error) {
	// Build JSON patch document
	var patchDocument []webapi.JsonPatchOperation

	for field, value := range fields {
		op := webapi.OperationValues.Replace
		path := fmt.Sprintf("/fields/%s", field)
		patchDocument = append(patchDocument, webapi.JsonPatchOperation{
			Op:    &op,
			Path:  &path,
			Value: value,
		})
	}

	validateOnly := false
	// Update work item
	workItem, err := c.workItemClient.UpdateWorkItem(c.ctx, workitemtracking.UpdateWorkItemArgs{
		Id:           &id,
		Document:     &patchDocument,
		ValidateOnly: &validateOnly,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update work item %d: %w", id, err)
	}

	return workItem, nil
}

// DeleteWorkItem deletes a work item
func (c *Client) DeleteWorkItem(id int) error {
	_, err := c.workItemClient.DeleteWorkItem(c.ctx, workitemtracking.DeleteWorkItemArgs{
		Id: &id,
	})

	if err != nil {
		return fmt.Errorf("failed to delete work item %d: %w", id, err)
	}

	return nil
}
