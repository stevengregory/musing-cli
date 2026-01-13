package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/docker"
	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/stevengregory/musing-cli/internal/ui"
)

// Styles using Lip Gloss (matching monitor.go)
var (
	devHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF00FF")). // Magenta/purple
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF00FF")).
			Padding(0, 2).
			MarginBottom(1)
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Manage development stack",
	Long:  `Start, stop, and manage the development stack with Docker Compose.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle stop flag
		stop, _ := cmd.Flags().GetBool("stop")
		if stop {
			return stopServices()
		}

		// Get other flags
		rebuild, _ := cmd.Flags().GetBool("rebuild")
		data, _ := cmd.Flags().GetBool("data")
		logs, _ := cmd.Flags().GetBool("logs")

		// Start services
		return startServices(rebuild, data, logs)
	},
}

func init() {
	devCmd.Flags().Bool("rebuild", false, "Force rebuild all Docker images")
	devCmd.Flags().Bool("data", false, "Deploy MongoDB data after starting services")
	devCmd.Flags().Bool("logs", false, "Follow logs after starting services")
	devCmd.Flags().Bool("stop", false, "Stop all services and exit")
}

func stopServices() error {
	fmt.Println(devHeaderStyle.Render("Stopping Development Stack"))

	// Change to project root directory
	if err := changeToProjectRoot(); err != nil {
		fmt.Println()
		ui.Error("Could not find project root")
		ui.Info("Run this command from inside a project with .musing.yaml")
		os.Exit(1)
	}

	if err := ui.SpinWithBubbles("Stopping all services...", "docker", "compose", "down"); err != nil {
		ui.Error("Failed to stop services")
		return err
	}

	fmt.Println()
	ui.Success("All services stopped")
	return nil
}

// changeToProjectRoot changes the working directory to the main project root
// (the directory containing compose.yaml), intelligently handling git worktrees
func changeToProjectRoot() error {
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	return os.Chdir(projectRoot)
}

func startServices(rebuild, shouldDeployData, followLogs bool) error {
	fmt.Println(devHeaderStyle.Render("Development Stack"))

	// Change to project root directory
	projectRoot := config.MustFindProjectRoot()
	if err := os.Chdir(projectRoot); err != nil {
		return fmt.Errorf("failed to change to project root: %w", err)
	}

	// Ensure Docker is running (auto-start if not)
	if err := docker.EnsureRunning(false); err != nil {
		ui.Error(err.Error())
		return err
	}
	ui.Success("Docker is running")

	// Check for missing API repositories
	if err := checkAPIRepos(); err != nil {
		return err
	}

	// Build images if requested (stop containers first if rebuilding)
	if rebuild {
		if err := ui.SpinWithBubbles("Stopping containers for rebuild...", "docker", "compose", "down"); err != nil {
			// Ignore errors on stop
		}
		if err := ui.SpinWithBubbles("Building images (this may take several minutes)...", "docker", "compose", "build", "--no-cache"); err != nil {
			ui.Error("Failed to build images")
			return err
		}
		ui.Success("Images built successfully")
	}

	// Start services
	if err := ui.SpinWithBubbles("Starting services...", "docker", "compose", "up", "-d"); err != nil {
		ui.Error("Failed to start services")
		return err
	}
	ui.Success("Services started")

	// Wait for services to be ready
	fmt.Println()
	ui.Info("Waiting for services to be ready...")
	time.Sleep(5 * time.Second)

	// Deploy data if requested (calls deploy command internally)
	if shouldDeployData {
		fmt.Println()
		if err := deployData("all", "dev"); err != nil {
			return err
		}
	}

	// Print service URLs with health checks
	printServiceStatus()

	// Follow logs if requested
	if followLogs {
		fmt.Println()
		ui.Info("Following logs (Ctrl+C to exit)...")
		fmt.Println()
		return docker.ComposeLogs(true)
	}

	return nil
}

func checkAPIRepos() error {
	repos := config.GetAPIRepos()
	var missing []string

	for _, repo := range repos {
		if _, err := os.Stat(repo); os.IsNotExist(err) {
			missing = append(missing, repo)
		}
	}

	if len(missing) > 0 {
		fmt.Println()
		ui.Warning("Missing API repositories:")
		for _, repo := range missing {
			fmt.Printf("  • %s\n", filepath.Base(repo))
		}

		fmt.Println()
		ui.Info("Docker Compose will fail without these repositories.")

		if !ui.Confirm("Continue anyway?", false) {
			return fmt.Errorf("cancelled by user")
		}
	}

	return nil
}

func printServiceStatus() {
	fmt.Println()

	cfg := config.GetConfig()
	if cfg == nil {
		ui.Error("No configuration loaded")
		return
	}

	// Check Docker Desktop
	dockerRunning := docker.CheckRunning() == nil

	// Organize services by type
	var apis, frontends []config.ServiceConfig
	for _, svc := range cfg.Services {
		switch svc.Type {
		case "api":
			apis = append(apis, svc)
		case "frontend":
			frontends = append(frontends, svc)
		}
	}

	// Define styles
	sectionHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF00FF"))

	checkmarkStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Bold(true)

	// Docker section
	fmt.Println(sectionHeaderStyle.Render("━━━ Docker ━━━"))
	fmt.Println()
	if dockerRunning {
		fmt.Printf("  %s %-25s\n",
			checkmarkStyle.Render("✓"),
			"Docker Desktop")
	} else {
		fmt.Printf("  %s %-25s\n",
			errorStyle.Render("✗"),
			"Docker Desktop")
	}
	fmt.Println()

	// Database section (from database config)
	fmt.Println(sectionHeaderStyle.Render("━━━ Database ━━━"))
	fmt.Println()
	dbStatus := health.CheckPort(cfg.Database.DevPort)
	if dbStatus.Open {
		fmt.Printf("  %s %-25s :%-6d\n",
			checkmarkStyle.Render("✓"),
			cfg.Database.Type,
			cfg.Database.DevPort)
	} else {
		fmt.Printf("  %s %-25s :%-6d\n",
			errorStyle.Render("✗"),
			cfg.Database.Type,
			cfg.Database.DevPort)
	}
	fmt.Println()

	// API Services section
	if len(apis) > 0 {
		fmt.Println(sectionHeaderStyle.Render(fmt.Sprintf("━━━ API Services (%d) ━━━", len(apis))))
		fmt.Println()
		for _, api := range apis {
			status := health.CheckPort(api.Port)
			if status.Open {
				fmt.Printf("  %s %-25s :%-6d\n",
					checkmarkStyle.Render("✓"),
					api.Name,
					api.Port)
			} else {
				fmt.Printf("  %s %-25s :%-6d\n",
					errorStyle.Render("✗"),
					api.Name,
					api.Port)
			}
		}
		fmt.Println()
	}

	// Frontend section
	if len(frontends) > 0 {
		fmt.Println(sectionHeaderStyle.Render("━━━ Frontend ━━━"))
		fmt.Println()
		for _, fe := range frontends {
			status := health.CheckPort(fe.Port)
			if status.Open {
				fmt.Printf("  %s %-25s :%-6d\n",
					checkmarkStyle.Render("✓"),
					fe.Name,
					fe.Port)
			} else {
				fmt.Printf("  %s %-25s :%-6d\n",
					errorStyle.Render("✗"),
					fe.Name,
					fe.Port)
			}
		}
	}

	fmt.Println()
	ui.Info("Use 'musing deploy' to populate MongoDB with data")
	ui.Info("Use 'musing monitor' for live monitoring dashboard")
	ui.Info("Use 'musing dev --stop' to stop all services")
}
