package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsContainedPath(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		path     string
		expected bool
	}{
		{
			name:     "contained simple path",
			baseDir:  "/home/user/project",
			path:     "/home/user/project/file.md",
			expected: true,
		},
		{
			name:     "contained nested path",
			baseDir:  "/home/user/project",
			path:     "/home/user/project/subdir/file.md",
			expected: true,
		},
		{
			name:     "not contained - parent traversal",
			baseDir:  "/home/user/project",
			path:     "/home/user/project/../other/file.md",
			expected: false,
		},
		{
			name:     "not contained - sibling directory",
			baseDir:  "/home/user/project",
			path:     "/home/user/other/file.md",
			expected: false,
		},
		{
			name:     "not contained - absolute outside",
			baseDir:  "/home/user/project",
			path:     "/etc/passwd",
			expected: false,
		},
		{
			name:     "base dir itself is contained",
			baseDir:  "/home/user/project",
			path:     "/home/user/project",
			expected: true,
		},
		{
			name:     "prefix attack - similar name",
			baseDir:  "/home/user/project",
			path:     "/home/user/project-evil/file.md",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isContainedPath(tc.baseDir, tc.path)
			if result != tc.expected {
				t.Errorf("isContainedPath(%q, %q) = %v, want %v",
					tc.baseDir, tc.path, result, tc.expected)
			}
		})
	}
}

func TestIsArtifactFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"markdown file", "test.md", true},
		{"jsonl file", "data.jsonl", true},
		{"text file", "readme.txt", false},
		{"go file", "main.go", false},
		{"no extension", "Makefile", false},
		{"hidden md", ".hidden.md", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isArtifactFile(tc.filename)
			if result != tc.expected {
				t.Errorf("isArtifactFile(%q) = %v, want %v",
					tc.filename, result, tc.expected)
			}
		})
	}
}

func TestParseMarkdownField(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		field    string
		expected string
		found    bool
	}{
		{
			name:     "standard bold field",
			line:     "**ID**: L1-test",
			field:    "ID",
			expected: "L1-test",
			found:    true,
		},
		{
			name:     "bold field with trailing colon",
			line:     "**ID:** L1-test",
			field:    "ID",
			expected: "L1-test",
			found:    true,
		},
		{
			name:     "list item field",
			line:     "- **Maturity**: candidate",
			field:    "Maturity",
			expected: "candidate",
			found:    true,
		},
		{
			name:     "field not found",
			line:     "**Other**: value",
			field:    "ID",
			expected: "",
			found:    false,
		},
		{
			name:     "value with spaces",
			line:     "**Status**: TEMPERED",
			field:    "Status",
			expected: "TEMPERED",
			found:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, found := parseMarkdownField(tc.line, tc.field)
			if found != tc.found {
				t.Errorf("parseMarkdownField(%q, %q) found = %v, want %v",
					tc.line, tc.field, found, tc.found)
			}
			if result != tc.expected {
				t.Errorf("parseMarkdownField(%q, %q) = %q, want %q",
					tc.line, tc.field, result, tc.expected)
			}
		})
	}
}

func TestExpandFilePatterns(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "temper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	// Create test files
	testFiles := []string{
		"file1.md",
		"file2.md",
		"data.jsonl",
		"readme.txt",
		"subdir/nested.md",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	t.Run("glob pattern", func(t *testing.T) {
		files, err := expandFilePatterns(tmpDir, []string{"*.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d: %v", len(files), files)
		}
	})

	t.Run("literal file", func(t *testing.T) {
		files, err := expandFilePatterns(tmpDir, []string{"file1.md"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("expected 1 file, got %d: %v", len(files), files)
		}
	})

	t.Run("directory non-recursive", func(t *testing.T) {
		temperRecursive = false
		files, err := expandFilePatterns(tmpDir, []string{tmpDir})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should find file1.md, file2.md, data.jsonl (not readme.txt, not nested.md)
		if len(files) != 3 {
			t.Errorf("expected 3 files, got %d: %v", len(files), files)
		}
	})

	t.Run("directory recursive", func(t *testing.T) {
		temperRecursive = true
		defer func() { temperRecursive = false }()

		files, err := expandFilePatterns(tmpDir, []string{tmpDir})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should find file1.md, file2.md, data.jsonl, subdir/nested.md
		if len(files) != 4 {
			t.Errorf("expected 4 files, got %d: %v", len(files), files)
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		_, err := expandFilePatterns(tmpDir, []string{"../../../etc/passwd"})
		if err == nil {
			t.Error("expected error for path traversal, got nil")
		}
	})
}

func TestParseArtifactMetadata(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "temper-meta-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	t.Run("parse markdown artifact", func(t *testing.T) {
		content := `# Learning: Test Pattern

**ID**: L1-test-pattern
**Maturity**: candidate
**Utility**: 0.75
**Confidence**: 0.8
**Status**: TEMPERED
**Schema Version**: 1

## Summary
This is a test learning.
`
		path := filepath.Join(tmpDir, "test-learning.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		meta, err := parseArtifactMetadata(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if meta.ID != "L1-test-pattern" {
			t.Errorf("expected ID 'L1-test-pattern', got %q", meta.ID)
		}
		if meta.Maturity != "candidate" {
			t.Errorf("expected Maturity 'candidate', got %q", meta.Maturity)
		}
		if meta.Utility != 0.75 {
			t.Errorf("expected Utility 0.75, got %f", meta.Utility)
		}
		if !meta.Tempered {
			t.Error("expected Tempered to be true")
		}
	})

	t.Run("fallback to filename for ID", func(t *testing.T) {
		content := `# Learning without ID

Just some content.
`
		path := filepath.Join(tmpDir, "unnamed-learning.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		meta, err := parseArtifactMetadata(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if meta.ID != "unnamed-learning" {
			t.Errorf("expected ID 'unnamed-learning', got %q", meta.ID)
		}
	})
}

func TestValidateArtifact(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "temper-validate-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	t.Run("valid artifact", func(t *testing.T) {
		content := `# Test

**ID**: L1-valid
**Maturity**: candidate
**Utility**: 0.8
`
		path := filepath.Join(tmpDir, "valid.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		result := validateArtifact(path, "provisional", 0.5, 0)
		if !result.Valid {
			t.Errorf("expected valid artifact, got issues: %v", result.Issues)
		}
	})

	t.Run("low utility rejected", func(t *testing.T) {
		content := `# Test

**ID**: L1-low-util
**Maturity**: candidate
**Utility**: 0.3
`
		path := filepath.Join(tmpDir, "low-util.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		result := validateArtifact(path, "provisional", 0.5, 0)
		if result.Valid {
			t.Error("expected invalid artifact for low utility")
		}
	})

	t.Run("low maturity rejected", func(t *testing.T) {
		content := `# Test

**ID**: L1-low-mat
**Maturity**: provisional
**Utility**: 0.8
`
		path := filepath.Join(tmpDir, "low-mat.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		result := validateArtifact(path, "candidate", 0.5, 0)
		if result.Valid {
			t.Error("expected invalid artifact for low maturity")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		result := validateArtifact(filepath.Join(tmpDir, "nonexistent.md"), "provisional", 0.5, 0)
		if result.Valid {
			t.Error("expected invalid for nonexistent file")
		}
	})
}
