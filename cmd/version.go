package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	versionInfo = struct {
		version string
		commit  string
		date    string
		builtBy string
	}{
		version: "dev",
		commit:  "none",
		date:    "unknown",
		builtBy: "unknown",
	}
)

// SetVersionInfo sets the version information from main package
func SetVersionInfo(version, commit, date, builtBy string) {
	versionInfo.version = version
	versionInfo.commit = commit
	versionInfo.date = date
	versionInfo.builtBy = builtBy
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print version information including version number, commit hash, build date, and Go version.`,
	Run: func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Azure Boards CLI (azb)\n")
		fmt.Fprintf(out, "Version:    %s\n", versionInfo.version)
		fmt.Fprintf(out, "Commit:     %s\n", versionInfo.commit)
		fmt.Fprintf(out, "Built:      %s\n", versionInfo.date)
		fmt.Fprintf(out, "Built by:   %s\n", versionInfo.builtBy)
		fmt.Fprintf(out, "Go version: %s\n", runtime.Version())
		fmt.Fprintf(out, "OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
