package cmd

import (
	"fmt"
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
	Use:   "deploy [collection] [env]",
	Short: "Deploy MongoDB data collections",
	Long: `Deploy MongoDB data collections to development or production environment.

Examples:
  musing deploy              # All collections to dev
  musing deploy news         # Deploy news to dev
  musing deploy news prod    # Deploy news to prod`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		collection := "all"
		env := "dev"
		if len(args) > 0 {
			collection = args[0]
		}
		if len(args) > 1 {
			env = args[1]
		}

		if env != "dev" && env != "prod" {
			return fmt.Errorf("invalid environment: %s (use 'dev' or 'prod')", env)
		}

		return deployData(collection, env)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// First arg: collection names
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
			for alias := range cfg.Database.Aliases {
				names = append(names, alias)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 {
			// Second arg: environment
			return []string{"dev", "prod"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

func deployData(collection, env string) error {
	projectRoot, cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	if collection != "all" {
		if target, ok := cfg.Database.Aliases[collection]; ok {
			collection = target
		}
	}

	fmt.Println(deployHeaderStyle.Render(fmt.Sprintf("%s Deployment - %s", cfg.Database.Type, env)))

	var mongoURI string
	var port int
	tunnelWasOpened := false

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

		// Auto-start tunnel if not open
		status := health.CheckPort(port)
		if !status.Open {
			ui.Info("Opening SSH tunnel...")
			if err := tunnelStart(); err != nil {
				return fmt.Errorf("failed to open SSH tunnel: %w", err)
			}
			tunnelWasOpened = true
		} else {
			ui.Success("SSH tunnel is open")
		}
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

	// Auto-close tunnel if we opened it
	if tunnelWasOpened {
		fmt.Println()
		ui.Info("Closing SSH tunnel...")
		if err := tunnelStop(); err != nil {
			ui.Error(fmt.Sprintf("Failed to close tunnel: %v", err))
		}
	}

	return nil
}
