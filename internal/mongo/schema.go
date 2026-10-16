package mongo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// DeployOptions configures optional validators by discovered collection source key.
type DeployOptions struct {
	ProjectRoot string
	Schemas     map[string]string
}

// Deployment is a locally checked deployment, ready to run against a database.
type Deployment struct {
	collections []preparedCollection
}

type preparedCollection struct {
	key        string
	collection Collection
	schema     []byte
	input      []byte
}

var (
	lookPath   = exec.LookPath
	runCommand = runDeploymentCommand
)

func runDeploymentCommand(name string, args []string, input []byte) error {
	command := exec.Command(name, args...)
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func deploymentOptions(options []DeployOptions) (DeployOptions, error) {
	if len(options) > 1 {
		return DeployOptions{}, fmt.Errorf("expected at most one deployment options value")
	}
	if len(options) == 1 {
		return options[0], nil
	}
	return DeployOptions{}, nil
}

// PrepareCollection checks a collection and snapshots any schema-backed input.
// It performs no database operations or external commands.
func PrepareCollection(collectionKey, dataDir string, options DeployOptions) (*Deployment, error) {
	return prepareDeployment(collectionKey, false, dataDir, options)
}

// PrepareAll checks every selected source and schema before any collection is changed.
func PrepareAll(dataDir string, options DeployOptions) (*Deployment, error) {
	return prepareDeployment("", true, dataDir, options)
}

func prepareDeployment(collectionKey string, all bool, dataDir string, options DeployOptions) (*Deployment, error) {
	collections, err := DiscoverCollections(dataDir)
	if err != nil {
		return nil, err
	}
	schemas, err := schemaPaths(collections, dataDir, options)
	if err != nil {
		return nil, err
	}
	keys := getCollectionKeys(collections)
	if !all {
		if _, exists := collections[collectionKey]; !exists {
			return nil, fmt.Errorf("collection not found: %s (available: %v)", collectionKey, keys)
		}
		keys = []string{collectionKey}
	}

	deployment := &Deployment{}
	hasSchema := false
	for _, key := range keys {
		prepared := preparedCollection{key: key, collection: collections[key]}
		if path, configured := schemas[key]; configured {
			hasSchema = true
			prepared.schema, err = readSchema(path)
			if err != nil {
				return nil, fmt.Errorf("collection %q: read schema: %w", key, err)
			}
			prepared.input, prepared.collection.IsArray, err = snapshotCollection(prepared.collection)
		} else if len(prepared.collection.Files) > 0 {
			prepared.input, err = directoryJSONLines(prepared.collection.Files)
		}
		if err != nil {
			return nil, fmt.Errorf("collection %q: prepare source: %w", key, err)
		}
		deployment.collections = append(deployment.collections, prepared)
	}
	if hasSchema {
		for _, name := range []string{"mongosh", "mongoimport"} {
			if _, err := lookPath(name); err != nil {
				return nil, fmt.Errorf("schema deployment requires %s in PATH: %w", name, err)
			}
		}
	}
	return deployment, nil
}

func schemaPaths(collections map[string]Collection, dataDir string, options DeployOptions) (map[string]string, error) {
	paths := make(map[string]string, len(options.Schemas))
	if len(options.Schemas) == 0 {
		return paths, nil
	}
	if !filepath.IsAbs(options.ProjectRoot) {
		return nil, fmt.Errorf("an absolute project root is required for collection schemas")
	}
	dataPath, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	resolvedDataPath, err := filepath.EvalSymlinks(dataPath)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	sourcePaths := make(map[string]bool)
	for _, collection := range collections {
		files := collection.Files
		if collection.File != "" {
			files = []string{collection.File}
		}
		for _, file := range files {
			if resolved, err := filepath.EvalSymlinks(file); err == nil {
				absolute, err := filepath.Abs(resolved)
				if err == nil {
					sourcePaths[absolute] = true
				}
			}
		}
	}
	keys := make([]string, 0, len(options.Schemas))
	for key := range options.Schemas {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, found := collections[key]; !found {
			return nil, fmt.Errorf("schema collection key %q not found (available: %v)", key, getCollectionKeys(collections))
		}
		path := options.Schemas[key]
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("schema path for collection %q must not be empty", key)
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(options.ProjectRoot, path)
		}
		path = filepath.Clean(path)
		if pathInside(dataPath, path) {
			return nil, fmt.Errorf("schema for collection %q must be outside the data directory", key)
		}
		// Missing unselected schema files do not block another collection. Existing
		// symlinks still need checking so a schema cannot also become source data.
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			if pathInside(resolvedDataPath, resolved) || sourcePaths[resolved] {
				return nil, fmt.Errorf("schema for collection %q is reachable inside the data directory", key)
			}
		}
		paths[key] = path
	}
	return paths, nil
}

func pathInside(dir, path string) bool {
	relative, err := filepath.Rel(dir, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func readSchema(path string) ([]byte, error) {
	data, err := readJSONObject(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s: schema JSON contains invalid UTF-8", path)
	}
	var fields map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&fields); err != nil {
		return nil, fmt.Errorf("%s: invalid schema JSON: %w", path, err)
	}
	for _, key := range []string{"$jsonSchema", "validator", "validationAction", "validationLevel", "indexes"} {
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("%s: unsupported schema wrapper %q; provide the plain JSON Schema object", path, key)
		}
	}
	if err := validateSchemaNumbers(fields); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return data, nil
}

