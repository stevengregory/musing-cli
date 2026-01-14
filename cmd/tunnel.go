package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/health"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage SSH tunnel to production database",
	Long:  `Start, stop, or check status of SSH tunnel for production database access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to start
		return tunnelStart()
	},
}

var tunnelStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start SSH tunnel",
	Long:  `Start SSH tunnel for production database access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tunnelStart()
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop SSH tunnel",
	Long:  `Stop the SSH tunnel to production database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tunnelStop()
	},
}

var tunnelStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check tunnel status",
	Long:  `Check if the SSH tunnel is currently running.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tunnelStatus()
	},
}

func init() {
	tunnelCmd.AddCommand(tunnelStartCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelStatusCmd)
}

func tunnelStart() error {
	config.MustFindProjectRoot()
	cfg := config.GetConfig()

	if cfg.Production == nil {
		return fmt.Errorf("production configuration not found in .musing.yaml")
	}

	prodPort := cfg.Database.ProdPort
	if prodPort == 0 {
		prodPort = 27019 // Default prod port
	}

	// Check if tunnel is already running
	if health.CheckPort(prodPort).Open {
		return tunnelStatus()
	}

	// Build SSH command
	remotePort := cfg.Production.RemoteDBPort
	if remotePort == 0 {
		remotePort = 27017 // Default MongoDB port
	}

	sshArgs := []string{
		"-f", // Fork to background
		"-N", // No remote command
	}

	// Add SSH key if specified
	if cfg.Production.SSHKeyPath != "" {
		// Expand ~ to home directory
		keyPath := cfg.Production.SSHKeyPath
		if strings.HasPrefix(keyPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				keyPath = strings.Replace(keyPath, "~", homeDir, 1)
			}
		}
		sshArgs = append(sshArgs, "-i", keyPath)
	}

	sshArgs = append(sshArgs, "-L", fmt.Sprintf("%d:localhost:%d", prodPort, remotePort))
	sshArgs = append(sshArgs, cfg.Production.Server)

	// Start SSH tunnel in background
	cmd := exec.Command("ssh", sshArgs...)

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(warningStyle.Render("✗") + " Failed to start SSH tunnel")
		fmt.Println()

		// Check if it's likely a password/key issue
		if strings.Contains(string(output), "Permission denied") ||
		   strings.Contains(string(output), "password") {
			fmt.Println("SSH authentication failed. Please ensure:")
			fmt.Println("  1. SSH key authentication is set up (recommended)")
			fmt.Println("  2. Or run: ssh-copy-id " + cfg.Production.Server)
			fmt.Println()
			return fmt.Errorf("SSH authentication required")
		}

		if len(output) > 0 {
			fmt.Println("SSH error:", string(output))
		}
		return fmt.Errorf("failed to start SSH tunnel: %w", err)
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	fmt.Println()
	fmt.Println(successStyle.Render("✓") + " SSH tunnel started")
	fmt.Println(infoStyle.Render("  Local port:  ") + strconv.Itoa(prodPort))
	fmt.Println(infoStyle.Render("  Remote:      ") + cfg.Production.Server)
	fmt.Println(infoStyle.Render("  Remote port: ") + strconv.Itoa(remotePort))
	fmt.Println()
	fmt.Println("  Use 'musing tunnel stop' to close the tunnel")

	return nil
}

func tunnelStop() error {
	config.MustFindProjectRoot()
	cfg := config.GetConfig()

	prodPort := cfg.Database.ProdPort
	if prodPort == 0 {
		prodPort = 27019
	}

	// Check if tunnel is running
	if !health.CheckPort(prodPort).Open {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println()
		fmt.Println(warningStyle.Render("✓") + " SSH tunnel is not running")
		return nil
	}

	// Find process using the port
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", prodPort))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find tunnel process (is lsof installed?): %w", err)
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return fmt.Errorf("no process found on port %d", prodPort)
	}

	// Kill the process
	if err := exec.Command("kill", pid).Run(); err != nil {
		return fmt.Errorf("failed to stop tunnel: %w", err)
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Println()
	fmt.Println(successStyle.Render("✓") + " SSH tunnel stopped")
	return nil
}

func tunnelStatus() error {
	config.MustFindProjectRoot()
	cfg := config.GetConfig()

	if cfg.Production == nil {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(warningStyle.Render("✗") + " Production configuration not found in .musing.yaml")
		return nil
	}

	prodPort := cfg.Database.ProdPort
	if prodPort == 0 {
		prodPort = 27019
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	fmt.Println()
	fmt.Println(headerStyle.Render("SSH Tunnel Status"))
	fmt.Println()

	portStatus := health.CheckPort(prodPort)
	var statusIcon string
	var statusText string

	if portStatus.Open {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		statusIcon = successStyle.Render("●")
		statusText = successStyle.Render("Running")
	} else {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		statusIcon = errorStyle.Render("●")
		statusText = errorStyle.Render("Stopped")
	}

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	fmt.Printf("%s Status:      %s\n", statusIcon, statusText)
	fmt.Println(infoStyle.Render("  Local port:  ") + strconv.Itoa(prodPort))
	fmt.Println(infoStyle.Render("  Remote:      ") + cfg.Production.Server)

	remotePort := cfg.Production.RemoteDBPort
	if remotePort == 0 {
		remotePort = 27017
	}
	fmt.Println(infoStyle.Render("  Remote port: ") + strconv.Itoa(remotePort))

	if !portStatus.Open {
		fmt.Println()
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		fmt.Println(dimStyle.Render("  Run 'musing tunnel start' to start the tunnel"))
	}

	fmt.Println()

	return nil
}

