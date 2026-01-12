package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

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
					fmt.Println(bashCompletionScript)
					return nil
				},
			},
			{
				Name:  "zsh",
				Usage: "Generate zsh completion script",
				Action: func(c *cli.Context) error {
					fmt.Println(zshCompletionScript)
					return nil
				},
			},
		},
	}
}

const bashCompletionScript = `#! /bin/bash

_musing_bash_autocomplete() {
    if [[ "${COMP_WORDS[0]}" != "source" ]]; then
        local cur opts base
        COMPREPLY=()
        cur="${COMP_WORDS[COMP_CWORD]}"
        if [[ "$cur" == "-"* ]]; then
            opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} ${cur} --generate-bash-completion )
        else
            opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
        fi
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}

complete -o bashdefault -o default -o nospace -F _musing_bash_autocomplete musing
`

const zshCompletionScript = `#compdef musing

_musing_zsh_autocomplete() {
    local -a opts
    local cur
    cur=${words[-1]}
    if [[ "$cur" == "-"* ]]; then
        opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} ${cur} --generate-bash-completion)}")
    else
        opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} --generate-bash-completion)}")
    fi

    if [[ "${opts[1]}" != "" ]]; then
        _describe 'values' opts
    else
        _files
    fi
}

compdef _musing_zsh_autocomplete musing
`
