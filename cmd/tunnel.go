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
	Long:  `Start SSH tunnel for production database access. Use --stop flag to close the tunnel.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stop, _ := cmd.Flags().GetBool("stop")

		if stop {
			return tunnelStop()
		}

		// Default behavior: start tunnel or show status if already running
		return tunnelStart()
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
	tunnelCmd.Flags().Bool("stop", false, "Stop SSH tunnel")
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
		"-N", // No remote command
		"-L", fmt.Sprintf("%d:localhost:%d", prodPort, remotePort),
		cfg.Production.Server,
	}

	// Start SSH tunnel in background
	cmd := exec.Command("ssh", sshArgs...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH tunnel: %w", err)
	}

	// Save PID for later stopping
	pidFile := getTunnelPIDFile()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		// Non-fatal, just warn
		fmt.Printf("Warning: Could not save tunnel PID: %v\n", err)
	}

	// Give it a moment to establish
	// time.Sleep(1 * time.Second)

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	fmt.Println(successStyle.Render("✓") + " SSH tunnel started")
	fmt.Println(infoStyle.Render("  Local port:  ") + strconv.Itoa(prodPort))
	fmt.Println(infoStyle.Render("  Remote:      ") + cfg.Production.Server)
	fmt.Println(infoStyle.Render("  Remote port: ") + strconv.Itoa(remotePort))

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
		fmt.Println(warningStyle.Render("✓") + " SSH tunnel is not running")
		return nil
	}

	// Try to kill by PID first
	pidFile := getTunnelPIDFile()
	if data, err := os.ReadFile(pidFile); err == nil {
		pid := strings.TrimSpace(string(data))
		if err := exec.Command("kill", pid).Run(); err == nil {
			os.Remove(pidFile)
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
			fmt.Println(successStyle.Render("✓") + " SSH tunnel stopped")
			return nil
		}
	}

	// Fallback: kill by port
	// Find process using the port
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", prodPort))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find tunnel process: %w", err)
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return fmt.Errorf("no process found on port %d", prodPort)
	}

	// Kill the process
	if err := exec.Command("kill", pid).Run(); err != nil {
		return fmt.Errorf("failed to stop tunnel: %w", err)
	}

	os.Remove(pidFile)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
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
		fmt.Println(dimStyle.Render("  Run 'musing tunnel' to start the tunnel"))
	}

	fmt.Println()

	return nil
}

func getTunnelPIDFile() string {
	return "/tmp/musing-tunnel.pid"
}
