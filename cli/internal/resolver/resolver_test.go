package resolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestLearnings(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	learningsDir := filepath.Join(root, ".agents", "learnings")
	patternsDir := filepath.Join(root, ".agents", "patterns")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(patternsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// JSONL learning
	if err := os.WriteFile(filepath.Join(learningsDir, "L001.jsonl"), []byte(`{"id":"L001","title":"test"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Markdown learning
	if err := os.WriteFile(filepath.Join(learningsDir, "L002.md"), []byte("---\nid: L002\ntitle: test\n---\n# L002\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Learning with long filename for glob matching
	if err := os.WriteFile(filepath.Join(learningsDir, "learning-003.jsonl"), []byte(`{"id":"003","title":"three"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Pattern file
	if err := os.WriteFile(filepath.Join(patternsDir, "retry-backoff.md"), []byte("# Retry Backoff\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Learning with frontmatter ID different from filename
	if err := os.WriteFile(filepath.Join(learningsDir, "some-file.md"), []byte("---\nid: learn-2026-02-21-backend-detection\ntitle: Backend Detection\n---\n# Content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestFileResolver_Resolve(t *testing.T) {
	root := setupTestLearnings(t)
	r := NewFileResolver(root)

	tests := []struct {
		name     string
		id       string
		wantBase string
		wantErr  bool
	}{
		{
			name:     "resolve by ID with jsonl extension",
			id:       "L001",
			wantBase: "L001.jsonl",
		},
		{
			name:     "resolve by ID with md extension",
			id:       "L002",
			wantBase: "L002.md",
		},
		{
			name:     "resolve by filename without extension",
			id:       "learning-003",
			wantBase: "learning-003.jsonl",
		},
		{
			name:     "resolve by glob (partial ID)",
			id:       "003",
			wantBase: "learning-003.jsonl",
		},
		{
			name:     "resolve pattern by name",
			id:       "retry-backoff",
			wantBase: "retry-backoff.md",
		},
		{
			name:     "resolve by frontmatter ID",
			id:       "learn-2026-02-21-backend-detection",
			wantBase: "some-file.md",
		},
		{
			name:    "not found returns error",
			id:      "nonexistent-xyz-999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := r.Resolve(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if tt.wantBase != "" && filepath.Base(path) != tt.wantBase {
				t.Errorf("Resolve(%q) = %q, want base %q", tt.id, path, tt.wantBase)
			}
		})
	}
}

func TestFileResolver_Resolve_PoolID(t *testing.T) {
	root := t.TempDir()
	learningsDir := filepath.Join(root, ".agents", "learnings")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// File named after the learning ID (without pend- prefix)
	if err := os.WriteFile(filepath.Join(learningsDir, "fix-auth-bug.md"), []byte("# Fix Auth Bug\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewFileResolver(root)
	path, err := r.Resolve("pend-fix-auth-bug")
	if err != nil {
		t.Fatalf("Resolve(pend-fix-auth-bug) error = %v", err)
	}
	if filepath.Base(path) != "fix-auth-bug.md" {
		t.Errorf("Resolve(pend-fix-auth-bug) = %q, want base fix-auth-bug.md", path)
	}
}

func TestFileResolver_Resolve_AbsolutePath(t *testing.T) {
	root := t.TempDir()
	learningsDir := filepath.Join(root, ".agents", "learnings")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		t.Fatal(err)
	}
	absPath := filepath.Join(learningsDir, "L001.jsonl")
	if err := os.WriteFile(absPath, []byte(`{"id":"L001"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewFileResolver(root)
	path, err := r.Resolve(absPath)
	if err != nil {
		t.Fatalf("Resolve(absolute path) error = %v", err)
	}
	if path != absPath {
		t.Errorf("Resolve(absolute path) = %q, want %q", path, absPath)
	}
}

func TestFileResolver_Resolve_ParentWalk(t *testing.T) {
	// Create a nested structure: root/.agents/learnings/L001.md
	// Then resolve from root/sub/dir
	root := t.TempDir()
	learningsDir := filepath.Join(root, ".agents", "learnings")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(learningsDir, "L001.md"), []byte("# L001\n"), 0644); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	r := NewFileResolver(subDir)
	path, err := r.Resolve("L001")
	if err != nil {
		t.Fatalf("Resolve from subdir error = %v", err)
	}
	if filepath.Base(path) != "L001.md" {
		t.Errorf("Resolve from subdir = %q, want base L001.md", path)
	}
}

func TestFileResolver_Resolve_NotFoundError(t *testing.T) {
	root := t.TempDir()
	r := NewFileResolver(root)
	_, err := r.Resolve("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent learning")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want containing 'not found'", err.Error())
	}
}

func TestFileResolver_ImplementsInterface(t *testing.T) {
	var _ LearningResolver = &FileResolver{}
}

func TestFileResolver_DiscoverAll(t *testing.T) {
	root := setupTestLearnings(t)
	r := NewFileResolver(root)

	files, err := r.DiscoverAll()
	if err != nil {
		t.Fatalf("DiscoverAll() error = %v", err)
	}

	// setupTestLearnings creates: L001.jsonl, L002.md, learning-003.jsonl, some-file.md in learnings/
	// and retry-backoff.md in patterns/
	if len(files) != 5 {
		t.Errorf("DiscoverAll() returned %d files, want 5", len(files))
		for _, f := range files {
			t.Logf("  %s", f)
		}
	}

	// Verify known files are present
	bases := make(map[string]bool)
	for _, f := range files {
		bases[filepath.Base(f)] = true
	}
	for _, want := range []string{"L001.jsonl", "L002.md", "learning-003.jsonl", "some-file.md", "retry-backoff.md"} {
		if !bases[want] {
			t.Errorf("DiscoverAll() missing %s", want)
		}
	}
}

func TestFileResolver_DiscoverAll_Empty(t *testing.T) {
	root := t.TempDir()
	r := NewFileResolver(root)

	files, err := r.DiscoverAll()
	if err != nil {
		t.Fatalf("DiscoverAll() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("DiscoverAll() on empty dir returned %d files, want 0", len(files))
	}
}
