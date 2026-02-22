package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	cleanupRunID          string
	cleanupAll            bool
	cleanupPruneWorktrees bool
	cleanupPruneBranches  bool
	cleanupDryRun         bool
	cleanupStaleAfter     time.Duration
)

func init() {
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up stale RPI runs",
		Long: `Detect and clean up stale RPI phased runs.

A run is considered stale if it has no active heartbeat, no live tmux session,
and is not at a terminal phase (completed). Stale runs are marked with terminal
metadata so they no longer appear as "running" or "unknown".

Examples:
  ao rpi cleanup --all --dry-run     # Preview cleanup actions
  ao rpi cleanup --all               # Clean up all stale runs
  ao rpi cleanup --run-id abc123     # Clean up a specific run
  ao rpi cleanup --all --prune-worktrees  # Also run git worktree prune`,
		RunE: runRPICleanup,
	}
	cleanupCmd.Flags().StringVar(&cleanupRunID, "run-id", "", "Clean up a specific run by ID")
	cleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "Clean up all stale runs")
	cleanupCmd.Flags().BoolVar(&cleanupPruneBranches, "prune-branches", false, "Delete legacy RPI branches (rpi/*, codex/auto-rpi-*)")
	cleanupCmd.Flags().BoolVar(&cleanupPruneWorktrees, "prune-worktrees", false, "Run 'git worktree prune' after cleanup")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be done without making changes")
	cleanupCmd.Flags().DurationVar(&cleanupStaleAfter, "stale-after", 0, "Only clean runs older than this age (0 disables age filtering)")
	rpiCmd.AddCommand(cleanupCmd)
}

func runRPICleanup(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	return executeRPICleanup(cwd, cleanupRunID, cleanupAll, cleanupPruneWorktrees, cleanupPruneBranches, cleanupDryRun, cleanupStaleAfter)
}

func executeRPICleanup(cwd, runID string, all, prune, pruneBranches bool, dryRun bool, staleAfter time.Duration) error {
	if !all && runID == "" {
		return fmt.Errorf("specify --all or --run-id <id>")
	}

	staleRuns := collectStaleRuns(cwd, runID, staleAfter)

	if len(staleRuns) == 0 {
		fmt.Println("No stale runs found.")
	} else {
		processStaleRuns(cwd, staleRuns, dryRun)
	}

	runCleanupPostActions(cwd, runID, all, prune, pruneBranches, dryRun)
	return nil
}

// collectStaleRuns gathers deduplicated stale runs across all search roots,
// optionally filtered to a specific runID.
func collectStaleRuns(cwd, runID string, staleAfter time.Duration) []staleRunEntry {
	roots := collectSearchRoots(cwd)
	var staleRuns []staleRunEntry
	seen := make(map[string]struct{})
	now := time.Now()

	for _, root := range roots {
		entries := findStaleRunsWithMinAge(root, staleAfter, now)
		for _, e := range entries {
			if _, ok := seen[e.runID]; ok {
				continue
			}
			seen[e.runID] = struct{}{}
			if runID != "" && e.runID != runID {
				continue
			}
			staleRuns = append(staleRuns, e)
		}
	}
	return staleRuns
}

// processStaleRuns iterates over stale runs, marking or cleaning each one.
func processStaleRuns(cwd string, staleRuns []staleRunEntry, dryRun bool) {
	for _, sr := range staleRuns {
		if dryRun {
			reportDryRunCleanup(sr)
			continue
		}
		cleanStaleRun(cwd, sr)
	}
}

// reportDryRunCleanup prints what would happen for a stale run without making changes.
func reportDryRunCleanup(sr staleRunEntry) {
	if sr.terminal == "" {
		fmt.Printf("[dry-run] Would mark run %s as stale (reason: %s)\n", sr.runID, sr.reason)
	} else {
		fmt.Printf("[dry-run] Would clean terminal run %s (%s)\n", sr.runID, sr.reason)
	}
	if sr.worktreePath != "" {
		if _, err := os.Stat(sr.worktreePath); err == nil {
			fmt.Printf("[dry-run] Would remove worktree: %s\n", sr.worktreePath)
		}
	}
}

