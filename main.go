package main

import (
	"log"
	"os"

	"github.com/stevengregory/musing-cli/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "musing",
		Usage: "Development tooling for musing-tu project",
		Commands: []*cli.Command{
			cmd.DevCommand(),
			cmd.DeployCommand(),
			cmd.MonitorCommand(),
		},
	}

	// Reorder args to allow flags after arguments (like most CLIs)
	args := reorderArgs(os.Args)

	if err := app.Run(args); err != nil {
		log.Fatal(err)
	}
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
