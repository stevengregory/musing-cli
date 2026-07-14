package mongo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Collection represents a discovered MongoDB collection
type Collection struct {
	Name    string   // MongoDB collection name derived from the source key
	File    string   // Full path to a single JSON source file
	Files   []string // Ordered JSON object files for a directory-backed collection
	IsArray bool     // Whether a single source file contains a JSON array
}

// DiscoverCollections scans the data directory for JSON files and immediate
// subdirectories containing one JSON object per file.
func DiscoverCollections(dataDir string) (map[string]Collection, error) {
	collections := make(map[string]Collection)

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			collection, found, err := discoverDirectoryCollection(dataDir, entry.Name())
			if err != nil {
				return nil, err
			}
			if found {
				if err := addCollection(collections, entry.Name(), collection); err != nil {
					return nil, err
				}
			}
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		fileName := entry.Name()
		key := strings.TrimSuffix(fileName, ".json")
		filePath := filepath.Join(dataDir, fileName)

		isArray, err := isJSONArray(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect %s: %w", fileName, err)
		}

		collection := Collection{
			Name:    mongoCollectionName(key),
			File:    filePath,
			IsArray: isArray,
		}
		if err := addCollection(collections, key, collection); err != nil {
			return nil, err
		}
	}

	return collections, nil
}

func discoverDirectoryCollection(dataDir, name string) (Collection, bool, error) {
	dirPath := filepath.Join(dataDir, name)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return Collection{}, false, fmt.Errorf("failed to read collection directory %s: %w", name, err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		if _, err := readJSONObject(filePath); err != nil {
			return Collection{}, false, fmt.Errorf("failed to inspect %s: %w", filepath.Join(name, entry.Name()), err)
		}
		files = append(files, filePath)
	}

	if len(files) == 0 {
		return Collection{}, false, nil
	}

	return Collection{
		Name:  mongoCollectionName(name),
		Files: files,
	}, true, nil
}

func addCollection(collections map[string]Collection, key string, collection Collection) error {
	if existing, ok := collections[key]; ok {
		return fmt.Errorf(
			"duplicate collection key %q from %s and %s",
			key,
			collectionSource(existing),
			collectionSource(collection),
		)
	}
	for existingKey, existing := range collections {
		if existing.Name == collection.Name {
			return fmt.Errorf(
				"collection keys %q and %q both map to MongoDB collection %q",
				existingKey,
				key,
				collection.Name,
			)
		}
	}
	collections[key] = collection
	return nil
}

func collectionSource(collection Collection) string {
	if collection.File != "" {
		return collection.File
	}
	if len(collection.Files) > 0 {
		return filepath.Dir(collection.Files[0])
	}
	return "unknown source"
}

func mongoCollectionName(key string) string {
	return strings.ReplaceAll(key, "-", "_")
}

// isJSONArray checks if a JSON file contains an array at the root level
func isJSONArray(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false, fmt.Errorf("empty JSON file")
	}
	if !json.Valid(trimmed) {
		return false, fmt.Errorf("invalid JSON")
	}

	switch trimmed[0] {
	case '[':
		return true, nil
	case '{':
		return false, nil
	default:
		return false, fmt.Errorf("expected a JSON object or array")
	}
}

func readJSONObject(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty JSON file")
	}
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("invalid JSON")
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("expected one JSON object")
	}

	return trimmed, nil
}

func directoryJSONLines(files []string) ([]byte, error) {
	var result bytes.Buffer

	for _, filePath := range files {
		document, err := readJSONObject(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}
		if err := json.Compact(&result, document); err != nil {
			return nil, fmt.Errorf("failed to compact %s: %w", filePath, err)
		}
		result.WriteByte('\n')
	}

	return result.Bytes(), nil
}

func prepareMongoImport(uri, db string, collection Collection) ([]string, []byte, error) {
	args := []string{
		"--uri", uri,
		"--db", db,
		"--collection", collection.Name,
	}

	var input []byte
	var err error
	if len(collection.Files) > 0 {
		input, err = directoryJSONLines(collection.Files)
		if err != nil {
			return nil, nil, err
		}
	} else {
		args = append(args, "--file", collection.File)
	}

	args = append(args, "--drop")
	if collection.IsArray {
		args = append(args, "--jsonArray")
	}

	return args, input, nil
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

	args, input, err := prepareMongoImport(uri, db, coll)
	if err != nil {
		return err
	}

	cmd := exec.Command("mongoimport", args...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
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

	for _, key := range getCollectionKeys(collections) {
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
	sort.Strings(keys)
	return keys
}
