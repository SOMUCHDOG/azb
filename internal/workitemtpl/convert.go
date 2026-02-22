package workitemtpl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"

	"github.com/SOMUCHDOG/azb/internal/api"
	"github.com/SOMUCHDOG/azb/internal/templates"
)

// ConvertToTemplate converts an Azure DevOps work item to a reusable template.
func ConvertToTemplate(client *api.Client, wi *workitemtracking.WorkItem) *templates.Template {
	template := &templates.Template{
		Name:        getStringField(wi, "System.Title"),
		Type:        getStringField(wi, "System.WorkItemType"),
		Description: fmt.Sprintf("Template created from work item #%d", *wi.Id),
		Fields:      make(map[string]interface{}),
	}

	relevantFields := []string{
		"System.Title",
		"System.Description",
		"System.State",
		"System.Tags",
		"Microsoft.VSTS.Common.Priority",
		"Microsoft.VSTS.Common.AcceptanceCriteria",
		"System.AreaPath",
		"System.IterationPath",
		"Custom.ApplicationName",
	}

	if wi.Fields != nil {
		for _, fieldName := range relevantFields {
			if value, ok := (*wi.Fields)[fieldName]; ok {
				template.Fields[fieldName] = value
			}
		}

		assignedTo := getEmailField(wi, "System.AssignedTo")
		if assignedTo != "" {
			template.Fields["System.AssignedTo"] = assignedTo
		}
	}

	if wi.Relations != nil {
		for _, rel := range *wi.Relations {
			if rel.Rel == nil || rel.Url == nil {
				continue
			}

			relType := *rel.Rel
			switch relType {
			case "System.LinkTypes.Hierarchy-Reverse":
				parentID := ExtractWorkItemIDFromURL(*rel.Url)
				if parentID > 0 {
					if template.Relations == nil {
						template.Relations = &templates.Relations{}
					}
					template.Relations.ParentID = parentID
				}
			case "System.LinkTypes.Hierarchy-Forward":
				childID := ExtractWorkItemIDFromURL(*rel.Url)
				if childID <= 0 {
					continue
				}
				if template.Relations == nil {
					template.Relations = &templates.Relations{}
				}

				childTitle := fmt.Sprintf("Child Work Item #%d", childID)
				childType := "Task"
				childDescription := ""
				childAssignedTo := ""
				if client != nil {
					childWI, err := client.GetWorkItem(childID)
					if err == nil && childWI != nil {
						childTitle = getStringField(childWI, "System.Title")
						childDescription = getStringField(childWI, "System.Description")
						childAssignedTo = getEmailField(childWI, "System.AssignedTo")
						if childWorkItemType := getStringField(childWI, "System.WorkItemType"); childWorkItemType != "" {
							childType = childWorkItemType
						}
					}
				}

				template.Relations.Children = append(template.Relations.Children, templates.ChildWorkItem{
					Title:       childTitle,
					Type:        childType,
					Description: childDescription,
					AssignedTo:  childAssignedTo,
				})
			}
		}
	}

	return template
}

func ExtractWorkItemIDFromURL(url string) int {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return 0
	}
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return id
}

func getStringField(wi *workitemtracking.WorkItem, fieldName string) string {
	if wi.Fields == nil {
		return ""
	}
	if value, ok := (*wi.Fields)[fieldName]; ok {
		if fieldName == "System.AssignedTo" || fieldName == "System.CreatedBy" || fieldName == "System.ChangedBy" {
			if identityMap, ok := value.(map[string]interface{}); ok {
				if displayName, ok := identityMap["displayName"].(string); ok {
					return displayName
				}
				if uniqueName, ok := identityMap["uniqueName"].(string); ok {
					return uniqueName
				}
			}
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}

func getEmailField(wi *workitemtracking.WorkItem, fieldName string) string {
	if wi.Fields == nil {
		return ""
	}
	if value, ok := (*wi.Fields)[fieldName]; ok {
		if fieldName == "System.AssignedTo" || fieldName == "System.CreatedBy" || fieldName == "System.ChangedBy" {
			if identityMap, ok := value.(map[string]interface{}); ok {
				if uniqueName, ok := identityMap["uniqueName"].(string); ok {
					return uniqueName
				}
				if displayName, ok := identityMap["displayName"].(string); ok {
					return displayName
				}
			}
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}
