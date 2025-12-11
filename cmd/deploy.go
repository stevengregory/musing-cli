package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/stevengregory/musing-cli/internal/mongo"
	"github.com/stevengregory/musing-cli/internal/ui"
	"github.com/urfave/cli/v2"
)

func DeployCommand() *cli.Command {
	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy MongoDB data collections",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "env",
				Usage: "Environment: dev or prod",
				Value: "dev",
			},
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
	ui.Header(fmt.Sprintf("📦 MongoDB Deployment - %s", env))
	fmt.Println()

	var mongoURI string
	var port int

	if env == "prod" {
		port = config.MongoProdPort
		mongoURI = fmt.Sprintf("mongodb://localhost:%d", port)
		ui.Info(fmt.Sprintf("Deploying to PRODUCTION (localhost:%d)", port))

		// Check if tunnel is open
		status := health.CheckPort(port)
		if !status.Open {
			ui.Error(fmt.Sprintf("MongoDB tunnel not open on port %d", port))
			ui.Info("Run 'ssh -f -N -L 27019:localhost:27018 root@stevengregory.io' first")
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

	// Get project root (go up from musing-cli directory)
	cwd, _ := os.Getwd()
	projectRoot := filepath.Dir(cwd)
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
