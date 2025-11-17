package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Organization       string `mapstructure:"organization"`
	Project            string `mapstructure:"project"`
	DefaultAreaPath    string `mapstructure:"default_area_path"`
	DefaultIteration   string `mapstructure:"default_iteration"`
	CacheTTL           int    `mapstructure:"cache_ttl"`
	DefaultView        string `mapstructure:"default_view"`
	PersonalAccessToken string `mapstructure:"personal_access_token"`
}

// Load loads the configuration from file and environment variables
func Load() (*Config, error) {
	var cfg Config

	// Unmarshal the config into our struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to file
func Save(cfg *Config) error {
	viper.Set("organization", cfg.Organization)
	viper.Set("project", cfg.Project)
	viper.Set("default_area_path", cfg.DefaultAreaPath)
	viper.Set("default_iteration", cfg.DefaultIteration)
	viper.Set("cache_ttl", cfg.CacheTTL)
	viper.Set("default_view", cfg.DefaultView)

	// Don't save PAT in config file - use auth package for that
	// viper.Set("personal_access_token", cfg.PersonalAccessToken)

	// Get config file path
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// Create default config path
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(home, ".azure-boards-cli")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		configFile = filepath.Join(configDir, "config.yaml")
	}

	return viper.WriteConfigAs(configFile)
}

// EnsureConfigDir ensures the config directory exists
func EnsureConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".azure-boards-cli")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// SetDefaults sets default configuration values
func SetDefaults() {
	viper.SetDefault("cache_ttl", 300)
	viper.SetDefault("default_view", "assigned-to-me")
}