// cleanStaleRun marks a non-terminal run as stale and removes orphaned worktrees.
func cleanStaleRun(cwd string, sr staleRunEntry) {
	if sr.terminal == "" {
		if err := markRunStale(sr); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to mark run %s as stale: %v\n", sr.runID, err)
			return
		}
		fmt.Printf("Marked run %s as stale (reason: %s)\n", sr.runID, sr.reason)
	} else {
		fmt.Printf("Cleaning terminal run %s (%s)\n", sr.runID, sr.reason)
	}

	removeStaleWorktreeIfExists(cwd, sr)
}

// removeStaleWorktreeIfExists removes the worktree directory associated with
// a stale run if it still exists on disk.
func removeStaleWorktreeIfExists(cwd string, sr staleRunEntry) {
	if sr.worktreePath == "" {
		return
	}
	if _, statErr := os.Stat(sr.worktreePath); statErr != nil {
		return
	}
	repoRoot := resolveCleanupRepoRoot(cwd, sr.worktreePath)
	if rmErr := removeOrphanedWorktree(repoRoot, sr.worktreePath, sr.runID); rmErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not remove worktree %s: %v\n", sr.worktreePath, rmErr)
	} else {
		fmt.Printf("Removed worktree: %s\n", sr.worktreePath)
	}
}

// runCleanupPostActions runs worktree pruning and legacy branch cleanup after
// stale run processing.
func runCleanupPostActions(cwd, runID string, all, prune, pruneBranches, dryRun bool) {
	if prune && !dryRun {
		if err := pruneWorktrees(cwd); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: git worktree prune failed: %v\n", err)
		}
	}
	if pruneBranches {
		if err := cleanupLegacyRPIBranches(cwd, runID, all, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: legacy branch cleanup failed: %v\n", err)
		}
	}
}

// deleteLegacyBranches iterates over candidate branches, skipping active ones,
// and deletes (or dry-run reports) each.
func deleteLegacyBranches(cwd string, candidates []string, activeBranches map[string]bool, dryRun bool) {
	for _, name := range candidates {
		if activeBranches[name] {
			fmt.Printf("Skipping active branch: %s\n", name)
			continue
		}
		if dryRun {
			fmt.Printf("[dry-run] Would delete branch: %s\n", name)
			continue
		}
		cmd := exec.Command("git", "branch", "-D", name)
		cmd.Dir = cwd
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete branch %s: %v\n", name, err)
			continue
		}
		fmt.Printf("Deleted branch: %s\n", name)
	}
}

// cleanupLegacyRPIBranches removes legacy RPI branches for the selected scope.
func cleanupLegacyRPIBranches(cwd, runID string, all, dryRun bool) error {
	runID = strings.TrimSpace(runID)
	if runID == "" && !all {
		return fmt.Errorf("specify --all or --run-id to prune branches")
	}
	if strings.TrimSpace(cwd) == "" {
		return fmt.Errorf("cleanup branch command missing repository path")
	}

	candidates, err := collectLegacyRPIBranches(cwd, runID, all)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Println("No legacy RPI branches found for cleanup.")
		return nil
	}

	activeBranches, err := checkedOutBranchSet(cwd)
	if err != nil {
		return err
	}

	deleteLegacyBranches(cwd, candidates, activeBranches, dryRun)
	return nil
}

func collectLegacyRPIBranches(cwd, runID string, all bool) ([]string, error) {
	branchPatterns := []string{}
	if all {
		branchPatterns = append(branchPatterns, "rpi/*", "codex/auto-rpi-*")
	} else {
		branchPatterns = append(branchPatterns, "rpi/"+runID)
	}

	seen := map[string]struct{}{}
	var branches []string

	for _, pattern := range branchPatterns {
		refPattern := "refs/heads/" + pattern
		cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", refPattern)
		cmd.Dir = cwd
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("list branches (%s): %w", pattern, err)
		}

		for _, raw := range strings.Split(string(out), "\n") {
			name := strings.TrimSpace(raw)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				branches = append(branches, name)
			}
		}
	}
	return branches, nil
}

