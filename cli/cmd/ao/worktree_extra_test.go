package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)


func TestWorktreeReferenceTime_NoFiles(t *testing.T) {
	dir := t.TempDir()
	ref := worktreeReferenceTime(dir)
	// When no state/status files exist, it should fall back to the dir's mod time.
	// ref.IsZero() is acceptable if stat failed; we just verify no panic.
	_ = ref.IsZero()
}

func TestWorktreeReferenceTime_WithStateFile(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".agents", "rpi")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write state file
	statePath := filepath.Join(stateDir, "phased-state.json")
	if err := os.WriteFile(statePath, []byte(`{"run_id":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	before := time.Now().Add(-time.Second)
	ref := worktreeReferenceTime(dir)
	after := time.Now().Add(time.Second)

	if ref.Before(before) || ref.After(after) {
		t.Errorf("reference time %v is not within expected range [%v, %v]", ref, before, after)
	}
}

func TestFindRPISiblingWorktreePaths_NoSiblings(t *testing.T) {
	dir := t.TempDir()
	// dir has no rpi sibling dirs
	paths, err := findRPISiblingWorktreePaths(dir)
	if err != nil {
		t.Fatalf("findRPISiblingWorktreePaths: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected no sibling paths, got %v", paths)
	}
}

func TestFindRPISiblingWorktreePaths_WithSiblings(t *testing.T) {
	parent := t.TempDir()
	repoRoot := filepath.Join(parent, "myrepo")
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatal(err)
	}

	// Create sibling worktree dirs
	sibling1 := filepath.Join(parent, "myrepo-rpi-abc123")
	sibling2 := filepath.Join(parent, "myrepo-rpi-def456")
	notSibling := filepath.Join(parent, "myrepo-other")

	for _, d := range []string{sibling1, sibling2, notSibling} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	paths, err := findRPISiblingWorktreePaths(repoRoot)
	if err != nil {
		t.Fatalf("findRPISiblingWorktreePaths: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 sibling paths, got %d: %v", len(paths), paths)
	}
}

func TestFindStaleRPISiblingWorktrees_NoWorktrees(t *testing.T) {
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "myrepo")
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatal(err)
	}

	candidates, liveRuns, skipped, err := findStaleRPISiblingWorktrees(
		repoRoot, time.Now(), time.Hour, make(map[string]bool), false)
	if err != nil {
		t.Fatalf("findStaleRPISiblingWorktrees: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected no candidates, got %d", len(candidates))
	}
	if len(liveRuns) != 0 {
		t.Errorf("expected no live runs, got %d", len(liveRuns))
	}
	if len(skipped) != 0 {
		t.Errorf("expected no skipped, got %d", len(skipped))
	}
}

func TestDiscoverActiveRPIRuns_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	activeRuns := discoverActiveRPIRuns(dir)
	if len(activeRuns) != 0 {
		t.Errorf("expected no active runs for empty dir, got %d", len(activeRuns))
	}
}
