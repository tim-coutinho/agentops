package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FireState holds the current state of the FIRE loop
type FireState struct {
	EpicID   string   `json:"epic_id"`
	Rig      string   `json:"rig"`
	Ready    []string `json:"ready"`   // Issues that can be ignited
	Burning  []string `json:"burning"` // Issues in_progress
	Reaped   []string `json:"reaped"`  // Issues closed
	Blocked  []string `json:"blocked"` // Issues waiting on deps
	ConvoyID string   `json:"convoy_id"`
}

// RetryInfo tracks retry state for an issue
type RetryInfo struct {
	IssueID      string    `json:"issue_id"`
	Attempt      int       `json:"attempt"`
	LastAttempt  time.Time `json:"last_attempt"`
	NextAttempt  time.Time `json:"next_attempt"`
	FailureNotes []string  `json:"failure_notes"`
}

// FireConfig configures the FIRE loop
type FireConfig struct {
	EpicID       string
	Rig          string
	MaxPolecats  int
	PollInterval time.Duration
	MaxRetries   int
	BackoffBase  time.Duration
}

// DefaultFireConfig returns sensible defaults
func DefaultFireConfig() FireConfig {
	return FireConfig{
		MaxPolecats:  4,
		PollInterval: 30 * time.Second,
		MaxRetries:   3,
		BackoffBase:  30 * time.Second,
	}
}

// =============================================================================
// FIRE Loop Entry Point
// =============================================================================

// runFireIteration executes one FIND-IGNITE-REAP-ESCALATE cycle.
// Returns true when the loop should exit (all issues complete).
func runFireIteration(cfg FireConfig, retryQueue map[string]*RetryInfo) (bool, error) {
	state, err := findPhase(cfg.EpicID)
	if err != nil {
		return false, fmt.Errorf("FIND failed: %w", err)
	}

	if isComplete(state) {
		fmt.Println("✅ FIRE complete: all issues closed")
		return true, nil
	}

	printState(state)

	ignited, err := ignitePhase(state, cfg, retryQueue)
	if err != nil {
		fmt.Printf("⚠️  IGNITE error: %v\n", err)
	}
	if len(ignited) > 0 {
		fmt.Printf("🚀 Ignited: %v\n", ignited)
	}

	reaped, failures, err := reapPhase(state)
	if err != nil {
		fmt.Printf("⚠️  REAP error: %v\n", err)
	}
	if len(reaped) > 0 {
		fmt.Printf("✅ Reaped: %v\n", reaped)
	}

	escalated, err := escalatePhase(failures, retryQueue, cfg)
	if err != nil {
		fmt.Printf("⚠️  ESCALATE error: %v\n", err)
	}
	if len(escalated) > 0 {
		fmt.Printf("🚨 Escalated: %v\n", escalated)
	}

	return false, nil
}

// RunFireLoop runs the autonomous FIRE loop until completion
func RunFireLoop(cfg FireConfig) error {
	fmt.Printf("🔥 FIRE Loop starting for epic %s on rig %s\n", cfg.EpicID, cfg.Rig)
	fmt.Printf("   Max polecats: %d, Poll interval: %s\n", cfg.MaxPolecats, cfg.PollInterval)

	retryQueue := make(map[string]*RetryInfo)
	iteration := 0

	for {
		iteration++
		fmt.Printf("\n━━━ FIRE Iteration %d ━━━\n", iteration)

		done, err := runFireIteration(cfg, retryQueue)
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		fmt.Printf("💤 Sleeping %s...\n", cfg.PollInterval)
		time.Sleep(cfg.PollInterval)
	}
}

// =============================================================================
// FIND Phase
// =============================================================================

