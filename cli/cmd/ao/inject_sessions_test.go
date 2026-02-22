package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSessionFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("JSONL session", func(t *testing.T) {
		data := map[string]interface{}{
			"summary": "Worked on authentication module",
		}
		line, _ := json.Marshal(data)
		path := filepath.Join(tmpDir, "session1.jsonl")
		if err := os.WriteFile(path, line, 0644); err != nil {
			t.Fatal(err)
		}

		s, err := parseSessionFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Summary != "Worked on authentication module" {
			t.Errorf("Summary = %q, want %q", s.Summary, "Worked on authentication module")
		}
		if s.Date == "" {
			t.Error("expected non-empty Date")
		}
	})

	t.Run("markdown session", func(t *testing.T) {
		content := `# Session Summary

Implemented new database migration system.
`
		path := filepath.Join(tmpDir, "session2.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		s, err := parseSessionFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Summary != "Implemented new database migration system." {
			t.Errorf("Summary = %q, want %q", s.Summary, "Implemented new database migration system.")
		}
	})

	t.Run("empty markdown", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty.md")
		if err := os.WriteFile(path, []byte("# Title\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}

		s, err := parseSessionFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Summary != "" {
			t.Errorf("Summary = %q, want empty (only headings and separators)", s.Summary)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := parseSessionFile(filepath.Join(tmpDir, "nope.jsonl"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("invalid JSONL", func(t *testing.T) {
		path := filepath.Join(tmpDir, "bad.jsonl")
		if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
			t.Fatal(err)
		}

		s, err := parseSessionFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Invalid JSON should result in empty summary
		if s.Summary != "" {
			t.Errorf("Summary = %q, want empty for invalid JSON", s.Summary)
		}
	})

	t.Run("long summary truncated", func(t *testing.T) {
		longSummary := make([]byte, 200)
		for i := range longSummary {
			longSummary[i] = 'a'
		}
		data := map[string]interface{}{
			"summary": string(longSummary),
		}
		line, _ := json.Marshal(data)
		path := filepath.Join(tmpDir, "long.jsonl")
		if err := os.WriteFile(path, line, 0644); err != nil {
			t.Fatal(err)
		}

		s, err := parseSessionFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.Summary) > 153 { // 150 + "..."
			t.Errorf("Summary length = %d, want at most 153 (truncated)", len(s.Summary))
		}
	})
}

func TestCollectRecentSessions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create sessions directory
	sessionsDir := filepath.Join(tmpDir, ".agents", "ao", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	data1 := map[string]interface{}{"summary": "Auth work"}
	line1, _ := json.Marshal(data1)
	if err := os.WriteFile(filepath.Join(sessionsDir, "s1.jsonl"), line1, 0644); err != nil {
		t.Fatal(err)
	}

	data2 := map[string]interface{}{"summary": "Database migration"}
	line2, _ := json.Marshal(data2)
	if err := os.WriteFile(filepath.Join(sessionsDir, "s2.jsonl"), line2, 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(sessionsDir, "s3.md"), []byte("# Summary\n\nWorked on testing"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("collects all sessions", func(t *testing.T) {
		got, err := collectRecentSessions(tmpDir, "", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("got %d sessions, want 3", len(got))
		}
	})

	t.Run("filters by query", func(t *testing.T) {
		got, err := collectRecentSessions(tmpDir, "auth", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d sessions for 'auth', want 1", len(got))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		got, err := collectRecentSessions(tmpDir, "", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) > 2 {
			t.Errorf("got %d sessions, want at most 2", len(got))
		}
	})

	t.Run("no sessions directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		got, err := collectRecentSessions(emptyDir, "", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
