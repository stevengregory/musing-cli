package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_CollectionSchemas(t *testing.T) {
	root := t.TempDir()
	writeProjectFiles(t, root, `
database:
  type: MongoDB
  schemas:
    news-items: schemas/news-items.schema.json
services:
  - name: news-api
    type: api
    alias: news
    collection: news-items
`)
	if err := loadConfig(filepath.Join(root, ".musing.yaml")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { currentConfig = nil })
	if got := GetConfig().Database.Schemas["news-items"]; got != "schemas/news-items.schema.json" {
		t.Fatalf("schema source key/path changed: %q", got)
	}
}

func TestLoadConfig_RejectsUnsupportedSchemaConfig(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{"list", "database:\n  type: MongoDB\n  schemas: [news]\n"},
		{"scalar", "database:\n  type: MongoDB\n  schemas: schemas/news.json\n"},
		{"options instead of path", "database:\n  type: MongoDB\n  schemas:\n    news: {path: schemas/news.json, validationAction: warn}\n"},
		{"numeric path", "database:\n  type: MongoDB\n  schemas:\n    news: 42\n"},
		{"null path", "database:\n  type: MongoDB\n  schemas:\n    news: null\n"},
		{"empty path", "database:\n  type: MongoDB\n  schemas:\n    news: ''\n"},
		{"blank source key", "database:\n  type: MongoDB\n  schemas:\n    ' ': schemas/news.json\n"},
		{"numeric source key", "database:\n  type: MongoDB\n  schemas:\n    42: schemas/news.json\n"},
		{"duplicate source key", "database:\n  type: MongoDB\n  schemas:\n    news: schemas/one.json\n    news: schemas/two.json\n"},
		{"other database type", "database:\n  type: postgres\n  schemas:\n    news: schemas/news.json\n"},
		{"missing database type", "database:\n  schemas:\n    news: schemas/news.json\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFiles(t, root, tt.yaml)
			if err := loadConfig(filepath.Join(root, ".musing.yaml")); err == nil || !strings.Contains(err.Error(), "schemas") {
				t.Fatalf("expected schema configuration error, got %v", err)
			}
		})
	}
}

func TestLoadConfig_OmittedSchemasPreservesExistingConfig(t *testing.T) {
	root := t.TempDir()
	writeProjectFiles(t, root, "database:\n  name: mydb\n  dataDir: data\n")
	if err := loadConfig(filepath.Join(root, ".musing.yaml")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { currentConfig = nil })
	if len(GetConfig().Database.Schemas) != 0 {
		t.Fatalf("unexpected schemas: %v", GetConfig().Database.Schemas)
	}
}