func findPhase(epicID string) (*FireState, error) {
	state := &FireState{
		EpicID: epicID,
	}

	// Get ready issues
	ready, err := bdReady(epicID)
	if err != nil {
		return nil, fmt.Errorf("bd ready: %w", err)
	}
	state.Ready = ready

	// Get in_progress issues
	burning, err := bdListByStatus(epicID, "in_progress")
	if err != nil {
		return nil, fmt.Errorf("bd list in_progress: %w", err)
	}
	state.Burning = burning

	// Get closed issues
	reaped, err := bdListByStatus(epicID, "closed")
	if err != nil {
		return nil, fmt.Errorf("bd list closed: %w", err)
	}
	state.Reaped = reaped

	// Get blocked issues
	blocked, err := bdBlocked(epicID)
	if err != nil {
		// Non-fatal - blocked detection is best-effort
		VerbosePrintf("Warning: bd blocked failed: %v\n", err)
	}
	state.Blocked = blocked

	return state, nil
}

// =============================================================================
// IGNITE Phase
// =============================================================================

// collectDueRetries returns retries from the queue whose NextAttempt has passed, up to capacity.
// Matching entries are removed from retryQueue.
func collectDueRetries(retryQueue map[string]*RetryInfo, capacity int) []string {
	now := time.Now()
	var due []string
	for issueID, info := range retryQueue {
		if now.After(info.NextAttempt) {
			due = append(due, issueID)
			delete(retryQueue, issueID)
			if len(due) >= capacity {
				break
			}
		}
	}
	return due
}

// containsString reports whether slice contains s.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// collectReadyIssues appends fresh ready issues not already queued, up to capacity.
func collectReadyIssues(ready []string, already []string, capacity int) []string {
	var out []string
	out = append(out, already...)
	for _, issueID := range ready {
		if containsString(out, issueID) {
			continue
		}
		out = append(out, issueID)
		if len(out) >= capacity {
			break
		}
	}
	return out
}

// slingIssues spawns polecats for each issue via gt sling, returning successfully ignited IDs.
func slingIssues(toIgnite []string, rig string) []string {
	var ignited []string
	for _, issueID := range toIgnite {
		if err := gtSling(issueID, rig); err != nil {
			fmt.Printf("⚠️  Failed to sling %s: %v\n", issueID, err)
			continue
		}
		ignited = append(ignited, issueID)
	}
	return ignited
}

func ignitePhase(state *FireState, cfg FireConfig, retryQueue map[string]*RetryInfo) ([]string, error) {
	capacity := cfg.MaxPolecats - len(state.Burning)
	if capacity <= 0 {
		VerbosePrintf("At capacity (%d burning, max %d)\n", len(state.Burning), cfg.MaxPolecats)
		return nil, nil
	}

	toIgnite := collectDueRetries(retryQueue, capacity)
	toIgnite = collectReadyIssues(state.Ready, toIgnite, capacity)

	if len(toIgnite) == 0 {
		return nil, nil
	}

	return slingIssues(toIgnite, cfg.Rig), nil
}

// =============================================================================
// REAP Phase
// =============================================================================

func reapPhase(state *FireState) ([]string, []string, error) {
	var reaped []string
	var failures []string

	// Check each burning issue for completion
	for _, issueID := range state.Burning {
		status, err := bdShowStatus(issueID)
		if err != nil {
			VerbosePrintf("Warning: couldn't check %s: %v\n", issueID, err)
			continue
		}

		switch status {
		case "closed":
			reaped = append(reaped, issueID)
		case "ready", "pending":
			// Issue was reset - treat as failure
			failures = append(failures, issueID)
		}
		// "in_progress" means still burning - no action
	}

	return reaped, failures, nil
}

// =============================================================================
// ESCALATE Phase
// =============================================================================

