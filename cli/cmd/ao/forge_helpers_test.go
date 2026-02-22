package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boshu2/agentops/cli/internal/storage"
	"github.com/boshu2/agentops/cli/internal/types"
)

func TestExtractSnippet(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog and continues running"

	tests := []struct {
		name     string
		startIdx int
		maxLen   int
		wantNot  string
		check    func(string) bool
		desc     string
	}{
		{
			name:     "from start",
			startIdx: 0,
			maxLen:   15,
			check:    func(s string) bool { return len(s) > 0 && len(s) <= 20 },
			desc:     "should return short snippet with ellipsis",
		},
		{
			name:     "from middle",
			startIdx: 10,
			maxLen:   20,
			check:    func(s string) bool { return len(s) > 0 },
			desc:     "should return snippet from middle",
		},
		{
			name:     "past end",
			startIdx: len(content) + 10,
			maxLen:   20,
			check:    func(s string) bool { return s == "" },
			desc:     "should return empty for past-end index",
		},
		{
			name:     "negative start clamped",
			startIdx: -5,
			maxLen:   10,
			check:    func(s string) bool { return len(s) > 0 },
			desc:     "should clamp negative start to 0",
		},
		{
			name:     "maxLen covers entire content",
			startIdx: 0,
			maxLen:   1000,
			check:    func(s string) bool { return s == content },
			desc:     "should return full content when maxLen exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSnippet(content, tt.startIdx, tt.maxLen)
			if !tt.check(got) {
				t.Errorf("extractSnippet(%d, %d) = %q; %s", tt.startIdx, tt.maxLen, got, tt.desc)
			}
		})
	}
}

