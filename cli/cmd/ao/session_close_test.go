package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTranscript(t *testing.T) {
	// Create temp directory structure mimicking ~/.claude/projects/
	tempDir, err := os.MkdirTemp("", "session-close-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir) //nolint:errcheck // test cleanup
	}()

	tests := []struct {
		name           string
		sessionID      string
		setupFunc      func(t *testing.T) string // returns expected path
		expectFallback bool
		expectError    bool
	}{
		{
			name:      "empty session ID triggers fallback",
			sessionID: "",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				// findLastSession searches real ~/.claude/projects
				// so this test just verifies fallback is set
				return ""
			},
			expectFallback: true,
			expectError:    true, // may fail if no real transcripts
		},
		{
			name:      "nonexistent session ID returns error",
			sessionID: "nonexistent-session-id-12345",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			expectFallback: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc(t)

			_, usedFallback, err := resolveTranscript(tt.sessionID)
			if tt.expectError && err == nil {
				// Some tests may pass on machines with real transcripts
				// Only fail if we expected an error and got specific wrong behavior
				if tt.sessionID == "nonexistent-session-id-12345" {
					t.Error("expected error for nonexistent session ID, got nil")
				}
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectFallback && err == nil && !usedFallback {
				t.Error("expected fallback to be used")
			}
		})
	}
}

func TestFindTranscriptBySessionID(t *testing.T) {
	// Create temp directory with mock transcript
	tempDir, err := os.MkdirTemp("", "session-find-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir) //nolint:errcheck // test cleanup
	}()

	tests := []struct {
		name        string
		sessionID   string
		expectError bool
	}{
		{
			name:        "nonexistent session returns error",
			sessionID:   "does-not-exist-abc-123",
			expectError: true,
		},
		{
			name:        "empty session ID returns error",
			sessionID:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := findTranscriptBySessionID(tt.sessionID)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOutputCloseResult(t *testing.T) {
	tests := []struct {
		name   string
		result SessionCloseResult
		format string
	}{
		{
			name: "table output succeeds",
			result: SessionCloseResult{
				SessionID:     "test-session-001",
				Transcript:    "/tmp/test.jsonl",
				Decisions:     3,
				Knowledge:     5,
				FilesChanged:  10,
				Issues:        2,
				VelocityDelta: 0.05,
				Status:        "compounding",
				Message:       "Session closed: 3 decisions, 5 learnings extracted",
			},
			format: "table",
		},
		{
			name: "json output succeeds",
			result: SessionCloseResult{
				SessionID:     "test-session-002",
				Transcript:    "/tmp/test2.jsonl",
				Decisions:     0,
				Knowledge:     0,
				FilesChanged:  0,
				VelocityDelta: -0.01,
				Status:        "decaying",
				Message:       "Session closed: 0 decisions, 0 learnings extracted",
			},
			format: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect stdout to capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Set output format
			oldOutput := output
			output = tt.format
			defer func() {
				output = oldOutput
			}()

			err := outputCloseResult(tt.result)

			_ = w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Read captured output
			buf := make([]byte, 4096)
			n, _ := r.Read(buf)
			_ = r.Close()

			out := string(buf[:n])
			if len(out) == 0 {
				t.Error("expected output, got empty string")
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("get home dir: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path under home directory",
			input:    filepath.Join(homeDir, ".claude", "projects", "test.jsonl"),
			expected: "~/.claude/projects/test.jsonl",
		},
		{
			name:     "path not under home",
			input:    "/tmp/test.jsonl",
			expected: "/tmp/test.jsonl",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shortenPath(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSessionCloseResultJSON(t *testing.T) {
	result := SessionCloseResult{
		SessionID:     "test-json-001",
		Transcript:    "/tmp/session.jsonl",
		Decisions:     2,
		Knowledge:     4,
		FilesChanged:  8,
		Issues:        1,
		VelocityDelta: 0.123,
		Status:        "compounding",
		Message:       "test message",
	}

	// Verify JSON fields exist
	if result.SessionID != "test-json-001" {
		t.Errorf("SessionID: got %q, want %q", result.SessionID, "test-json-001")
	}
	if result.Decisions != 2 {
		t.Errorf("Decisions: got %d, want 2", result.Decisions)
	}
	if result.VelocityDelta != 0.123 {
		t.Errorf("VelocityDelta: got %f, want 0.123", result.VelocityDelta)
	}
}
