package api

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// Client wraps the Azure DevOps API client
type Client struct {
	connection         *azuredevops.Connection
	workItemClient     workitemtracking.Client
	coreClient         core.Client
	organizationURL    string
	project            string
	ctx                context.Context
}

// NewClient creates a new Azure DevOps API client
func NewClient(organizationURL, project, token string) (*Client, error) {
	if organizationURL == "" {
		return nil, fmt.Errorf("organization URL is required")
	}

	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	if token == "" {
		return nil, fmt.Errorf("personal access token is required")
	}

	// Create a connection to Azure DevOps
	connection := azuredevops.NewPatConnection(organizationURL, token)

	ctx := context.Background()

	// Create work item tracking client
	workItemClient, err := workitemtracking.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work item client: %w", err)
	}

	// Create core client
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	return &Client{
		connection:      connection,
		workItemClient:  workItemClient,
		coreClient:      coreClient,
		organizationURL: organizationURL,
		project:         project,
		ctx:             ctx,
	}, nil
}

// GetOrganizationURL returns the organization URL
func (c *Client) GetOrganizationURL() string {
	return c.organizationURL
}

// GetProject returns the project name
func (c *Client) GetProject() string {
	return c.project
}

// GetContext returns the context
func (c *Client) GetContext() context.Context {
	return c.ctx
}
