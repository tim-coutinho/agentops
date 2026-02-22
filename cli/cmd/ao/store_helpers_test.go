package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for pure helper functions in store.go

func TestExtractCategoryAndTags(t *testing.T) {
	t.Run("YAML frontmatter with category and tags", func(t *testing.T) {
		content := "---\ncategory: testing\ntags: [go, coverage, rpi]\n---\n\n# My Learning"
		cat, tags := extractCategoryAndTags(content)
		if cat != "testing" {
			t.Errorf("category = %q, want %q", cat, "testing")
		}
		if len(tags) != 3 {
			t.Errorf("tags count = %d, want 3; tags = %v", len(tags), tags)
		}
	})

	t.Run("YAML frontmatter category only", func(t *testing.T) {
		content := "---\ncategory: \"go-patterns\"\n---\n\ncontent here"
		cat, tags := extractCategoryAndTags(content)
		if cat != "go-patterns" {
			t.Errorf("category = %q, want %q", cat, "go-patterns")
		}
		if len(tags) != 0 {
			t.Errorf("expected no tags, got %v", tags)
		}
	})

	t.Run("markdown format category and tags", func(t *testing.T) {
		content := "# My Learning\n\n**Category**: workflow\n**Tags**: automation, ci, testing"
		cat, tags := extractCategoryAndTags(content)
		if cat != "workflow" {
			t.Errorf("category = %q, want %q", cat, "workflow")
		}
		if len(tags) != 3 {
			t.Errorf("tags count = %d, want 3; tags = %v", len(tags), tags)
		}
	})

	t.Run("no metadata returns empty", func(t *testing.T) {
		content := "# My Learning\n\nJust some plain content."
		cat, tags := extractCategoryAndTags(content)
		if cat != "" {
			t.Errorf("expected empty category, got %q", cat)
		}
		if len(tags) != 0 {
			t.Errorf("expected no tags, got %v", tags)
		}
	})

	t.Run("empty content returns empty", func(t *testing.T) {
		cat, tags := extractCategoryAndTags("")
		if cat != "" {
			t.Errorf("expected empty category, got %q", cat)
		}
		if len(tags) != 0 {
			t.Errorf("expected no tags, got %v", tags)
		}
	})

	t.Run("YAML frontmatter category YAML wins over markdown", func(t *testing.T) {
		content := "---\ncategory: yaml-category\n---\n**Category**: markdown-category"
		cat, _ := extractCategoryAndTags(content)
		if cat != "yaml-category" {
			t.Errorf("category = %q, want %q", cat, "yaml-category")
		}
	})

	t.Run("tags with quoted values in YAML", func(t *testing.T) {
		content := "---\ntags: [\"go\", \"test\"]\n---\n"
		_, tags := extractCategoryAndTags(content)
		if len(tags) != 2 {
			t.Errorf("expected 2 tags, got %d: %v", len(tags), tags)
		}
	})
}

func TestCreateIndexEntry(t *testing.T) {
	t.Run("creates entry for learning file", func(t *testing.T) {
		dir := t.TempDir()
		learningsDir := filepath.Join(dir, ".agents", "learnings")
		if err := os.MkdirAll(learningsDir, 0755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(learningsDir, "my-learning.md")
		content := "# My Learning\n\nThis is a test learning.\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		entry, err := createIndexEntry(path, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if entry.Type != "learning" {
			t.Errorf("Type = %q, want %q", entry.Type, "learning")
		}
		if entry.Title != "My Learning" {
			t.Errorf("Title = %q, want %q", entry.Title, "My Learning")
		}
		if entry.Path != path {
			t.Errorf("Path = %q, want %q", entry.Path, path)
		}
	})

	t.Run("creates entry for pattern file", func(t *testing.T) {
		dir := t.TempDir()
		patternsDir := filepath.Join(dir, ".agents", "patterns")
		if err := os.MkdirAll(patternsDir, 0755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(patternsDir, "mutex-pattern.md")
		if err := os.WriteFile(path, []byte("# Mutex Pattern\n\nUse sync.Mutex.\n"), 0644); err != nil {
			t.Fatal(err)
		}
		entry, err := createIndexEntry(path, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if entry.Type != "pattern" {
			t.Errorf("Type = %q, want %q", entry.Type, "pattern")
		}
	})

	t.Run("categorize=true extracts category and tags", func(t *testing.T) {
		dir := t.TempDir()
		learningsDir := filepath.Join(dir, ".agents", "learnings")
		if err := os.MkdirAll(learningsDir, 0755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(learningsDir, "tagged.md")
		content := "---\ncategory: testing\ntags: [go, coverage]\n---\n\n# Tagged Learning\n\nContent here.\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		entry, err := createIndexEntry(path, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if entry.Category != "testing" {
			t.Errorf("Category = %q, want %q", entry.Category, "testing")
		}
		// Keywords should include category and tag values
		kws := strings.Join(entry.Keywords, " ")
		if !strings.Contains(kws, "testing") {
			t.Errorf("Keywords should contain category 'testing', got: %v", entry.Keywords)
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := createIndexEntry("/tmp/definitely-not-existing-12345.md", false)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}
