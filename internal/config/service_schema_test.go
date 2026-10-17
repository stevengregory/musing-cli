package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfig_ServiceSchemaPreservesServiceFields(t *testing.T) {
	root := t.TempDir()
	writeProjectFiles(t, root, `
database:
  type: MongoDB
  name: project
  dataDir: data
services:
  - name: random-quotes-api
    port: 8083
    type: api
    alias: quotes
    collection: random_quotes
    schema: random_quotes.schema.json
    futureOption: preserved-unknown-field-behavior
  - name: web
    port: 3000
    type: frontend
`)
	if err := loadConfig(filepath.Join(root, ".musing.yaml")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { currentConfig = nil })
	cfg := GetConfig()
	want := []ServiceConfig{
		{Name: "random-quotes-api", Port: 8083, Type: "api", Alias: "quotes", Collection: "random_quotes", Schema: "random_quotes.schema.json"},
		{Name: "web", Port: 3000, Type: "frontend"},
	}
	if !reflect.DeepEqual(cfg.Services, want) {
		t.Fatalf("service fields changed: got %+v, want %+v", cfg.Services, want)
	}
	if cfg.Database.SchemaDir != "schemas" {
		t.Fatalf("default schema directory: got %q", cfg.Database.SchemaDir)
	}
	paths, err := cfg.ResolveCollectionSchemas(root)
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := map[string]string{"random_quotes": filepath.Join(root, "schemas", "random_quotes.schema.json")}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("schema resolution inferred another filename or key: got %v, want %v", paths, wantPaths)
	}
}

func TestLoadConfig_RejectsInvalidServiceSchema(t *testing.T) {
	for _, value := range []string{
		"true", "false", "42", "1.5", "null", "~", "[]", "{}", "''", "' '",
		"'nested/file.json'", "'nested\\file.json'", "'/tmp/file.json'", "'C:\\schemas\\file.json'", "'C:file.json'", "'c:file.json'", "'\\\\server\\share\\file.json'", "'news.json:stream'", `"news\0.json"`,
		"'./file.json'", "'../file.json'", "'file'", "'file.'", "'.json'", "'.'", "'..'",
	} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, "database:\n  type: MongoDB\nservices:\n  - name: news-api\n    collection: news-items\n    schema: "+value+"\n")
			if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil {
				t.Fatalf("invalid service schema %s was accepted", value)
			}
		})
	}
}

func TestLoadConfig_RejectsInvalidSchemaDirectory(t *testing.T) {
	for _, value := range []string{"true", "42", "null", "~", "[]", "{}", "''", "' '"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, "database:\n  schemaDir: "+value+"\n")
			if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil || !strings.Contains(err.Error(), "schemaDir") {
				t.Fatalf("invalid schema directory must fail clearly: %v", err)
			}
		})
	}
}

func TestLoadConfig_ServiceSchemaKeepsServicesAsList(t *testing.T) {
	root := t.TempDir()
	writeProjectFiles(t, root, `
database:
  type: MongoDB
services:
  news-api:
    collection: news-items
    schema: news.schema.json
`)
	if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil {
		t.Fatal("schema support must not convert services from a list to a map")
	}
}

func TestLoadConfig_ValidatesMergedAndAliasedSchemaFields(t *testing.T) {
	for _, value := range []string{"1.2", "true", "null", "[]", "{}"} {
		t.Run("service merge "+value, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, "defaults: &defaults\n  schema: "+value+"\n  collection: news-items\ndatabase:\n  type: MongoDB\nservices:\n  - <<: *defaults\n    name: news-api\n")
			if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil || !strings.Contains(err.Error(), "schema") {
				t.Fatalf("merged schema must remain a nonempty filename string: %v", err)
			}
		})
		t.Run("directory merge "+value, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, "defaults: &defaults\n  schemaDir: "+value+"\ndatabase:\n  <<: *defaults\n")
			if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil || !strings.Contains(err.Error(), "schemaDir") {
				t.Fatalf("merged schemaDir must remain a nonempty path string: %v", err)
			}
		})
	}
}

