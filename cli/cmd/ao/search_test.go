package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyResultType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"learnings path", "/foo/.agents/learnings/L42.md", "learning"},
		{"patterns path", "/foo/.agents/patterns/mutex.md", "pattern"},
		{"retros path", "/foo/.agents/retros/2026-01.md", "retro"},
		{"research path", "/foo/.agents/research/auth.md", "research"},
		{"sessions path", "/foo/.agents/ao/sessions/s1.md", "session"},
		{"decisions path", "/foo/.agents/decisions/use-go.md", "decision"},
		{"unknown path", "/foo/bar/baz.md", "knowledge"},
		{"case insensitive", "/foo/LEARNINGS/test.md", "learning"},
		{"empty path", "", "knowledge"},
		{"nested learnings", "/a/b/learnings/deep/nested.md", "learning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyResultType(tt.path)
			if got != tt.want {
				t.Errorf("classifyResultType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFilterByType(t *testing.T) {
	results := []searchResult{
		{Path: "a.md", Type: "session"},
		{Path: "b.md", Type: "learning"},
		{Path: "c.md", Type: "session"},
		{Path: "d.md", Type: "pattern"},
	}

	tests := []struct {
		name       string
		filterType string
		wantCount  int
	}{
		{"filter sessions", "session", 2},
		{"filter learnings", "learning", 1},
		{"filter patterns", "pattern", 1},
		{"filter nonexistent", "retro", 0},
		{"empty filter returns all", "", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByType(results, tt.filterType)
			if len(got) != tt.wantCount {
				t.Errorf("filterByType(%q) returned %d results, want %d", tt.filterType, len(got), tt.wantCount)
			}
		})
	}
}

func TestCalculateCASSScore(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantMin float64
		wantMax float64
	}{
		{
			name:    "all defaults",
			data:    map[string]any{},
			wantMin: 0.124, // 0.5 * 1.0 * 0.5 = 0.25
			wantMax: 0.251,
		},
		{
			name: "established maturity",
			data: map[string]any{
				"maturity":   "established",
				"utility":    0.8,
				"confidence": 0.9,
			},
			wantMin: 1.07, // 0.8 * 1.5 * 0.9 = 1.08
			wantMax: 1.09,
		},
		{
			name: "anti-pattern low weight",
			data: map[string]any{
				"maturity":   "anti-pattern",
				"utility":    0.5,
				"confidence": 0.5,
			},
			wantMin: 0.074, // 0.5 * 0.3 * 0.5 = 0.075
			wantMax: 0.076,
		},
		{
			name: "candidate maturity",
			data: map[string]any{
				"maturity":   "candidate",
				"utility":    1.0,
				"confidence": 1.0,
			},
			wantMin: 1.19, // 1.0 * 1.2 * 1.0 = 1.2
			wantMax: 1.21,
		},
		{
			name: "provisional maturity explicit",
			data: map[string]any{
				"maturity":   "provisional",
				"utility":    0.5,
				"confidence": 0.5,
			},
			wantMin: 0.249, // 0.5 * 1.0 * 0.5 = 0.25
			wantMax: 0.251,
		},
		{
			name: "zero utility uses default",
			data: map[string]any{
				"utility": 0.0,
			},
			wantMin: 0.249, // default 0.5 * 1.0 * 0.5 = 0.25
			wantMax: 0.251,
		},
		{
			name: "negative utility uses default",
			data: map[string]any{
				"utility": -1.0,
			},
			wantMin: 0.249,
			wantMax: 0.251,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCASSScore(tt.data)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateCASSScore() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGetFileContext(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file with matching content
	content := `Line one
This line contains the QUERY term
Another unrelated line
Also has query in it
Third match with query here
Fourth query should be excluded (max 3)
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("finds matching lines", func(t *testing.T) {
		ctx := getFileContext(path, "query")
		if ctx == "" {
			t.Error("expected non-empty context")
		}
		// Should contain up to MaxContextLines matches
		lines := splitNonEmpty(ctx)
		if len(lines) > MaxContextLines {
			t.Errorf("got %d context lines, want at most %d", len(lines), MaxContextLines)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		ctx := getFileContext(path, "QUERY")
		if ctx == "" {
			t.Error("expected case-insensitive match")
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		ctx := getFileContext(path, "nonexistent_xyz_999")
		if ctx != "" {
			t.Errorf("expected empty context, got %q", ctx)
		}
	})

	t.Run("nonexistent file returns empty", func(t *testing.T) {
		ctx := getFileContext(filepath.Join(tmpDir, "nope.md"), "query")
		if ctx != "" {
			t.Errorf("expected empty context for missing file, got %q", ctx)
		}
	})

	// Test line truncation
	t.Run("long lines are truncated", func(t *testing.T) {
		longLine := "query " + string(make([]byte, ContextLineMaxLength+50))
		longPath := filepath.Join(tmpDir, "long.md")
		if err := os.WriteFile(longPath, []byte(longLine), 0644); err != nil {
			t.Fatal(err)
		}
		ctx := getFileContext(longPath, "query")
		// Each line should be at most ContextLineMaxLength + "..."
		for _, line := range splitNonEmpty(ctx) {
			if len(line) > ContextLineMaxLength+3 {
				t.Errorf("line length %d exceeds max %d+3", len(line), ContextLineMaxLength)
			}
		}
	})
}

// splitNonEmpty splits a string by newlines and removes empty strings.
func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range splitLines(s) {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestSearchJSONL(t *testing.T) {
	tmpDir := t.TempDir()

	// Create JSONL files
	data1 := map[string]any{
		"id":      "L1",
		"summary": "Authentication patterns for Go services",
		"content": "Use middleware for auth",
	}
	line1, _ := json.Marshal(data1)
	if err := os.WriteFile(filepath.Join(tmpDir, "auth.jsonl"), line1, 0644); err != nil {
		t.Fatal(err)
	}

	data2 := map[string]any{
		"id":      "L2",
		"summary": "Database connection pooling",
		"content": "Pool connections for efficiency",
	}
	line2, _ := json.Marshal(data2)
	if err := os.WriteFile(filepath.Join(tmpDir, "db.jsonl"), line2, 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("finds matching JSONL", func(t *testing.T) {
		results, err := searchJSONL("auth", tmpDir, 10)
		if err != nil {
			t.Fatalf("searchJSONL() error = %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
		if len(results) > 0 && results[0].Context == "" {
			t.Error("expected non-empty context from summary field")
		}
	})

	t.Run("no match", func(t *testing.T) {
		results, err := searchJSONL("kubernetes", tmpDir, 10)
		if err != nil {
			t.Fatalf("searchJSONL() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		// Both files contain common words
		results, err := searchJSONL("for", tmpDir, 1)
		if err != nil {
			t.Fatalf("searchJSONL() error = %v", err)
		}
		if len(results) > 1 {
			t.Errorf("got %d results, want at most 1", len(results))
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		results, err := searchJSONL("test", emptyDir, 10)
		if err != nil {
			t.Fatalf("searchJSONL() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("got %d results from empty dir, want 0", len(results))
		}
	})
}

func TestSearchFilesWithFixtures(t *testing.T) {
	tmp := t.TempDir()

	// Create a sessions directory with sample session files
	sessDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a session file with searchable content
	sessionContent := `# Session: abc-123

## Summary
Implemented mutex pattern for concurrent access to shared state.

## Learnings
- Use sync.RWMutex for read-heavy workloads
- Always defer Unlock() calls
`
	if err := os.WriteFile(filepath.Join(sessDir, "session-abc-123.md"), []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write another session
	session2 := `# Session: def-456

## Summary
Database migration patterns for PostgreSQL.
`
	if err := os.WriteFile(filepath.Join(sessDir, "session-def-456.md"), []byte(session2), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("finds matching content", func(t *testing.T) {
		results, err := searchFiles("mutex", sessDir, 10)
		if err != nil {
			t.Fatalf("searchFiles() error: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected at least 1 result for 'mutex'")
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		results, err := searchFiles("kubernetes_xyz_nonexistent", sessDir, 10)
		if err != nil {
			t.Fatalf("searchFiles() error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("returns results from both sources", func(t *testing.T) {
		results, err := searchFiles("Session", sessDir, 10)
		if err != nil {
			t.Fatalf("searchFiles() error: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected results for 'Session'")
		}
	})
}

func TestSearchFilesNoData(t *testing.T) {
	tmp := t.TempDir()
	// Use an empty (but existing) directory — grep returns error for nonexistent dirs
	emptyDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	results, err := searchFiles("test", emptyDir, 10)
	if err != nil {
		t.Fatalf("searchFiles() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty dir, got %d", len(results))
	}
}

func TestParseGrepResults(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		pattern    string
		query      string
		useRipgrep bool
		wantCount  int
	}{
		{
			name:       "ripgrep output (no filtering needed)",
			output:     "/tmp/test/a.md\n/tmp/test/b.md\n",
			pattern:    "*.md",
			query:      "test",
			useRipgrep: true,
			wantCount:  2,
		},
		{
			name:       "grep output filtered by pattern",
			output:     "/tmp/test/a.md\n/tmp/test/b.txt\n/tmp/test/c.md\n",
			pattern:    "*.md",
			query:      "test",
			useRipgrep: false,
			wantCount:  2,
		},
		{
			name:       "empty output",
			output:     "",
			pattern:    "*.md",
			query:      "test",
			useRipgrep: true,
			wantCount:  0,
		},
		{
			name:       "only newlines",
			output:     "\n\n\n",
			pattern:    "*.md",
			query:      "test",
			useRipgrep: true,
			wantCount:  0,
		},
		{
			name:       "grep no pattern filter",
			output:     "/tmp/test/a.md\n",
			pattern:    "",
			query:      "test",
			useRipgrep: false,
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGrepResults([]byte(tt.output), tt.pattern, tt.query, tt.useRipgrep)
			if len(got) != tt.wantCount {
				t.Errorf("parseGrepResults() returned %d results, want %d", len(got), tt.wantCount)
			}
			// All results should have Type = "session"
			for _, r := range got {
				if r.Type != "session" {
					t.Errorf("result Type = %q, want %q", r.Type, "session")
				}
			}
		})
	}
}

func TestSearchFilesCombinedLimitEnforcement(t *testing.T) {
	tmp := t.TempDir()
	sessDir := filepath.Join(tmp, "sessions")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create multiple markdown files
	for i := 1; i <= 5; i++ {
		content := "test content with searchable term\n"
		path := filepath.Join(sessDir, "session-"+string(rune('a'+i-1))+".md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create multiple JSONL files
	for i := 1; i <= 5; i++ {
		data := map[string]any{
			"id":      "L" + string(rune('0'+i)),
			"summary": "searchable term in JSONL content",
		}
		line, _ := json.Marshal(data)
		path := filepath.Join(sessDir, "learning-"+string(rune('a'+i-1))+".jsonl")
		if err := os.WriteFile(path, line, 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("enforces combined limit after dedup", func(t *testing.T) {
		limit := 3
		results, err := searchFiles("searchable", sessDir, limit)
		if err != nil {
			t.Fatalf("searchFiles() error: %v", err)
		}

		// Total files: 5 md + 5 jsonl = 10 unique results
		// After dedup, should be exactly the limit (3)
		if len(results) > limit {
			t.Errorf("searchFiles() returned %d results, want at most %d", len(results), limit)
		}
	})

	t.Run("no limit enforcement when limit is 0", func(t *testing.T) {
		results, err := searchFiles("searchable", sessDir, 0)
		if err != nil {
			t.Fatalf("searchFiles() error: %v", err)
		}

		// Should return all unique results without limit
		if len(results) == 0 {
			t.Error("expected results when limit=0, got none")
		}
	})

	t.Run("limit larger than results", func(t *testing.T) {
		limit := 100
		results, err := searchFiles("searchable", sessDir, limit)
		if err != nil {
			t.Fatalf("searchFiles() error: %v", err)
		}

		// Should return all available results (10)
		if len(results) > limit {
			t.Errorf("searchFiles() returned %d results, want at most %d", len(results), limit)
		}
	})
}
