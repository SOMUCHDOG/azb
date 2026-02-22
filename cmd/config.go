package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/SOMUCHDOG/azb/internal/config"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and modify Azure Boards CLI configuration.`,
	}

	configGetCmd = &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  `Get the value of a configuration key.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigGet,
	}

	configSetCmd = &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  `Set the value of a configuration key.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	}

	configListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all configuration",
		Long:  `Display all configuration values.`,
		RunE:  runConfigList,
	}
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(newConfigKeyCmd("organization", "Azure DevOps organization"))
	configCmd.AddCommand(newConfigKeyCmd("project", "Azure DevOps project"))
	configCmd.AddCommand(newConfigKeyCmd("default_area_path", "Default area path"))
	configCmd.AddCommand(newConfigKeyCmd("default_iteration", "Default iteration path"))
	configCmd.AddCommand(newConfigKeyCmd("default_view", "Default dashboard view"))
}

func newConfigKeyCmd(key, description string) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s [value]", key),
		Short: fmt.Sprintf("Get or set %s", description),
		Long: fmt.Sprintf(
			"Get or set %s.\n\nExamples:\n  azb config %s\n  azb config %s <value>",
			description,
			key,
			key,
		),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runConfigGet(cmd, []string{key})
			}
			return runConfigSet(cmd, []string{key, args[0]})
		},
	}
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := viper.Get(key)

	if value == nil {
		fmt.Printf("%s is not set\n", key)
		return nil
	}

	fmt.Printf("%s = %v\n", key, value)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	viper.Set(key, value)

	// Load current config
	cfg, err := config.Load()
	if err != nil {
		// Create new config if it doesn't exist
		cfg = &config.Config{}
	}

	// Update the specific field based on key
	switch key {
	case "organization":
		cfg.Organization = value
	case "project":
		cfg.Project = value
	case "default_area_path":
		cfg.DefaultAreaPath = value
	case "default_iteration":
		cfg.DefaultIteration = value
	case "default_view":
		cfg.DefaultView = value
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Configuration:")
	fmt.Printf("  organization:        %s\n", cfg.Organization)
	fmt.Printf("  project:             %s\n", cfg.Project)
	fmt.Printf("  default_area_path:   %s\n", cfg.DefaultAreaPath)
	fmt.Printf("  default_iteration:   %s\n", cfg.DefaultIteration)
	fmt.Printf("  cache_ttl:           %d\n", cfg.CacheTTL)
	fmt.Printf("  default_view:        %s\n", cfg.DefaultView)

	// Show the computed organization URL for debugging
	if cfg.Organization != "" {
		fmt.Printf("\nComputed organization URL: https://dev.azure.com/%s\n", cfg.Organization)
	}

	return nil
}
