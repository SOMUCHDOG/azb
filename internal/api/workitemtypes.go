package api

import (
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// GetWorkItemType retrieves the definition for a work item type
func (c *Client) GetWorkItemType(workItemTypeName string) (*workitemtracking.WorkItemType, error) {
	workItemType, err := c.workItemClient.GetWorkItemType(c.ctx, workitemtracking.GetWorkItemTypeArgs{
		Project: &c.project,
		Type:    &workItemTypeName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get work item type '%s': %w", workItemTypeName, err)
	}

	return workItemType, nil
}

// GetWorkItemTypes retrieves all work item types for the project
func (c *Client) GetWorkItemTypes() (*[]workitemtracking.WorkItemType, error) {
	workItemTypes, err := c.workItemClient.GetWorkItemTypes(c.ctx, workitemtracking.GetWorkItemTypesArgs{
		Project: &c.project,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get work item types: %w", err)
	}

	return workItemTypes, nil
}

// GetRequiredFields returns a list of required field reference names for a work item type
func (c *Client) GetRequiredFields(workItemTypeName string) ([]string, error) {
	workItemType, err := c.GetWorkItemType(workItemTypeName)
	if err != nil {
		return nil, err
	}

	var requiredFields []string

	if workItemType.Fields != nil {
		for _, field := range *workItemType.Fields {
			if field.AlwaysRequired != nil && *field.AlwaysRequired {
				if field.ReferenceName != nil {
					requiredFields = append(requiredFields, *field.ReferenceName)
				}
			}
		}
	}

	return requiredFields, nil
}

// GetFieldDefinition returns detailed information about a field for a work item type
func (c *Client) GetFieldDefinition(workItemTypeName, fieldReferenceName string) (*workitemtracking.WorkItemTypeFieldInstance, error) {
	workItemType, err := c.GetWorkItemType(workItemTypeName)
	if err != nil {
		return nil, err
	}

	if workItemType.Fields != nil {
		for _, field := range *workItemType.Fields {
			if field.ReferenceName != nil && *field.ReferenceName == fieldReferenceName {
				return &field, nil
			}
		}
	}

	return nil, fmt.Errorf("field '%s' not found for work item type '%s'", fieldReferenceName, workItemTypeName)
}