func validateSchemaNumbers(value any) error {
	switch value := value.(type) {
	case map[string]any:
		for _, nested := range value {
			if err := validateSchemaNumbers(nested); err != nil {
				return err
			}
		}
	case []any:
		for _, nested := range value {
			if err := validateSchemaNumbers(nested); err != nil {
				return err
			}
		}
	case json.Number:
		number, err := value.Float64()
		if err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
			return fmt.Errorf("schema number cannot be represented as a finite JavaScript number")
		}
		// JSON.parse uses JavaScript numbers; larger integer values can silently
		// change a validator's bounds or enum members after the collection is dropped.
		if math.Trunc(number) == number && math.Abs(number) > 9007199254740991 {
			return fmt.Errorf("schema integer exceeds JavaScript's safe integer range (+/-9007199254740991)")
		}
		if number == 0 {
			for _, digit := range value.String() {
				if digit == 'e' || digit == 'E' {
					break
				}
				if digit >= '1' && digit <= '9' {
					return fmt.Errorf("nonzero schema number underflows to zero in JavaScript")
				}
			}
		}
	}
	return nil
}

func snapshotCollection(collection Collection) ([]byte, bool, error) {
	if len(collection.Files) > 0 {
		data, err := directoryJSONLines(collection.Files)
		if err == nil && !utf8.Valid(data) {
			err = fmt.Errorf("directory source JSON contains invalid UTF-8")
		}
		return data, false, err
	}
	data, err := os.ReadFile(collection.File)
	if err != nil {
		return nil, false, err
	}
	if !utf8.Valid(data) {
		return nil, false, fmt.Errorf("%s: source JSON contains invalid UTF-8", collection.File)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || !json.Valid(data) {
		return nil, false, fmt.Errorf("%s: invalid or empty source JSON", collection.File)
	}
	if data[0] == '{' {
		return data, false, nil
	}
	if data[0] != '[' {
		return nil, false, fmt.Errorf("%s: expected a JSON object or array of objects", collection.File)
	}
	var documents []json.RawMessage
	if err := json.Unmarshal(data, &documents); err != nil {
		return nil, false, err
	}
	for index, document := range documents {
		if value := bytes.TrimSpace(document); len(value) == 0 || value[0] != '{' {
			return nil, false, fmt.Errorf("%s: array item %d must be a JSON object", collection.File, index+1)
		}
	}
	return data, true, nil
}

// Deploy runs the prepared collections in source-key order, stopping at the first
// failure. Validator setup drops and recreates a collection before importing it.
func (deployment *Deployment) Deploy(uri, db string) error {
	if deployment == nil {
		return fmt.Errorf("deployment is not prepared")
	}
	// The server parses every selected schema before any collection, including
	// legacy imports earlier in source-key order, can be dropped or replaced.
	for _, prepared := range deployment.collections {
		if prepared.schema == nil {
			continue
		}
		args := []string{"--quiet", "--norc", uri, "--eval", validatorPreflightScript(db, prepared.collection.Name, prepared.schema)}
		if err := runCommand("mongosh", args, nil); err != nil {
			return fmt.Errorf("collection %q: server schema preflight failed: %w", prepared.key, err)
		}
	}
	for _, prepared := range deployment.collections {
		collection := prepared.collection
		if prepared.schema != nil {
			args := []string{"--quiet", "--norc", uri, "--eval", validatorScript(db, collection.Name, prepared.schema)}
			if err := runCommand("mongosh", args, nil); err != nil {
				return fmt.Errorf("collection %q: drop/create validator failed: %w", prepared.key, err)
			}
		}
		args := legacyMongoImportArgs(uri, db, collection)
		if prepared.schema != nil {
			args = []string{"--uri", uri, "--db", db, "--collection", collection.Name, "--stopOnError", "--maintainInsertionOrder"}
			if collection.IsArray {
				args = append(args, "--jsonArray")
			}
		}
		if err := runCommand("mongoimport", args, prepared.input); err != nil {
			return fmt.Errorf("collection %q: import failed: %w", prepared.key, err)
		}
	}
	return nil
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func validatorPreflightScript(database, collection string, schema []byte) string {
	return fmt.Sprintf(`(() => {
  const name = %s;
  try {
    const target = db.getSiblingDB(%s);
    const schema = JSON.parse(%s);
    const checked = target.runCommand({
      explain: {
        find: name,
        filter: {$jsonSchema: schema},
        collation: {locale: "simple"}
      },
      verbosity: "queryPlanner"
    });
    if (!checked || checked.ok !== 1) throw checked;
  } catch (error) {
    const code = error && typeof error.code === "number" ? error.code : "unknown";
    print("Schema preflight failed for collection " + name + " (MongoDB error code " + code + ").");
    exit(1);
  }
})();`, jsonString(collection), jsonString(database), jsonString(string(schema)))
}

func validatorScript(database, collection string, schema []byte) string {
	return fmt.Sprintf(`(() => {
  let step = "drop";
  const name = %s;
  try {
    const target = db.getSiblingDB(%s);
    const schema = JSON.parse(%s);
    let dropped;
    try {
      dropped = target.runCommand({drop: name});
    } catch (error) {
      if (!error || error.code !== 26) throw error;
      dropped = {ok: 0, code: 26};
    }
    if (!dropped || (dropped.ok !== 1 && dropped.code !== 26)) throw dropped;
    step = "create";
    const created = target.runCommand({
      create: name,
      validator: {$jsonSchema: schema},
      validationLevel: "strict",
      validationAction: "error"
    });
    if (!created || created.ok !== 1) throw created;
  } catch (error) {
    const code = error && typeof error.code === "number" ? error.code : "unknown";
    print("Failed to " + step + " collection " + name + " (MongoDB error code " + code + ").");
    exit(1);
  }
})();`, jsonString(collection), jsonString(database), jsonString(string(schema)))
}
