package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLogAndFailPhase(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rpi.log")
	stateDir := filepath.Join(dir, ".agents", "rpi")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	state := newTestPhasedState().WithRunID("test-run-123")
	err := logAndFailPhase(state, "research", logPath, dir, fmt.Errorf("research failed"))

	if err == nil {
		t.Fatal("expected logAndFailPhase to return the error")
	}
	if err.Error() != "research failed" {
		t.Errorf("error = %q, want 'research failed'", err.Error())
	}
	if state.TerminalStatus != "failed" {
		t.Errorf("TerminalStatus = %q, want failed", state.TerminalStatus)
	}
	if state.TerminatedAt == "" {
		t.Error("TerminatedAt should be set")
	}
}
