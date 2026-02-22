package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCycleFailureError_Unwrap tests the Unwrap method on cycleFailureError.
func TestCycleFailureError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	wrapped := wrapCycleFailure(cycleFailureTask, "stage", inner)

	var cfe *cycleFailureError
	// Unwrap should expose the inner error
	if !errors.As(wrapped, &cfe) {
		t.Fatalf("expected *cycleFailureError, got %T", wrapped)
	}
	unwrapped := cfe.Unwrap()
	if unwrapped == nil {
		t.Fatal("Unwrap should return non-nil")
	}
	if !strings.Contains(unwrapped.Error(), "inner error") {
		t.Errorf("Unwrap = %v, want to contain 'inner error'", unwrapped)
	}
}

func TestWrapCycleFailure_NilError(t *testing.T) {
	result := wrapCycleFailure(cycleFailureTask, "stage", nil)
	if result != nil {
		t.Errorf("wrapCycleFailure with nil should return nil, got %v", result)
	}
}

func TestWrapCycleFailure_AlreadyWrapped(t *testing.T) {
	inner := fmt.Errorf("base error")
	wrapped := wrapCycleFailure(cycleFailureTask, "stage1", inner)
	rewrapped := wrapCycleFailure(cycleFailureInfrastructure, "stage2", wrapped)

	// Should preserve the first wrap, not create a double-wrap
	var cfe *cycleFailureError
	if !errors.As(rewrapped, &cfe) {
		t.Fatalf("expected *cycleFailureError, got %T", rewrapped)
	}
	if cfe.kind != cycleFailureTask {
		t.Errorf("kind = %q, want task (first wrap should be preserved)", cfe.kind)
	}
}

func TestRunSupervisorGates_Off(t *testing.T) {
	cfg := rpiLoopSupervisorConfig{
		GatePolicy: loopGatePolicyOff,
	}
	if err := runSupervisorGates(t.TempDir(), cfg); err != nil {
		t.Errorf("expected no error for off gate policy, got: %v", err)
	}
}

func TestRunSupervisorGates_BestEffort_MissingScript(t *testing.T) {
	cfg := rpiLoopSupervisorConfig{
		GatePolicy:         loopGatePolicyBestEffort,
		ValidateFastScript: "nonexistent-gate.sh",
		CommandTimeout:     time.Second,
	}
	// Missing script in best-effort mode should not return error
	if err := runSupervisorGates(t.TempDir(), cfg); err != nil {
		t.Errorf("best-effort gate with missing script should not fail, got: %v", err)
	}
}

func TestRunSupervisorGates_Required_MissingScript(t *testing.T) {
	cfg := rpiLoopSupervisorConfig{
		GatePolicy:         loopGatePolicyRequired,
		ValidateFastScript: "nonexistent-gate.sh",
		CommandTimeout:     time.Second,
	}
	err := runSupervisorGates(t.TempDir(), cfg)
	if err == nil {
		t.Fatal("required gate with missing script should fail")
	}
}

func TestRunSupervisorGates_Required_PassingScript(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "gate.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	prevRunner := loopCommandRunner
	defer func() { loopCommandRunner = prevRunner }()
	loopCommandRunner = func(_ string, _ time.Duration, name string, args ...string) error {
		return nil // simulate pass
	}

	cfg := rpiLoopSupervisorConfig{
		GatePolicy:         loopGatePolicyRequired,
		ValidateFastScript: scriptPath,
		CommandTimeout:     time.Second,
	}
	if err := runSupervisorGates(dir, cfg); err != nil {
		t.Errorf("required gate with passing script should not fail, got: %v", err)
	}
}

func TestRunSupervisorGates_Required_FailingScript(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "gate.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	prevRunner := loopCommandRunner
	defer func() { loopCommandRunner = prevRunner }()
	loopCommandRunner = func(_ string, _ time.Duration, name string, args ...string) error {
		return fmt.Errorf("gate failed")
	}

	cfg := rpiLoopSupervisorConfig{
		GatePolicy:         loopGatePolicyRequired,
		ValidateFastScript: scriptPath,
		CommandTimeout:     time.Second,
	}
	if err := runSupervisorGates(dir, cfg); err == nil {
		t.Fatal("required gate with failing script should return error")
	}
}

