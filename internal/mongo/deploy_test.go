package mongo

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDiscoverCollections(t *testing.T) {
	dataDir := t.TempDir()

	files := map[string]string{
		"news.json":          `[{ "title": "a" }]`,
		"feature-flags.json": `{ "enabled": true }`,
		"README.md":          `ignored`,
	}

	for name, content := range files {
		path := filepath.Join(dataDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write fixture %s: %v", name, err)
		}
	}

	collections, err := DiscoverCollections(dataDir)
	if err != nil {
		t.Fatalf("DiscoverCollections returned error: %v", err)
	}

	if len(collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(collections))
	}

	news := collections["news"]
	if news.Name != "news" {
		t.Fatalf("news collection name mismatch: got %q", news.Name)
	}
	if !news.IsArray {
		t.Fatalf("news should be detected as json array")
	}

	flags := collections["feature-flags"]
	if flags.Name != "feature_flags" {
		t.Fatalf("hyphenated collection should convert to underscores, got %q", flags.Name)
	}
	if flags.IsArray {
		t.Fatalf("feature-flags should be detected as json object")
	}
}

func TestDiscoverCollections_EmptyJSONReturnsError(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "empty.json")
	if err := os.WriteFile(path, []byte(" \n\t "), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	_, err := DiscoverCollections(dataDir)
	if err == nil {
		t.Fatal("expected error for empty json file, got nil")
	}
}

func TestGetCollectionKeys(t *testing.T) {
	collections := map[string]Collection{
		"b": {Name: "b"},
		"a": {Name: "a"},
	}

	keys := getCollectionKeys(collections)
	sort.Strings(keys)
	want := []string{"a", "b"}

	if len(keys) != len(want) {
		t.Fatalf("expected %d keys, got %d", len(want), len(keys))
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys mismatch at %d: got %q, want %q", i, keys[i], want[i])
		}
	}
}
