package api

import (
	"fmt"
	"strings"
)

// NormalizeOrganizationURL normalizes the organization input to a proper Azure DevOps URL
func NormalizeOrganizationURL(org string) string {
	org = strings.TrimSpace(org)

	// If it's already a full URL, return it
	if strings.HasPrefix(org, "https://dev.azure.com/") {
		// Remove trailing slash if present
		return strings.TrimSuffix(org, "/")
	}

	// If it starts with http:// or https://, but not the correct domain, try to extract org name
	if strings.HasPrefix(org, "http://") || strings.HasPrefix(org, "https://") {
		// Try to extract organization name from URL
		parts := strings.Split(org, "/")
		if len(parts) > 0 {
			org = parts[len(parts)-1]
		}
	}

	// Remove dev.azure.com if user included it
	org = strings.TrimPrefix(org, "dev.azure.com/")
	org = strings.TrimPrefix(org, "dev.azure.com")

	// Remove any leading/trailing slashes
	org = strings.Trim(org, "/")

	// Build proper URL
	return fmt.Sprintf("https://dev.azure.com/%s", org)
}
