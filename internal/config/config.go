package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ProjectConfig represents the .musing.yaml configuration
type ProjectConfig struct {
	Services    []ServiceConfig   `yaml:"services"`
	Database    DatabaseConfig    `yaml:"database"`
	Production  *ProductionConfig `yaml:"production,omitempty"`  // Optional production config
	APIReposDir string            `yaml:"apiReposDir,omitempty"` // Optional. Directory holding API repos. Relative paths resolve against the project root's parent. Absolute and ~-prefixed paths are honored as-is. Defaults to the project root's parent (sibling layout).
}

// ServiceConfig represents a service in the stack
type ServiceConfig struct {
	Name       string `yaml:"name"`
	Port       int    `yaml:"port"`
	Type       string `yaml:"type"`                 // frontend, api, database
	Alias      string `yaml:"alias,omitempty"`      // Optional: short CLI name for `musing deploy`. Falls back to data file key when omitted.
	Collection string `yaml:"collection,omitempty"` // Optional: data file or directory key. When omitted, defaults to Alias.
	Schema     string `yaml:"schema,omitempty"`     // Optional schema filename under database.schemaDir; requires an explicit Collection.
}

func (service *ServiceConfig) UnmarshalYAML(node *yaml.Node) error {
	filename, declared, err := schemaStringField(node, "schema", "service schema")
	if err != nil {
		return err
	}
	if declared {
		if err := validateSchemaFilename(filename); err != nil {
			return err
		}
	}
	type serviceFields ServiceConfig
	var decoded serviceFields
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	*service = ServiceConfig(decoded)
	return nil
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"` // mongodb, postgres, etc
	Name     string `yaml:"name"` // Database name
	DevPort  int    `yaml:"devPort"`
	ProdPort int    `yaml:"prodPort"`
	DataDir  string `yaml:"dataDir"` // Relative path to data directory
	// Schemas retains source-key -> schema paths relative to the project root.
	//
	// Deprecated: use ServiceConfig.Schema and DatabaseConfig.SchemaDir.
	Schemas   CollectionSchemas `yaml:"schemas,omitempty"`
	SchemaDir string            `yaml:"schemaDir,omitempty"` // Service schema directory, relative to the project root; defaults to schemas.
}

func (database *DatabaseConfig) UnmarshalYAML(node *yaml.Node) error {
	if _, _, err := schemaStringField(node, "schemaDir", "database.schemaDir"); err != nil {
		return err
	}
	type databaseFields DatabaseConfig
	decoded := databaseFields{SchemaDir: "schemas"}
	if err := node.Decode(&decoded); err != nil {
		return err
	}
	*database = DatabaseConfig(decoded)
	return nil
}

func schemaStringField(node *yaml.Node, name, context string) (string, bool, error) {
	// Let YAML apply mapping-merge precedence before inspecting scalar types;
	// decoding straight into string fields would coerce numeric and boolean values.
	var fields map[string]yaml.Node
	if err := node.Decode(&fields); err != nil {
		return "", false, err
	}
	value, declared := fields[name]
	if !declared {
		return "", false, nil
	}
	resolved := &value
	seen := make(map[*yaml.Node]bool)
	for resolved.Kind == yaml.AliasNode {
		if resolved.Alias == nil || seen[resolved] {
			return "", true, fmt.Errorf("%s must be a nonempty string", context)
		}
		seen[resolved] = true
		resolved = resolved.Alias
	}
	if resolved.Kind != yaml.ScalarNode || resolved.Tag != "!!str" || strings.TrimSpace(resolved.Value) == "" {
		return "", true, fmt.Errorf("%s must be a nonempty string", context)
	}
	return resolved.Value, true, nil
}

func validateSchemaFilename(filename string) error {
	extension := filepath.Ext(filename)
	if strings.TrimSpace(filename) == "" || strings.ContainsAny(filename, "/\\:\x00") || filepath.IsAbs(filename) ||
		extension == "" || extension == "." || len(extension) == len(filename) {
		return fmt.Errorf("service schema must be a filename with an extension, without directories or drive prefixes")
	}
	return nil
}

// CollectionSchemas maps exact data source keys to schema files, without
// introducing validator modes or other collection options.
type CollectionSchemas map[string]string

