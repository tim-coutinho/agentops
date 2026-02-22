package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCancelTargets_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	targets := discoverCancelTargets([]string{dir}, "", nil)
	if len(targets) != 0 {
		t.Errorf("expected no targets for empty dir, got %d", len(targets))
	}
}

func TestDiscoverRunRegistryTargets_EmptyRunsDir(t *testing.T) {
	dir := t.TempDir()
	// No .agents/rpi/runs directory
	targets := discoverRunRegistryTargets(dir, "", nil, make(map[string]struct{}))
	if len(targets) != 0 {
		t.Errorf("expected no targets when runs dir missing, got %d", len(targets))
	}
}

func TestDiscoverSupervisorLeaseTargets_NoLease(t *testing.T) {
	dir := t.TempDir()
	targets := discoverSupervisorLeaseTargets(dir, "", nil, make(map[string]struct{}))
	if len(targets) != 0 {
		t.Errorf("expected no targets when no lease file, got %d", len(targets))
	}
}

func TestDiscoverSupervisorLeaseTargets_InvalidLease(t *testing.T) {
	dir := t.TempDir()
	leaseDir := filepath.Join(dir, ".agents", "rpi")
	if err := os.MkdirAll(leaseDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write invalid JSON
	if err := os.WriteFile(filepath.Join(leaseDir, "supervisor.lock"), []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}
	targets := discoverSupervisorLeaseTargets(dir, "", nil, make(map[string]struct{}))
	if len(targets) != 0 {
		t.Errorf("expected no targets for invalid lease JSON, got %d", len(targets))
	}
}

func TestDiscoverSupervisorLeaseTargets_LeaseWithInvalidPID(t *testing.T) {
	dir := t.TempDir()
	leaseDir := filepath.Join(dir, ".agents", "rpi")
	if err := os.MkdirAll(leaseDir, 0755); err != nil {
		t.Fatal(err)
	}
	meta := supervisorLeaseMetadata{
		RunID: "test-run",
		PID:   0, // invalid
	}
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(leaseDir, "supervisor.lock"), data, 0644); err != nil {
		t.Fatal(err)
	}
	targets := discoverSupervisorLeaseTargets(dir, "", nil, make(map[string]struct{}))
	if len(targets) != 0 {
		t.Errorf("expected no targets for lease with invalid PID, got %d", len(targets))
	}
}

func TestFilterKillablePIDs_FiltersSelfAndZero(t *testing.T) {
	selfPID := 12345
	pids := []int{0, 1, selfPID, 100, 200}
	filtered := filterKillablePIDs(pids, selfPID)
	for _, pid := range filtered {
		if pid == 0 || pid == 1 || pid == selfPID {
			t.Errorf("filtered list should not contain PID %d", pid)
		}
	}
	// Should keep 100 and 200
	if len(filtered) != 2 {
		t.Errorf("expected 2 killable PIDs, got %d: %v", len(filtered), filtered)
	}
}

func TestFilterKillablePIDs_Empty(t *testing.T) {
	filtered := filterKillablePIDs(nil, 100)
	if len(filtered) != 0 {
		t.Errorf("expected empty result for nil input, got %v", filtered)
	}
}

func TestFilterKillablePIDs_Deduplication(t *testing.T) {
	pids := []int{100, 100, 200, 200}
	filtered := filterKillablePIDs(pids, 999)
	if len(filtered) != 2 {
		t.Errorf("expected 2 unique PIDs after dedup, got %d: %v", len(filtered), filtered)
	}
}

func TestCollectRunProcessPIDs_EmptyState(t *testing.T) {
	state := &phasedState{
		RunID:           "test-run",
		OrchestratorPID: 0, // invalid PID
	}
	pids := collectRunProcessPIDs(state, nil)
	if len(pids) != 0 {
		t.Errorf("expected no PIDs for empty/invalid state, got %d: %v", len(pids), pids)
	}
}

func TestCollectRunProcessPIDs_MatchByWorktreePath(t *testing.T) {
	procs := []processInfo{
		{PID: 1234, PPID: 1, Command: "/some/path /worktrees/myrepo-rpi-abc123"},
		{PID: 5678, PPID: 1, Command: "some other process"},
	}
	state := &phasedState{
		RunID:        "abc123",
		WorktreePath: "/worktrees/myrepo-rpi-abc123",
	}
	pids := collectRunProcessPIDs(state, procs)
	found := false
	for _, pid := range pids {
		if pid == 1234 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected PID 1234 in result for worktree path match, got: %v", pids)
	}
}