func checkedOutBranchSet(cwd string) (map[string]bool, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	active := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		const prefix = "branch "
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		ref := strings.TrimPrefix(line, prefix)
		ref = strings.TrimSpace(ref)
		const refsHeads = "refs/heads/"
		if strings.HasPrefix(ref, refsHeads) {
			active[strings.TrimPrefix(ref, refsHeads)] = true
		}
	}

	return active, nil
}

// resolveCleanupRepoRoot picks a controller worktree root to execute
// `git worktree remove` against. It prefers a sibling worktree in the same
// parent directory as targetWorktree, avoiding attempts to remove a worktree
// from within itself.
func resolveCleanupRepoRoot(cwd, targetWorktree string) string {
	target := filepath.Clean(targetWorktree)
	targetParent := filepath.Dir(target)

	roots := collectSearchRoots(cwd)
	for _, root := range roots {
		cleanRoot := filepath.Clean(root)
		if cleanRoot == target {
			continue
		}
		if filepath.Dir(cleanRoot) == targetParent {
			return cleanRoot
		}
	}
	return cwd
}

type staleRunEntry struct {
	runID        string
	root         string
	statePath    string
	reason       string
	worktreePath string
	terminal     string
}

// findStaleRuns scans the registry for runs that are not active and not completed.
func findStaleRuns(root string) []staleRunEntry {
	return findStaleRunsWithMinAge(root, 0, time.Now())
}

// classifyRunEntry reads and parses a run's state file, returning a staleRunEntry
// if the run qualifies as stale. Returns ok=false if the run is active or
// cannot be parsed.
func classifyRunEntry(runID, root, runsDir string, minAge time.Duration, now time.Time) (staleRunEntry, bool) {
	statePath := filepath.Join(runsDir, runID, phasedStateFile)
	data, err := os.ReadFile(statePath)
	if err != nil {
		return staleRunEntry{}, false
	}
	state, err := parsePhasedState(data)
	if err != nil || state.RunID == "" {
		return staleRunEntry{}, false
	}

	if state.TerminalStatus != "" {
		return checkTerminalRunStale(runID, root, statePath, state, minAge, now)
	}
	return checkNonTerminalRunStale(runID, root, statePath, state, minAge, now)
}

// findStaleRunsWithMinAge scans the registry for runs that are not active and
// not completed, optionally filtering to runs older than minAge.
func findStaleRunsWithMinAge(root string, minAge time.Duration, now time.Time) []staleRunEntry {
	runsDir := filepath.Join(root, ".agents", "rpi", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil
	}

	var stale []staleRunEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if sr, ok := classifyRunEntry(entry.Name(), root, runsDir, minAge, now); ok {
			stale = append(stale, sr)
		}
	}
	return stale
}

// checkTerminalRunStale returns a staleRunEntry for a terminal run if it qualifies for cleanup.
func checkTerminalRunStale(runID, root, statePath string, state *phasedState, minAge time.Duration, now time.Time) (staleRunEntry, bool) {
	if state.TerminalStatus == "completed" {
		return staleRunEntry{}, false
	}
	if state.WorktreePath == "" {
		return staleRunEntry{}, false
	}
	if _, statErr := os.Stat(state.WorktreePath); statErr != nil {
		return staleRunEntry{}, false
	}
	if minAge > 0 {
		candidateAt := cmp.Or(state.TerminatedAt, state.StartedAt)
		parsedAt, parseErr := time.Parse(time.RFC3339, candidateAt)
		if parseErr != nil || now.Sub(parsedAt) < minAge {
			return staleRunEntry{}, false
		}
	}
	reason := cmp.Or(state.TerminalReason, "terminal status: "+state.TerminalStatus)
	return staleRunEntry{
		runID:        runID,
		root:         root,
		statePath:    statePath,
		reason:       reason,
		worktreePath: state.WorktreePath,
		terminal:     state.TerminalStatus,
	}, true
}

