package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// CheckRunning checks if Docker daemon is running
func CheckRunning() error {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker is not running")
	}

	return nil
}

// IsDockerDesktopInstalled checks if Docker Desktop is installed on macOS
func IsDockerDesktopInstalled() bool {
	_, err := os.Stat("/Applications/Docker.app")
	return err == nil
}

// EnsureRunning checks Docker status and starts it if needed
func EnsureRunning(promptUser bool) error {
	// Check if already running
	if err := CheckRunning(); err == nil {
		return nil // Docker is running
	}

	// Check if Docker Desktop is installed
	if !IsDockerDesktopInstalled() {
		return fmt.Errorf("Docker Desktop is not installed. Please install from https://docker.com")
	}

	// Prompt user if requested
	if promptUser {
		fmt.Println("\n⚠️  Docker is not running.")
		fmt.Print("Start Docker Desktop? [Y/n]: ")

		var response string
		fmt.Scanln(&response)

		if response == "n" || response == "N" {
			return fmt.Errorf("Docker is required to continue")
		}
	} else {
		fmt.Println("\n⚠️  Docker is not running. Starting Docker Desktop...")
	}

	// Start Docker Desktop
	if err := startDockerDesktop(); err != nil {
		return fmt.Errorf("failed to start Docker Desktop: %w", err)
	}

	// Wait for Docker to be ready
	fmt.Println("⏳ Waiting for Docker to be ready (this may take 30-60 seconds)...")
	return waitForReady(90 * time.Second)
}

// startDockerDesktop starts Docker Desktop app
func startDockerDesktop() error {
	// Try official Docker Desktop CLI first (v4.37+)
	cmd := exec.Command("docker", "desktop", "start")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback to open command (works on all versions)
	cmd = exec.Command("open", "-a", "Docker", "-g") // -g = don't bring to front
	return cmd.Run()
}

// waitForReady polls docker info until ready or timeout
func waitForReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	lastProgress := 0

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Docker after %v", time.Since(startTime))

		case <-ticker.C:
			if CheckRunning() == nil {
				fmt.Println("✅ Docker is ready!")
				return nil
			}

			// Show progress every 10 seconds
			elapsed := int(time.Since(startTime).Seconds())
			if elapsed > 0 && elapsed%10 == 0 && elapsed != lastProgress {
				fmt.Printf("   Still waiting... (%ds)\n", elapsed)
				lastProgress = elapsed
			}
		}
	}
}

// ComposeUp starts services with docker compose
func ComposeUp() error {
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ComposeDown stops all services
func ComposeDown() error {
	cmd := exec.Command("docker", "compose", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ComposeBuild builds images
func ComposeBuild(noCache bool) error {
	args := []string{"compose", "build"}
	if noCache {
		args = append(args, "--no-cache")
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ComposeLogs follows logs from all services
func ComposeLogs(follow bool) error {
	args := []string{"compose", "logs"}
	if follow {
		args = append(args, "-f")
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
