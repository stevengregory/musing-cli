package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/docker"
	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/stevengregory/musing-cli/internal/ui"
	"github.com/urfave/cli/v2"
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

func DevCommand() *cli.Command {
	return &cli.Command{
		Name:  "dev",
		Usage: "Manage development stack",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "rebuild",
				Usage: "Force rebuild all Docker images",
			},
			&cli.BoolFlag{
				Name:  "data",
				Usage: "Deploy MongoDB data after starting services",
			},
			&cli.BoolFlag{
				Name:  "logs",
				Usage: "Follow logs after starting services",
			},
			&cli.BoolFlag{
				Name:  "stop",
				Usage: "Stop all services and exit",
			},
		},
		Action: func(c *cli.Context) error {
			// Handle stop flag
			if c.Bool("stop") {
				return stopServices()
			}

			// Start services
			return startServices(c.Bool("rebuild"), c.Bool("data"), c.Bool("logs"))
		},
	}
}

func stopServices() error {
	fmt.Println(devHeaderStyle.Render("Stopping Development Stack"))

	// Change to project root directory
	if err := changeToProjectRoot(); err != nil {
		ui.Error("Failed to find project root directory")
		return err
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
	// Strategy: Always target ~/Repos/steven (the main repo), never worktrees
	home := os.Getenv("HOME")
	mainRepoPath := filepath.Join(home, "Repos", "steven")

	// Check if main repo exists and has compose.yaml
	composePath := filepath.Join(mainRepoPath, "compose.yaml")
	if _, err := os.Stat(composePath); err == nil {
		return os.Chdir(mainRepoPath)
	}

	// Fallback: Search for compose.yaml starting from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Check current directory
	if _, err := os.Stat("compose.yaml"); err == nil {
		// If we're in a worktree, skip it and continue searching
		if !isWorktree(cwd) {
			return nil
		}
	}

	// Check parent directory
	parent := filepath.Dir(cwd)
	parentCompose := filepath.Join(parent, "compose.yaml")
	if _, err := os.Stat(parentCompose); err == nil {
		if !isWorktree(parent) {
			return os.Chdir(parent)
		}
	}

	// Search sibling directories
	entries, err := os.ReadDir(parent)
	if err != nil {
		return fmt.Errorf("could not read parent directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			siblingPath := filepath.Join(parent, entry.Name())
			siblingCompose := filepath.Join(siblingPath, "compose.yaml")
			if _, err := os.Stat(siblingCompose); err == nil {
				if !isWorktree(siblingPath) {
					return os.Chdir(siblingPath)
				}
			}
		}
	}

	return fmt.Errorf("could not find compose.yaml in main repository")
}

// isWorktree checks if a directory is a git worktree (not the main repository)
func isWorktree(path string) bool {
	// Check if .git is a file (worktrees have .git as a file pointing to the real git dir)
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// In a worktree, .git is a file. In main repo, .git is a directory
	return !info.IsDir()
}

func startServices(rebuild, shouldDeployData, followLogs bool) error {
	fmt.Println(devHeaderStyle.Render("Development Stack"))

	// Change to project root directory (parent of musing-cli)
	if err := changeToProjectRoot(); err != nil {
		ui.Error("Failed to find project root directory")
		return err
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

	// Check Docker Desktop
	dockerRunning := docker.CheckRunning() == nil

	// Check all services
	mongoStatus := health.CheckPort(config.MongoDevPort)
	angularStatus := health.CheckPort(config.AngularPort)

	var apiStatuses []struct {
		name   string
		port   int
		status health.PortStatus
	}

	for _, svc := range config.APIServices {
		status := health.CheckPort(svc.Port)
		apiStatuses = append(apiStatuses, struct {
			name   string
			port   int
			status health.PortStatus
		}{svc.Name, svc.Port, status})
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

	// Database section
	fmt.Println(sectionHeaderStyle.Render("━━━ Database ━━━"))
	fmt.Println()
	if mongoStatus.Open {
		fmt.Printf("  %s %-25s :%-6d\n",
			checkmarkStyle.Render("✓"),
			"MongoDB",
			config.MongoDevPort)
	} else {
		fmt.Printf("  %s %-25s :%-6d\n",
			errorStyle.Render("✗"),
			"MongoDB",
			config.MongoDevPort)
	}
	fmt.Println()

	// API Services section
	fmt.Println(sectionHeaderStyle.Render(fmt.Sprintf("━━━ API Services (%d) ━━━", len(apiStatuses))))
	fmt.Println()
	for _, api := range apiStatuses {
		if api.status.Open {
			fmt.Printf("  %s %-25s :%-6d\n",
				checkmarkStyle.Render("✓"),
				api.name,
				api.port)
		} else {
			fmt.Printf("  %s %-25s :%-6d\n",
				errorStyle.Render("✗"),
				api.name,
				api.port)
		}
	}
	fmt.Println()

	// Frontend section
	fmt.Println(sectionHeaderStyle.Render("━━━ Frontend ━━━"))
	fmt.Println()
	if angularStatus.Open {
		fmt.Printf("  %s %-25s :%-6d\n",
			checkmarkStyle.Render("✓"),
			"Angular",
			config.AngularPort)
	} else {
		fmt.Printf("  %s %-25s :%-6d\n",
			errorStyle.Render("✗"),
			"Angular",
			config.AngularPort)
	}

	fmt.Println()
	ui.Info("Use 'musing deploy' to populate MongoDB with data")
	ui.Info("Use 'musing monitor' for live monitoring dashboard")
	ui.Info("Use 'musing dev --stop' to stop all services")
}
