package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeResult(t *testing.T) {
	tests := []struct {
		name       string
		checks     []doctorCheck
		wantResult string
		wantFails  bool
	}{
		{
			name: "all pass",
			checks: []doctorCheck{
				{Name: "a", Status: "pass", Required: true},
				{Name: "b", Status: "pass", Required: true},
			},
			wantResult: "HEALTHY",
			wantFails:  false,
		},
		{
			name: "one failure",
			checks: []doctorCheck{
				{Name: "a", Status: "pass", Required: true},
				{Name: "b", Status: "fail", Required: true},
			},
			wantResult: "UNHEALTHY",
			wantFails:  true,
		},
		{
			name: "warnings only",
			checks: []doctorCheck{
				{Name: "a", Status: "pass", Required: true},
				{Name: "b", Status: "warn", Required: false},
			},
			wantResult: "HEALTHY",
			wantFails:  false,
		},
		{
			name: "mixed failures and warnings",
			checks: []doctorCheck{
				{Name: "a", Status: "fail", Required: true},
				{Name: "b", Status: "warn", Required: false},
				{Name: "c", Status: "pass", Required: true},
			},
			wantResult: "UNHEALTHY",
			wantFails:  true,
		},
		{
			name:       "empty checks",
			checks:     []doctorCheck{},
			wantResult: "HEALTHY",
			wantFails:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := computeResult(tt.checks)
			if output.Result != tt.wantResult {
				t.Errorf("computeResult() result = %q, want %q", output.Result, tt.wantResult)
			}
			if tt.wantFails && output.Summary == "all checks passed" {
				t.Error("expected failure in summary")
			}
			if !tt.wantFails && len(tt.checks) > 0 && !hasWarns(tt.checks) {
				expected := fmt.Sprintf("%d/%d checks passed", len(tt.checks), len(tt.checks))
				if output.Summary != expected {
					t.Errorf("expected %q, got %q", expected, output.Summary)
				}
			}
		})
	}
}

func hasWarns(checks []doctorCheck) bool {
	for _, c := range checks {
		if c.Status == "warn" {
			return true
		}
	}
	return false
}

func TestCountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("empty directory", func(t *testing.T) {
		got := countFiles(tmpDir)
		if got != 0 {
			t.Errorf("countFiles(empty) = %d, want 0", got)
		}
	})

	t.Run("with files", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(tmpDir, "a.md"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "b.md"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
			t.Fatal(err)
		}

		got := countFiles(tmpDir)
		if got != 2 {
			t.Errorf("countFiles() = %d, want 2 (should not count directories)", got)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		got := countFiles(filepath.Join(tmpDir, "nonexistent"))
		if got != 0 {
			t.Errorf("countFiles(nonexistent) = %d, want 0", got)
		}
	})
}

// --- Integration tests for doctor check functions ---

// chdirTemp changes to a temp dir and returns a cleanup function.
func chdirTemp(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	return tmp
}

func TestCheckKnowledgeBase(t *testing.T) {
	t.Run("initialized", func(t *testing.T) {
		tmp := chdirTemp(t)
		if err := os.MkdirAll(filepath.Join(tmp, ".agents", "ao"), 0755); err != nil {
			t.Fatal(err)
		}
		result := checkKnowledgeBase()
		if result.Status != "pass" {
			t.Errorf("status=%q, want pass (detail: %s)", result.Status, result.Detail)
		}
	})

	t.Run("not initialized", func(t *testing.T) {
		chdirTemp(t)
		result := checkKnowledgeBase()
		if result.Status != "fail" {
			t.Errorf("status=%q, want fail (detail: %s)", result.Status, result.Detail)
		}
	})
}

func TestCheckKnowledgeFreshness(t *testing.T) {
	t.Run("recent session", func(t *testing.T) {
		tmp := chdirTemp(t)
		sessDir := filepath.Join(tmp, ".agents", "ao", "sessions")
		if err := os.MkdirAll(sessDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sessDir, "session-1.md"), []byte("recent"), 0644); err != nil {
			t.Fatal(err)
		}

		result := checkKnowledgeFreshness()
		if result.Status != "pass" {
			t.Errorf("status=%q, want pass (detail: %s)", result.Status, result.Detail)
		}
	})

	t.Run("no sessions", func(t *testing.T) {
		tmp := chdirTemp(t)
		if err := os.MkdirAll(filepath.Join(tmp, ".agents", "ao", "sessions"), 0755); err != nil {
			t.Fatal(err)
		}

		result := checkKnowledgeFreshness()
		if result.Status != "warn" {
			t.Errorf("status=%q, want warn (detail: %s)", result.Status, result.Detail)
		}
	})

	t.Run("no sessions dir", func(t *testing.T) {
		chdirTemp(t)
		result := checkKnowledgeFreshness()
		if result.Status != "warn" {
			t.Errorf("status=%q, want warn (detail: %s)", result.Status, result.Detail)
		}
	})
}