func (schemas *CollectionSchemas) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("database.schemas must map collection source keys to schema file paths")
	}
	paths := make(CollectionSchemas, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || strings.TrimSpace(key.Value) == "" {
			return fmt.Errorf("database.schemas keys must be nonempty collection source strings")
		}
		if value.Kind != yaml.ScalarNode || value.Tag != "!!str" || strings.TrimSpace(value.Value) == "" {
			return fmt.Errorf("database.schemas[%q] must be a nonempty schema file path", key.Value)
		}
		if _, exists := paths[key.Value]; exists {
			return fmt.Errorf("duplicate database.schemas source key %q", key.Value)
		}
		paths[key.Value] = value.Value
	}
	*schemas = paths
	return nil
}

// ProductionConfig represents optional production deployment settings
type ProductionConfig struct {
	Server       string `yaml:"server"`       // SSH server (e.g., "root@your-server.com")
	RemoteDBPort int    `yaml:"remoteDBPort"` // Remote database port (typically same as devPort)
	SSHKeyPath   string `yaml:"sshKeyPath"`   // Optional SSH key path (e.g., "~/.ssh/digital-ocean/id_ed25519")
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
	config.Database.SchemaDir = "schemas"
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}
	absolutePath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("resolve project configuration path: %w", err)
	}
	if _, err := config.ResolveCollectionSchemas(filepath.Dir(absolutePath)); err != nil {
		return err
	}

	currentConfig = &config
	return nil
}

// ResolveCollectionSchemas merges legacy schema paths and service schema filenames.
// It returns absolute paths without reading schema files or inspecting collections.
func (config *ProjectConfig) ResolveCollectionSchemas(projectRoot string) (map[string]string, error) {
	if config == nil {
		return nil, fmt.Errorf("project configuration is required to resolve collection schemas")
	}
	schemaDir := config.Database.SchemaDir
	if schemaDir == "" {
		schemaDir = "schemas"
	} else if strings.TrimSpace(schemaDir) == "" {
		return nil, fmt.Errorf("database.schemaDir must be a nonempty directory path")
	}
	paths := make(map[string]string, len(config.Database.Schemas))
	hasSchemas := len(config.Database.Schemas) > 0
	for _, service := range config.Services {
		hasSchemas = hasSchemas || service.Schema != ""
	}
	if !hasSchemas {
		return paths, nil
	}
	if !strings.EqualFold(config.Database.Type, "MongoDB") {
		return nil, fmt.Errorf("collection schemas require database.type MongoDB")
	}
	if !filepath.IsAbs(projectRoot) {
		return nil, fmt.Errorf("an absolute project root is required to resolve collection schemas")
	}
	resolve := func(path string) string {
		if !filepath.IsAbs(path) {
			path = filepath.Join(projectRoot, path)
		}
		return filepath.Clean(path)
	}
	add := func(collection, path string) error {
		if existing, found := paths[collection]; found && existing != path {
			return fmt.Errorf("conflicting schemas for collection %q: %s and %s", collection, existing, path)
		}
		paths[collection] = path
		return nil
	}
	keys := make([]string, 0, len(config.Database.Schemas))
	for key := range config.Database.Schemas {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		path := config.Database.Schemas[key]
		if strings.TrimSpace(key) == "" || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("database.schemas requires nonempty collection source keys and schema file paths")
		}
		if err := add(key, resolve(path)); err != nil {
			return nil, err
		}
	}
	for _, service := range config.Services {
		if service.Schema == "" {
			continue
		}
		if err := validateSchemaFilename(service.Schema); err != nil {
			return nil, fmt.Errorf("service %q: %w", service.Name, err)
		}
		if strings.TrimSpace(service.Collection) == "" {
			return nil, fmt.Errorf("service %q schema requires an explicit nonempty collection", service.Name)
		}
		if err := add(service.Collection, filepath.Join(resolve(schemaDir), service.Schema)); err != nil {
			return nil, err
		}
	}
	return paths, nil
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

	parentDir := resolveAPIReposParent(projectRoot, currentConfig.APIReposDir)
	var repos []string

	for _, svc := range currentConfig.Services {
		if svc.Type == "api" {
			repoPath := filepath.Join(parentDir, svc.Name)
			repos = append(repos, repoPath)
		}
	}

	return repos
}

// resolveAPIReposParent picks the directory that contains API repos.
// Empty input falls back to the project root's parent (sibling layout).
// "~/" is expanded to the user's home directory; absolute paths are honored
// as-is; otherwise the value is treated as relative to the project root's parent.
func resolveAPIReposParent(projectRoot, apiReposDir string) string {
	if apiReposDir == "" {
		return filepath.Dir(projectRoot)
	}
	if strings.HasPrefix(apiReposDir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, apiReposDir[2:])
		}
	}
	if filepath.IsAbs(apiReposDir) {
		return apiReposDir
	}
	return filepath.Join(filepath.Dir(projectRoot), apiReposDir)
}
