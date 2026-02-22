package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCanonicalSessionID is in feedback_loop_test.go

func TestFindAgentsSubdir(t *testing.T) {
	// Build a mock rig structure
	tmpDir := t.TempDir()

	// Create rig root with marker
	rigRoot := filepath.Join(tmpDir, "myrig")
	if err := os.MkdirAll(filepath.Join(rigRoot, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create .agents/learnings at rig root
	learningsDir := filepath.Join(rigRoot, ".agents", "learnings")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a nested work dir
	workDir := filepath.Join(rigRoot, "crew", "worker")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("finds subdir walking up", func(t *testing.T) {
		got := findAgentsSubdir(workDir, "learnings")
		if got != learningsDir {
			t.Errorf("findAgentsSubdir() = %q, want %q", got, learningsDir)
		}
	})

	t.Run("returns empty when not found", func(t *testing.T) {
		got := findAgentsSubdir(workDir, "nonexistent")
		if got != "" {
			t.Errorf("findAgentsSubdir() = %q, want empty", got)
		}
	})

	t.Run("stops at rig root", func(t *testing.T) {
		// Create agents dir above rig root - should not be found
		if err := os.MkdirAll(filepath.Join(tmpDir, ".agents", "patterns"), 0755); err != nil {
			t.Fatal(err)
		}
		got := findAgentsSubdir(workDir, "patterns")
		if got != "" {
			t.Errorf("findAgentsSubdir() = %q, want empty (should stop at rig root)", got)
		}
	})

	t.Run("finds in current dir", func(t *testing.T) {
		got := findAgentsSubdir(rigRoot, "learnings")
		if got != learningsDir {
			t.Errorf("findAgentsSubdir() = %q, want %q", got, learningsDir)
		}
	})
}

func TestFormatKnowledgeMarkdown(t *testing.T) {
	t.Run("empty knowledge", func(t *testing.T) {
		k := &injectedKnowledge{
			Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		}
		got := formatKnowledgeMarkdown(k)
		if !strings.Contains(got, "No prior knowledge found") {
			t.Error("expected 'No prior knowledge found' message for empty knowledge")
		}
		if !strings.Contains(got, "Injected Knowledge") {
			t.Error("expected header")
		}
	})

	t.Run("with learnings", func(t *testing.T) {
		k := &injectedKnowledge{
			Learnings: []learning{
				{ID: "L1", Title: "Auth Pattern", Summary: "Use middleware"},
				{ID: "L2", Title: "No Summary"},
			},
			Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		}
		got := formatKnowledgeMarkdown(k)
		if !strings.Contains(got, "Recent Learnings") {
			t.Error("expected 'Recent Learnings' section")
		}
		if !strings.Contains(got, "L1") {
			t.Error("expected L1 ID in output")
		}
		if !strings.Contains(got, "Use middleware") {
			t.Error("expected summary in output")
		}
		// L2 has no summary, should use title
		if !strings.Contains(got, "No Summary") {
			t.Error("expected title fallback for L2")
		}
	})

	t.Run("with patterns", func(t *testing.T) {
		k := &injectedKnowledge{
			Patterns: []pattern{
				{Name: "Mutex", Description: "Use sync.Mutex for shared state"},
				{Name: "NoDesc"},
			},
			Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		}
		got := formatKnowledgeMarkdown(k)
		if !strings.Contains(got, "Active Patterns") {
			t.Error("expected 'Active Patterns' section")
		}
		if !strings.Contains(got, "Mutex") {
			t.Error("expected pattern name")
		}
	})

	t.Run("with sessions", func(t *testing.T) {
		k := &injectedKnowledge{
			Sessions: []session{
				{Date: "2026-02-10", Summary: "Worked on auth"},
			},
			Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		}
		got := formatKnowledgeMarkdown(k)
		if !strings.Contains(got, "Recent Sessions") {
			t.Error("expected 'Recent Sessions' section")
		}
	})

	t.Run("with OL constraints", func(t *testing.T) {
		k := &injectedKnowledge{
			OLConstraints: []olConstraint{
				{Pattern: "no-eval", Detection: "eval() usage found"},
			},
			Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		}
		got := formatKnowledgeMarkdown(k)
		if !strings.Contains(got, "Olympus Constraints") {
			t.Error("expected 'Olympus Constraints' section")
		}
	})
}

func TestTrimToCharBudget(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		budget int
		check  func(string) bool
		desc   string
	}{
		{
			name:   "under budget passes through",
			input:  "short",
			budget: 100,
			check:  func(s string) bool { return s == "short" },
			desc:   "should return unchanged",
		},
		{
			name:   "over budget truncated",
			input:  "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10",
			budget: 30,
			check: func(s string) bool {
				return len(s) <= 80 && strings.Contains(s, "truncated")
			},
			desc: "should be truncated with marker",
		},
		{
			name:   "exact budget",
			input:  "exact",
			budget: 5,
			check:  func(s string) bool { return s == "exact" },
			desc:   "should return unchanged at exact budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimToCharBudget(tt.input, tt.budget)
			if !tt.check(got) {
				t.Errorf("trimToCharBudget(): %s; got %q", tt.desc, got)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"empty", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateText(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestCollectOLConstraints(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("no .ol directory returns nil", func(t *testing.T) {
		got, err := collectOLConstraints(tmpDir, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("no quarantine.json returns nil", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(tmpDir, ".ol", "constraints"), 0755); err != nil {
			t.Fatal(err)
		}
		got, err := collectOLConstraints(tmpDir, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	// Create quarantine.json
	constraints := []olConstraint{
		{Pattern: "no-eval", Detection: "eval() usage found in hooks"},
		{Pattern: "no-force-push", Detection: "git push --force detected"},
	}
	data, _ := json.Marshal(constraints)
	quarantinePath := filepath.Join(tmpDir, ".ol", "constraints", "quarantine.json")
	if err := os.MkdirAll(filepath.Dir(quarantinePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quarantinePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("loads all constraints without query", func(t *testing.T) {
		got, err := collectOLConstraints(tmpDir, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("got %d constraints, want 2", len(got))
		}
	})

	t.Run("filters by query", func(t *testing.T) {
		got, err := collectOLConstraints(tmpDir, "eval")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d constraints, want 1", len(got))
		}
		if len(got) > 0 && got[0].Pattern != "no-eval" {
			t.Errorf("got pattern %q, want %q", got[0].Pattern, "no-eval")
		}
	})

	t.Run("query no match", func(t *testing.T) {
		got, err := collectOLConstraints(tmpDir, "kubernetes")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d constraints, want 0", len(got))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		badDir := t.TempDir()
		badPath := filepath.Join(badDir, ".ol", "constraints", "quarantine.json")
		if err := os.MkdirAll(filepath.Dir(badPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(badPath, []byte("not json"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := collectOLConstraints(badDir, "")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestCollectLearnings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .agents/learnings/ directory
	learningsDir := filepath.Join(tmpDir, ".agents", "learnings")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create markdown learning
	mdContent := `# Mutex Pattern

Always use sync.Mutex for shared state access.
`
	if err := os.WriteFile(filepath.Join(learningsDir, "mutex.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create JSONL learning
	jsonlData := map[string]interface{}{
		"id":      "L2",
		"title":   "Database Pooling",
		"summary": "Use connection pooling for database access",
		"utility": 0.8,
	}
	line, _ := json.Marshal(jsonlData)
	if err := os.WriteFile(filepath.Join(learningsDir, "db.jsonl"), line, 0644); err != nil {
		t.Fatal(err)
	}

	// Create superseded learning
	supersededContent := `---
superseded_by: L99
---
# Old Learning
`
	if err := os.WriteFile(filepath.Join(learningsDir, "old.md"), []byte(supersededContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("collects non-superseded learnings", func(t *testing.T) {
		got, err := collectLearnings(tmpDir, "", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should get 2 (mutex.md + db.jsonl), not 3 (old.md is superseded)
		if len(got) != 2 {
			t.Errorf("got %d learnings, want 2 (superseded should be filtered)", len(got))
			for _, l := range got {
				t.Logf("  %s: %s (superseded=%v)", l.ID, l.Title, l.Superseded)
			}
		}
	})

	t.Run("filters by query", func(t *testing.T) {
		got, err := collectLearnings(tmpDir, "mutex", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d learnings for 'mutex', want 1", len(got))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		got, err := collectLearnings(tmpDir, "", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) > 1 {
			t.Errorf("got %d learnings, want at most 1", len(got))
		}
	})

	t.Run("no learnings directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		got, err := collectLearnings(emptyDir, "", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("learnings sorted by composite score", func(t *testing.T) {
		got, err := collectLearnings(tmpDir, "", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) >= 2 {
			// Verify sorted descending by composite score
			if got[0].CompositeScore < got[1].CompositeScore {
				t.Errorf("learnings not sorted by composite score: first=%.4f < second=%.4f",
					got[0].CompositeScore, got[1].CompositeScore)
			}
		}
	})
}

func TestParseLearningJSONL(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("basic JSONL", func(t *testing.T) {
		data := map[string]interface{}{
			"id":      "L42",
			"title":   "Test Learning",
			"summary": "A test summary",
			"utility": 0.7,
		}
		line, _ := json.Marshal(data)
		path := filepath.Join(tmpDir, "basic.jsonl")
		if err := os.WriteFile(path, line, 0644); err != nil {
			t.Fatal(err)
		}

		l, err := parseLearningJSONL(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.ID != "L42" {
			t.Errorf("ID = %q, want %q", l.ID, "L42")
		}
		if l.Title != "Test Learning" {
			t.Errorf("Title = %q, want %q", l.Title, "Test Learning")
		}
		if l.Utility != 0.7 {
			t.Errorf("Utility = %f, want 0.7", l.Utility)
		}
	})

	t.Run("superseded JSONL", func(t *testing.T) {
		data := map[string]interface{}{
			"id":            "L1",
			"title":         "Old",
			"superseded_by": "L99",
		}
		line, _ := json.Marshal(data)
		path := filepath.Join(tmpDir, "superseded.jsonl")
		if err := os.WriteFile(path, line, 0644); err != nil {
			t.Fatal(err)
		}

		l, err := parseLearningJSONL(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !l.Superseded {
			t.Error("expected Superseded = true")
		}
	})

	t.Run("content fallback for summary", func(t *testing.T) {
		data := map[string]interface{}{
			"id":      "L3",
			"title":   "Title Only",
			"content": "This is the content used as summary",
		}
		line, _ := json.Marshal(data)
		path := filepath.Join(tmpDir, "content.jsonl")
		if err := os.WriteFile(path, line, 0644); err != nil {
			t.Fatal(err)
		}

		l, err := parseLearningJSONL(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.Summary == "" {
			t.Error("expected summary from content field")
		}
	})

	t.Run("default utility", func(t *testing.T) {
		data := map[string]interface{}{
			"id":    "L4",
			"title": "No Utility",
		}
		line, _ := json.Marshal(data)
		path := filepath.Join(tmpDir, "no-utility.jsonl")
		if err := os.WriteFile(path, line, 0644); err != nil {
			t.Fatal(err)
		}

		l, err := parseLearningJSONL(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.Utility != 0.5 {
			t.Errorf("Utility = %f, want 0.5 (InitialUtility)", l.Utility)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := parseLearningJSONL(filepath.Join(tmpDir, "nope.jsonl"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "bad.jsonl")
		if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
			t.Fatal(err)
		}

		l, err := parseLearningJSONL(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Invalid JSON should return default learning with ID from filename
		if l.ID != "bad.jsonl" {
			t.Errorf("ID = %q, want %q", l.ID, "bad.jsonl")
		}
	})
}