func escalatePhase(failures []string, retryQueue map[string]*RetryInfo, cfg FireConfig) ([]string, error) {
	var escalated []string

	for _, issueID := range failures {
		info, exists := retryQueue[issueID]
		if !exists {
			info = &RetryInfo{
				IssueID: issueID,
				Attempt: 0,
			}
		}

		info.Attempt++
		info.LastAttempt = time.Now()

		if info.Attempt >= cfg.MaxRetries {
			// ESCALATE - mark as blocker
			if err := bdAddLabel(issueID, "BLOCKER"); err != nil {
				fmt.Printf("⚠️  Failed to add BLOCKER label to %s: %v\n", issueID, err)
			}

			// Send mail to human
			msg := fmt.Sprintf("AUTO-ESCALATED: %s failed %d attempts. Human review required.", issueID, info.Attempt)
			if err := sendMail("mayor", msg, "blocker"); err != nil {
				fmt.Printf("⚠️  Failed to mail escalation for %s: %v\n", issueID, err)
			}

			escalated = append(escalated, issueID)
			delete(retryQueue, issueID)
		} else {
			// Schedule retry with exponential backoff
			backoff := cfg.BackoffBase * time.Duration(1<<(info.Attempt-1))
			info.NextAttempt = time.Now().Add(backoff)
			retryQueue[issueID] = info

			fmt.Printf("📅 Scheduled retry for %s in %s (attempt %d/%d)\n",
				issueID, backoff, info.Attempt, cfg.MaxRetries)
		}
	}

	return escalated, nil
}

// =============================================================================
// Helper: bd CLI calls
// =============================================================================

func bdReady(epicID string) ([]string, error) {
	args := []string{"ready", "-o", "json"}
	if epicID != "" {
		args = append(args, "--parent", epicID)
	}

	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseBeadIDs(output)
}

func bdListByStatus(epicID, status string) ([]string, error) {
	args := []string{"list", "--status", status, "-o", "json"}
	if epicID != "" {
		args = append(args, "--parent", epicID)
	}

	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseBeadIDs(output)
}

func bdBlocked(epicID string) ([]string, error) {
	args := []string{"blocked", "-o", "json"}
	if epicID != "" {
		args = append(args, "--parent", epicID)
	}

	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseBeadIDs(output)
}

func bdShowStatus(issueID string) (string, error) {
	cmd := exec.Command("bd", "show", issueID, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var bead struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(output, &bead); err != nil {
		return "", err
	}

	return bead.Status, nil
}

func bdAddLabel(issueID, label string) error {
	cmd := exec.Command("bd", "update", issueID, "--labels", label)
	return cmd.Run()
}

func parseBeadIDs(jsonOutput []byte) ([]string, error) {
	if len(jsonOutput) == 0 {
		return nil, nil
	}

	var beads []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(jsonOutput, &beads); err != nil {
		// Try parsing as single object
		var single struct {
			ID string `json:"id"`
		}
		if err2 := json.Unmarshal(jsonOutput, &single); err2 != nil {
			return nil, err
		}
		if single.ID != "" {
			return []string{single.ID}, nil
		}
		return nil, nil
	}

	ids := make([]string, 0, len(beads))
	for _, b := range beads {
		ids = append(ids, b.ID)
	}
	return ids, nil
}

// =============================================================================
// Helper: gt CLI calls
// =============================================================================

func gtSling(issueID, rig string) error {
	args := []string{"sling", issueID}
	if rig != "" {
		args = append(args, rig)
	}

	VerbosePrintf("Running: gt %s\n", strings.Join(args, " "))

	cmd := exec.Command("gt", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// =============================================================================
// Helper: ao mail
// =============================================================================

func sendMail(to, body, msgType string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	msg := Message{
		From:      "fire-loop",
		To:        to,
		Body:      body,
		Type:      msgType,
		Timestamp: time.Now(),
	}

	mailDir := filepath.Join(cwd, ".agents", "mail")
	if err := os.MkdirAll(mailDir, 0700); err != nil {
		return err
	}

	messagesPath := filepath.Join(mailDir, "messages.jsonl")
	f, err := os.OpenFile(messagesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // write already flushed, close best-effort
	}()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// =============================================================================
// Helper: State
// =============================================================================

func isComplete(state *FireState) bool {
	return len(state.Ready) == 0 && len(state.Burning) == 0
}

func printState(state *FireState) {
	total := len(state.Ready) + len(state.Burning) + len(state.Reaped) + len(state.Blocked)
	fmt.Printf("📊 State: %d ready, %d burning, %d reaped, %d blocked (total: %d)\n",
		len(state.Ready), len(state.Burning), len(state.Reaped), len(state.Blocked), total)
}
