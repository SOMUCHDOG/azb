package tui

import (
	"testing"

	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// Mock client for testing - doesn't make real API calls
type mockWorkItemClient struct {
	workItems map[int]*workitemtracking.WorkItem
}

func (m *mockWorkItemClient) getWorkItem(id int) (*workitemtracking.WorkItem, error) {
	if wi, ok := m.workItems[id]; ok {
		return wi, nil
	}
	return nil, nil
}

func TestExtractWorkItemIDFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{
			name:     "valid work item URL",
			url:      "https://dev.azure.com/org/project/_apis/wit/workItems/123",
			expected: 123,
		},
		{
			name:     "work item URL with query params",
			url:      "https://dev.azure.com/org/project/_apis/wit/workItems/456?api-version=7.0",
			expected: 0, // Query params make the last segment non-numeric
		},
		{
			name:     "simple ID path",
			url:      "/workItems/789",
			expected: 789,
		},
		{
			name:     "invalid URL without ID",
			url:      "https://dev.azure.com/org/project",
			expected: 0,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: 0,
		},
		{
			name:     "non-numeric ID",
			url:      "https://dev.azure.com/org/project/_apis/wit/workItems/abc",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWorkItemIDFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("extractWorkItemIDFromURL(%q) = %d, want %d", tt.url, result, tt.expected)
			}
		})
	}
}

