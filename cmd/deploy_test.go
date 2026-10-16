package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stevengregory/musing-cli/internal/config"
)

func schemaDeployFixture(t *testing.T) (string, *config.ProjectConfig) {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"data", "schemas", "nested/schemas", "tools"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []struct{ name, contents string }{
		{"data/news-items.json", `{"title":"A story"}`},
		{"schemas/news-items.json", `{"bsonType":"object","required":["title"]}`},
	} {
		if err := os.WriteFile(filepath.Join(root, file.name), []byte(file.contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Preflight only looks these up; an accidental execution would fail rather
	// than reach a real database tool from the user's PATH.
	for _, tool := range []string{"mongosh", "mongoimport"} {
		if err := os.WriteFile(filepath.Join(root, "tools", tool), nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", filepath.Join(root, "tools"))
	return root, &config.ProjectConfig{
		Database: config.DatabaseConfig{
			Type: "MongoDB", DataDir: "data",
			Schemas: config.CollectionSchemas{"news-items": "schemas/news-items.json"},
		},
		Services: []config.ServiceConfig{{Type: "api", Alias: "news", Collection: "news-items"}},
	}
}

func TestPrepareDataDeployment_UsesSourceKeyAfterAlias(t *testing.T) {
	root, cfg := schemaDeployFixture(t)
	chdirCmdTest(t, filepath.Join(root, "nested"))
	key, deployment, err := prepareDataDeployment(root, cfg, "news")
	if err != nil {
		t.Fatal(err)
	}
	if key != "news-items" || deployment == nil {
		t.Fatalf("alias did not resolve to the source key: key=%q plan=%v", key, deployment)
	}
}

func TestPrepareDataDeployment_DefaultAliasCollection(t *testing.T) {
	root, cfg := schemaDeployFixture(t)
	cfg.Services = []config.ServiceConfig{{Type: "api", Alias: "news-items"}}
	key, deployment, err := prepareDataDeployment(root, cfg, "news-items")
	if err != nil || key != "news-items" || deployment == nil {
		t.Fatalf("default alias resolution failed: key=%q plan=%v err=%v", key, deployment, err)
	}
}

func TestPrepareDataDeployment_RejectsNormalizedSchemaKeyAndAliasKey(t *testing.T) {
	for _, wrongKey := range []string{"news_items", "news"} {
		t.Run(wrongKey, func(t *testing.T) {
			root, cfg := schemaDeployFixture(t)
			cfg.Database.Schemas = config.CollectionSchemas{wrongKey: "schemas/news-items.json"}
			if _, _, err := prepareDataDeployment(root, cfg, "news"); err == nil || !strings.Contains(err.Error(), wrongKey) {
				t.Fatalf("expected invalid source key %q error, got %v", wrongKey, err)
			}
		})
	}
}

func TestPrepareDataDeployment_SchemaPathDoesNotUseCallerDirectory(t *testing.T) {
	root, cfg := schemaDeployFixture(t)
	if err := os.Rename(filepath.Join(root, "schemas/news-items.json"), filepath.Join(root, "nested/schemas/news-items.json")); err != nil {
		t.Fatal(err)
	}
	chdirCmdTest(t, filepath.Join(root, "nested"))
	if _, _, err := prepareDataDeployment(root, cfg, "news"); err == nil {
		t.Fatal("accepted a schema found only in the caller's directory")
	}
}

func TestPrepareDataDeployment_LegacyImportDoesNotNeedSchemaTools(t *testing.T) {
	root, cfg := schemaDeployFixture(t)
	cfg.Database.Schemas = nil
	t.Setenv("PATH", t.TempDir())
	key, deployment, err := prepareDataDeployment(root, cfg, "news")
	if err != nil || key != "news-items" || deployment == nil {
		t.Fatalf("legacy preflight unexpectedly requires schema tools: key=%q plan=%v err=%v", key, deployment, err)
	}
}
