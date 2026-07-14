package mongo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestDiscoverCollections_DirectoryCollection(t *testing.T) {
	dataDir := t.TempDir()
	dirPath := filepath.Join(dataDir, "blog-posts")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("failed to create collection directory: %v", err)
	}

	files := map[string]string{
		"002-second.json": `{"slug":"second","title":"Curly quote: “second”"}`,
		"001-first.json":  `{"slug":"first","title":"First"}`,
		"README.md":       `ignored`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dirPath, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write fixture %s: %v", name, err)
		}
	}

	collections, err := DiscoverCollections(dataDir)
	if err != nil {
		t.Fatalf("DiscoverCollections returned error: %v", err)
	}
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	blog := collections["blog-posts"]
	if blog.Name != "blog_posts" {
		t.Fatalf("directory collection name mismatch: got %q", blog.Name)
	}
	if blog.File != "" {
		t.Fatalf("directory collection should not have a single file: %q", blog.File)
	}
	if blog.IsArray {
		t.Fatal("directory collection should stream individual JSON objects")
	}
	if len(blog.Files) != 2 {
		t.Fatalf("expected 2 JSON object files, got %d", len(blog.Files))
	}
	if got := filepath.Base(blog.Files[0]); got != "001-first.json" {
		t.Fatalf("directory files should be ordered by name, first got %q", got)
	}
	if got := filepath.Base(blog.Files[1]); got != "002-second.json" {
		t.Fatalf("directory files should be ordered by name, second got %q", got)
	}

	data, err := directoryJSONLines(blog.Files)
	if err != nil {
		t.Fatalf("directoryJSONLines returned error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 newline-delimited documents, got %d", len(lines))
	}
	posts := make([]struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}, len(lines))
	for i, line := range lines {
		if err := json.Unmarshal([]byte(line), &posts[i]); err != nil {
			t.Fatalf("directory payload line %d is invalid JSON: %v", i+1, err)
		}
	}
	if len(posts) != 2 || posts[0].Slug != "first" || posts[1].Slug != "second" {
		t.Fatalf("unexpected combined posts: %+v", posts)
	}
	if posts[1].Title != "Curly quote: “second”" {
		t.Fatalf("unicode content changed: %q", posts[1].Title)
	}
}

func TestPrepareMongoImport_DirectoryCollectionUsesStdin(t *testing.T) {
	dataDir := t.TempDir()
	first := filepath.Join(dataDir, "first.json")
	second := filepath.Join(dataDir, "second.json")
	if err := os.WriteFile(first, []byte("{\n  \"slug\": \"first\"\n}"), 0o644); err != nil {
		t.Fatalf("failed writing first fixture: %v", err)
	}
	if err := os.WriteFile(second, []byte("{\n  \"slug\": \"second\"\n}"), 0o644); err != nil {
		t.Fatalf("failed writing second fixture: %v", err)
	}

	args, input, err := prepareMongoImport("mongodb://localhost:27017", "test", Collection{
		Name:  "blog_posts",
		Files: []string{first, second},
	})
	if err != nil {
		t.Fatalf("prepareMongoImport returned error: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--file") {
		t.Fatalf("directory import should read stdin, got args %v", args)
	}
	if strings.Contains(joined, "--jsonArray") {
		t.Fatalf("directory import should use newline-delimited objects, got args %v", args)
	}
	if !strings.Contains(joined, "--collection blog_posts") || !strings.Contains(joined, "--drop") {
		t.Fatalf("directory import args missing collection or drop: %v", args)
	}
	if got, want := string(input), "{\"slug\":\"first\"}\n{\"slug\":\"second\"}\n"; got != want {
		t.Fatalf("stdin payload mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestPrepareMongoImport_ArrayFileKeepsExistingFlags(t *testing.T) {
	args, input, err := prepareMongoImport("mongodb://localhost:27017", "test", Collection{
		Name:    "news",
		File:    "/tmp/news.json",
		IsArray: true,
	})
	if err != nil {
		t.Fatalf("prepareMongoImport returned error: %v", err)
	}
	if input != nil {
		t.Fatalf("single-file import should not provide stdin, got %q", input)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--file /tmp/news.json") || !strings.Contains(joined, "--jsonArray") {
		t.Fatalf("array-file import flags changed: %v", args)
	}
}

func TestDiscoverCollections_DirectoryRequiresJSONObjectFiles(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantError string
	}{
		{name: "array", content: `[{"slug":"post"}]`, wantError: "expected one JSON object"},
		{name: "invalid", content: `{"slug":`, wantError: "invalid JSON"},
		{name: "empty", content: " \n\t ", wantError: "empty JSON file"},
		{name: "primitive", content: `"post"`, wantError: "expected one JSON object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()
			dirPath := filepath.Join(dataDir, "blog-posts")
			if err := os.Mkdir(dirPath, 0o755); err != nil {
				t.Fatalf("failed to create collection directory: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dirPath, "post.json"), []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write fixture: %v", err)
			}

			_, err := DiscoverCollections(dataDir)
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDiscoverCollections_DuplicateFileAndDirectoryKeyReturnsError(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "blog-posts.json"), []byte(`[]`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	dirPath := filepath.Join(dataDir, "blog-posts")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("failed to create collection directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "post.json"), []byte(`{"slug":"post"}`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	_, err := DiscoverCollections(dataDir)
	if err == nil {
		t.Fatal("expected error for duplicate file and directory collection keys")
	}
	if !strings.Contains(err.Error(), `duplicate collection key "blog-posts"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverCollections_DuplicateMongoCollectionNameReturnsError(t *testing.T) {
	dataDir := t.TempDir()
	for _, name := range []string{"blog-posts.json", "blog_posts.json"} {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte(`[]`), 0o644); err != nil {
			t.Fatalf("failed to write fixture %s: %v", name, err)
		}
	}

	_, err := DiscoverCollections(dataDir)
	if err == nil {
		t.Fatal("expected error for source keys targeting the same MongoDB collection")
	}
	if !strings.Contains(err.Error(), `both map to MongoDB collection "blog_posts"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverCollections_DirectoryWithoutJSONFilesIsIgnored(t *testing.T) {
	dataDir := t.TempDir()
	dirPath := filepath.Join(dataDir, "notes")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "README.md"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	collections, err := DiscoverCollections(dataDir)
	if err != nil {
		t.Fatalf("DiscoverCollections returned error: %v", err)
	}
	if len(collections) != 0 {
		t.Fatalf("expected no collections, got %v", collections)
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

func TestDiscoverCollections_InvalidJSONReturnsError(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "invalid.json")
	if err := os.WriteFile(path, []byte(`{"broken":`), 0o644); err != nil {
		t.Fatalf("failed writing fixture: %v", err)
	}

	_, err := DiscoverCollections(dataDir)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCollectionKeys(t *testing.T) {
	collections := map[string]Collection{
		"b": {Name: "b"},
		"a": {Name: "a"},
	}

	keys := getCollectionKeys(collections)
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
