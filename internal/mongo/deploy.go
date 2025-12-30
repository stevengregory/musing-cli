package mongo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Collection represents a discovered MongoDB collection
type Collection struct {
	Name    string // Collection name (derived from filename)
	File    string // Full path to JSON file
	IsArray bool   // Auto-detected by inspecting file
}

// DiscoverCollections scans the data directory and auto-discovers JSON files
func DiscoverCollections(dataDir string) (map[string]Collection, error) {
	collections := make(map[string]Collection)

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		fileName := entry.Name()
		collectionName := strings.TrimSuffix(fileName, ".json")
		collectionName = strings.ReplaceAll(collectionName, "-", "_")

		filePath := filepath.Join(dataDir, fileName)

		// Auto-detect if file contains an array
		isArray, err := isJSONArray(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect %s: %w", fileName, err)
		}

		// Use filename without extension as the key
		key := strings.TrimSuffix(fileName, ".json")

		collections[key] = Collection{
			Name:    collectionName,
			File:    filePath,
			IsArray: isArray,
		}
	}

	return collections, nil
}

// isJSONArray checks if a JSON file contains an array at the root level
func isJSONArray(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	// Trim whitespace and check first character
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return false, fmt.Errorf("empty JSON file")
	}

	// Check if first character is '[' (array) or '{' (object)
	return trimmed[0] == '[', nil
}

// DeployCollection imports a single collection into MongoDB
func DeployCollection(uri, db, collectionKey, dataDir string) error {
	collections, err := DiscoverCollections(dataDir)
	if err != nil {
		return err
	}

	coll, exists := collections[collectionKey]
	if !exists {
		return fmt.Errorf("collection not found: %s (available: %v)", collectionKey, getCollectionKeys(collections))
	}

	args := []string{
		"--uri", uri,
		"--db", db,
		"--collection", coll.Name,
		"--file", coll.File,
		"--drop",
	}

	if coll.IsArray {
		args = append(args, "--jsonArray")
	}

	cmd := exec.Command("mongoimport", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// DeployAll imports all discovered collections
func DeployAll(uri, db, dataDir string) error {
	collections, err := DiscoverCollections(dataDir)
	if err != nil {
		return err
	}

	for key := range collections {
		if err := DeployCollection(uri, db, key, dataDir); err != nil {
			return fmt.Errorf("failed to deploy %s: %w", key, err)
		}
	}

	return nil
}

// getCollectionKeys returns a slice of collection keys for error messages
func getCollectionKeys(collections map[string]Collection) []string {
	keys := make([]string, 0, len(collections))
	for k := range collections {
		keys = append(keys, k)
	}
	return keys
}
