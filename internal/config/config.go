package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// FindProjectRoot reads the project root path from ~/.musingrc config file
func FindProjectRoot() (string, error) {
	home := os.Getenv("HOME")
	configPath := filepath.Join(home, ".musingrc")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("~/.musingrc not found - create it with your project path (e.g., /Users/you/Repos/project)")
	}

	// Trim whitespace and clean the path
	projectPath := filepath.Clean(strings.TrimSpace(string(data)))
	if !hasComposeFile(projectPath) {
		return "", fmt.Errorf("~/.musingrc points to %s which does not contain compose.yaml", projectPath)
	}

	return projectPath, nil
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
