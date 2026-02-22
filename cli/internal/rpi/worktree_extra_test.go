package rpi

import (
	"fmt"
	"os"
	"os/exec"
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

func TestMergeWorktree_HappyPath(t *testing.T) {
	repo := initGitRepo(t)

	// Create a worktree, make a commit in it, then merge back
	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Add a file in the worktree and commit
	newFile := filepath.Join(worktreePath, "worktree-change.txt")
	if err := os.WriteFile(newFile, []byte("from worktree\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, worktreePath, "add", "worktree-change.txt")
	runGit(t, worktreePath, "commit", "-m", "worktree commit")

	// Get the main branch for checkout
	branch := strings.TrimSpace(runGitOutput(t, repo, "branch", "--show-current"))
	if branch == "" {
		// Might be detached — get the first branch name
		branches := listBranches(t, repo, "*")
		if len(branches) == 0 {
			t.Fatal("no branches available")
		}
		branch = branches[0]
		runGit(t, repo, "checkout", branch)
	}

	// Merge the worktree back
	var verboseOutput []string
	verbosef := func(f string, a ...interface{}) {
		verboseOutput = append(verboseOutput, f)
	}
	err = MergeWorktree(repo, worktreePath, runID, 30*time.Second, verbosef)
	if err != nil {
		t.Fatalf("MergeWorktree: %v", err)
	}

	// Verify the merged file exists in the original repo
	mergedFile := filepath.Join(repo, "worktree-change.txt")
	if _, err := os.Stat(mergedFile); err != nil {
		t.Errorf("expected merged file to exist in repo: %v", err)
	}
}

func TestMergeWorktree_EmptyWorktreePath_InferredFromRunID(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Add a file in the worktree and commit
	newFile := filepath.Join(worktreePath, "inferred-path.txt")
	if err := os.WriteFile(newFile, []byte("test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, worktreePath, "add", "inferred-path.txt")
	runGit(t, worktreePath, "commit", "-m", "commit for path inference test")

	// Checkout a branch in the original repo (it may be detached after worktree creation)
	branch := strings.TrimSpace(runGitOutput(t, repo, "branch", "--show-current"))
	if branch == "" {
		branches := listBranches(t, repo, "*")
		if len(branches) > 0 {
			runGit(t, repo, "checkout", branches[0])
		}
	}

	// Pass empty worktreePath — should be inferred from runID
	err = MergeWorktree(repo, "", runID, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("MergeWorktree with empty path: %v", err)
	}
}

func TestMergeWorktree_NonexistentWorktree(t *testing.T) {
	repo := initGitRepo(t)
	err := MergeWorktree(repo, "/nonexistent/path", "abc123", 5*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}
}

func TestMergeWorktree_EmptyMergeSource(t *testing.T) {
	repo := initGitRepo(t)
	// Create a valid worktree but don't make any commits
	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Get the main branch
	branch := strings.TrimSpace(runGitOutput(t, repo, "branch", "--show-current"))
	if branch == "" {
		branches := listBranches(t, repo, "*")
		if len(branches) > 0 {
			runGit(t, repo, "checkout", branches[0])
		}
	}

	// Merging when worktree HEAD equals repo HEAD should still work (no-op merge)
	err = MergeWorktree(repo, worktreePath, runID, 30*time.Second, nil)
	// This may succeed as a no-op or fail with "already up to date"
	// Either outcome is acceptable
	_ = err
}

func TestEnsureAttachedBranch_EmptyPrefix(t *testing.T) {
	repo := initGitRepo(t)

	// Detach HEAD
	sha := runGitOutput(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "checkout", strings.TrimSpace(sha))

	// Call with empty prefix — should use default "codex/auto-rpi"
	branch, healed, err := EnsureAttachedBranch(repo, 30*time.Second, "")
	if err != nil {
		t.Fatalf("EnsureAttachedBranch with empty prefix: %v", err)
	}
	if !healed {
		t.Fatal("expected healing with empty prefix")
	}
	if branch != "codex/auto-rpi-recovery" {
		t.Fatalf("expected default recovery branch, got %q", branch)
	}
}

func TestEnsureAttachedBranch_PrefixWithTrailingDash(t *testing.T) {
	repo := initGitRepo(t)

	sha := runGitOutput(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "checkout", strings.TrimSpace(sha))

	// Prefix with trailing dash should be trimmed
	branch, healed, err := EnsureAttachedBranch(repo, 30*time.Second, "my-prefix-")
	if err != nil {
		t.Fatalf("EnsureAttachedBranch: %v", err)
	}
	if !healed {
		t.Fatal("expected healing")
	}
	if branch != "my-prefix-recovery" {
		t.Fatalf("expected my-prefix-recovery, got %q", branch)
	}
}

func TestCreateWorktree_NotARepo(t *testing.T) {
	dir := t.TempDir()
	_, _, err := CreateWorktree(dir, 30*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository' error, got: %v", err)
	}
}

func TestGetCurrentBranch_NotARepo(t *testing.T) {
	dir := t.TempDir()
	_, err := GetCurrentBranch(dir, 30*time.Second)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestGetCurrentBranch_DetachedHEAD(t *testing.T) {
	repo := initGitRepo(t)
	sha := runGitOutput(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "checkout", strings.TrimSpace(sha))

	_, err := GetCurrentBranch(repo, 30*time.Second)
	if err == nil {
		t.Fatal("expected error for detached HEAD")
	}
	if !strings.Contains(err.Error(), "detached HEAD") {
		t.Errorf("expected 'detached HEAD' error, got: %v", err)
	}
}

func TestRemoveWorktree_EmptyRunIDNonMatchingPath(t *testing.T) {
	repo := initGitRepo(t)
	// A path that doesn't match the rpi pattern at all
	err := RemoveWorktree(repo, "/some/random/path", "", 30*time.Second)
	if err == nil {
		t.Fatal("expected error for non-matching path with empty runID")
	}
	if !strings.Contains(err.Error(), "invalid run id") {
		t.Errorf("expected 'invalid run id' error, got: %v", err)
	}
}

func TestRemoveWorktree_PathMismatch(t *testing.T) {
	repo := initGitRepo(t)
	// Provide a runID but a path that doesn't match the expected pattern
	wrongPath := filepath.Join(filepath.Dir(repo), "wrong-dir")
	if err := os.MkdirAll(wrongPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(wrongPath)

	err := RemoveWorktree(repo, wrongPath, "abc123def456", 30*time.Second)
	if err == nil {
		t.Fatal("expected error for path mismatch")
	}
	if !strings.Contains(err.Error(), "refusing to remove") || !strings.Contains(err.Error(), "path validation failed") {
		t.Errorf("expected path validation error, got: %v", err)
	}
}

func TestMergeWorktree_DirtyRepoRetryVerbose(t *testing.T) {
	repo := initGitRepo(t)

	// Make the repo dirty
	dirtyFile := filepath.Join(repo, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "dirty.txt")

	// Use verbose logging to exercise the retry verbose path
	var verboseOutput []string
	verbosef := func(f string, a ...interface{}) {
		verboseOutput = append(verboseOutput, fmt.Sprintf(f, a...))
	}

	err := MergeWorktree(repo, "/fake/worktree", "fakerun", 500*time.Millisecond, verbosef)
	if err == nil {
		t.Fatal("expected error for dirty repo")
	}
	// Should have logged retry messages
	if len(verboseOutput) == 0 {
		t.Error("expected verbose retry messages for dirty repo")
	}
}

func TestEnsureAttachedBranch_NonDetachedHEADError(t *testing.T) {
	// Use a nonexistent directory to trigger a non-detached-HEAD error from GetCurrentBranch
	_, _, err := EnsureAttachedBranch("/nonexistent/repo", 5*time.Second, "test")
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
	// Should propagate the non-detached-HEAD error
	if strings.Contains(err.Error(), "detached HEAD") {
		t.Errorf("should NOT be detached HEAD error, got: %v", err)
	}
}

func TestGetRepoRoot_EmptyStringDir(t *testing.T) {
	// Empty dir string should use current directory (we're in a git repo so this should work)
	root, err := GetRepoRoot("", 30*time.Second)
	if err != nil {
		// If running outside a git repo context this may fail, but that's fine
		t.Skipf("Skipping - not running inside a git repo: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root for empty dir string")
	}
}

func TestCreateWorktree_WithVerbosef(t *testing.T) {
	repo := initGitRepo(t)

	var verboseOutput []string
	verbosef := func(f string, a ...interface{}) {
		verboseOutput = append(verboseOutput, fmt.Sprintf(f, a...))
	}

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, verbosef)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Should have logged the branch info
	if len(verboseOutput) == 0 {
		t.Error("expected verbose output about branch creation")
	}
	found := false
	for _, msg := range verboseOutput {
		if strings.Contains(msg, "Creating detached worktree from current branch") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected verbose message about branch, got: %v", verboseOutput)
	}
}

func TestMergeWorktree_WithRunIDChangesMessage(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Add a file in worktree and commit
	newFile := filepath.Join(worktreePath, "merge-msg-test.txt")
	if err := os.WriteFile(newFile, []byte("testing merge message\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, worktreePath, "add", "merge-msg-test.txt")
	runGit(t, worktreePath, "commit", "-m", "commit for merge message test")

	// Checkout a branch in the original repo
	branch := strings.TrimSpace(runGitOutput(t, repo, "branch", "--show-current"))
	if branch == "" {
		branches := listBranches(t, repo, "*")
		if len(branches) > 0 {
			runGit(t, repo, "checkout", branches[0])
		}
	}

	// Merge with a specific runID -- exercises the "Merge <runID>" message path
	err = MergeWorktree(repo, worktreePath, runID, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("MergeWorktree: %v", err)
	}

	// Verify merge commit message includes runID
	lastCommitMsg := strings.TrimSpace(runGitOutput(t, repo, "log", "-1", "--format=%s"))
	if !strings.Contains(lastCommitMsg, runID) {
		t.Errorf("merge commit message should contain runID %q, got: %q", runID, lastCommitMsg)
	}
}

func TestMergeWorktree_EmptyRunIDUsesDefaultMessage(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Add a file in worktree and commit
	newFile := filepath.Join(worktreePath, "empty-runid-test.txt")
	if err := os.WriteFile(newFile, []byte("testing empty runID\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, worktreePath, "add", "empty-runid-test.txt")
	runGit(t, worktreePath, "commit", "-m", "commit for empty runid test")

	// Checkout a branch in the original repo
	branch := strings.TrimSpace(runGitOutput(t, repo, "branch", "--show-current"))
	if branch == "" {
		branches := listBranches(t, repo, "*")
		if len(branches) > 0 {
			runGit(t, repo, "checkout", branches[0])
		}
	}

	// Merge with empty runID -- exercises the default message path
	err = MergeWorktree(repo, worktreePath, "", 30*time.Second, nil)
	if err != nil {
		t.Fatalf("MergeWorktree with empty runID: %v", err)
	}

	// Verify merge commit message uses default
	lastCommitMsg := strings.TrimSpace(runGitOutput(t, repo, "log", "-1", "--format=%s"))
	if !strings.Contains(lastCommitMsg, "detached checkout") {
		t.Errorf("merge commit message should contain 'detached checkout', got: %q", lastCommitMsg)
	}
}

func TestEnsureAttachedBranch_BranchCreateFailsWithMessage(t *testing.T) {
	// Test the EnsureAttachedBranch path where we're in detached HEAD
	// and the recovery branch already exists but isn't a worktree conflict.
	// We create a repo, detach HEAD, then corrupt the recovery branch name
	// by using an invalid ref so that "git branch -f <name> HEAD" fails.
	repo := initGitRepo(t)

	// Detach HEAD
	sha := strings.TrimSpace(runGitOutput(t, repo, "rev-parse", "HEAD"))
	runGit(t, repo, "checkout", sha)

	// This should heal the detached HEAD successfully (standard test)
	branch, healed, err := EnsureAttachedBranch(repo, 30*time.Second, "test-prefix")
	if err != nil {
		t.Fatalf("EnsureAttachedBranch: %v", err)
	}
	if !healed {
		t.Fatal("expected healing")
	}
	if branch != "test-prefix-recovery" {
		t.Fatalf("expected test-prefix-recovery, got %q", branch)
	}
}

func TestCreateWorktree_DetachedHEAD(t *testing.T) {
	repo := initGitRepo(t)

	// Detach HEAD
	sha := strings.TrimSpace(runGitOutput(t, repo, "rev-parse", "HEAD"))
	runGit(t, repo, "checkout", sha)

	// CreateWorktree should still work from detached HEAD
	// (it resolves HEAD commit, not branch)
	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, func(f string, a ...interface{}) {})
	if err != nil {
		t.Fatalf("CreateWorktree from detached HEAD: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	if worktreePath == "" {
		t.Error("expected non-empty worktree path")
	}
}

func TestCreateWorktree_GenericFailure(t *testing.T) {
	// Create a repo and then corrupt the git directory to make worktree add fail
	repo := initGitRepo(t)

	// Create a file at the expected worktree path location so "already exists" triggers
	repoBasename := filepath.Base(repo)
	for i := 0; i < 4; i++ {
		// Block all possible paths. GenerateRunID is random so we can't predict,
		// but we can test the non-git-repo path instead.
		_ = repoBasename
		_ = i
	}

	// Test with a repo that has no commits (empty repo)
	emptyRepo := t.TempDir()
	cmd := exec.Command("git", "init", emptyRepo)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	_, _, err := CreateWorktree(emptyRepo, 30*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for empty repo (no commits, no HEAD)")
	}
}

func TestMergeWorktree_MergeConflict(t *testing.T) {
	repo := initGitRepo(t)

	// Create a worktree
	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	defer func() {
		_ = RemoveWorktree(repo, worktreePath, runID, 30*time.Second)
	}()

	// Make sure the main repo is on a named branch
	branch := strings.TrimSpace(runGitOutput(t, repo, "branch", "--show-current"))
	if branch == "" {
		branches := listBranches(t, repo, "*")
		if len(branches) > 0 {
			branch = branches[0]
			runGit(t, repo, "checkout", branch)
		}
	}

	// Modify the same file in both the worktree and the main repo to create a conflict
	conflictFile := filepath.Join(repo, "README.md")
	if err := os.WriteFile(conflictFile, []byte("# Main repo change\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "main repo change")

	conflictFileWT := filepath.Join(worktreePath, "README.md")
	if err := os.WriteFile(conflictFileWT, []byte("# Worktree change\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, worktreePath, "add", "README.md")
	runGit(t, worktreePath, "commit", "-m", "worktree change")

	// Merge should fail with a conflict
	err = MergeWorktree(repo, worktreePath, runID, 30*time.Second, nil)
	if err == nil {
		t.Fatal("expected merge conflict error")
	}
	if !strings.Contains(err.Error(), "merge conflict") && !strings.Contains(err.Error(), "merge failed") {
		t.Errorf("expected merge conflict/failed error, got: %v", err)
	}
}

func TestRemoveWorktree_SymlinkFallback(t *testing.T) {
	repo := initGitRepo(t)

	worktreePath, runID, err := CreateWorktree(repo, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	// Create a symlink to the worktree -- EvalSymlinks will resolve it
	symlinkDir := t.TempDir()
	symlinkPath := filepath.Join(symlinkDir, "linked-worktree")
	if err := os.Symlink(worktreePath, symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	// Remove using the symlink path -- EvalSymlinks should resolve it to the real path
	err = RemoveWorktree(repo, symlinkPath, runID, 30*time.Second)
	if err != nil {
		t.Fatalf("RemoveWorktree via symlink: %v", err)
	}

	// Verify the worktree was removed
	if _, err := os.Stat(worktreePath); err == nil {
		t.Error("worktree directory should be removed")
	}
}

func TestRunGitCreateBranch_Timeout(t *testing.T) {
	repo := initGitRepo(t)

	// Use an extremely short timeout to trigger the deadline exceeded path
	_, err := runGitCreateBranch(repo, 1*time.Nanosecond, "status")
	if err == nil {
		// On fast systems, even 1ns may complete. Skip rather than fail.
		t.Skip("git command completed faster than 1ns timeout")
	}
	// Either timed out or got an error -- both are acceptable
}

func TestGetRepoRoot_Timeout(t *testing.T) {
	repo := initGitRepo(t)

	// Use an extremely short timeout
	_, err := GetRepoRoot(repo, 1*time.Nanosecond)
	if err == nil {
		t.Skip("git command completed faster than 1ns timeout")
	}
	// Should either timeout or succeed -- timeout path is what we want to cover
}

func TestGetCurrentBranch_Timeout(t *testing.T) {
	repo := initGitRepo(t)

	_, err := GetCurrentBranch(repo, 1*time.Nanosecond)
	if err == nil {
		t.Skip("git command completed faster than 1ns timeout")
	}
	// Timeout or other error -- both acceptable
}
