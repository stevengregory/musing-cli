package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	MongoDevPort  = 27018
	MongoProdPort = 27019
	AngularPort   = 3000
)

// API service configurations
type ServiceConfig struct {
	Name string
	Port int
	Path string // Health check path
}

var APIServices = []ServiceConfig{
	{Name: "networks-api", Port: 8085, Path: "/health"},
	{Name: "random-facts-api", Port: 8082, Path: "/health"},
	{Name: "alcohol-free-api", Port: 8081, Path: "/health"},
	{Name: "random-quotes-api", Port: 8083, Path: "/health"},
	{Name: "news-api", Port: 8084, Path: "/health"},
	{Name: "about-me-api", Port: 8086, Path: "/health"},
	{Name: "featured-item-api", Port: 8087, Path: "/health"},
	{Name: "bitcoin-price-api", Port: 8088, Path: "/health"},
}

// FindProjectRoot searches upward from CWD for a directory containing .musing marker file
func FindProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Search upward from current directory
	dir := currentDir
	for {
		// Check if this directory contains .musing file
		musingPath := filepath.Join(dir, ".musing")
		if _, err := os.Stat(musingPath); err == nil {
			// Found .musing file, verify compose.yaml exists
			if hasComposeFile(dir) {
				return dir, nil
			}
			return "", fmt.Errorf("found .musing at %s but no compose.yaml", dir)
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no .musing file found (searched upward from %s)", currentDir)
}

// hasComposeFile checks if directory contains compose.yaml
func hasComposeFile(dir string) bool {
	composePath := filepath.Join(dir, "compose.yaml")
	_, err := os.Stat(composePath)
	return err == nil
}

// GetAPIRepos returns paths to expected API repositories
// Dynamically discovers repos relative to project root
func GetAPIRepos() []string {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		// Fallback to empty list if project root not found
		return []string{}
	}

	parentDir := filepath.Dir(projectRoot)
	var repos []string

	for _, svc := range APIServices {
		repoPath := filepath.Join(parentDir, svc.Name)
		repos = append(repos, repoPath)
	}

	return repos
}
