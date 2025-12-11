package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/stevengregory/musing-cli/internal/ui"
	"github.com/urfave/cli/v2"
)

func StatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show live development stack status",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "watch",
				Usage: "Continuously monitor services (updates every 2s)",
				Value: false,
			},
		},
		Action: func(c *cli.Context) error {
			watch := c.Bool("watch")

			if watch {
				return runWatchMode()
			}

			return showStatus()
		},
	}
}

func showStatus() error {
	ui.Header("🚀 Development Stack Status")

	// Check core services
	fmt.Println()
	ui.Style("Core Services", "--foreground", "212", "--bold", "--underline")
	fmt.Println()

	// Check MongoDB
	mongoStatus := health.CheckPort(config.MongoDevPort)
	if mongoStatus.Open {
		ui.ServiceStatus("MongoDB", "running", config.MongoDevPort, health.FormatLatency(mongoStatus.Latency))
	} else {
		ui.ServiceStatus("MongoDB", "down", config.MongoDevPort, "timeout")
	}

	// Check Angular
	frontendStatus := health.CheckPort(config.AngularPort)
	if frontendStatus.Open {
		ui.ServiceStatus("Angular", "running", config.AngularPort, health.FormatLatency(frontendStatus.Latency))
	} else {
		ui.ServiceStatus("Angular", "down", config.AngularPort, "timeout")
	}

	// Check API services
	fmt.Println()
	ui.Style("API Services", "--foreground", "212", "--bold", "--underline")
	fmt.Println()

	for _, svc := range config.APIServices {
		status := health.CheckPort(svc.Port)
		if status.Open {
			ui.ServiceStatus(svc.Name, "running", svc.Port, health.FormatLatency(status.Latency))
		} else {
			ui.ServiceStatus(svc.Name, "down", svc.Port, "timeout")
		}
	}

	fmt.Println()
	ui.Info("Use 'musing status --watch' for live monitoring")

	return nil
}

func runWatchMode() error {
	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Render initial state
	ui.ClearScreen()
	renderWatchScreen()

	for {
		select {
		case <-sigChan:
			// Clean exit on Ctrl+C
			fmt.Println()
			ui.Info("Stopped monitoring")
			return nil

		case <-ticker.C:
			ui.ClearScreen()
			renderWatchScreen()
		}
	}
}

func renderWatchScreen() {
	// Header
	ui.Header("🚀 Development Stack - Live Monitor")

	// Current time
	fmt.Println()
	ui.Style(fmt.Sprintf("Last updated: %s", time.Now().Format("15:04:05")),
		"--foreground", "246",
		"--italic",
	)

	// Core Services Section
	fmt.Println()
	renderCoreServices()

	// API Services Section
	fmt.Println()
	renderAPIServices()

	// Footer
	fmt.Println()
	ui.Style("Press Ctrl+C to stop monitoring",
		"--foreground", "246",
		"--italic",
	)
}

func renderCoreServices() {
	ui.Style("┌─ Core Services ────────────────────────────────────────┐",
		"--foreground", "212",
	)

	// MongoDB
	mongoStatus := health.CheckPort(config.MongoDevPort)
	if mongoStatus.Open {
		fmt.Print("│ ")
		ui.ServiceStatus("MongoDB", "running", config.MongoDevPort, health.FormatLatency(mongoStatus.Latency))
	} else {
		fmt.Print("│ ")
		ui.ServiceStatus("MongoDB", "down", config.MongoDevPort, "timeout")
	}

	// Angular
	frontendStatus := health.CheckPort(config.AngularPort)
	if frontendStatus.Open {
		fmt.Print("│ ")
		ui.ServiceStatus("Angular", "running", config.AngularPort, health.FormatLatency(frontendStatus.Latency))
	} else {
		fmt.Print("│ ")
		ui.ServiceStatus("Angular", "down", config.AngularPort, "timeout")
	}

	ui.Style("└────────────────────────────────────────────────────────┘",
		"--foreground", "212",
	)
}

func renderAPIServices() {
	ui.Style("┌─ API Services (8) ─────────────────────────────────────┐",
		"--foreground", "212",
	)

	for _, svc := range config.APIServices {
		status := health.CheckPort(svc.Port)
		fmt.Print("│ ")
		if status.Open {
			ui.ServiceStatus(svc.Name, "running", svc.Port, health.FormatLatency(status.Latency))
		} else {
			ui.ServiceStatus(svc.Name, "down", svc.Port, "timeout")
		}
	}

	ui.Style("└────────────────────────────────────────────────────────┘",
		"--foreground", "212",
	)
}
