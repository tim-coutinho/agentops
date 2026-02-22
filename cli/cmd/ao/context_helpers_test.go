package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tests for pure helper functions in context.go

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "empty string returns zero time",
			input: "",
			want:  time.Time{},
		},
		{
			name:  "whitespace only returns zero time",
			input: "   ",
			want:  time.Time{},
		},
		{
			name:  "valid RFC3339 timestamp",
			input: "2026-01-15T10:30:00Z",
			want:  time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "valid RFC3339Nano timestamp",
			input: "2026-01-15T10:30:00.123456789Z",
			want:  time.Date(2026, 1, 15, 10, 30, 0, 123456789, time.UTC),
		},
		{
			name:  "invalid string returns zero time",
			input: "not-a-timestamp",
			want:  time.Time{},
		},
		{
			name:  "RFC3339 with timezone offset",
			input: "2026-01-15T10:30:00+05:00",
			want:  time.Date(2026, 1, 15, 5, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimestamp(tt.input)
			if !got.Equal(tt.want) {
				t.Errorf("parseTimestamp(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty raw message",
			input: "",
			want:  "",
		},
		{
			name:  "plain string JSON",
			input: `"hello world"`,
			want:  "hello world",
		},
		{
			name:  "array with text field",
			input: `[{"type":"text","text":"hello from array"}]`,
			want:  "hello from array",
		},
		{
			name:  "array with empty text fields skips to next",
			input: `[{"type":"text","text":""},{"type":"text","text":"second"}]`,
			want:  "second",
		},
		{
			name:  "invalid JSON returns empty",
			input: `{invalid}`,
			want:  "",
		},
		{
			name:  "array with no text field returns empty",
			input: `[{"type":"image"}]`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextContent(json.RawMessage(tt.input))
			if got != tt.want {
				t.Errorf("extractTextContent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		wantN int
	}{
		{
			name:  "empty string returns 0",
			input: "",
			wantN: 0,
		},
		{
			name:  "whitespace only returns 0",
			input: "   ",
			wantN: 0,
		},
		{
			name:  "short text returns at least 1",
			input: "hi",
			wantN: 1,
		},
		{
			name:  "40 chars returns 10 tokens",
			input: "1234567890123456789012345678901234567890",
			wantN: 10,
		},
		{
			name:  "4 chars returns 1 token",
			input: "abcd",
			wantN: 1,
		},
		{
			name:  "8 chars returns 2 tokens",
			input: "abcdefgh",
			wantN: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokens(tt.input)
			if got != tt.wantN {
				t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, got, tt.wantN)
			}
		})
	}
}

func TestInferAgentRole(t *testing.T) {
	tests := []struct {
		name         string
		agentName    string
		explicitRole string
		want         string
	}{
		{
			name:         "explicit role takes priority",
			agentName:    "worker",
			explicitRole: "reviewer",
			want:         "reviewer",
		},
		{
			name:         "empty agent name returns empty",
			agentName:    "",
			explicitRole: "",
			want:         "",
		},
		{
			name:         "admiral maps to team-lead",
			agentName:    "admiral-01",
			explicitRole: "",
			want:         "team-lead",
		},
		{
			name:         "captain maps to team-lead",
			agentName:    "captain-crunch",
			explicitRole: "",
			want:         "team-lead",
		},
		{
			name:         "lead maps to team-lead",
			agentName:    "team-lead",
			explicitRole: "",
			want:         "team-lead",
		},
		{
			name:         "leader maps to team-lead",
			agentName:    "project-leader",
			explicitRole: "",
			want:         "team-lead",
		},
		{
			name:         "mayor maps to team-lead",
			agentName:    "mayor",
			explicitRole: "",
			want:         "team-lead",
		},
		{
			name:         "reviewer maps to review",
			agentName:    "reviewer-01",
			explicitRole: "",
			want:         "review",
		},
		{
			name:         "judge maps to review",
			agentName:    "judge",
			explicitRole: "",
			want:         "review",
		},
		{
			name:         "worker maps to worker",
			agentName:    "worker-01",
			explicitRole: "",
			want:         "worker",
		},
		{
			name:         "crew maps to worker",
			agentName:    "crew-member",
			explicitRole: "",
			want:         "worker",
		},
		{
			name:         "unknown maps to agent",
			agentName:    "something-random",
			explicitRole: "",
			want:         "agent",
		},
		{
			name:         "explicit role whitespace trimmed",
			agentName:    "worker",
			explicitRole: "  custom-role  ",
			want:         "custom-role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferAgentRole(tt.agentName, tt.explicitRole)
			if got != tt.want {
				t.Errorf("inferAgentRole(%q, %q) = %q, want %q", tt.agentName, tt.explicitRole, got, tt.want)
			}
		})
	}
}

func TestRemainingPercent(t *testing.T) {
	tests := []struct {
		name         string
		usagePercent float64
		want         float64
	}{
		{name: "0% usage = 1.0 remaining", usagePercent: 0.0, want: 1.0},
		{name: "0.5 usage = 0.5 remaining", usagePercent: 0.5, want: 0.5},
		{name: "1.0 usage = 0.0 remaining", usagePercent: 1.0, want: 0.0},
		{name: "over 1.0 clamps to 0", usagePercent: 1.5, want: 0.0},
		{name: "negative usage clamps to 1", usagePercent: -0.1, want: 1.0},
		{name: "0.25 usage = 0.75 remaining", usagePercent: 0.25, want: 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := remainingPercent(tt.usagePercent)
			if got != tt.want {
				t.Errorf("remainingPercent(%v) = %v, want %v", tt.usagePercent, got, tt.want)
			}
		})
	}
}

