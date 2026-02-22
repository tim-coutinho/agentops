package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecuteRPICleanup_RequiresAllOrRunID(t *testing.T) {
	err := executeRPICleanup(t.TempDir(), "", false, false, false, false, 0)
	if err == nil {
		t.Fatal("expected error when neither --all nor --run-id specified")
	}
	if !strings.Contains(err.Error(), "--all") || !strings.Contains(err.Error(), "--run-id") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteRPICleanup_AllNoStaleRuns(t *testing.T) {
	// Directory with no runs — should succeed with "No stale runs found."
	err := executeRPICleanup(t.TempDir(), "", true, false, false, false, 0)
	if err != nil {
		t.Errorf("expected no error for clean dir, got: %v", err)
	}
}

func TestExecuteRPICleanup_DryRunWithStaleRun(t *testing.T) {
	tmpDir := t.TempDir()
	runID := "dry-run-test"
	runDir := filepath.Join(tmpDir, ".agents", "rpi", "runs", runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	state := map[string]any{
		"schema_version": 1,
		"run_id":         runID,
		"goal":           "test goal",
		"phase":          2,
		"worktree_path":  "/nonexistent/worktree",
		"started_at":     time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(runDir, phasedStateFile), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Dry run should not modify state
	err := executeRPICleanup(tmpDir, "", true, false, false, true, 0)
	if err != nil {
		t.Errorf("dry run cleanup: %v", err)
	}

	// State file should be unchanged (no terminal_status written)
	updated, _ := os.ReadFile(filepath.Join(runDir, phasedStateFile))
	var updatedState map[string]any
	_ = json.Unmarshal(updated, &updatedState)
	if _, ok := updatedState["terminal_status"]; ok {
		t.Error("dry run should not modify state file")
	}
}

func TestExecuteRPICleanup_SpecificRunID(t *testing.T) {
	tmpDir := t.TempDir()
	targetID := "target-run"
	otherID := "other-run"

	for _, runID := range []string{targetID, otherID} {
		runDir := filepath.Join(tmpDir, ".agents", "rpi", "runs", runID)
		if err := os.MkdirAll(runDir, 0755); err != nil {
			t.Fatal(err)
		}
		state := map[string]any{
			"schema_version": 1,
			"run_id":         runID,
			"goal":           "test",
			"phase":          2,
			"worktree_path":  "/nonexistent/path",
			"started_at":     time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
		}
		data, _ := json.Marshal(state)
		if err := os.WriteFile(filepath.Join(runDir, phasedStateFile), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Cleanup only target run
	err := executeRPICleanup(tmpDir, targetID, false, false, false, false, 0)
	if err != nil {
		t.Errorf("targeted cleanup: %v", err)
	}

	// Target run should be marked stale
	targetData, _ := os.ReadFile(filepath.Join(tmpDir, ".agents", "rpi", "runs", targetID, phasedStateFile))
	var targetState map[string]any
	_ = json.Unmarshal(targetData, &targetState)
	if targetState["terminal_status"] != "stale" {
		t.Errorf("target run should be stale, got terminal_status = %v", targetState["terminal_status"])
	}

	// Other run should not be modified
	otherData, _ := os.ReadFile(filepath.Join(tmpDir, ".agents", "rpi", "runs", otherID, phasedStateFile))
	var otherState map[string]any
	_ = json.Unmarshal(otherData, &otherState)
	if _, ok := otherState["terminal_status"]; ok {
		t.Error("other run should not be marked stale")
	}
}

func TestRemoveOrphanedWorktree_SafetyChecks(t *testing.T) {
	// Test path that's not a sibling of repo root
	err := removeOrphanedWorktree("/home/user/repo", "/tmp/random-dir", "abc123")
	if err == nil {
		t.Fatal("expected error for non-sibling worktree path")
	}
	if !strings.Contains(err.Error(), "sibling") {
		t.Errorf("expected 'sibling' in error, got: %v", err)
	}
}

func TestRemoveOrphanedWorktree_RepoRootSameAsWorktree(t *testing.T) {
	err := removeOrphanedWorktree("/home/user/repo", "/home/user/repo", "abc123")
	if err == nil {
		t.Fatal("expected error when worktree path equals repo root")
	}
	if !strings.Contains(err.Error(), "repo root") {
		t.Errorf("expected 'repo root' in error, got: %v", err)
	}
}

func TestPruneWorktrees_InGitRepo(t *testing.T) {
	// Create a minimal git repo to test pruneWorktrees
	dir := t.TempDir()
	initCmd := exec.Command("git", "init")
	initCmd.Dir = dir
	if err := initCmd.Run(); err != nil {
		t.Skip("git not available")
	}
	if err := exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("git config user.email: %v", err)
	}
	if err := exec.Command("git", "-C", dir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("git config user.name: %v", err)
	}
	if err := exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// pruneWorktrees should succeed in a valid git repo
	err := pruneWorktrees(dir)
	if err != nil {
		t.Errorf("pruneWorktrees in valid repo: %v", err)
	}
}

func TestCleanupLegacyRPIBranches_EmptyCwd(t *testing.T) {
	// With all=true and empty cwd, should get the empty cwd error
	err := cleanupLegacyRPIBranches("", "", true, false)
	if err == nil {
		t.Fatal("expected error for empty cwd")
	}
	if !strings.Contains(err.Error(), "repository path") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCleanupLegacyRPIBranches_NeitherAllNorRunID(t *testing.T) {
	err := cleanupLegacyRPIBranches("/some/dir", "", false, false)
	if err == nil {
		t.Fatal("expected error when neither --all nor --run-id specified")
	}
	if !strings.Contains(err.Error(), "--all") || !strings.Contains(err.Error(), "--run-id") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMarkRunStale_WithFlatStateSync(t *testing.T) {
	tmpDir := t.TempDir()
	runID := "sync-test-run"

	// Create run-specific state file
	runDir := filepath.Join(tmpDir, ".agents", "rpi", "runs", runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(runDir, phasedStateFile)
	state := map[string]any{
		"schema_version": 1,
		"run_id":         runID,
		"goal":           "test",
		"phase":          2,
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Create flat state file with same run ID
	flatDir := filepath.Join(tmpDir, ".agents", "rpi")
	flatPath := filepath.Join(flatDir, phasedStateFile)
	flatState := map[string]any{
		"run_id": runID,
		"phase":  2,
	}
	flatData, _ := json.Marshal(flatState)
	if err := os.WriteFile(flatPath, flatData, 0644); err != nil {
		t.Fatal(err)
	}

	sr := staleRunEntry{
		runID:     runID,
		statePath: statePath,
		root:      tmpDir,
		reason:    "test stale",
	}

	if err := markRunStale(sr); err != nil {
		t.Fatalf("markRunStale: %v", err)
	}

	// Check flat state was also updated
	updatedFlat, _ := os.ReadFile(flatPath)
	var updatedFlatState map[string]any
	_ = json.Unmarshal(updatedFlat, &updatedFlatState)
	if updatedFlatState["terminal_status"] != "stale" {
		t.Error("flat state file should also be marked stale")
	}
}

func TestMarkRunStale_MissingStateFile(t *testing.T) {
	sr := staleRunEntry{
		runID:     "missing-run",
		statePath: "/nonexistent/path/state.json",
		root:      t.TempDir(),
		reason:    "test",
	}
	if err := markRunStale(sr); err == nil {
		t.Fatal("expected error for missing state file")
	}
}
