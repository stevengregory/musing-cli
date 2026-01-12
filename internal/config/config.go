package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig represents the .musing.yaml configuration
type ProjectConfig struct {
	Services   []ServiceConfig   `yaml:"services"`
	Database   DatabaseConfig    `yaml:"database"`
	Production *ProductionConfig `yaml:"production,omitempty"` // Optional production config
}

// ServiceConfig represents a service in the stack
type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
	Type string `yaml:"type"` // frontend, api, database
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"` // mongodb, postgres, etc
	Name     string `yaml:"name"` // Database name
	DevPort  int    `yaml:"devPort"`
	ProdPort int    `yaml:"prodPort"`
	DataDir  string `yaml:"dataDir"` // Relative path to data directory
}

// ProductionConfig represents optional production deployment settings
type ProductionConfig struct {
	Server       string `yaml:"server"`       // SSH server (e.g., "root@your-server.com")
	RemoteDBPort int    `yaml:"remoteDBPort"` // Remote database port (typically same as devPort)
}

var currentConfig *ProjectConfig

// FindProjectRoot searches upward from CWD for a directory containing .musing.yaml
// and loads the project configuration
func FindProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Search upward from current directory
	dir := currentDir
	for {
		// Check if this directory contains .musing.yaml file
		musingPath := filepath.Join(dir, ".musing.yaml")
		if _, err := os.Stat(musingPath); err == nil {
			// Found .musing.yaml file, load config
			if err := loadConfig(musingPath); err != nil {
				return "", fmt.Errorf("failed to load config from %s: %w", musingPath, err)
			}

			// Verify compose.yaml exists
			if !hasComposeFile(dir) {
				return "", fmt.Errorf("found .musing.yaml at %s but no compose.yaml", dir)
			}

			return dir, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no .musing.yaml file found (searched upward from %s)", currentDir)
}

// loadConfig reads and parses the .musing.yaml configuration file
func loadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	currentConfig = &config
	return nil
}

// GetConfig returns the loaded project configuration
func GetConfig() *ProjectConfig {
	return currentConfig
}

// MustFindProjectRoot finds the project root or exits with a helpful error message
func MustFindProjectRoot() string {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		fmt.Println()
		fmt.Println("\033[31m✗\033[0m Could not find project root")
		fmt.Println("\033[36mℹ\033[0m Run this command from inside a project with .musing.yaml")
		os.Exit(1)
	}
	return projectRoot
}

// hasComposeFile checks if directory contains compose.yaml
func hasComposeFile(dir string) bool {
	composePath := filepath.Join(dir, "compose.yaml")
	_, err := os.Stat(composePath)
	return err == nil
}

// GetAPIRepos returns paths to expected API repositories
// Dynamically discovers repos relative to project root based on config
func GetAPIRepos() []string {
	if currentConfig == nil {
		return []string{}
	}

	projectRoot, err := FindProjectRoot()
	if err != nil {
		return []string{}
	}

	parentDir := filepath.Dir(projectRoot)
	var repos []string

	for _, svc := range currentConfig.Services {
		if svc.Type == "api" {
			repoPath := filepath.Join(parentDir, svc.Name)
			repos = append(repos, repoPath)
		}
	}

	return repos
}