// checkNonTerminalRunStale returns a staleRunEntry for an inactive, non-completed run if it qualifies.
func checkNonTerminalRunStale(runID, root, statePath string, state *phasedState, minAge time.Duration, now time.Time) (staleRunEntry, bool) {
	isActive, _ := determineRunLiveness(root, state)
	if isActive {
		return staleRunEntry{}, false
	}
	if state.Phase >= completedPhaseNumber(*state) {
		return staleRunEntry{}, false
	}
	if minAge > 0 {
		startedAt, parseErr := time.Parse(time.RFC3339, state.StartedAt)
		if parseErr != nil || now.Sub(startedAt) < minAge {
			return staleRunEntry{}, false
		}
	}
	reason := "no heartbeat, no tmux session"
	if state.WorktreePath != "" {
		if _, statErr := os.Stat(state.WorktreePath); statErr != nil {
			reason = "worktree missing"
		}
	}
	return staleRunEntry{
		runID:        runID,
		root:         root,
		statePath:    statePath,
		reason:       reason,
		worktreePath: state.WorktreePath,
	}, true
}

// updateFlatStateIfMatches updates the flat (root-level) state file with stale
// metadata when its run_id matches the given runID.
func updateFlatStateIfMatches(flatPath, runID, reason, terminatedAt string) {
	flatData, fErr := os.ReadFile(flatPath)
	if fErr != nil {
		return
	}
	var flatRaw map[string]any
	if json.Unmarshal(flatData, &flatRaw) != nil {
		return
	}
	if flatRunID, ok := flatRaw["run_id"].(string); !ok || flatRunID != runID {
		return
	}
	flatRaw["terminal_status"] = "stale"
	flatRaw["terminal_reason"] = reason
	flatRaw["terminated_at"] = terminatedAt
	if flatUpdated, mErr := json.MarshalIndent(flatRaw, "", "  "); mErr == nil {
		flatUpdated = append(flatUpdated, '\n')
		_ = writePhasedStateAtomic(flatPath, flatUpdated)
	}
}

// markRunStale writes terminal metadata to the state file.
func markRunStale(sr staleRunEntry) error {
	data, err := os.ReadFile(sr.statePath)
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	raw["terminal_status"] = "stale"
	raw["terminal_reason"] = sr.reason
	raw["terminated_at"] = time.Now().Format(time.RFC3339)

	updated, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	updated = append(updated, '\n')

	if err := writePhasedStateAtomic(sr.statePath, updated); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	flatPath := filepath.Join(sr.root, ".agents", "rpi", phasedStateFile)
	updateFlatStateIfMatches(flatPath, sr.runID, sr.reason, raw["terminated_at"].(string))

	return nil
}

// removeOrphanedWorktree removes a worktree directory and any legacy branch marker.
func removeOrphanedWorktree(repoRoot, worktreePath, runID string) error {
	// Safety: validate that worktreePath is a sibling of the repo root (same parent dir).
	// Worktrees are created as ../repo-rpi-<id>/ — siblings of the repo, not children.
	// This prevents corrupted state files from directing os.RemoveAll at unrelated paths.
	repoParent := filepath.Dir(filepath.Clean(repoRoot))
	wtParent := filepath.Dir(filepath.Clean(worktreePath))
	if wtParent != repoParent {
		return fmt.Errorf("worktree path %q is not a sibling of repo %q; refusing removal", worktreePath, repoRoot)
	}
	// Additional safety: never remove the repo root itself.
	cleanWT := filepath.Clean(worktreePath)
	if cleanWT == filepath.Clean(repoRoot) {
		return fmt.Errorf("worktree path %q is the repo root; refusing removal", worktreePath)
	}

	// Force remove the worktree.
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		// If worktree remove fails (already pruned), just remove the directory.
		if rmErr := os.RemoveAll(worktreePath); rmErr != nil {
			return fmt.Errorf("git worktree remove: %s; manual rm: %w", string(out), rmErr)
		}
	}

	// Delete legacy branch marker if present.
	if strings.TrimSpace(runID) != "" {
		branchName := "rpi/" + runID
		branchCmd := exec.Command("git", "branch", "-D", branchName)
		branchCmd.Dir = repoRoot
		_ = branchCmd.Run() // Best-effort; branch may not exist.
	}

	return nil
}

// pruneWorktrees runs `git worktree prune`.
func pruneWorktrees(cwd string) error {
	fmt.Println("Running: git worktree prune")
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
