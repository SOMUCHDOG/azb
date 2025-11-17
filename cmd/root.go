package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "ab",
		Short: "Azure Boards CLI - Manage work items from your terminal",
		Long: `Azure Boards CLI is a cross-platform command-line interface for managing
Azure Boards work items. It provides both a Terminal UI dashboard for
interactive work and traditional CLI commands for automation and scripting.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no subcommand is provided, launch the dashboard
			// For now, we'll just show help
			cmd.Help()
		},
	}
)

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.azure-boards-cli/config.yaml)")
	rootCmd.PersistentFlags().String("org", "", "Azure DevOps organization")
	rootCmd.PersistentFlags().String("project", "", "Azure DevOps project")

	// Bind flags to viper
	viper.BindPFlag("organization", rootCmd.PersistentFlags().Lookup("org"))
	viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Search config in home directory with name ".azure-boards-cli"
		configPath := home + "/.azure-boards-cli"
		viper.AddConfigPath(configPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		// Config file loaded successfully
	}
}
