package main

import (
	"github.com/stevengregory/musing-cli/cmd"
)

var (
	version = "0.1.0" // Updated by ldflags during release
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version information
	cmd.Version = version
	cmd.Commit = commit
	cmd.Date = date

	// Execute root command
	cmd.Execute()
}