func TestLastSpaceIndex(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"with spaces", "hello world", 5},
		{"trailing space", "hello ", 5},
		{"no spaces", "helloworld", -1},
		{"empty", "", -1},
		{"single space", " ", 0},
		{"multiple spaces", "a b c", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastSpaceIndex(tt.s)
			if got != tt.want {
				t.Errorf("lastSpaceIndex(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestExtractIssueIDs(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst string
	}{
		{"single issue", "Fixed ol-0001 in commit", 1, "ol-0001"},
		{"multiple issues", "Closes ol-0001 and at-v123", 2, "ol-0001"},
		{"no issues", "This has no issue references", 0, ""},
		{"compound ID", "See gt-abc-def for details", 1, "gt-abc-def"},
		{"uppercase ignored", "Not OL-0001", 0, ""},
		{"mixed text", "Working on ag-m0r with issues ag-oke and cm-0012", 3, "ag-m0r"},
		{"empty string", "", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIssueIDs(tt.content)
			if len(got) != tt.wantCount {
				t.Errorf("extractIssueIDs(%q) returned %d IDs, want %d; got %v", tt.content, len(got), tt.wantCount, got)
			}
			if tt.wantFirst != "" && len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("first ID = %q, want %q", got[0], tt.wantFirst)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world!", 8, "hello..."},
		{"empty", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestDedup(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{"no duplicates", []string{"a", "b", "c"}, 3},
		{"all duplicates", []string{"a", "a", "a"}, 1},
		{"mixed", []string{"a", "b", "a", "c", "b"}, 3},
		{"empty", []string{}, 0},
		{"single", []string{"x"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedup(tt.input)
			if len(got) != tt.want {
				t.Errorf("dedup() returned %d items, want %d; got %v", len(got), tt.want, got)
			}
			// Verify order preserved
			if len(tt.input) > 0 && len(got) > 0 && got[0] != tt.input[0] {
				t.Errorf("first element = %q, want %q (order should be preserved)", got[0], tt.input[0])
			}
		})
	}
}

func TestSplitMarkdownSections(t *testing.T) {
	t.Run("empty string returns one section", func(t *testing.T) {
		got := splitMarkdownSections("")
		if len(got) != 1 {
			t.Errorf("expected 1 section, got %d", len(got))
		}
	})

	t.Run("no headings returns one section", func(t *testing.T) {
		content := "just some text\nno headings here"
		got := splitMarkdownSections(content)
		if len(got) != 1 {
			t.Errorf("expected 1 section for no-heading content, got %d", len(got))
		}
		if got[0] != content {
			t.Errorf("section = %q, want %q", got[0], content)
		}
	})

	t.Run("h1 heading splits into 2 sections", func(t *testing.T) {
		content := "intro text\n# Heading\ncontent under heading"
		got := splitMarkdownSections(content)
		if len(got) != 2 {
			t.Errorf("expected 2 sections, got %d: %v", len(got), got)
		}
		if !strings.HasPrefix(got[1], "# Heading") {
			t.Errorf("second section should start with heading, got %q", got[1])
		}
	})

	t.Run("h2 heading splits correctly", func(t *testing.T) {
		content := "intro\n## Section A\ncontent A\n## Section B\ncontent B"
		got := splitMarkdownSections(content)
		if len(got) != 3 {
			t.Errorf("expected 3 sections, got %d", len(got))
		}
	})

	t.Run("content starting with heading has one section per heading", func(t *testing.T) {
		content := "# First\nstuff\n# Second\nmore stuff"
		got := splitMarkdownSections(content)
		if len(got) != 2 {
			t.Errorf("expected 2 sections, got %d: %v", len(got), got)
		}
	})

	t.Run("h3 not treated as heading boundary", func(t *testing.T) {
		content := "intro\n### Not a split point\nstill in intro"
		got := splitMarkdownSections(content)
		if len(got) != 1 {
			t.Errorf("expected 1 section for h3-only content, got %d", len(got))
		}
	})
}

func TestGenerateSummary(t *testing.T) {
	t.Run("uses first decision when available", func(t *testing.T) {
		decisions := []string{"First decision", "Second decision"}
		got := generateSummary(decisions, nil, time.Now())
		if !strings.Contains(got, "First decision") {
			t.Errorf("expected summary to contain first decision, got %q", got)
		}
	})

	t.Run("falls back to knowledge when no decisions", func(t *testing.T) {
		knowledge := []string{"Key learning"}
		got := generateSummary(nil, knowledge, time.Now())
		if !strings.Contains(got, "Key learning") {
			t.Errorf("expected summary to contain knowledge, got %q", got)
		}
	})

	t.Run("uses date when no decisions or knowledge", func(t *testing.T) {
		refDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		got := generateSummary(nil, nil, refDate)
		if !strings.Contains(got, "2024-03-15") {
			t.Errorf("expected summary to contain date, got %q", got)
		}
	})

	t.Run("truncates long decision to SummaryMaxLength", func(t *testing.T) {
		longDecision := strings.Repeat("x", SummaryMaxLength+50)
		got := generateSummary([]string{longDecision}, nil, time.Now())
		if len(got) > SummaryMaxLength {
			t.Errorf("summary length %d exceeds max %d", len(got), SummaryMaxLength)
		}
	})
}

func TestCountLines(t *testing.T) {
	t.Run("returns 0 for nonexistent file", func(t *testing.T) {
		got := countLines("/tmp/definitely-does-not-exist-12345.txt")
		if got != 0 {
			t.Errorf("expected 0 for nonexistent file, got %d", got)
		}
	})

	t.Run("counts newlines in temp file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		content := "line1\nline2\nline3\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got := countLines(path)
		if got != 3 {
			t.Errorf("expected 3 newlines, got %d", got)
		}
	})

	t.Run("empty file has 0 lines", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.txt")
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		got := countLines(path)
		if got != 0 {
			t.Errorf("expected 0 for empty file, got %d", got)
		}
	})
}

func TestNoFilesError(t *testing.T) {
	t.Run("quiet mode returns nil", func(t *testing.T) {
		err := noFilesError(true, "no files found")
		if err != nil {
			t.Errorf("expected nil for quiet mode, got %v", err)
		}
	})

	t.Run("non-quiet returns error with message", func(t *testing.T) {
		err := noFilesError(false, "no files found")
		if err == nil {
			t.Error("expected error for non-quiet mode")
		}
		if err.Error() != "no files found" {
			t.Errorf("expected %q, got %q", "no files found", err.Error())
		}
	})
}

func TestForgeWarnf(t *testing.T) {
	// forgeWarnf should not panic regardless of quiet flag
	forgeWarnf(true, "should not print: %s", "test")
	forgeWarnf(false, "test warning (goes to stderr): %s\n", "msg")
}

func TestCollectFilesFromPatterns(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "test1.txt")
	file2 := filepath.Join(dir, "test2.txt")
	for _, f := range []string{file1, file2} {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("glob pattern matches files", func(t *testing.T) {
		pattern := filepath.Join(dir, "*.txt")
		files, err := collectFilesFromPatterns([]string{pattern}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d: %v", len(files), files)
		}
	})

	t.Run("literal path included directly", func(t *testing.T) {
		files, err := collectFilesFromPatterns([]string{file1}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("expected 1 file, got %d", len(files))
		}
	})

	t.Run("non-existent literal path excluded", func(t *testing.T) {
		files, err := collectFilesFromPatterns([]string{"/no/such/file.txt"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("expected 0 files for non-existent path, got %d", len(files))
		}
	})

	t.Run("matchFilter filters files", func(t *testing.T) {
		pattern := filepath.Join(dir, "*.txt")
		filter := func(s string) bool { return strings.Contains(s, "test1") }
		files, err := collectFilesFromPatterns([]string{pattern}, filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("expected 1 filtered file, got %d: %v", len(files), files)
		}
	})

	t.Run("empty patterns returns empty", func(t *testing.T) {
		files, err := collectFilesFromPatterns(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})
}

func TestCloneHooksMap(t *testing.T) {
	t.Run("clones hooks sub-map from rawSettings", func(t *testing.T) {
		rawSettings := map[string]any{
			"hooks": map[string]any{
				"SessionStart": []any{"cmd1"},
				"PreToolUse":   []any{"cmd2"},
			},
		}
		got := cloneHooksMap(rawSettings)
		if len(got) != 2 {
			t.Errorf("expected 2 entries, got %d: %v", len(got), got)
		}
		if _, ok := got["SessionStart"]; !ok {
			t.Error("expected SessionStart key in clone")
		}
	})

	t.Run("no hooks key returns empty map", func(t *testing.T) {
		rawSettings := map[string]any{"other": "value"}
		got := cloneHooksMap(rawSettings)
		if len(got) != 0 {
			t.Errorf("expected empty map, got %d entries", len(got))
		}
	})

	t.Run("empty rawSettings returns empty map", func(t *testing.T) {
		got := cloneHooksMap(map[string]any{})
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("hooks key with wrong type returns empty map", func(t *testing.T) {
		rawSettings := map[string]any{"hooks": "not-a-map"}
		got := cloneHooksMap(rawSettings)
		if len(got) != 0 {
			t.Errorf("expected empty map for wrong type, got %d entries", len(got))
		}
	})
}

func TestExtractFilePathsFromTool(t *testing.T) {
	t.Run("extracts file_path from tool input", func(t *testing.T) {
		tool := types.ToolCall{
			Name: "Read",
			Input: map[string]any{
				"file_path": "/some/path/file.go",
			},
		}
		state := &transcriptState{seenFiles: make(map[string]bool)}
		extractFilePathsFromTool(tool, state)
		if len(state.filesChanged) != 1 || state.filesChanged[0] != "/some/path/file.go" {
			t.Errorf("expected filesChanged=[/some/path/file.go], got %v", state.filesChanged)
		}
	})

	t.Run("extracts path from tool input", func(t *testing.T) {
		tool := types.ToolCall{
			Name: "Bash",
			Input: map[string]any{
				"path": "/another/path.sh",
			},
		}
		state := &transcriptState{seenFiles: make(map[string]bool)}
		extractFilePathsFromTool(tool, state)
		if len(state.filesChanged) != 1 || state.filesChanged[0] != "/another/path.sh" {
			t.Errorf("expected filesChanged=[/another/path.sh], got %v", state.filesChanged)
		}
	})

	t.Run("deduplicates seen files", func(t *testing.T) {
		state := &transcriptState{seenFiles: make(map[string]bool)}
		tool := types.ToolCall{
			Name:  "Read",
			Input: map[string]any{"file_path": "/dup.go"},
		}
		extractFilePathsFromTool(tool, state)
		extractFilePathsFromTool(tool, state)
		if len(state.filesChanged) != 1 {
			t.Errorf("expected 1 file after dedup, got %d", len(state.filesChanged))
		}
	})

	t.Run("nil input is skipped", func(t *testing.T) {
		tool := types.ToolCall{Name: "Bash", Input: nil}
		state := &transcriptState{seenFiles: make(map[string]bool)}
		extractFilePathsFromTool(tool, state)
		if len(state.filesChanged) != 0 {
			t.Errorf("expected no files for nil input, got %v", state.filesChanged)
		}
	})
}

func TestExtractIssueRefs(t *testing.T) {
	t.Run("extracts issue IDs from content", func(t *testing.T) {
		state := &transcriptState{seenIssues: make(map[string]bool)}
		extractIssueRefs("Fixed ol-0001 and ag-m0r in this session", state)
		if len(state.issues) != 2 {
			t.Errorf("expected 2 issues, got %v", state.issues)
		}
	})

	t.Run("deduplicates repeated issue IDs", func(t *testing.T) {
		state := &transcriptState{seenIssues: make(map[string]bool)}
		extractIssueRefs("ol-0001 and ol-0001 again", state)
		if len(state.issues) != 1 {
			t.Errorf("expected 1 issue after dedup, got %v", state.issues)
		}
	})

	t.Run("no issues in content leaves state empty", func(t *testing.T) {
		state := &transcriptState{seenIssues: make(map[string]bool)}
		extractIssueRefs("no issue refs here", state)
		if len(state.issues) != 0 {
			t.Errorf("expected 0 issues, got %v", state.issues)
		}
	})
}

func TestExtractToolRefs(t *testing.T) {
	t.Run("counts tool calls in session", func(t *testing.T) {
		session := &storage.Session{ToolCalls: make(map[string]int)}
		state := &transcriptState{seenFiles: make(map[string]bool)}
		tools := []types.ToolCall{
			{Name: "Read", Input: map[string]any{"file_path": "/a.go"}},
			{Name: "Bash", Input: nil},
			{Name: "Read", Input: map[string]any{"file_path": "/b.go"}},
		}
		extractToolRefs(tools, session, state)
		if session.ToolCalls["Read"] != 2 {
			t.Errorf("expected Read count=2, got %d", session.ToolCalls["Read"])
		}
		if session.ToolCalls["Bash"] != 1 {
			t.Errorf("expected Bash count=1, got %d", session.ToolCalls["Bash"])
		}
		if len(state.filesChanged) != 2 {
			t.Errorf("expected 2 files, got %v", state.filesChanged)
		}
	})

	t.Run("tool_result name is not counted", func(t *testing.T) {
		session := &storage.Session{ToolCalls: make(map[string]int)}
		state := &transcriptState{seenFiles: make(map[string]bool)}
		tools := []types.ToolCall{{Name: "tool_result"}}
		extractToolRefs(tools, session, state)
		if _, ok := session.ToolCalls["tool_result"]; ok {
			t.Error("tool_result should not be counted")
		}
	})
}

func TestUpdateSessionMeta(t *testing.T) {
	t.Run("sets session ID from first message with ID", func(t *testing.T) {
		session := &storage.Session{}
		msg := types.TranscriptMessage{SessionID: "abc-123"}
		updateSessionMeta(session, msg)
		if session.ID != "abc-123" {
			t.Errorf("expected ID=abc-123, got %q", session.ID)
		}
	})

	t.Run("does not overwrite existing session ID", func(t *testing.T) {
		session := &storage.Session{ID: "original-id"}
		msg := types.TranscriptMessage{SessionID: "new-id"}
		updateSessionMeta(session, msg)
		if session.ID != "original-id" {
			t.Errorf("expected ID=original-id, got %q", session.ID)
		}
	})

	t.Run("sets date from first message timestamp", func(t *testing.T) {
		session := &storage.Session{}
		ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		msg := types.TranscriptMessage{Timestamp: ts}
		updateSessionMeta(session, msg)
		if !session.Date.Equal(ts) {
			t.Errorf("expected date=%v, got %v", ts, session.Date)
		}
	})

	t.Run("updates date to earlier timestamp", func(t *testing.T) {
		later := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		earlier := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		session := &storage.Session{Date: later}
		msg := types.TranscriptMessage{Timestamp: earlier}
		updateSessionMeta(session, msg)
		if !session.Date.Equal(earlier) {
			t.Errorf("expected date updated to earlier=%v, got %v", earlier, session.Date)
		}
	})
}
