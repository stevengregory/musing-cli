package config

import (
	"os"
	"path/filepath"
	"testing"
)

func realPath(t *testing.T, p string) string {
	t.Helper()
	rp, err := filepath.EvalSymlinks(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(rp)
}

func writeProjectFiles(t *testing.T, root string, yaml string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("failed writing compose.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".musing.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("failed writing .musing.yaml: %v", err)
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
		currentConfig = nil
	})

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
}

func TestFindProjectRoot_FindsParentAndLoadsConfig(t *testing.T) {
	root := t.TempDir()
	writeProjectFiles(t, root, `
database:
  type: MongoDB
  name: mydb
  devPort: 27018
  prodPort: 27019
  dataDir: data
services:
  - name: news-api
    port: 8080
    type: api
`)

	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	chdirForTest(t, nested)

	foundRoot, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot returned error: %v", err)
	}
	if realPath(t, foundRoot) != realPath(t, root) {
		t.Fatalf("root mismatch: got %q want %q", foundRoot, root)
	}

	cfg := GetConfig()
	if cfg == nil {
		t.Fatal("expected config to be loaded")
	}
	if cfg.Database.Name != "mydb" {
		t.Fatalf("expected database name mydb, got %q", cfg.Database.Name)
	}
}

func TestFindProjectRoot_NoComposeReturnsError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".musing.yaml"), []byte("database: {}\n"), 0o644); err != nil {
		t.Fatalf("failed writing .musing.yaml: %v", err)
	}
	chdirForTest(t, root)

	_, err := FindProjectRoot()
	if err == nil {
		t.Fatal("expected error when compose.yaml is missing")
	}
}

func TestGetAPIRepos_ReturnsOnlyAPIServiceRepos(t *testing.T) {
	root := t.TempDir()
	writeProjectFiles(t, root, `
database:
  type: MongoDB
  name: mydb
  devPort: 27018
  prodPort: 27019
  dataDir: data
services:
  - name: api-one
    port: 8080
    type: api
  - name: web
    port: 3000
    type: frontend
  - name: api-two
    port: 8081
    type: api
`)
	chdirForTest(t, root)

	_, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot returned error: %v", err)
	}

	parent := filepath.Dir(root)
	want1 := filepath.Join(parent, "api-one")
	want2 := filepath.Join(parent, "api-two")
	if err := os.MkdirAll(want1, 0o755); err != nil {
		t.Fatalf("failed to create expected api-one path: %v", err)
	}
	if err := os.MkdirAll(want2, 0o755); err != nil {
		t.Fatalf("failed to create expected api-two path: %v", err)
	}

	repos := GetAPIRepos()
	if len(repos) != 2 {
		t.Fatalf("expected 2 api repos, got %d", len(repos))
	}

	if realPath(t, repos[0]) != realPath(t, want1) || realPath(t, repos[1]) != realPath(t, want2) {
		t.Fatalf("unexpected repos: got %v, want [%s %s]", repos, want1, want2)
	}
}
