package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/stevengregory/musing-cli/internal/mongo"
	"github.com/stevengregory/musing-cli/internal/ui"
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

var deployCmd = &cobra.Command{
	Use:   "deploy [collection]",
	Short: "Deploy MongoDB data collections",
	Long:  `Deploy MongoDB data collections to development or production environment.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		collection := "all"
		if len(args) > 0 {
			collection = args[0]
		}

		env, _ := cmd.Flags().GetString("env")
		return deployData(collection, env)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Dynamic completion for collection names
		config.MustFindProjectRoot()
		cfg := config.GetConfig()
		if cfg == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		collections, err := mongo.DiscoverCollections(cfg.Database.DataDir)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var names []string
		for name := range collections {
			names = append(names, name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	deployCmd.Flags().StringP("env", "e", "dev", "Environment: dev or prod")

	// Add completion for env flag
	deployCmd.RegisterFlagCompletionFunc("env", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "prod"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func deployData(collection, env string) error {
	// Find and load project configuration
	config.MustFindProjectRoot()

	cfg := config.GetConfig()
	if cfg == nil {
		fmt.Println()
		ui.Error("No configuration loaded")
		ui.Info("Run 'musing dev' first to initialize the project")
		os.Exit(1)
	}

	fmt.Println(deployHeaderStyle.Render(fmt.Sprintf("%s Deployment - %s", cfg.Database.Type, env)))

	var mongoURI string
	var port int

	if env == "prod" {
		port = cfg.Database.ProdPort
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
			ui.Error(fmt.Sprintf("%s tunnel not open on port %d", cfg.Database.Type, port))

			// Generate helpful SSH tunnel command
			tunnelCmd := generateTunnelCommand(cfg)
			ui.Info(fmt.Sprintf("Open SSH tunnel first: %s", tunnelCmd))
			return fmt.Errorf("production %s not accessible", cfg.Database.Type)
		}
		ui.Success("SSH tunnel is open")
	} else {
		port = cfg.Database.DevPort
		mongoURI = fmt.Sprintf("mongodb://localhost:%d", port)
		ui.Info(fmt.Sprintf("Deploying to DEVELOPMENT (localhost:%d)", port))

		// Check if dev database is running
		status := health.CheckPort(port)
		if !status.Open {
			ui.Error(fmt.Sprintf("%s not running on port %d", cfg.Database.Type, port))
			ui.Info("Run 'musing dev' first to start the development stack")
			return fmt.Errorf("development %s not accessible", cfg.Database.Type)
		}
		ui.Success(fmt.Sprintf("%s is running", cfg.Database.Type))
	}

	// Get data directory from project root
	projectRoot, _ := config.FindProjectRoot() // Already validated above
	dataDir := filepath.Join(projectRoot, cfg.Database.DataDir)

	fmt.Println()

	if collection == "all" {
		ui.Info("Deploying all collections...")
		if err := mongo.DeployAll(mongoURI, cfg.Database.Name, dataDir); err != nil {
			ui.Error(fmt.Sprintf("Failed to deploy: %v", err))
			return err
		}
		ui.Success("All collections deployed successfully!")
	} else {
		ui.Info(fmt.Sprintf("Deploying collection: %s", collection))
		if err := mongo.DeployCollection(mongoURI, cfg.Database.Name, collection, dataDir); err != nil {
			ui.Error(fmt.Sprintf("Failed to deploy: %v", err))
			return err
		}
		ui.Success(fmt.Sprintf("Collection '%s' deployed successfully!", collection))
	}

	return nil
}

// generateTunnelCommand creates the SSH tunnel command from config
func generateTunnelCommand(cfg *config.ProjectConfig) string {
	// Default values if production config not set
	server := "<your-server>"
	remotePort := cfg.Database.DevPort

	// Use config values if available
	if cfg.Production != nil {
		if cfg.Production.Server != "" {
			server = cfg.Production.Server
		}
		if cfg.Production.RemoteDBPort != 0 {
			remotePort = cfg.Production.RemoteDBPort
		}
	}

	return fmt.Sprintf("ssh -f -N -L %d:localhost:%d %s",
		cfg.Database.ProdPort, remotePort, server)
}