func TestLoadConfig_SchemaMergePrecedenceAndStringAliases(t *testing.T) {
	for _, test := range []struct {
		name      string
		merge     string
		direct    string
		wantError bool
	}{
		{name: "valid alias and merged fields", merge: "*good"},
		{name: "direct valid schema overrides invalid merge", merge: "*bad", direct: "    schema: *filename\n"},
		{name: "first merge wins", merge: "[*good, *bad]"},
		{name: "invalid first merge wins", merge: "[*bad, *good]", wantError: true},
		{name: "direct invalid schema overrides valid merge", merge: "*good", direct: "    schema: *invalid\n", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, `
filename: &filename news.schema.json
invalid: &invalid 1.2
directory: &directory config/validators
good: &good
  schema: *filename
  collection: news-items
  port: 8080
  alias: news
  type: api
bad: &bad
  schema: *invalid
  collection: news-items
dbDefaults: &dbDefaults
  type: MongoDB
  schemaDir: *directory
database:
  <<: *dbDefaults
services:
  - <<: `+test.merge+"\n    name: news-api\n"+test.direct)
			err := loadConfig(filepath.Join(root, ".musing.yaml"))
			if test.wantError {
				if err == nil || !strings.Contains(err.Error(), "schema") {
					t.Fatalf("effective invalid schema type was accepted: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { currentConfig = nil })
			cfg := GetConfig()
			paths, err := cfg.ResolveCollectionSchemas(root)
			if err != nil || paths["news-items"] != filepath.Join(root, "config", "validators", "news.schema.json") {
				t.Fatalf("valid merged fields or aliases changed: %v, %v", paths, err)
			}
			if test.merge == "*good" && test.direct == "" && cfg.Services[0].Port != 8080 {
				t.Fatalf("non-schema service merge behavior changed: %+v", cfg.Services[0])
			}
		})
	}
}

func TestLoadConfig_ServiceSchemaRequiresExplicitCollection(t *testing.T) {
	for _, collection := range []string{"", "    collection: ''\n", "    collection: ' '\n"} {
		t.Run(collection, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, "database:\n  type: MongoDB\nservices:\n  - name: news-api\n    alias: news\n"+collection+"    schema: news.schema.json\n")
			if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil || !strings.Contains(err.Error(), "collection") {
				t.Fatalf("schema must require explicit collection, got %v", err)
			}
		})
	}
}

func TestResolveCollectionSchemas_DirectoryAndLegacyPaths(t *testing.T) {
	root := t.TempDir()
	absDirectory := filepath.Join(t.TempDir(), "absolute-schemas")
	for _, schemaDir := range []string{"", "config/validators", absDirectory} {
		t.Run(schemaDir, func(t *testing.T) {
			cfg := ProjectConfig{
				Database: DatabaseConfig{Type: "mOnGoDb", SchemaDir: schemaDir, Schemas: CollectionSchemas{"legacy-items": "old/legacy.schema.json"}},
				Services: []ServiceConfig{{Name: "news", Collection: "news-items", Schema: "news.schema.json"}},
			}
			paths, err := cfg.ResolveCollectionSchemas(root)
			if err != nil {
				t.Fatal(err)
			}
			directory := schemaDir
			if directory == "" {
				directory = "schemas"
			}
			if !filepath.IsAbs(directory) {
				directory = filepath.Join(root, directory)
			}
			want := map[string]string{
				"legacy-items": filepath.Join(root, "old", "legacy.schema.json"),
				"news-items":   filepath.Join(directory, "news.schema.json"),
			}
			if !reflect.DeepEqual(paths, want) {
				t.Fatalf("incorrect schema path origins: got %v, want %v", paths, want)
			}
		})
	}
}

func TestResolveCollectionSchemas_IdenticalAndConflictingDeclarations(t *testing.T) {
	root := t.TempDir()
	for _, test := range []struct {
		name      string
		legacy    CollectionSchemas
		services  []ServiceConfig
		wantError bool
	}{
		{name: "identical legacy and service", legacy: CollectionSchemas{"news-items": "schemas/nested/../news.schema.json"}, services: []ServiceConfig{{Collection: "news-items", Schema: "news.schema.json"}}},
		{name: "identical absolute legacy and service", legacy: CollectionSchemas{"news-items": filepath.Join(root, "schemas", "news.schema.json")}, services: []ServiceConfig{{Collection: "news-items", Schema: "news.schema.json"}}},
		{name: "identical service declarations", services: []ServiceConfig{{Collection: "news-items", Schema: "news.schema.json"}, {Collection: "news-items", Schema: "news.schema.json"}}},
		{name: "legacy service conflict", legacy: CollectionSchemas{"news-items": "other/news.schema.json"}, services: []ServiceConfig{{Collection: "news-items", Schema: "news.schema.json"}}, wantError: true},
		{name: "service service conflict", services: []ServiceConfig{{Collection: "news-items", Schema: "one.schema.json"}, {Collection: "news-items", Schema: "two.schema.json"}}, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := ProjectConfig{Database: DatabaseConfig{Type: "MongoDB", Schemas: test.legacy}, Services: test.services}
			paths, err := cfg.ResolveCollectionSchemas(root)
			if test.wantError {
				if err == nil || !strings.Contains(err.Error(), "conflicting") {
					t.Fatalf("expected conflicting schema error, got %v", err)
				}
				return
			}
			if err != nil || len(paths) != 1 || paths["news-items"] != filepath.Join(root, "schemas", "news.schema.json") {
				t.Fatalf("identical declarations should merge: paths=%v err=%v", paths, err)
			}
		})
	}
}