func TestExtractIssueID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no issue ID returns empty", input: "no issue here", want: ""},
		{name: "ag- prefix extracted", input: "working on ag-123", want: "ag-123"},
		{name: "uppercase AG- extracted and lowercased", input: "AG-abc", want: "ag-abc"},
		{name: "mixed case normalized", input: "Ag-XYZ", want: "ag-xyz"},
		{name: "empty string returns empty", input: "", want: ""},
		{name: "longer ID", input: "task ag-abc123 complete", want: "ag-abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIssueID(tt.input)
			if got != tt.want {
				t.Errorf("extractIssueID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTmuxTargetFromPaneID(t *testing.T) {
	tests := []struct {
		name   string
		paneID string
		want   string
	}{
		{name: "empty pane ID returns empty", paneID: "", want: ""},
		{name: "in-process returns empty", paneID: "in-process", want: ""},
		{name: "pane with dot returns session:window", paneID: "mysession:0.1", want: "mysession:0"},
		{name: "pane without dot returns as-is", paneID: "mysession", want: "mysession"},
		{name: "whitespace trimmed", paneID: "  mysession:0.1  ", want: "mysession:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tmuxTargetFromPaneID(tt.paneID)
			if got != tt.want {
				t.Errorf("tmuxTargetFromPaneID(%q) = %q, want %q", tt.paneID, got, tt.want)
			}
		})
	}
}

func TestTmuxSessionFromTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "empty target returns empty", target: "", want: ""},
		{name: "session:window returns session", target: "mysession:0", want: "mysession"},
		{name: "no colon returns target as-is", target: "mysession", want: "mysession"},
		{name: "whitespace trimmed", target: "  mysession:1  ", want: "mysession"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tmuxSessionFromTarget(tt.target)
			if got != tt.want {
				t.Errorf("tmuxSessionFromTarget(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestNonZeroOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		fallback int
		want     int
	}{
		{name: "positive value returned", value: 5, fallback: 10, want: 5},
		{name: "zero value returns fallback", value: 0, fallback: 10, want: 10},
		{name: "negative value returns fallback", value: -1, fallback: 10, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nonZeroOrDefault(tt.value, tt.fallback)
			if got != tt.want {
				t.Errorf("nonZeroOrDefault(%d, %d) = %d, want %d", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestTruncateDisplay(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		max   int
		want  string
	}{
		{name: "short string unchanged", s: "hello", max: 10, want: "hello"},
		{name: "exact length unchanged", s: "hello", max: 5, want: "hello"},
		{name: "truncated with ellipsis", s: "hello world", max: 8, want: "hello..."},
		{name: "max<=3 truncates without ellipsis", s: "hello", max: 3, want: "hel"},
		{name: "max=0 is empty", s: "hello", max: 0, want: ""},
		{name: "empty string unchanged", s: "", max: 10, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateDisplay(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncateDisplay(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestToRepoRelative(t *testing.T) {
	t.Run("relative path computed", func(t *testing.T) {
		cwd := "/home/user/project"
		full := "/home/user/project/src/main.go"
		got := toRepoRelative(cwd, full)
		if got != "src/main.go" {
			t.Errorf("toRepoRelative(%q, %q) = %q, want %q", cwd, full, got, "src/main.go")
		}
	})
	t.Run("empty path returns empty", func(t *testing.T) {
		got := toRepoRelative("/cwd", "")
		if got != "" {
			t.Errorf("toRepoRelative with empty path = %q, want \"\"", got)
		}
	})
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple string unchanged", input: "hello", want: "hello"},
		{name: "spaces replaced with dashes", input: "hello world", want: "hello-world"},
		{name: "special chars replaced", input: "abc/def:ghi", want: "abc-def-ghi"},
		{name: "leading/trailing dashes trimmed", input: "/hello/", want: "hello"},
		{name: "empty string returns session", input: "", want: "session"},
		{name: "all special chars returns session", input: "!!!", want: "session"},
		{name: "alphanumeric preserved", input: "abc-123_def.txt", want: "abc-123_def.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeForFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeForFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestContextAssignmentIsEmpty(t *testing.T) {
	t.Run("empty assignment returns true", func(t *testing.T) {
		a := contextAssignment{}
		if !a.isEmpty() {
			t.Error("empty contextAssignment.isEmpty() should return true")
		}
	})

	t.Run("whitespace-only fields returns true", func(t *testing.T) {
		a := contextAssignment{AgentName: "  ", AgentRole: "\t"}
		if !a.isEmpty() {
			t.Error("whitespace-only contextAssignment.isEmpty() should return true")
		}
	})

	t.Run("non-empty agent name returns false", func(t *testing.T) {
		a := contextAssignment{AgentName: "worker-01"}
		if a.isEmpty() {
			t.Error("contextAssignment with AgentName should not be empty")
		}
	})

	t.Run("non-empty issue ID returns false", func(t *testing.T) {
		a := contextAssignment{IssueID: "ag-123"}
		if a.isEmpty() {
			t.Error("contextAssignment with IssueID should not be empty")
		}
	})
}

func TestAssignmentFromStatus(t *testing.T) {
	status := contextSessionStatus{
		AgentName:   "  worker-01  ",
		AgentRole:   "  worker  ",
		TeamName:    "  team-a  ",
		IssueID:     "  ag-123  ",
		TmuxPaneID:  "  pane-1  ",
		TmuxTarget:  "  target-1  ",
		TmuxSession: "  session-1  ",
	}
	got := assignmentFromStatus(status)
	if got.AgentName != "worker-01" {
		t.Errorf("AgentName = %q, want %q", got.AgentName, "worker-01")
	}
	if got.IssueID != "ag-123" {
		t.Errorf("IssueID = %q, want %q", got.IssueID, "ag-123")
	}
	if got.TmuxSession != "session-1" {
		t.Errorf("TmuxSession = %q, want %q", got.TmuxSession, "session-1")
	}
}

func TestReadPersistedAssignment(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file returns not found", func(t *testing.T) {
		_, ok := readPersistedAssignment(dir, "nonexistent-session")
		if ok {
			t.Error("expected readPersistedAssignment to return false for missing file")
		}
	})

	t.Run("valid assignment file returned", func(t *testing.T) {
		contextDir := filepath.Join(dir, ".agents", "ao", "context")
		if err := os.MkdirAll(contextDir, 0755); err != nil {
			t.Fatal(err)
		}
		snapshot := contextAssignmentSnapshot{
			AgentName: "worker-01",
			AgentRole: "worker",
			IssueID:   "ag-456",
			UpdatedAt: "2026-01-01T00:00:00Z",
		}
		data, _ := json.Marshal(snapshot)
		sessionID := "test-session-123"
		filename := "assignment-" + sanitizeForFilename(sessionID) + ".json"
		if err := os.WriteFile(filepath.Join(contextDir, filename), data, 0644); err != nil {
			t.Fatal(err)
		}
		got, ok := readPersistedAssignment(dir, sessionID)
		if !ok {
			t.Fatal("expected readPersistedAssignment to return true for existing file")
		}
		if got.AgentName != "worker-01" {
			t.Errorf("AgentName = %q, want %q", got.AgentName, "worker-01")
		}
		if got.IssueID != "ag-456" {
			t.Errorf("IssueID = %q, want %q", got.IssueID, "ag-456")
		}
	})

	t.Run("empty assignment returns not found", func(t *testing.T) {
		contextDir := filepath.Join(dir, ".agents", "ao", "context")
		snapshot := contextAssignmentSnapshot{} // all empty
		data, _ := json.Marshal(snapshot)
		sessionID := "empty-session"
		filename := "assignment-" + sanitizeForFilename(sessionID) + ".json"
		os.WriteFile(filepath.Join(contextDir, filename), data, 0644)
		_, ok := readPersistedAssignment(dir, sessionID)
		if ok {
			t.Error("expected readPersistedAssignment to return false for empty snapshot")
		}
	})

	t.Run("invalid JSON returns not found", func(t *testing.T) {
		contextDir := filepath.Join(dir, ".agents", "ao", "context")
		sessionID := "bad-json-session"
		filename := "assignment-" + sanitizeForFilename(sessionID) + ".json"
		os.WriteFile(filepath.Join(contextDir, filename), []byte("{invalid json}"), 0644)
		_, ok := readPersistedAssignment(dir, sessionID)
		if ok {
			t.Error("expected readPersistedAssignment to return false for invalid JSON")
		}
	})
}

func TestMergePersistedAssignment(t *testing.T) {
	dir := t.TempDir()

	t.Run("nil status is no-op", func(t *testing.T) {
		// Should not panic
		mergePersistedAssignment(dir, nil)
	})

	t.Run("empty session ID is no-op", func(t *testing.T) {
		status := &contextSessionStatus{}
		mergePersistedAssignment(dir, status)
		// Should not panic or modify
	})

	t.Run("merges persisted agent name into empty field", func(t *testing.T) {
		contextDir := filepath.Join(dir, ".agents", "ao", "context")
		os.MkdirAll(contextDir, 0755)

		sessionID := "merge-test-session"
		snapshot := contextAssignmentSnapshot{
			AgentName: "persisted-worker",
			IssueID:   "ag-789",
		}
		data, _ := json.Marshal(snapshot)
		filename := "assignment-" + sanitizeForFilename(sessionID) + ".json"
		os.WriteFile(filepath.Join(contextDir, filename), data, 0644)

		status := &contextSessionStatus{
			SessionID: sessionID,
			AgentName: "", // empty - should be merged from persisted
		}
		mergePersistedAssignment(dir, status)
		if status.AgentName != "persisted-worker" {
			t.Errorf("AgentName = %q, want %q", status.AgentName, "persisted-worker")
		}
		if status.IssueID != "ag-789" {
			t.Errorf("IssueID = %q, want %q", status.IssueID, "ag-789")
		}
	})

	t.Run("existing field not overwritten by persisted", func(t *testing.T) {
		contextDir := filepath.Join(dir, ".agents", "ao", "context")
		sessionID := "no-overwrite-session"
		snapshot := contextAssignmentSnapshot{
			AgentName: "persisted-worker",
		}
		data, _ := json.Marshal(snapshot)
		filename := "assignment-" + sanitizeForFilename(sessionID) + ".json"
		os.WriteFile(filepath.Join(contextDir, filename), data, 0644)

		status := &contextSessionStatus{
			SessionID: sessionID,
			AgentName: "current-worker",
		}
		mergePersistedAssignment(dir, status)
		if status.AgentName != "current-worker" {
			t.Errorf("AgentName should not be overwritten: got %q, want %q", status.AgentName, "current-worker")
		}
	})
}

func TestContextWithTimeout(t *testing.T) {
	t.Run("positive timeout creates WithTimeout", func(t *testing.T) {
		ctx, cancel := contextWithTimeout(100 * time.Millisecond)
		defer cancel()
		if ctx == nil {
			t.Error("expected non-nil context")
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("expected context with deadline")
		}
		if deadline.IsZero() {
			t.Error("expected non-zero deadline")
		}
	})

	t.Run("zero timeout creates WithCancel", func(t *testing.T) {
		ctx, cancel := contextWithTimeout(0)
		defer cancel()
		if ctx == nil {
			t.Error("expected non-nil context")
		}
		_, ok := ctx.Deadline()
		if ok {
			t.Error("expected context without deadline for zero timeout")
		}
	})

	t.Run("negative timeout creates WithCancel", func(t *testing.T) {
		ctx, cancel := contextWithTimeout(-1)
		defer cancel()
		if ctx == nil {
			t.Error("expected non-nil context")
		}
		_, ok := ctx.Deadline()
		if ok {
			t.Error("expected context without deadline for negative timeout")
		}
	})
}

func TestActionForStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		stale  bool
		want   string
	}{
		{name: "CRITICAL → handoff_now", status: "CRITICAL", stale: false, want: "handoff_now"},
		{name: "WARNING → checkpoint", status: "WARNING", stale: false, want: "checkpoint_and_prepare_handoff"},
		{name: "OPTIMAL not stale → continue", status: "OPTIMAL", stale: false, want: "continue"},
		{name: "stale non-optimal → recover_dead", status: "WARNING", stale: true, want: "recover_dead_session"},
		{name: "stale OPTIMAL → investigate", status: "OPTIMAL", stale: true, want: "investigate_stale_session"},
		{name: "default not stale → continue", status: "unknown", stale: false, want: "continue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actionForStatus(tt.status, tt.stale)
			if got != tt.want {
				t.Errorf("actionForStatus(%q, %v) = %q, want %q", tt.status, tt.stale, got, tt.want)
			}
		})
	}
}

func TestReadinessRank(t *testing.T) {
	tests := []struct {
		readiness string
		want      int
	}{
		{readiness: "CRITICAL", want: 0},
		{readiness: "RED", want: 1},
		{readiness: "AMBER", want: 2},
		{readiness: "GREEN", want: 3},
		{readiness: "unknown", want: 4},
		{readiness: "  GREEN  ", want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.readiness, func(t *testing.T) {
			got := readinessRank(tt.readiness)
			if got != tt.want {
				t.Errorf("readinessRank(%q) = %d, want %d", tt.readiness, got, tt.want)
			}
		})
	}
}

func TestHookMessageForStatus(t *testing.T) {
	t.Run("handoff_now returns critical message", func(t *testing.T) {
		status := contextSessionStatus{
			Action:           "handoff_now",
			UsagePercent:     0.95,
			Readiness:        "CRITICAL",
			RemainingPercent: 0.05,
		}
		got := hookMessageForStatus(status)
		if !strings.Contains(got, "CRITICAL") {
			t.Errorf("expected CRITICAL in message, got: %s", got)
		}
	})

	t.Run("checkpoint returns warning message", func(t *testing.T) {
		status := contextSessionStatus{
			Action:           "checkpoint_and_prepare_handoff",
			UsagePercent:     0.75,
			Readiness:        "WARNING",
			RemainingPercent: 0.25,
		}
		got := hookMessageForStatus(status)
		if !strings.Contains(got, "WARNING") {
			t.Errorf("expected WARNING in message, got: %s", got)
		}
	})

	t.Run("recover_dead with restart attempt success", func(t *testing.T) {
		status := contextSessionStatus{
			Action:         "recover_dead_session",
			RestartAttempt: true,
			RestartSuccess: true,
			TmuxSession:    "my-session",
		}
		got := hookMessageForStatus(status)
		if !strings.Contains(got, "auto-restarted") {
			t.Errorf("expected 'auto-restarted' in message, got: %s", got)
		}
	})

	t.Run("recover_dead with restart attempt failure", func(t *testing.T) {
		status := contextSessionStatus{
			Action:         "recover_dead_session",
			RestartAttempt: true,
			RestartSuccess: false,
			RestartMessage: "tmux not found",
		}
		got := hookMessageForStatus(status)
		if !strings.Contains(got, "auto-restart failed") {
			t.Errorf("expected 'auto-restart failed' in message, got: %s", got)
		}
	})

	t.Run("recover_dead without restart attempt with message", func(t *testing.T) {
		status := contextSessionStatus{
			Action:         "recover_dead_session",
			RestartAttempt: false,
			RestartMessage: "session unresponsive",
		}
		got := hookMessageForStatus(status)
		if !strings.Contains(got, "session unresponsive") {
			t.Errorf("expected restart message in output, got: %s", got)
		}
	})

	t.Run("recover_dead without restart attempt and no message", func(t *testing.T) {
		status := contextSessionStatus{
			Action:         "recover_dead_session",
			RestartAttempt: false,
		}
		got := hookMessageForStatus(status)
		if !strings.Contains(got, "stale") {
			t.Errorf("expected 'stale' in message, got: %s", got)
		}
	})

	t.Run("default with RED readiness returns hull message", func(t *testing.T) {
		status := contextSessionStatus{
			Action:           "continue",
			Readiness:        "RED",
			RemainingPercent: 0.35,
		}
		got := hookMessageForStatus(status)
		if !strings.Contains(got, "RED") {
			t.Errorf("expected RED in message, got: %s", got)
		}
	})

	t.Run("default non-red returns empty", func(t *testing.T) {
		status := contextSessionStatus{
			Action:    "continue",
			Readiness: "GREEN",
		}
		got := hookMessageForStatus(status)
		if got != "" {
			t.Errorf("expected empty message for continue+green, got: %s", got)
		}
	})
}

func TestDisplayOrDash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string returns dash", "", "-"},
		{"whitespace only returns dash", "   ", "-"},
		{"non-empty returns trimmed value", "hello", "hello"},
		{"leading/trailing whitespace trimmed", "  value  ", "value"},
		{"tab whitespace returns dash", "\t", "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := displayOrDash(tt.input)
			if got != tt.want {
				t.Errorf("displayOrDash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text unchanged", "hello world", "hello world"},
		{"newlines replaced with spaces", "hello\nworld", "hello world"},
		{"carriage returns replaced", "hello\rworld", "hello world"},
		{"multiple spaces collapsed", "hello   world", "hello world"},
		{"leading/trailing whitespace removed", "  hello world  ", "hello world"},
		{"mixed newlines and spaces", "  hello\n  world  \n", "hello world"},
		{"empty string", "", ""},
		{"only whitespace", "   \n   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLine(tt.input)
			if got != tt.want {
				t.Errorf("normalizeLine(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReadFileTail(t *testing.T) {
	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := readFileTail("/tmp/definitely-does-not-exist-xyz.txt", 1024)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("returns empty for empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.txt")
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := readFileTail(path, 1024)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty result, got %d bytes", len(got))
		}
	})

	t.Run("returns full content when smaller than maxBytes", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "small.txt")
		content := "hello\nworld\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := readFileTail(path, 10000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != content {
			t.Errorf("expected full content, got %q", string(got))
		}
	})

	t.Run("returns tail when content exceeds maxBytes", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "large.txt")
		// Write 200 bytes of content
		content := strings.Repeat("abcdefghij\n", 20) // 220 bytes
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := readFileTail(path, 50) // Only read last 50 bytes
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) == 0 {
			t.Error("expected non-empty tail")
		}
		if len(got) >= len(content) {
			t.Errorf("expected tail shorter than full content (%d bytes), got %d", len(content), len(got))
		}
	})
}

func TestFindTeamMemberByName(t *testing.T) {
	t.Run("empty name returns false", func(t *testing.T) {
		_, _, ok := findTeamMemberByName("")
		if ok {
			t.Error("expected false for empty agent name")
		}
	})

	t.Run("finds member in team config", func(t *testing.T) {
		dir := t.TempDir()
		teamDir := filepath.Join(dir, ".claude", "teams", "my-team")
		if err := os.MkdirAll(teamDir, 0755); err != nil {
			t.Fatal(err)
		}

		config := map[string]interface{}{
			"members": []map[string]interface{}{
				{"name": "Alice", "role": "worker"},
				{"name": "Bob", "role": "lead"},
			},
		}
		data, _ := json.Marshal(config)
		if err := os.WriteFile(filepath.Join(teamDir, "config.json"), data, 0644); err != nil {
			t.Fatal(err)
		}

		old := os.Getenv("HOME")
		os.Setenv("HOME", dir)
		defer os.Setenv("HOME", old)

		teamName, member, ok := findTeamMemberByName("Alice")
		if !ok {
			t.Error("expected true for Alice")
		}
		if teamName != "my-team" {
			t.Errorf("teamName = %q, want %q", teamName, "my-team")
		}
		if member.Name != "Alice" {
			t.Errorf("member.Name = %q, want %q", member.Name, "Alice")
		}
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		dir := t.TempDir()
		teamDir := filepath.Join(dir, ".claude", "teams", "team-a")
		if err := os.MkdirAll(teamDir, 0755); err != nil {
			t.Fatal(err)
		}

		config := map[string]interface{}{
			"members": []map[string]interface{}{
				{"name": "Worker-1"},
			},
		}
		data, _ := json.Marshal(config)
		os.WriteFile(filepath.Join(teamDir, "config.json"), data, 0644) //nolint:errcheck // test setup
		old := os.Getenv("HOME")
		os.Setenv("HOME", dir)
		defer os.Setenv("HOME", old)

		_, member, ok := findTeamMemberByName("worker-1")
		if !ok {
			t.Error("expected case-insensitive match for worker-1")
		}
		if member.Name != "Worker-1" {
			t.Errorf("member.Name = %q, want %q", member.Name, "Worker-1")
		}
	})

	t.Run("nonexistent agent returns false", func(t *testing.T) {
		dir := t.TempDir()
		teamDir := filepath.Join(dir, ".claude", "teams", "team-b")
		os.MkdirAll(teamDir, 0755)                                                        //nolint:errcheck // test setup
		data, _ := json.Marshal(map[string]interface{}{"members": []map[string]interface{}{{"name": "Bob"}}})
		os.WriteFile(filepath.Join(teamDir, "config.json"), data, 0644) //nolint:errcheck // test setup
		old := os.Getenv("HOME")
		os.Setenv("HOME", dir)
		defer os.Setenv("HOME", old)

		_, _, ok := findTeamMemberByName("Charlie")
		if ok {
			t.Error("expected false for nonexistent agent")
		}
	})
}
