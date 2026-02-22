package rpi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateRunID(t *testing.T) {
	id := GenerateRunID()
	if len(id) != 12 {
		t.Errorf("GenerateRunID length = %d, want 12", len(id))
	}
	// Should be hex
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("GenerateRunID contains non-hex char %q in %q", c, id)
			break
		}
	}
	// Should be unique-ish
	id2 := GenerateRunID()
	if id == id2 {
		t.Logf("Warning: two consecutive GenerateRunID calls returned same value %q (very unlikely)", id)
	}
}

func TestGetRepoRoot_ValidRepo(t *testing.T) {
	repo := initGitRepo(t)
	root, err := GetRepoRoot(repo, 30*time.Second)
	if err != nil {
		t.Fatalf("GetRepoRoot: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty repo root")
	}
}

func TestGetRepoRoot_NotARepo(t *testing.T) {
	dir := t.TempDir()
	_, err := GetRepoRoot(dir, 30*time.Second)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error should mention 'not a git repository', got: %v", err)
	}
}

func TestGetRepoRoot_EmptyDir(t *testing.T) {
	// Empty string dir means current directory; test with a non-repo temp dir instead
	dir := t.TempDir()
	_, err := GetRepoRoot(dir, 30*time.Second)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestCreateWorktree_HappyPath(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, func(f string, a ...interface{}) {})
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		// cleanup
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	if worktreePath == "" {
		t.Error("expected non-empty worktree path")
	}
	if runID == "" {
		t.Error("expected non-empty runID")
	}
	if len(runID) != 12 {
		t.Errorf("runID length = %d, want 12", len(runID))
	}

	// Verify the worktree directory exists
	if _, err := os.Stat(worktreePath); err != nil {
		t.Errorf("worktree directory should exist: %v", err)
	}

	// Verify .agents/rpi was created inside worktree
	agentsDir := filepath.Join(worktreePath, ".agents", "rpi")
	if _, err := os.Stat(agentsDir); err != nil {
		t.Errorf(".agents/rpi directory should exist in worktree: %v", err)
	}
}

func TestCreateWorktree_NilVerbosef(t *testing.T) {
	repo := initGitRepo(t)

	// nil verbosef should not panic
	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree with nil verbosef: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	if worktreePath == "" {
		t.Error("expected non-empty worktree path")
	}
}

func TestRemoveWorktree_HappyPath(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if err := RemoveWorktree(repo, worktreePath, runID, 30*time.Second); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	// After removal, directory should not exist
	if _, err := os.Stat(worktreePath); err == nil {
		t.Error("worktree directory should not exist after removal")
	}
}

func TestRemoveWorktree_PathValidation(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Try to remove with a path that doesn't match the expected pattern
	err = RemoveWorktree(repo, t.TempDir(), runID, 30*time.Second)
	if err == nil {
		t.Fatal("expected error when removing path that doesn't match expected pattern")
	}
	if !strings.Contains(err.Error(), "path validation failed") && !strings.Contains(err.Error(), "refusing to remove") {
		t.Errorf("expected path validation error, got: %v", err)
	}
}

func TestRemoveWorktree_EmptyRunID_InferredFromPath(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	// Remove with empty runID — should be inferred from path
	if err := RemoveWorktree(repo, worktreePath, "", 30*time.Second); err != nil {
		t.Fatalf("RemoveWorktree with empty runID: %v", err)
	}
	_ = runID
}

func TestRpiRunIDFromWorktree(t *testing.T) {
	cases := []struct {
		repoRoot     string
		worktreePath string
		wantRunID    string
	}{
		{
			repoRoot:     "/home/user/myrepo",
			worktreePath: "/home/user/myrepo-rpi-abc123def456",
			wantRunID:    "abc123def456",
		},
		{
			repoRoot:     "/home/user/myrepo",
			worktreePath: "/home/user/other-dir",
			wantRunID:    "", // wrong prefix
		},
		{
			repoRoot:     "/home/user/myrepo",
			worktreePath: "/home/user/myrepo-rpi-",
			wantRunID:    "", // empty suffix
		},
	}

	for _, tc := range cases {
		got := rpiRunIDFromWorktree(tc.repoRoot, tc.worktreePath)
		if got != tc.wantRunID {
			t.Errorf("rpiRunIDFromWorktree(%q, %q) = %q, want %q",
				tc.repoRoot, tc.worktreePath, got, tc.wantRunID)
		}
	}
}

func TestIsBranchBusyInWorktree(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"", false},
		{"fatal: 'main' is already used by worktree at '/foo/bar'", true},
		{"error: 'branch' is used by worktree", true},
		{"ALREADY USED BY WORKTREE", true}, // case insensitive
		{"some other git error", false},
	}
	for _, tc := range cases {
		got := isBranchBusyInWorktree(tc.msg)
		if got != tc.want {
			t.Errorf("isBranchBusyInWorktree(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestMergeWorktree_MissingBothPathAndRunID(t *testing.T) {
	repo := initGitRepo(t)
	err := MergeWorktree(repo, "", "", 5*time.Second, nil)
	if err == nil {
		t.Fatal("expected error when both worktreePath and runID are empty")
	}
	if !strings.Contains(err.Error(), "missing worktree path and run ID") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMergeWorktree_DirtyRepo(t *testing.T) {
	repo := initGitRepo(t)

	// Make the repo dirty by writing an untracked file
	dirtyFile := filepath.Join(repo, "uncommitted.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "uncommitted.txt")
	// Don't commit — staged changes make the index dirty vs HEAD

	err := MergeWorktree(repo, "/fake/path", "fakerunid", 100*time.Millisecond, nil)
	// Should fail because repo is dirty or because the worktree doesn't exist
	// Either way it should return an error
	if err == nil {
		t.Fatal("expected error for dirty/invalid merge scenario")
	}
}