func TestGetStringField(t *testing.T) {
	tests := []struct {
		name      string
		workItem  *workitemtracking.WorkItem
		fieldName string
		expected  string
	}{
		{
			name: "existing string field",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{
					"System.Title": "Test Work Item",
				},
			},
			fieldName: "System.Title",
			expected:  "Test Work Item",
		},
		{
			name: "non-existent field",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{},
			},
			fieldName: "System.Description",
			expected:  "",
		},
		{
			name: "nil fields",
			workItem: &workitemtracking.WorkItem{
				Fields: nil,
			},
			fieldName: "System.Title",
			expected:  "",
		},
		{
			name: "identity field with display name",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{
					"System.AssignedTo": map[string]interface{}{
						"displayName": "John Doe",
						"uniqueName":  "john.doe@example.com",
					},
				},
			},
			fieldName: "System.AssignedTo",
			expected:  "John Doe",
		},
		{
			name: "identity field with only unique name",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{
					"System.AssignedTo": map[string]interface{}{
						"uniqueName": "jane.doe@example.com",
					},
				},
			},
			fieldName: "System.AssignedTo",
			expected:  "jane.doe@example.com",
		},
		{
			name: "numeric field converted to string",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{
					"Microsoft.VSTS.Common.Priority": 1,
				},
			},
			fieldName: "Microsoft.VSTS.Common.Priority",
			expected:  "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringField(tt.workItem, tt.fieldName)
			if result != tt.expected {
				t.Errorf("getStringField() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetEmailField(t *testing.T) {
	tests := []struct {
		name      string
		workItem  *workitemtracking.WorkItem
		fieldName string
		expected  string
	}{
		{
			name: "identity field with both displayName and uniqueName - prefers uniqueName",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{
					"System.AssignedTo": map[string]interface{}{
						"displayName": "John Doe",
						"uniqueName":  "john.doe@example.com",
					},
				},
			},
			fieldName: "System.AssignedTo",
			expected:  "john.doe@example.com", // Prefer email over name
		},
		{
			name: "identity field with only displayName",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{
					"System.AssignedTo": map[string]interface{}{
						"displayName": "Jane Smith",
					},
				},
			},
			fieldName: "System.AssignedTo",
			expected:  "Jane Smith", // Fall back to display name
		},
		{
			name: "identity field with only uniqueName",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{
					"System.AssignedTo": map[string]interface{}{
						"uniqueName": "user@example.com",
					},
				},
			},
			fieldName: "System.AssignedTo",
			expected:  "user@example.com",
		},
		{
			name: "non-existent field",
			workItem: &workitemtracking.WorkItem{
				Fields: &map[string]interface{}{},
			},
			fieldName: "System.AssignedTo",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEmailField(tt.workItem, tt.fieldName)
			if result != tt.expected {
				t.Errorf("getEmailField() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetIntField(t *testing.T) {
	tests := []struct {
		name     string
		workItem *workitemtracking.WorkItem
		expected int
	}{
		{
			name: "work item with ID",
			workItem: &workitemtracking.WorkItem{
				Id: intPtr(123),
			},
			expected: 123,
		},
		{
			name: "work item without ID",
			workItem: &workitemtracking.WorkItem{
				Id: nil,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntField(tt.workItem, "System.Id")
			if result != tt.expected {
				t.Errorf("getIntField() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCleanAssignedTo(t *testing.T) {
	tests := []struct {
		name       string
		assignedTo string
		expected   string
	}{
		{
			name:       "name with email",
			assignedTo: "John Doe <john.doe@example.com>",
			expected:   "John Doe",
		},
		{
			name:       "name only",
			assignedTo: "Jane Smith",
			expected:   "Jane Smith",
		},
		{
			name:       "empty string",
			assignedTo: "",
			expected:   "",
		},
		{
			name:       "email only",
			assignedTo: "<user@example.com>",
			expected:   "<user@example.com>", // cleanAssignedTo returns the input if no name before <
		},
		{
			name:       "name with spaces before email",
			assignedTo: "Alice Johnson  <alice@example.com>",
			expected:   "Alice Johnson",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanAssignedTo(tt.assignedTo)
			if result != tt.expected {
				t.Errorf("cleanAssignedTo(%q) = %q, want %q", tt.assignedTo, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid filename",
			input:    "My Work Item",
			expected: "my work item",
		},
		{
			name:     "filename with invalid characters",
			input:    "Bug: Fix/Update <Component>",
			expected: "bug- fix-update -component-",
		},
		{
			name:     "filename with multiple invalid chars",
			input:    `Test\File:Name*With?Chars"<>|`,
			expected: "test-file-name-with-chars----", // Actual output has 4 dashes at end
		},
		{
			name:     "long filename gets truncated",
			input:    "This is a very long work item title that exceeds the maximum length and should be truncated",
			expected: "this is a very long work item title that exceeds t", // Actual truncation at 50 chars
		},
		{
			name:     "filename with leading/trailing spaces",
			input:    "  Spaced Filename  ",
			expected: "spaced filename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertWorkItemToTemplate_Fields(t *testing.T) {
	// Test with nil client since we're not fetching children
	workItemID := 123
	workItem := &workitemtracking.WorkItem{
		Id: &workItemID,
		Fields: &map[string]interface{}{
			"System.Title":       "Test User Story",
			"System.Description": "This is a test description",
			"System.State":       "Active",
			"System.AssignedTo": map[string]interface{}{
				"displayName": "John Doe",
				"uniqueName":  "john.doe@example.com",
			},
			"System.Tags":                              "tag1;tag2",
			"Microsoft.VSTS.Common.Priority":           2,
			"Microsoft.VSTS.Common.AcceptanceCriteria": "Acceptance criteria here",
			"System.AreaPath":                          "Project\\Area",
			"System.IterationPath":                     "Project\\Iteration1",
			"Custom.ApplicationName":                   "MyApp",
			"System.WorkItemType":                      "User Story",
		},
		Relations: nil,
	}

	template := convertWorkItemToTemplate(nil, workItem)

	// Test that template has correct name and type
	if template.Name != "Test User Story" {
		t.Errorf("Template.Name = %q, want %q", template.Name, "Test User Story")
	}

	if template.Type != "User Story" {
		t.Errorf("Template.Type = %q, want %q", template.Type, "User Story")
	}

	// Test that all expected fields are included (Issue #30 regression test)
	expectedFields := map[string]interface{}{
		"System.Title":                             "Test User Story",
		"System.Description":                       "This is a test description",
		"System.State":                             "Active", // Issue #30: Must include State
		"System.AssignedTo":                        "john.doe@example.com", // Email preferred over display name
		"System.Tags":                              "tag1;tag2",
		"Microsoft.VSTS.Common.Priority":           2,
		"Microsoft.VSTS.Common.AcceptanceCriteria": "Acceptance criteria here",
		"System.AreaPath":                          "Project\\Area",
		"System.IterationPath":                     "Project\\Iteration1",
		"Custom.ApplicationName":                   "MyApp", // Issue #30: Must include ApplicationName
	}

	for fieldName, expectedValue := range expectedFields {
		if actualValue, ok := template.Fields[fieldName]; !ok {
			t.Errorf("Template.Fields missing expected field %q", fieldName)
		} else if actualValue != expectedValue {
			t.Errorf("Template.Fields[%q] = %v, want %v", fieldName, actualValue, expectedValue)
		}
	}
}

func TestConvertWorkItemToTemplate_ParentRelationship(t *testing.T) {
	workItemID := 456
	parentURL := "https://dev.azure.com/org/project/_apis/wit/workItems/123"

	workItem := &workitemtracking.WorkItem{
		Id: &workItemID,
		Fields: &map[string]interface{}{
			"System.Title":        "Child Work Item",
			"System.WorkItemType": "Task",
		},
		Relations: &[]workitemtracking.WorkItemRelation{
			{
				Rel: strPtr("System.LinkTypes.Hierarchy-Reverse"),
				Url: &parentURL,
			},
		},
	}

	template := convertWorkItemToTemplate(nil, workItem)

	// Test that parent relationship is captured (Issue #30: relationships in template)
	if template.Relations == nil {
		t.Fatal("Template.Relations is nil, expected parent relationship")
	}

	if template.Relations.ParentID != 123 {
		t.Errorf("Template.Relations.ParentID = %d, want %d", template.Relations.ParentID, 123)
	}
}

func TestConvertWorkItemToTemplate_ChildRelationshipsWithoutClient(t *testing.T) {
	// Test that child relationship structure is created even without fetching details
	workItemID := 789
	child1URL := "https://dev.azure.com/org/project/_apis/wit/workItems/101"
	child2URL := "https://dev.azure.com/org/project/_apis/wit/workItems/102"

	workItem := &workitemtracking.WorkItem{
		Id: &workItemID,
		Fields: &map[string]interface{}{
			"System.Title":        "Parent User Story",
			"System.WorkItemType": "User Story",
		},
		Relations: &[]workitemtracking.WorkItemRelation{
			{
				Rel: strPtr("System.LinkTypes.Hierarchy-Forward"),
				Url: &child1URL,
			},
			{
				Rel: strPtr("System.LinkTypes.Hierarchy-Forward"),
				Url: &child2URL,
			},
		},
	}

	// Without a real client, child titles won't be fetched but structure should exist
	template := convertWorkItemToTemplate(nil, workItem)

	if template.Relations == nil {
		t.Fatal("Template.Relations is nil, expected child relationships")
	}

	// Issue #30: Must include child relationships
	if len(template.Relations.Children) != 2 {
		t.Fatalf("Template.Relations.Children length = %d, want %d", len(template.Relations.Children), 2)
	}

	// Without client, should have fallback titles
	child1 := template.Relations.Children[0]
	if child1.Title == "" {
		t.Error("Child[0].Title is empty, expected fallback title")
	}

	child2 := template.Relations.Children[1]
	if child2.Title == "" {
		t.Error("Child[1].Title is empty, expected fallback title")
	}

	// Child fields should not include System.Id (no longer stored in template)
	if child1.Fields != nil && child1.Fields["System.Id"] != nil {
		t.Error("Child[0].Fields[System.Id] should not be set")
	}
	if child2.Fields != nil && child2.Fields["System.Id"] != nil {
		t.Error("Child[1].Fields[System.Id] should not be set")
	}
}

func TestWorkItemItem_FilterValue(t *testing.T) {
	item := workItemItem{
		ID:         123,
		Title:      "Test Work Item",
		State:      "Active",
		AssignedTo: "John Doe",
	}

	filterValue := item.FilterValue()
	if filterValue != "Test Work Item" {
		t.Errorf("FilterValue() = %q, want %q", filterValue, "Test Work Item")
	}
}

// Test that extractWorkItemIDFromURL handles edge cases properly
func TestExtractWorkItemIDFromURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want int
	}{
		{
			name: "URL with trailing slash",
			url:  "https://dev.azure.com/org/project/_apis/wit/workItems/123/",
			want: 0, // Empty string after split
		},
		{
			name: "URL with only slashes",
			url:  "///",
			want: 0,
		},
		{
			name: "Very large ID number",
			url:  "https://dev.azure.com/org/_apis/wit/workItems/999999999",
			want: 999999999,
		},
		{
			name: "Negative number",
			url:  "/workItems/-123",
			want: -123, // Function accepts negative numbers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkItemIDFromURL(tt.url)
			if got != tt.want {
				t.Errorf("extractWorkItemIDFromURL(%q) = %d, want %d", tt.url, got, tt.want)
			}
		})
	}
}

// Helper functions for creating pointers
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
