package cmd

import (
	"fmt"
	"os"

	"github.com/stevengregory/musing-cli/internal/config"
)

// loadProjectConfig discovers the project root and returns the loaded config.
func loadProjectConfig() (string, *config.ProjectConfig, error) {
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return "", nil, fmt.Errorf("could not find project root: %w", err)
	}

	cfg := config.GetConfig()
	if cfg == nil {
		return "", nil, fmt.Errorf("no configuration loaded")
	}

	return projectRoot, cfg, nil
}

// changeToProjectRoot changes cwd to the discovered project root.
func changeToProjectRoot() (*config.ProjectConfig, error) {
	projectRoot, cfg, err := loadProjectConfig()
	if err != nil {
		return nil, err
	}

	if err := os.Chdir(projectRoot); err != nil {
		return nil, fmt.Errorf("failed to change to project root: %w", err)
	}

	return cfg, nil
}