func TestRunSupervisorGates_EmptyScriptPaths(t *testing.T) {
	cfg := rpiLoopSupervisorConfig{
		GatePolicy:         loopGatePolicyRequired,
		ValidateFastScript: "",
		SecurityGateScript: "",
		CommandTimeout:     time.Second,
	}
	// Empty script paths should be skipped without error
	if err := runSupervisorGates(t.TempDir(), cfg); err != nil {
		t.Errorf("empty gate paths should be skipped, got: %v", err)
	}
}

func TestRunGateScript_ScriptIsDirectory(t *testing.T) {
	dir := t.TempDir()
	// dir itself is a directory, not a file
	err := runGateScript(dir, dir, true, time.Second)
	if err == nil {
		t.Fatal("expected error when gate path is a directory (required mode)")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("expected 'directory' in error, got: %v", err)
	}
}

func TestRunGateScript_ScriptIsDirectoryOptional(t *testing.T) {
	dir := t.TempDir()
	// optional gate path is a directory: should skip without error
	err := runGateScript(dir, dir, false, time.Second)
	if err != nil {
		t.Errorf("optional directory gate should skip without error, got: %v", err)
	}
}

func TestIsLoopKillSwitchSet_EmptyPath(t *testing.T) {
	cfg := rpiLoopSupervisorConfig{KillSwitchPath: ""}
	set, err := isLoopKillSwitchSet(cfg)
	if err != nil {
		t.Fatalf("empty kill switch path should not error: %v", err)
	}
	if set {
		t.Fatal("empty kill switch path should return false")
	}
}

func TestIsLoopKillSwitchSet_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := rpiLoopSupervisorConfig{KillSwitchPath: dir}
	_, err := isLoopKillSwitchSet(cfg)
	if err == nil {
		t.Fatal("expected error when kill switch path is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("expected 'directory' in error, got: %v", err)
	}
}

func TestNormalizeLoopCommandTimeout_Zero(t *testing.T) {
	got := normalizeLoopCommandTimeout(0)
	if got != defaultLoopCommandTimeout {
		t.Errorf("normalizeLoopCommandTimeout(0) = %v, want %v", got, defaultLoopCommandTimeout)
	}
}

func TestNormalizeLoopCommandTimeout_Negative(t *testing.T) {
	got := normalizeLoopCommandTimeout(-1 * time.Second)
	if got != defaultLoopCommandTimeout {
		t.Errorf("normalizeLoopCommandTimeout(-1s) = %v, want %v", got, defaultLoopCommandTimeout)
	}
}

func TestNormalizeLoopCommandTimeout_Positive(t *testing.T) {
	want := 5 * time.Minute
	got := normalizeLoopCommandTimeout(want)
	if got != want {
		t.Errorf("normalizeLoopCommandTimeout(%v) = %v, want %v", want, got, want)
	}
}

func TestRunLoopCommandWithTimeout_Success(t *testing.T) {
	prevExec := loopExecCommandContext
	defer func() { loopExecCommandContext = prevExec }()
	loopExecCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}

	err := runLoopCommandWithTimeout(t.TempDir(), time.Minute, "true")
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}
}

func TestRunLoopCommandWithTimeout_Failure(t *testing.T) {
	prevExec := loopExecCommandContext
	defer func() { loopExecCommandContext = prevExec }()
	loopExecCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}

	err := runLoopCommandWithTimeout(t.TempDir(), time.Minute, "false")
	if err == nil {
		t.Fatal("expected failure")
	}
}

func TestRunLoopCommandOutputWithTimeout_Success(t *testing.T) {
	prevExec := loopExecCommandContext
	defer func() { loopExecCommandContext = prevExec }()
	loopExecCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "hello")
	}

	out, err := runLoopCommandOutputWithTimeout(t.TempDir(), time.Minute, "echo", "hello")
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}
	_ = out
}

func TestResolveLandingBranch_ExplicitBranch(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		return "", fmt.Errorf("should not be called")
	}

	branch, err := resolveLandingBranch(t.TempDir(), "my-branch", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "my-branch" {
		t.Errorf("branch = %q, want my-branch", branch)
	}
}

