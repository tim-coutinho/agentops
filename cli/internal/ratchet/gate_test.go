package ratchet

import (
	"testing"
	"time"
)

func TestBdCLITimeout(t *testing.T) {
	// Verify the timeout constant is set correctly
	if BdCLITimeout != 5*time.Second {
		t.Errorf("expected BdCLITimeout to be 5s, got %v", BdCLITimeout)
	}

	// Verify error message is correct
	expectedMsg := "bd CLI timeout after 5s"
	if ErrBdCLITimeout.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrBdCLITimeout.Error())
	}
}

func TestGateCheckerWithMissingBd(t *testing.T) {
	// This test verifies that findEpic handles command errors gracefully.
	// When bd is not installed or not in PATH, we should get an error but not hang.
	tmpDir := t.TempDir()
	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		// NewGateChecker may fail if the directory structure is not set up,
		// which is expected for this test
		t.Skip("GateChecker requires specific directory structure")
	}

	// Call findEpic - it will fail but should not hang
	start := time.Now()
	_, err = checker.findEpic("open")
	elapsed := time.Since(start)

	// The command should return quickly (within timeout) even if bd is not found
	if elapsed > BdCLITimeout+time.Second {
		t.Errorf("findEpic took too long (%v), expected to complete within timeout", elapsed)
	}

	// We expect an error (bd not found or no epic found), but not a timeout
	// unless the command is actually hanging
	if err == ErrBdCLITimeout {
		t.Error("unexpected timeout error - bd command should fail fast if not installed")
	}
}
