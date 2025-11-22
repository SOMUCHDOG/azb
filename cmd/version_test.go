package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	// Set test version info
	SetVersionInfo("1.0.0", "abc123", "2024-01-01", "test")

	// Execute version command directly via Run function
	var output bytes.Buffer

	// Create a test cobra command to capture output
	testCmd := &cobra.Command{}
	testCmd.SetOut(&output)

	// Call the Run function directly
	versionCmd.Run(testCmd, []string{})

	result := output.String()

	// Check that output contains expected information
	expectedStrings := []string{
		"Azure Boards CLI (azb)",
		"Version:    1.0.0",
		"Commit:     abc123",
		"Built:      2024-01-01",
		"Built by:   test",
		"Go version:",
		"OS/Arch:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("version output missing expected string: %q\nGot: %s", expected, result)
		}
	}
}

func TestSetVersionInfo(t *testing.T) {
	tests := []struct {
		version string
		commit  string
		date    string
		builtBy string
	}{
		{"1.0.0", "abc123", "2024-01-01", "goreleaser"},
		{"dev", "none", "unknown", "unknown"},
		{"2.5.3", "def456", "2024-12-31", "ci"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			SetVersionInfo(tt.version, tt.commit, tt.date, tt.builtBy)

			if versionInfo.version != tt.version {
				t.Errorf("versionInfo.version = %v, want %v", versionInfo.version, tt.version)
			}
			if versionInfo.commit != tt.commit {
				t.Errorf("versionInfo.commit = %v, want %v", versionInfo.commit, tt.commit)
			}
			if versionInfo.date != tt.date {
				t.Errorf("versionInfo.date = %v, want %v", versionInfo.date, tt.date)
			}
			if versionInfo.builtBy != tt.builtBy {
				t.Errorf("versionInfo.builtBy = %v, want %v", versionInfo.builtBy, tt.builtBy)
			}
		})
	}
}

func TestVersionCommand_DefaultValues(t *testing.T) {
	// Reset to default values
	SetVersionInfo("dev", "none", "unknown", "unknown")

	var output bytes.Buffer

	// Create a test cobra command to capture output
	testCmd := &cobra.Command{}
	testCmd.SetOut(&output)

	// Call the Run function directly
	versionCmd.Run(testCmd, []string{})

	result := output.String()

	// Check that default values are present
	if !strings.Contains(result, "Version:    dev") {
		t.Error("Expected default version 'dev'")
	}
	if !strings.Contains(result, "Commit:     none") {
		t.Error("Expected default commit 'none'")
	}
	if !strings.Contains(result, "Built:      unknown") {
		t.Error("Expected default date 'unknown'")
	}
}