func TestResolveLandingBranch_FallsBackToMain(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		return "", fmt.Errorf("not a repo")
	}

	branch, err := resolveLandingBranch(t.TempDir(), "", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("branch = %q, want main", branch)
	}
}

func TestResolveLandingBranch_UsesSymbolicRef(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		if name == "git" && len(args) > 0 && args[0] == "symbolic-ref" {
			return "origin/main", nil
		}
		return "", fmt.Errorf("unexpected call")
	}

	branch, err := resolveLandingBranch(t.TempDir(), "", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("branch = %q, want main", branch)
	}
}

func TestResolveLandingBranch_FallsBackToCurrentBranch(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	calls := 0
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		calls++
		if calls == 1 {
			// symbolic-ref fails
			return "", fmt.Errorf("no remote HEAD")
		}
		// rev-parse returns current branch
		return "feature-branch", nil
	}

	branch, err := resolveLandingBranch(t.TempDir(), "", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "feature-branch" {
		t.Errorf("branch = %q, want feature-branch", branch)
	}
}

func TestRepoHasChanges_NoChanges(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		return "", nil
	}

	dirty, err := repoHasChanges(t.TempDir(), time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dirty {
		t.Error("expected clean repo")
	}
}

func TestRepoHasChanges_WithChanges(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		return " M some-file.go\n", nil
	}

	dirty, err := repoHasChanges(t.TempDir(), time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dirty {
		t.Error("expected dirty repo")
	}
}

func TestCaptureLandingScope(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		if name == "git" && len(args) > 0 && args[0] == "diff" {
			return "file1.go\nfile2.go\n", nil
		}
		if name == "git" && len(args) > 0 && args[0] == "ls-files" {
			return "untracked.go\n", nil
		}
		return "", nil
	}

	scope, err := captureLandingScope(t.TempDir(), time.Minute)
	if err != nil {
		t.Fatalf("captureLandingScope: %v", err)
	}
	if scope == nil {
		t.Fatal("scope should not be nil")
	}
	if len(scope.baselineDirtyPaths) != 3 {
		t.Errorf("expected 3 baseline dirty paths, got %d", len(scope.baselineDirtyPaths))
	}
}

func TestRunSupervisorLanding_Off(t *testing.T) {
	cfg := rpiLoopSupervisorConfig{
		LandingPolicy: loopLandingPolicyOff,
	}
	err := runSupervisorLanding(t.TempDir(), cfg, 1, 1, "test-goal", nil)
	if err != nil {
		t.Errorf("landing policy off should return nil, got: %v", err)
	}
}

func TestRunSupervisorLanding_CommitPolicy_NoChanges(t *testing.T) {
	prevRunner := loopCommandOutputRunner
	defer func() { loopCommandOutputRunner = prevRunner }()
	loopCommandOutputRunner = func(_ string, _ time.Duration, name string, args ...string) (string, error) {
		// No dirty files
		return "", nil
	}

	cfg := rpiLoopSupervisorConfig{
		LandingPolicy:  loopLandingPolicyCommit,
		CommandTimeout: time.Minute,
	}
	err := runSupervisorLanding(t.TempDir(), cfg, 1, 1, "test-goal", &landingScope{
		baselineDirtyPaths: map[string]struct{}{},
	})
	if err != nil {
		t.Errorf("commit policy with no changes should return nil, got: %v", err)
	}
}

func TestRenderLandingCommitMessage_DefaultTemplate(t *testing.T) {
	msg := renderLandingCommitMessage("", 3, 2, "test-goal")
	if !strings.Contains(msg, "3") {
		t.Errorf("default template should include cycle number, got: %q", msg)
	}
}

func TestMaxCycleAttempts(t *testing.T) {
	cfg := rpiLoopSupervisorConfig{CycleRetries: 3}
	if got := cfg.MaxCycleAttempts(); got != 4 {
		t.Errorf("MaxCycleAttempts() = %d, want 4", got)
	}
}

func TestShouldContinueAfterFailure(t *testing.T) {
	stop := rpiLoopSupervisorConfig{FailurePolicy: loopFailurePolicyStop}
	if stop.ShouldContinueAfterFailure() {
		t.Error("stop policy should not continue after failure")
	}

	cont := rpiLoopSupervisorConfig{FailurePolicy: loopFailurePolicyContinue}
	if !cont.ShouldContinueAfterFailure() {
		t.Error("continue policy should continue after failure")
	}
}

