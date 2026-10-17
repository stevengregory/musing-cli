package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stevengregory/musing-cli/internal/config"
)

func serviceSchemaDeployFixture(t *testing.T, schemaDir string) (string, *config.ProjectConfig) {
	t.Helper()
	root, cfg := schemaDeployFixture(t)
	if err := os.Remove(filepath.Join(root, "data", "news-items.json")); err != nil {
		t.Fatal(err)
	}
	writeServiceSchemaFixture(t, root, "data/random_quotes.json", `[{"quote":"A thought"}]`)
	cfg.Database.Schemas = nil
	cfg.Database.SchemaDir = schemaDir
	cfg.Services = []config.ServiceConfig{{
		Name: "quotes-api", Type: "api", Alias: "quotes",
		Collection: "random_quotes", Schema: "shared-record-format.json",
	}}
	dir := schemaDir
	if dir == "" {
		dir = "schemas"
	}
	writeServiceSchemaFixture(t, root, filepath.Join(dir, "shared-record-format.json"), `{"bsonType":"object","required":["quote"]}`)
	return root, cfg
}

func writeServiceSchemaFixture(t *testing.T, root, name, contents string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPrepareDataDeployment_ServiceSchemaUsesExplicitCollection(t *testing.T) {
	for _, requested := range []string{"quotes", "random_quotes", "all"} {
		t.Run(requested, func(t *testing.T) {
			root, cfg := serviceSchemaDeployFixture(t, "")
			key, deployment, err := prepareDataDeployment(root, cfg, requested)
			wantKey := "random_quotes"
			if requested == "all" {
				wantKey = "all"
			}
			if err != nil || key != wantKey || deployment == nil {
				t.Fatalf("service schema preparation: key=%q plan=%v err=%v", key, deployment, err)
			}

			// A selected service schema must opt in to validator preflight even
			// though its filename shares neither the collection key nor alias.
			t.Setenv("PATH", t.TempDir())
			if _, _, err := prepareDataDeployment(root, cfg, requested); err == nil || !strings.Contains(err.Error(), "mongosh") {
				t.Fatalf("selected service schema did not require its tools: %v", err)
			}
		})
	}
}

func TestPrepareDataDeployment_ServiceSchemaDirectoryUsesProjectRoot(t *testing.T) {
	for _, schemaDir := range []string{"", "validation/records"} {
		name := schemaDir
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			root, cfg := serviceSchemaDeployFixture(t, schemaDir)
			dir := schemaDir
			if dir == "" {
				dir = "schemas"
			}
			chdirCmdTest(t, filepath.Join(root, "nested"))
			writeServiceSchemaFixture(t, root, filepath.Join("nested", dir, "shared-record-format.json"), `{"invalid":`)
			if _, deployment, err := prepareDataDeployment(root, cfg, "quotes"); err != nil || deployment == nil {
				t.Fatalf("schema directory used nested caller directory: plan=%v err=%v", deployment, err)
			}

			if err := os.Remove(filepath.Join(root, dir, "shared-record-format.json")); err != nil {
				t.Fatal(err)
			}
			writeServiceSchemaFixture(t, root, filepath.Join("nested", dir, "shared-record-format.json"), `{"bsonType":"object"}`)
			if _, _, err := prepareDataDeployment(root, cfg, "quotes"); err == nil {
				t.Fatal("accepted a schema found only relative to the caller's directory")
			}
		})
	}
}

func TestPrepareDataDeployment_ServiceSchemaRequiresFilenameOnly(t *testing.T) {
	for _, filename := range []string{"schemas/record.json", `schemas\record.json`, "../record.json", ".", "..", "/tmp/record.json", `C:\schemas\record.json`} {
		t.Run(filename, func(t *testing.T) {
			root, cfg := serviceSchemaDeployFixture(t, "")
			cfg.Services[0].Schema = filename
			// Invalid configuration must fail before source discovery or tools.
			cfg.Database.DataDir = "missing-data"
			t.Setenv("PATH", t.TempDir())
			if _, _, err := prepareDataDeployment(root, cfg, "quotes"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "schema") || strings.Contains(err.Error(), "failed to read data directory") {
				t.Fatalf("wanted schema filename rejection before source preflight, got %v", err)
			}
		})
	}
}

func TestPrepareDataDeployment_ServiceSchemaRequiresExplicitCollection(t *testing.T) {
	root, cfg := serviceSchemaDeployFixture(t, "")
	cfg.Services[0].Collection = ""
	cfg.Database.DataDir = "missing-data"
	if _, _, err := prepareDataDeployment(root, cfg, "quotes"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "collection") || strings.Contains(err.Error(), "failed to read data directory") {
		t.Fatalf("wanted explicit collection rejection before source preflight, got %v", err)
	}
}

