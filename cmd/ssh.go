package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stevengregory/musing-cli/internal/config"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Open interactive SSH session to production server",
	Long:  `Open an interactive SSH session to the production server configured in .musing.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config.MustFindProjectRoot()
		cfg := config.GetConfig()

		if cfg.Production == nil {
			return fmt.Errorf("production configuration not found in .musing.yaml")
		}

		// Build SSH command using shared helper
		sshArgs := buildSSHArgs(cfg, false) // false = no tunnel

		// Execute SSH interactively
		sshCmd := exec.Command("ssh", sshArgs...)
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		return sshCmd.Run()
	},
}

func init() {
	sshCmd.GroupID = "core"
}

// buildSSHArgs creates SSH arguments with optional tunnel configuration
func buildSSHArgs(cfg *config.ProjectConfig, withTunnel bool) []string {
	var args []string

	// Add SSH key if specified
	if cfg.Production.SSHKeyPath != "" {
		keyPath := expandHomeDir(cfg.Production.SSHKeyPath)
		args = append(args, "-i", keyPath)
	}

	// Add tunnel configuration if requested
	if withTunnel {
		args = append(args, "-f") // Fork to background
		args = append(args, "-N") // No remote command

		prodPort := cfg.Database.ProdPort
		if prodPort == 0 {
			prodPort = 27019
		}

		remotePort := cfg.Production.RemoteDBPort
		if remotePort == 0 {
			remotePort = 27017
		}

		args = append(args, "-L", fmt.Sprintf("%d:localhost:%d", prodPort, remotePort))
	}

	args = append(args, cfg.Production.Server)
	return args
}

// expandHomeDir expands ~ to the user's home directory
func expandHomeDir(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return strings.Replace(path, "~", homeDir, 1)
		}
	}
	return path
}
