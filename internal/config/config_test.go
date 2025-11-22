package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfig_Load(t *testing.T) {
	// Setup: Create a temporary config
	viper.Reset()
	viper.Set("organization", "test-org")
	viper.Set("project", "test-project")
	viper.Set("cache_ttl", 600)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Organization != "test-org" {
		t.Errorf("Expected organization 'test-org', got '%s'", cfg.Organization)
	}

	if cfg.Project != "test-project" {
		t.Errorf("Expected project 'test-project', got '%s'", cfg.Project)
	}

	if cfg.CacheTTL != 600 {
		t.Errorf("Expected cache_ttl 600, got %d", cfg.CacheTTL)
	}
}

func TestConfig_Save(t *testing.T) {
	// Setup: Create a temporary directory
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	viper.Reset()
	viper.SetConfigFile(configFile)

	cfg := &Config{
		Organization:     "test-org",
		Project:          "test-project",
		DefaultAreaPath:  "test-area",
		DefaultIteration: "test-iteration",
		CacheTTL:         300,
		DefaultView:      "assigned-to-me",
	}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("Config file was not created")
	}

	// Load and verify
	viper.Reset()
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loadedCfg.Organization != cfg.Organization {
		t.Errorf("Expected organization '%s', got '%s'", cfg.Organization, loadedCfg.Organization)
	}
}

func TestSetDefaults(t *testing.T) {
	viper.Reset()
	SetDefaults()

	if viper.GetInt("cache_ttl") != 300 {
		t.Errorf("Expected default cache_ttl 300, got %d", viper.GetInt("cache_ttl"))
	}

	if viper.GetString("default_view") != "assigned-to-me" {
		t.Errorf("Expected default_view 'assigned-to-me', got '%s'", viper.GetString("default_view"))
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() failed: %v", err)
	}

	if path == "" {
		t.Error("GetConfigPath() returned empty path")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("GetConfigPath() should return absolute path, got '%s'", path)
	}

	expectedSuffix := filepath.Join(".azure-boards-cli", "config.yaml")
	if !filepath.HasPrefix(filepath.Base(filepath.Dir(path))+"/"+filepath.Base(path), ".azure-boards-cli/config.yaml") {
		t.Errorf("Expected path to end with '%s', got '%s'", expectedSuffix, path)
	}
}
