package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/stevengregory/musing-cli/internal/mongo"
	"github.com/stevengregory/musing-cli/internal/ui"
	"github.com/urfave/cli/v2"
)

// Styles using Lip Gloss (matching monitor.go)
var (
	deployHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF00FF")). // Magenta/purple
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF00FF")).
			Padding(0, 2).
			MarginBottom(1)
)

func DeployCommand() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Usage:     "Deploy MongoDB data collections",
		ArgsUsage: "[collection] (use 'all' or specific collection name)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment: dev or prod",
				Value:   "dev",
			},
		},
		Before: func(c *cli.Context) error {
			// Reorder args to handle flags after arguments
			// This allows both "deploy --env prod news" and "deploy news --env prod"
			return nil
		},
		Action: func(c *cli.Context) error {
			collection := "all"
			if c.NArg() > 0 {
				collection = c.Args().Get(0)
			}

			env := c.String("env")
			return deployData(collection, env)
		},
	}
}

func deployData(collection, env string) error {
	fmt.Println(deployHeaderStyle.Render(fmt.Sprintf("MongoDB Deployment - %s", env)))

	var mongoURI string
	var port int

	if env == "prod" {
		port = config.MongoProdPort
		mongoURI = fmt.Sprintf("mongodb://localhost:%d", port)
		ui.Info(fmt.Sprintf("Deploying to PRODUCTION (localhost:%d)", port))

		// Confirm production deployment
		confirmMsg := fmt.Sprintf("Deploy '%s' to PRODUCTION?", collection)
		if !ui.Confirm(confirmMsg, false) {
			fmt.Println()
			ui.Info("Production deployment cancelled")
			return nil
		}

		// Check if tunnel is open
		status := health.CheckPort(port)
		if !status.Open {
			ui.Error(fmt.Sprintf("MongoDB tunnel not open on port %d", port))
			ui.Info(fmt.Sprintf("Open SSH tunnel first: ssh -f -N -L %d:localhost:%d root@<your-server>", config.MongoProdPort, config.MongoDevPort))
			return fmt.Errorf("production MongoDB not accessible")
		}
		ui.Success("SSH tunnel is open")
	} else {
		port = config.MongoDevPort
		mongoURI = fmt.Sprintf("mongodb://localhost:%d", port)
		ui.Info(fmt.Sprintf("Deploying to DEVELOPMENT (localhost:%d)", port))

		// Check if dev MongoDB is running
		status := health.CheckPort(port)
		if !status.Open {
			ui.Error(fmt.Sprintf("MongoDB not running on port %d", port))
			ui.Info("Run 'musing dev' first to start the development stack")
			return fmt.Errorf("development MongoDB not accessible")
		}
		ui.Success("MongoDB is running")
	}

	// Get data directory from project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		ui.Error("Could not find project root")
		return err
	}
	dataDir := filepath.Join(projectRoot, "data")

	fmt.Println()

	if collection == "all" {
		ui.Info("Deploying all collections...")
		if err := mongo.DeployAll(mongoURI, "me", dataDir); err != nil {
			ui.Error(fmt.Sprintf("Failed to deploy: %v", err))
			return err
		}
		ui.Success("All collections deployed successfully!")
	} else {
		ui.Info(fmt.Sprintf("Deploying collection: %s", collection))
		if err := mongo.DeployCollection(mongoURI, "me", collection, dataDir); err != nil {
			ui.Error(fmt.Sprintf("Failed to deploy: %v", err))
			return err
		}
		ui.Success(fmt.Sprintf("Collection '%s' deployed successfully!", collection))
	}

	return nil
}
