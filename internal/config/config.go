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

// FindProjectRoot searches for the project root containing compose.yaml
// Priority: 1) MUSING_PROJECT_ROOT env var, 2) ~/.musingrc config file, 3) Auto-discovery
func FindProjectRoot() (string, error) {
	// 1. Check environment variable first (highest priority)
	if envPath := os.Getenv("MUSING_PROJECT_ROOT"); envPath != "" {
		if hasComposeFile(envPath) {
			return envPath, nil
		}
		return "", fmt.Errorf("MUSING_PROJECT_ROOT=%s does not contain compose.yaml", envPath)
	}

	// 2. Check config file
	home := os.Getenv("HOME")
	configPath := filepath.Join(home, ".musingrc")
	if data, err := os.ReadFile(configPath); err == nil {
		projectPath := string(data)
		projectPath = filepath.Clean(projectPath)
		if hasComposeFile(projectPath) {
			return projectPath, nil
		}
		return "", fmt.Errorf("~/.musingrc points to %s which does not contain compose.yaml", projectPath)
	}

	// 3. Fallback to auto-discovery
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check current directory
	if hasComposeFile(cwd) && !isWorktree(cwd) {
		return cwd, nil
	}

	// Check parent directory
	parent := filepath.Dir(cwd)
	if hasComposeFile(parent) && !isWorktree(parent) {
		return parent, nil
	}

	// Search sibling directories
	entries, err := os.ReadDir(parent)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				siblingPath := filepath.Join(parent, entry.Name())
				if hasComposeFile(siblingPath) && !isWorktree(siblingPath) {
					return siblingPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find compose.yaml in project root")
}

// hasComposeFile checks if directory contains compose.yaml
func hasComposeFile(dir string) bool {
	composePath := filepath.Join(dir, "compose.yaml")
	_, err := os.Stat(composePath)
	return err == nil
}

// isWorktree checks if a directory is a git worktree (not the main repository)
func isWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// In a worktree, .git is a file. In main repo, .git is a directory
	return !info.IsDir()
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