func TestCheckSearchIndex(t *testing.T) {
	t.Run("index exists with content", func(t *testing.T) {
		tmp := chdirTemp(t)
		indexDir := filepath.Join(tmp, IndexDir)
		if err := os.MkdirAll(indexDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(indexDir, IndexFileName), []byte("{\"term\":\"hello\"}\n{\"term\":\"world\"}\n"), 0644); err != nil {
			t.Fatal(err)
		}

		result := checkSearchIndex()
		if result.Status != "pass" {
			t.Errorf("status=%q, want pass (detail: %s)", result.Status, result.Detail)
		}
	})

	t.Run("empty index", func(t *testing.T) {
		tmp := chdirTemp(t)
		indexDir := filepath.Join(tmp, IndexDir)
		if err := os.MkdirAll(indexDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(indexDir, IndexFileName), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		result := checkSearchIndex()
		if result.Status != "warn" {
			t.Errorf("status=%q, want warn (detail: %s)", result.Status, result.Detail)
		}
	})

	t.Run("no index", func(t *testing.T) {
		chdirTemp(t)
		result := checkSearchIndex()
		if result.Status != "warn" {
			t.Errorf("status=%q, want warn (detail: %s)", result.Status, result.Detail)
		}
	})
}

func TestCheckFlywheelHealth(t *testing.T) {
	t.Run("with learnings", func(t *testing.T) {
		tmp := chdirTemp(t)
		learningsDir := filepath.Join(tmp, ".agents", "ao", "learnings")
		if err := os.MkdirAll(learningsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(learningsDir, "L1.md"), []byte("learning 1"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(learningsDir, "L2.md"), []byte("learning 2"), 0644); err != nil {
			t.Fatal(err)
		}

		result := checkFlywheelHealth()
		if result.Status != "pass" {
			t.Errorf("status=%q, want pass (detail: %s)", result.Status, result.Detail)
		}
	})

	t.Run("no learnings", func(t *testing.T) {
		chdirTemp(t)
		result := checkFlywheelHealth()
		if result.Status != "warn" {
			t.Errorf("status=%q, want warn (detail: %s)", result.Status, result.Detail)
		}
	})

	t.Run("alt path learnings", func(t *testing.T) {
		tmp := chdirTemp(t)
		altDir := filepath.Join(tmp, ".agents", "learnings")
		if err := os.MkdirAll(altDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(altDir, "L1.md"), []byte("learning"), 0644); err != nil {
			t.Fatal(err)
		}

		result := checkFlywheelHealth()
		if result.Status != "pass" {
			t.Errorf("status=%q, want pass (detail: %s)", result.Status, result.Detail)
		}
	})
}

func TestCountHooksInMap(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want int
	}{
		{
			name: "flat hook arrays",
			raw: map[string]any{
				"PreToolUse":  []any{"hook1", "hook2"},
				"PostToolUse": []any{"hook3"},
			},
			want: 3,
		},
		{
			name: "empty map",
			raw:  map[string]any{},
			want: 0,
		},
		{
			name: "nested hooks map",
			raw: map[string]any{
				"hooks": map[string]any{
					"PreToolUse": []any{"h1"},
				},
			},
			want: 1,
		},
		{
			name: "nil input",
			raw:  nil,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countHooksInMap(tt.raw)
			if got != tt.want {
				t.Errorf("countHooksInMap() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountFileLines(t *testing.T) {
	tmp := t.TempDir()

	t.Run("file with lines", func(t *testing.T) {
		path := filepath.Join(tmp, "test.jsonl")
		if err := os.WriteFile(path, []byte("{\"a\":1}\n{\"b\":2}\n{\"c\":3}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got := countFileLines(path)
		if got != 3 {
			t.Errorf("countFileLines() = %d, want 3", got)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(tmp, "empty.jsonl")
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		got := countFileLines(path)
		if got != 0 {
			t.Errorf("countFileLines() = %d, want 0", got)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		got := countFileLines(filepath.Join(tmp, "nope"))
		if got != 0 {
			t.Errorf("countFileLines(nonexistent) = %d, want 0", got)
		}
	})

	t.Run("blank lines ignored", func(t *testing.T) {
		path := filepath.Join(tmp, "blanks.jsonl")
		if err := os.WriteFile(path, []byte("line1\n\n  \nline2\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got := countFileLines(path)
		if got != 2 {
			t.Errorf("countFileLines() = %d, want 2", got)
		}
	})
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1,000"},
		{1247, "1,247"},
		{1000000, "1,000,000"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.input), func(t *testing.T) {
			got := formatNumber(tt.input)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"days", 48 * time.Hour, "2d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.input)
			if got != tt.want {
				t.Errorf("formatDuration() = %q, want %q", got, tt.want)
			}
		})
	}
}
