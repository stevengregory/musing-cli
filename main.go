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
			cmd.StatusCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