func TestLoadConfig_RejectsConflictingSchemaDeclarations(t *testing.T) {
	root := t.TempDir()
	writeProjectFiles(t, root, `
database:
  type: MongoDB
  schemas:
    news-items: old/news.schema.json
services:
  - name: news-api
    collection: news-items
    schema: news.schema.json
`)
	if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("conflicts must fail during configuration loading: %v", err)
	}
}

func TestResolveCollectionSchemas_ValidatesProgrammaticConfig(t *testing.T) {
	root := t.TempDir()
	for _, test := range []struct {
		name string
		edit func(*ProjectConfig)
	}{
		{name: "blank schema", edit: func(cfg *ProjectConfig) { cfg.Services[0].Schema = " " }},
		{name: "nested filename", edit: func(cfg *ProjectConfig) { cfg.Services[0].Schema = "nested/news.json" }},
		{name: "windows filename", edit: func(cfg *ProjectConfig) { cfg.Services[0].Schema = `nested\news.json` }},
		{name: "drive prefix", edit: func(cfg *ProjectConfig) { cfg.Services[0].Schema = "C:news.json" }},
		{name: "alternate data stream", edit: func(cfg *ProjectConfig) { cfg.Services[0].Schema = "news.json:stream" }},
		{name: "NUL filename", edit: func(cfg *ProjectConfig) { cfg.Services[0].Schema = "news\x00.schema.json" }},
		{name: "missing extension", edit: func(cfg *ProjectConfig) { cfg.Services[0].Schema = "news" }},
		{name: "missing explicit collection", edit: func(cfg *ProjectConfig) { cfg.Services[0].Collection = "" }},
		{name: "blank explicit collection", edit: func(cfg *ProjectConfig) { cfg.Services[0].Collection = " " }},
		{name: "unsupported database", edit: func(cfg *ProjectConfig) { cfg.Database.Type = "postgres" }},
		{name: "missing database type", edit: func(cfg *ProjectConfig) { cfg.Database.Type = "" }},
		{name: "blank schema directory", edit: func(cfg *ProjectConfig) { cfg.Database.SchemaDir = " " }},
		{name: "blank legacy key", edit: func(cfg *ProjectConfig) { cfg.Database.Schemas = CollectionSchemas{" ": "old.json"} }},
		{name: "blank legacy path", edit: func(cfg *ProjectConfig) { cfg.Database.Schemas = CollectionSchemas{"other": " "} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := ProjectConfig{Database: DatabaseConfig{Type: "MongoDB"}, Services: []ServiceConfig{{Name: "news-api", Alias: "news", Collection: "news-items", Schema: "news.schema.json"}}}
			test.edit(&cfg)
			if _, err := cfg.ResolveCollectionSchemas(root); err == nil {
				t.Fatal("invalid programmatic schema configuration was accepted")
			}
		})
	}
	cfg := ProjectConfig{Database: DatabaseConfig{Type: "MongoDB", Schemas: CollectionSchemas{"news": "old.json"}}}
	if _, err := cfg.ResolveCollectionSchemas("relative"); err == nil {
		t.Fatal("enabled schema paths require an absolute project root")
	}
}

func TestLoadConfig_SchemaDirectoryAloneEnablesNothing(t *testing.T) {
	for _, database := range []string{"", "database: {}\n", "database:\n  type: postgres\n  schemaDir: validators\n"} {
		t.Run(fmt.Sprintf("%q", database), func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, database+"services:\n  - name: news-api\n    alias: news\n    collection: news-items\n")
			if err := loadConfig(filepath.Join(root, ".musing.yaml")); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { currentConfig = nil })
			cfg := GetConfig()
			if database != "database:\n  type: postgres\n  schemaDir: validators\n" && cfg.Database.SchemaDir != "schemas" {
				t.Fatalf("omitted schemaDir must default: %q", cfg.Database.SchemaDir)
			}
			paths, err := cfg.ResolveCollectionSchemas(root)
			if err != nil || len(paths) != 0 {
				t.Fatalf("schema directory or collection alone enabled schemas: %v, %v", paths, err)
			}
		})
	}
}