func TestPrepareDataDeployment_ServiceSchemasStayOutsideDataDirectory(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "direct"
		if linked {
			name = "symlink"
		}
		t.Run(name, func(t *testing.T) {
			root, cfg := serviceSchemaDeployFixture(t, "")
			writeServiceSchemaFixture(t, root, "data/shared-record-format.json", `{"bsonType":"object"}`)
			cfg.Database.SchemaDir = "data"
			if linked {
				if err := os.Symlink(filepath.Join(root, "data"), filepath.Join(root, "linked-schemas")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				cfg.Database.SchemaDir = "linked-schemas"
			}
			if _, deployment, err := prepareDataDeployment(root, cfg, "quotes"); err == nil || deployment != nil || !strings.Contains(err.Error(), "data directory") {
				t.Fatalf("schema inside imported data directory was accepted: plan=%v err=%v", deployment, err)
			}
		})
	}
}

func TestPrepareDataDeployment_SchemaDirectoryAloneDoesNotRequireTools(t *testing.T) {
	for _, schemaDir := range []string{"", "schemas-that-do-not-exist"} {
		t.Run(schemaDir, func(t *testing.T) {
			root, cfg := serviceSchemaDeployFixture(t, "")
			cfg.Database.SchemaDir = schemaDir
			cfg.Services[0].Schema = ""
			t.Setenv("PATH", t.TempDir())
			if key, deployment, err := prepareDataDeployment(root, cfg, "quotes"); err != nil || key != "random_quotes" || deployment == nil {
				t.Fatalf("schema directory opted an unconfigured collection in: key=%q plan=%v err=%v", key, deployment, err)
			}
		})
	}
}

func TestPrepareDataDeployment_ServiceAndLegacySchemasRemainCompatible(t *testing.T) {
	root, cfg := serviceSchemaDeployFixture(t, "validation/records")
	writeServiceSchemaFixture(t, root, "data/legacy-items.json", `{"label":"legacy"}`)
	legacyPath := writeServiceSchemaFixture(t, root, "old-validation/legacy.json", `{"invalid":`)
	cfg.Database.Schemas = config.CollectionSchemas{"legacy-items": "old-validation/legacy.json"}
	chdirCmdTest(t, filepath.Join(root, "nested"))
	if _, deployment, err := prepareDataDeployment(root, cfg, "quotes"); err != nil || deployment == nil {
		t.Fatalf("unselected legacy schema blocked the service collection: plan=%v err=%v", deployment, err)
	}
	if _, _, err := prepareDataDeployment(root, cfg, "all"); err == nil || !strings.Contains(err.Error(), "legacy-items") {
		t.Fatalf("deploy all ignored the legacy schema mapping: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte(`{"bsonType":"object"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, deployment, err := prepareDataDeployment(root, cfg, "all"); err != nil || deployment == nil {
		t.Fatalf("legacy path was not resolved from the project root independently of schemaDir: plan=%v err=%v", deployment, err)
	}
}

func TestPrepareDataDeployment_ConflictingSchemasFailBeforeSourcePreflight(t *testing.T) {
	for _, declaration := range []string{"legacy mapping", "another service"} {
		t.Run(declaration, func(t *testing.T) {
			root, cfg := serviceSchemaDeployFixture(t, "")
			if declaration == "legacy mapping" {
				cfg.Database.Schemas = config.CollectionSchemas{"random_quotes": "schemas/other.json"}
			} else {
				cfg.Services = append(cfg.Services, config.ServiceConfig{
					Name: "other-api", Type: "api", Collection: "random_quotes", Schema: "other.json",
				})
			}
			cfg.Database.DataDir = "missing-data"
			t.Setenv("PATH", t.TempDir())
			if _, _, err := prepareDataDeployment(root, cfg, "all"); err == nil || !strings.Contains(err.Error(), "random_quotes") || strings.Contains(err.Error(), "failed to read data directory") {
				t.Fatalf("wanted conflicting collection schemas to fail before source preflight, got %v", err)
			}
		})
	}
}

func TestPrepareDataDeployment_AllChecksLaterServiceSchemaBeforeDeployment(t *testing.T) {
	for _, content := range []string{"missing", `{"bsonType":`} {
		t.Run(content, func(t *testing.T) {
			root, cfg := serviceSchemaDeployFixture(t, "")
			writeServiceSchemaFixture(t, root, "data/a-legacy.json", `{"label":"legacy"}`)
			writeServiceSchemaFixture(t, root, "data/z-records.json", `{"title":"later"}`)
			cfg.Services = append(cfg.Services, config.ServiceConfig{
				Name: "later-api", Type: "api", Collection: "z-records", Schema: "later-format.json",
			})
			if content != "missing" {
				writeServiceSchemaFixture(t, root, "schemas/later-format.json", content)
			}
			if _, deployment, err := prepareDataDeployment(root, cfg, "all"); err == nil || deployment != nil || !strings.Contains(err.Error(), "z-records") {
				t.Fatalf("deploy all prepared despite invalid later schema: plan=%v err=%v", deployment, err)
			}
		})
	}
}
