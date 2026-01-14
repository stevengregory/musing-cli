package cmd

import (
	"fmt"
	"os"

	figure "github.com/common-nighthawk/go-figure"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	// Version information - set by ldflags during build
	Version = "0.1.0"
	Commit  = "none"
	Date    = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "musing",
	Short: "CLI for managing multi-service development stacks",
	Long:  `A CLI tool for managing multi-service development stacks with Docker, MongoDB, and microservices.`,
	// Don't show usage on errors
	SilenceUsage: true,
	// Don't print errors (we'll handle them ourselves)
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip banner during shell completion
		if cmd.Flag("help") != nil && cmd.Flag("help").Changed {
			return nil
		}

		// Show banner for all commands except monitor, version, and completion
		// Skip for root command (handled in Run)
		if cmd.Name() != "musing" && shouldShowBanner(cmd) {
			printBanner()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Show banner for root help
		printBanner()
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Errors are already printed by commands, just exit
		os.Exit(1)
	}
}

func init() {
	// Set version
	rootCmd.Version = Version

	// Custom version template to match previous behavior (just the version number)
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Custom help function to show banner
	originalHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Show banner for root command help only when using --help flag
		// (not when using Run function which already shows it)
		if cmd.Name() == "musing" && cmd.Flags().Changed("help") {
			printBanner()
		}
		originalHelpFunc(cmd, args)
	})

	// Set command groups first
	rootCmd.AddGroup(&cobra.Group{
		ID:    "core",
		Title: "Core Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "additional",
		Title: "Additional Commands:",
	})

	// Add core commands with group IDs
	devCmd.GroupID = "core"
	deployCmd.GroupID = "core"
	monitorCmd.GroupID = "core"
	sshCmd.GroupID = "core"
	tunnelCmd.GroupID = "core"

	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(tunnelCmd)

	// Enable built-in completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = false

	// Mark built-in commands as additional (done after they're created)
	rootCmd.SetHelpCommandGroupID("additional")
	rootCmd.SetCompletionCommandGroupID("additional")
}

// printBanner displays the ASCII art banner
func printBanner() {
	banner := figure.NewFigure("Musing", "", true)
	bannerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)
	fmt.Println(bannerStyle.Render(banner.String()))
}

// shouldShowBanner returns true if the banner should be displayed for this command
func shouldShowBanner(cmd *cobra.Command) bool {
	// Check if we're in shell completion mode
	if os.Getenv("COMP_LINE") != "" || os.Args[len(os.Args)-1] == cobra.ShellCompRequestCmd {
		return false
	}

	// Check if this is a version flag
	if cmd.Flags().Changed("version") {
		return false
	}

	// Skip banner for these commands and their subcommands
	current := cmd
	for current != nil {
		switch current.Name() {
		case "monitor", "completion", "bash", "zsh", "fish", "powershell", "help", cobra.ShellCompRequestCmd:
			return false
		}
		current = current.Parent()
	}

	return true
}