func TestResolveLoopSupervisorConfig_InvalidPolicies(t *testing.T) {
	t.Setenv("AGENTOPS_RPI_RUNTIME", "")
	t.Setenv("AGENTOPS_RPI_RUNTIME_MODE", "")
	t.Setenv("AGENTOPS_RPI_RUNTIME_COMMAND", "")
	t.Setenv("AGENTOPS_RPI_AO_COMMAND", "")
	t.Setenv("AGENTOPS_RPI_BD_COMMAND", "")
	t.Setenv("AGENTOPS_RPI_TMUX_COMMAND", "")

	prev := snapshotLoopSupervisorGlobals()
	defer restoreLoopSupervisorGlobals(prev)

	tests := []struct {
		name           string
		failurePolicy  string
		gatePolicy     string
		landingPolicy  string
		bdSyncPolicy   string
		wantErrContain string
	}{
		{
			name:           "invalid failure policy",
			failurePolicy:  "invalid",
			gatePolicy:     "off",
			landingPolicy:  "off",
			bdSyncPolicy:   "auto",
			wantErrContain: "invalid failure-policy",
		},
		{
			name:           "invalid gate policy",
			failurePolicy:  "stop",
			gatePolicy:     "invalid",
			landingPolicy:  "off",
			bdSyncPolicy:   "auto",
			wantErrContain: "invalid gate-policy",
		},
		{
			name:           "invalid landing policy",
			failurePolicy:  "stop",
			gatePolicy:     "off",
			landingPolicy:  "invalid",
			bdSyncPolicy:   "auto",
			wantErrContain: "invalid landing-policy",
		},
		{
			name:           "invalid bd-sync policy",
			failurePolicy:  "stop",
			gatePolicy:     "off",
			landingPolicy:  "off",
			bdSyncPolicy:   "invalid",
			wantErrContain: "invalid bd-sync-policy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rpiSupervisor = false
			rpiFailurePolicy = tc.failurePolicy
			rpiGatePolicy = tc.gatePolicy
			rpiLandingPolicy = tc.landingPolicy
			rpiBDSyncPolicy = tc.bdSyncPolicy
			rpiCycleRetries = 0
			rpiLeaseTTL = 2 * time.Minute
			rpiCommandTimeout = 0
			rpiAutoCleanStaleAfter = 24 * time.Hour

			cmd := newLoopSupervisorTestCommand()
			_, err := resolveLoopSupervisorConfig(cmd, t.TempDir())
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErrContain)
			}
			if !strings.Contains(err.Error(), tc.wantErrContain) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tc.wantErrContain)
			}
		})
	}
}

func TestResolveLoopSupervisorConfig_NegativeRetries(t *testing.T) {
	t.Setenv("AGENTOPS_RPI_RUNTIME", "")
	t.Setenv("AGENTOPS_RPI_RUNTIME_MODE", "")
	t.Setenv("AGENTOPS_RPI_RUNTIME_COMMAND", "")
	t.Setenv("AGENTOPS_RPI_AO_COMMAND", "")
	t.Setenv("AGENTOPS_RPI_BD_COMMAND", "")
	t.Setenv("AGENTOPS_RPI_TMUX_COMMAND", "")

	prev := snapshotLoopSupervisorGlobals()
	defer restoreLoopSupervisorGlobals(prev)

	rpiSupervisor = false
	rpiFailurePolicy = "stop"
	rpiGatePolicy = "off"
	rpiLandingPolicy = "off"
	rpiBDSyncPolicy = "auto"
	rpiCycleRetries = -1
	rpiLeaseTTL = 2 * time.Minute
	rpiCommandTimeout = 0
	rpiAutoCleanStaleAfter = 24 * time.Hour

	cmd := newLoopSupervisorTestCommand()
	_, err := resolveLoopSupervisorConfig(cmd, t.TempDir())
	if err == nil {
		t.Fatal("expected error for negative cycle-retries")
	}
	if !strings.Contains(err.Error(), "cycle-retries") {
		t.Errorf("error = %q, want 'cycle-retries'", err.Error())
	}
}
