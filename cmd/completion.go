package cmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

//go:embed autocomplete/bash_autocomplete
var bashScriptTemplate string

//go:embed autocomplete/zsh_autocomplete
var zshScriptTemplate string

// CompletionCommand returns the completion command
func CompletionCommand() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Generate shell completion scripts",
		Subcommands: []*cli.Command{
			{
				Name:  "bash",
				Usage: "Generate bash completion script",
				Action: func(c *cli.Context) error {
					// Replace $PROG placeholder with actual program name
					script := strings.ReplaceAll(bashScriptTemplate, "$PROG", "musing")
					fmt.Print(script)
					return nil
				},
			},
			{
				Name:  "zsh",
				Usage: "Generate zsh completion script",
				Action: func(c *cli.Context) error {
					// Replace $PROG placeholder with actual program name
					script := strings.ReplaceAll(zshScriptTemplate, "$PROG", "musing")
					fmt.Print(script)
					return nil
				},
			},
		},
	}
}
