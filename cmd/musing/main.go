package main

import (
	"fmt"
	"io"
	"log"
	"os"

	figure "github.com/common-nighthawk/go-figure"
	"github.com/charmbracelet/lipgloss"
	"github.com/stevengregory/musing-cli/cmd"
	"github.com/urfave/cli/v2"
)

var (
	version = "0.1.0" // Updated by ldflags during release
	commit  = "none"
	date    = "unknown"
)

// PrintBanner displays the ASCII art banner
func PrintBanner() {
	banner := figure.NewFigure("Musing", "", true)
	bannerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)
	fmt.Println(bannerStyle.Render(banner.String()))
}

// shouldShowBanner returns true if the banner should be displayed
func shouldShowBanner() bool {
	if len(os.Args) < 2 {
		return true
	}
	// Skip banner for monitor, version, and completion commands
	cmd := os.Args[1]
	return cmd != "monitor" && cmd != "--version" && cmd != "-v" && cmd != "completion"
}

func main() {
	// Disable default log output (timestamp prefixes)
	// This suppresses urfave/cli's internal error logging
	log.SetOutput(io.Discard)
	log.SetFlags(0)

	// Print ASCII art banner with color
	if shouldShowBanner() {
		PrintBanner()
	}

	// Custom version printer - outputs only the version number (like bun -v)
	// This provides clean, parseable output for scripts and automation
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}

	app := &cli.App{
		Name:                 "musing",
		Usage:                "CLI for managing multi-service development stacks",
		Version:              version,
		EnableBashCompletion: true, // Enable shell completion support
		ErrWriter:            io.Discard, // Suppress framework error output
		Commands: []*cli.Command{
			cmd.DevCommand(),
			cmd.DeployCommand(),
			cmd.MonitorCommand(),
			cmd.CompletionCommand(),
		},
	}

	// Reorder args to allow flags after arguments (like most CLIs)
	args := reorderArgs(os.Args)

	_ = app.Run(args)
}

// reorderArgs moves flags before positional arguments to work around urfave/cli limitation
func reorderArgs(args []string) []string {
	if len(args) <= 2 {
		return args
	}

	var flags []string
	var positional []string
	command := args[:2] // Keep program name and command

	for i := 2; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 0 && arg[0] == '-' {
			flags = append(flags, arg)
			// Check if next arg is the flag value
			if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				flags = append(flags, args[i+1])
				i++ // Skip next arg since we consumed it
			}
		} else {
			positional = append(positional, arg)
		}
	}

	// Rebuild: command + flags + positional
	result := append(command, flags...)
	result = append(result, positional...)
	return result
}
