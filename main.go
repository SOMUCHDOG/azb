package main

import (
	"github.com/casey/azure-boards-cli/cmd"
)

// Build-time variables injected by ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date, builtBy)
	cmd.Execute()
}
