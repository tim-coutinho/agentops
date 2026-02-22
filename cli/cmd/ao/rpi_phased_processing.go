package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

// gateFailError signals a gate check failure that may be retried.
type gateFailError struct {
	Phase    int
	Verdict  string
	Findings []finding
	Report   string
}

func (e *gateFailError) Error() string {
	return fmt.Sprintf("gate FAIL at phase %d: %s (report: %s)", e.Phase, e.Verdict, e.Report)
}

// postPhaseProcessing handles phase-specific post-processing.
func postPhaseProcessing(cwd string, state *phasedState, phaseNum int, logPath string) error {
	switch phaseNum {
	case 1:
		return processDiscoveryPhase(cwd, state, logPath)
	case 2:
		return processImplementationPhase(cwd, state, phaseNum, logPath)
	case 3:
		return processValidationPhase(cwd, state, phaseNum, logPath)
	}
	return nil
}

// processDiscoveryPhase handles post-processing for the discovery phase.
// It extracts the epic ID, detects fast path, and checks the pre-mortem verdict.
func processDiscoveryPhase(cwd string, state *phasedState, logPath string) error {
	epicID, err := extractEpicID(state.Opts.BDCommand)
	if err != nil {
		return fmt.Errorf("discovery phase: could not extract epic ID (implementation needs this): %w", err)
	}
	state.EpicID = epicID
	fmt.Printf("Epic ID: %s\n", epicID)
	logPhaseTransition(logPath, state.RunID, "discovery", fmt.Sprintf("extracted epic: %s", epicID))

	if !state.Opts.FastPath {
		fast, err := detectFastPath(state.EpicID, state.Opts.BDCommand)
		if err != nil {
			VerbosePrintf("Warning: fast-path detection failed (continuing without): %v\n", err)
		} else if fast {
			state.FastPath = true
			fmt.Println("Micro-epic detected — using fast path (--quick for gates)")
		}
	}

	report, err := findLatestCouncilReport(cwd, "pre-mortem", time.Time{}, state.EpicID)
	if err != nil {
		// Pre-mortem may not have run if the session handled retries internally
		// and ultimately gave up. Check if council report exists at all.
		VerbosePrintf("Warning: pre-mortem council report not found (session may have handled retries internally): %v\n", err)
		return nil
	}
	verdict, err := extractCouncilVerdict(report)
	if err != nil {
		VerbosePrintf("Warning: could not extract pre-mortem verdict: %v\n", err)
		return nil
	}
	state.Verdicts["pre_mortem"] = verdict
	fmt.Printf("Pre-mortem verdict: %s\n", verdict)
	logPhaseTransition(logPath, state.RunID, "discovery", fmt.Sprintf("pre-mortem verdict: %s report=%s", verdict, report))

	if verdict == "FAIL" {
		// Discovery session was instructed to retry internally.
		// If we still see FAIL here, it means all retries failed.
		findings, _ := extractCouncilFindings(report, 5)
		return &gateFailError{Phase: 1, Verdict: verdict, Findings: findings, Report: report}
	}
	return nil
}

// processImplementationPhase handles post-processing for the implementation phase.
// It validates the prior phase result and checks crank completion status.
func processImplementationPhase(cwd string, state *phasedState, phaseNum int, logPath string) error {
	if state.StartPhase <= 1 {
		if err := validatePriorPhaseResult(cwd, 1); err != nil {
			return fmt.Errorf("phase %d prerequisite not met: %w", phaseNum, err)
		}
	}
	if state.EpicID == "" {
		return nil
	}
	status, err := checkCrankCompletion(state.EpicID, state.Opts.BDCommand)
	if err != nil {
		VerbosePrintf("Warning: could not check crank completion (continuing to validation): %v\n", err)
		return nil
	}
	fmt.Printf("Crank status: %s\n", status)
	logPhaseTransition(logPath, state.RunID, "implementation", fmt.Sprintf("crank status: %s", status))
	if status == "BLOCKED" || status == "PARTIAL" {
		return &gateFailError{Phase: 2, Verdict: status, Report: "bd children " + state.EpicID}
	}
	return nil
}

// processValidationPhase handles post-processing for the validation phase.
// It validates the prior phase result, checks the vibe verdict, and optionally extracts
// the post-mortem verdict.
func processValidationPhase(cwd string, state *phasedState, phaseNum int, logPath string) error {
	if state.StartPhase <= 2 {
		if err := validatePriorPhaseResult(cwd, 2); err != nil {
			return fmt.Errorf("phase %d prerequisite not met: %w", phaseNum, err)
		}
	}
	report, err := findLatestCouncilReport(cwd, "vibe", time.Time{}, state.EpicID)
	if err != nil {
		return fmt.Errorf("validation phase: vibe report not found (phase may not have completed): %w", err)
	}
	verdict, err := extractCouncilVerdict(report)
	if err != nil {
		return fmt.Errorf("validation phase: could not extract vibe verdict from %s: %w", report, err)
	}
	state.Verdicts["vibe"] = verdict
	fmt.Printf("Vibe verdict: %s\n", verdict)
	logPhaseTransition(logPath, state.RunID, "validation", fmt.Sprintf("vibe verdict: %s report=%s", verdict, report))

	if verdict == "FAIL" {
		findings, _ := extractCouncilFindings(report, 5)
		return &gateFailError{Phase: 3, Verdict: verdict, Findings: findings, Report: report}
	}

	// Also extract post-mortem verdict if available (non-blocking)
	pmReport, err := findLatestCouncilReport(cwd, "post-mortem", time.Time{}, state.EpicID)
	if err == nil {
		pmVerdict, err := extractCouncilVerdict(pmReport)
		if err == nil {
			state.Verdicts["post_mortem"] = pmVerdict
			fmt.Printf("Post-mortem verdict: %s\n", pmVerdict)
			logPhaseTransition(logPath, state.RunID, "validation", fmt.Sprintf("post-mortem verdict: %s report=%s", pmVerdict, pmReport))
		}
	}
	return nil
}

func legacyGateAction(attempt, maxRetries int) types.MemRLAction {
	if attempt >= maxRetries {
		return types.MemRLActionEscalate
	}
	return types.MemRLActionRetry
}

func classifyGateFailureClass(phaseNum int, gateErr *gateFailError) types.MemRLFailureClass {
	if gateErr == nil {
		return ""
	}
	verdict := strings.ToUpper(strings.TrimSpace(gateErr.Verdict))
	if fc := classifyByPhase(phaseNum, verdict); fc != "" {
		return fc
	}
	return classifyByVerdict(verdict)
}

func classifyByPhase(phaseNum int, verdict string) types.MemRLFailureClass {
	switch phaseNum {
	case 1:
		if verdict == "FAIL" {
			return types.MemRLFailureClassPreMortemFail
		}
	case 2:
		switch verdict {
		case "BLOCKED":
			return types.MemRLFailureClassCrankBlocked
		case "PARTIAL":
			return types.MemRLFailureClassCrankPartial
		}
	case 3:
		if verdict == "FAIL" {
			return types.MemRLFailureClassVibeFail
		}
	}
	return ""
}

func classifyByVerdict(verdict string) types.MemRLFailureClass {
	switch verdict {
	case string(failReasonTimeout):
		return types.MemRLFailureClassPhaseTimeout
	case string(failReasonStall):
		return types.MemRLFailureClassPhaseStall
	case string(failReasonExit):
		return types.MemRLFailureClassPhaseExitError
	default:
		return types.MemRLFailureClass(strings.ToLower(verdict))
	}
}

func resolveGateRetryAction(state *phasedState, phaseNum int, gateErr *gateFailError, attempt int) (types.MemRLAction, types.MemRLPolicyDecision) {
	mode := types.GetMemRLMode()
	failureClass := classifyGateFailureClass(phaseNum, gateErr)
	metadataPresent := gateErr != nil && strings.TrimSpace(gateErr.Verdict) != ""

	decision := types.EvaluateDefaultMemRLPolicy(types.MemRLPolicyInput{
		Mode:            mode,
		FailureClass:    failureClass,
		Attempt:         attempt,
		MaxAttempts:     state.Opts.MaxRetries,
		MetadataPresent: metadataPresent,
	})

	legacy := legacyGateAction(attempt, state.Opts.MaxRetries)
	if mode == types.MemRLModeEnforce {
		return decision.Action, decision
	}
	return legacy, decision
}

// handleGateRetry manages retry logic for failed gates.
// spawnCwd is the working directory for spawned claude sessions (may be worktree).
func handleGateRetry(cwd string, state *phasedState, phaseNum int, gateErr *gateFailError, logPath string, spawnCwd string, statusPath string, allPhases []PhaseProgress, executor PhaseExecutor) (bool, error) {
	phaseName := phases[phaseNum-1].Name
	attemptKey := fmt.Sprintf("phase_%d", phaseNum)

	state.Attempts[attemptKey]++
	attempt := state.Attempts[attemptKey]
	maybeUpdateLiveStatus(state, statusPath, allPhases, phaseNum, "retrying after "+gateErr.Verdict, attempt, "")

	action, decision := resolveGateRetryAction(state, phaseNum, gateErr, attempt)
	logGateRetryMemRL(logPath, state.RunID, phaseName, decision, action)

	if action == types.MemRLActionEscalate {
		return performGateEscalation(state, phaseNum, attempt, gateErr, decision, action, phaseName, logPath, statusPath, allPhases)
	}

	fmt.Printf("%s: %s (attempt %d/%d) — retrying\n", phaseName, gateErr.Verdict, attempt, state.Opts.MaxRetries)
	logPhaseTransition(logPath, state.RunID, phaseName, fmt.Sprintf("RETRY attempt %d/%d verdict=%s report=%s", attempt, state.Opts.MaxRetries, gateErr.Verdict, gateErr.Report))

	retryCtx := &retryContext{
		Attempt:  attempt,
		Findings: gateErr.Findings,
		Verdict:  gateErr.Verdict,
	}

	retryPrompt, err := buildRetryPrompt(cwd, phaseNum, state, retryCtx)
	if err != nil {
		return false, fmt.Errorf("build retry prompt: %w", err)
	}

	if GetDryRun() {
		fmt.Printf("[dry-run] Would spawn retry: %s -p '%s'\n", effectiveRuntimeCommand(state.Opts.RuntimeCommand), retryPrompt)
		return false, nil
	}

	if err := executeWithStatus(executor, state, statusPath, allPhases, phaseNum, attempt, retryPrompt, spawnCwd, "running retry prompt", "retry failed"); err != nil {
		return false, fmt.Errorf("retry failed: %w", err)
	}

	rerunPrompt, err := buildPromptForPhase(cwd, phaseNum, state, nil)
	if err != nil {
		return false, fmt.Errorf("build rerun prompt: %w", err)
	}

	fmt.Printf("Re-running phase %d after retry\n", phaseNum)
	if err := executeWithStatus(executor, state, statusPath, allPhases, phaseNum, attempt, rerunPrompt, spawnCwd, "re-running phase", "rerun failed"); err != nil {
		return false, fmt.Errorf("rerun failed: %w", err)
	}

	return verifyGateAfterRetry(cwd, state, phaseNum, logPath, spawnCwd, statusPath, allPhases, executor, attempt)
}

func maybeUpdateLiveStatus(state *phasedState, statusPath string, allPhases []PhaseProgress, phaseNum int, status string, attempt int, errMsg string) {
	if state.Opts.LiveStatus {
		updateLivePhaseStatus(statusPath, allPhases, phaseNum, status, attempt, errMsg)
	}
}

func executeWithStatus(executor PhaseExecutor, state *phasedState, statusPath string, allPhases []PhaseProgress, phaseNum, attempt int, prompt, spawnCwd, runningMsg, failedMsg string) error {
	maybeUpdateLiveStatus(state, statusPath, allPhases, phaseNum, runningMsg, attempt, "")
	if err := executor.Execute(prompt, spawnCwd, state.RunID, phaseNum); err != nil {
		maybeUpdateLiveStatus(state, statusPath, allPhases, phaseNum, failedMsg, attempt, err.Error())
		return err
	}
	return nil
}

// logGateRetryMemRL logs the MemRL policy decision for a gate retry, if mode is not off.
func logGateRetryMemRL(logPath, runID, phaseName string, decision types.MemRLPolicyDecision, action types.MemRLAction) {
	if decision.Mode == types.MemRLModeOff {
		return
	}
	logPhaseTransition(
		logPath,
		runID,
		phaseName,
		fmt.Sprintf(
			"memrl policy mode=%s failure_class=%s attempt_bucket=%s policy_action=%s selected_action=%s rule=%s",
			decision.Mode,
			decision.FailureClass,
			decision.AttemptBucket,
			decision.Action,
			action,
			decision.RuleID,
		),
	)
}

// performGateEscalation handles the escalation path when the retry action is escalate.
// Returns (false, nil) to signal escalation without error (caller will handle reporting).
func performGateEscalation(state *phasedState, phaseNum, attempt int, gateErr *gateFailError, decision types.MemRLPolicyDecision, action types.MemRLAction, phaseName, logPath, statusPath string, allPhases []PhaseProgress) (bool, error) {
	msg := fmt.Sprintf(
		"%s escalated (mode=%s, action=%s, rule=%s, attempt=%d/%d). Last report: %s. Manual intervention needed.",
		phaseName,
		decision.Mode,
		action,
		decision.RuleID,
		attempt,
		state.Opts.MaxRetries,
		gateErr.Report,
	)
	fmt.Println(msg)
	if state.Opts.LiveStatus {
		updateLivePhaseStatus(statusPath, allPhases, phaseNum, "failed after retries", attempt, gateErr.Report)
	}
	logPhaseTransition(logPath, state.RunID, phaseName, msg)
	return false, nil
}

// verifyGateAfterRetry re-checks the gate after a retry session completes.
// If the gate still fails, it recurses into handleGateRetry.
func verifyGateAfterRetry(cwd string, state *phasedState, phaseNum int, logPath, spawnCwd, statusPath string, allPhases []PhaseProgress, executor PhaseExecutor, attempt int) (bool, error) {
	if err := postPhaseProcessing(cwd, state, phaseNum, logPath); err != nil {
		var gateErr *gateFailError
		if errors.As(err, &gateErr) {
			// Still failing — recurse
			return handleGateRetry(cwd, state, phaseNum, gateErr, logPath, spawnCwd, statusPath, allPhases, executor)
		}
		return false, err
	}
	if state.Opts.LiveStatus {
		updateLivePhaseStatus(statusPath, allPhases, phaseNum, "retry succeeded", attempt, "")
	}
	return true, nil
}

// --- Verdict extraction helpers ---

// extractCouncilVerdict reads a council report and returns the verdict (PASS/WARN/FAIL).
func extractCouncilVerdict(reportPath string) (string, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return "", fmt.Errorf("read report: %w", err)
	}

	re := regexp.MustCompile(`(?m)^## Council Verdict:\s*(PASS|WARN|FAIL)`)
	matches := re.FindSubmatch(data)
	if len(matches) < 2 {
		return "", fmt.Errorf("no verdict found in %s", reportPath)
	}
	return string(matches[1]), nil
}

// findLatestCouncilReport finds the most recent council report matching a pattern.
// When epicID is non-empty, reports whose filename contains the epicID are preferred.
// If no epic-scoped report is found, all pattern-matching reports are used as fallback.
func findLatestCouncilReport(cwd string, pattern string, notBefore time.Time, epicID string) (string, error) {
	councilDir := filepath.Join(cwd, ".agents", "council")
	entries, err := os.ReadDir(councilDir)
	if err != nil {
		return "", fmt.Errorf("read council directory: %w", err)
	}

	var matches []string
	var epicMatches []string
	for _, entry := range entries {
		fullPath, ok := matchCouncilEntry(entry, councilDir, pattern, notBefore)
		if !ok {
			continue
		}
		matches = append(matches, fullPath)
		if epicID != "" && strings.Contains(entry.Name(), epicID) {
			epicMatches = append(epicMatches, fullPath)
		}
	}

	selected := matches
	if len(epicMatches) > 0 {
		selected = epicMatches
	}

	if len(selected) == 0 {
		return "", fmt.Errorf("no council report matching %q found", pattern)
	}

	sort.Strings(selected)

	return selected[len(selected)-1], nil
}

func matchCouncilEntry(entry os.DirEntry, councilDir, pattern string, notBefore time.Time) (string, bool) {
	if entry.IsDir() {
		return "", false
	}
	name := entry.Name()
	if !strings.Contains(name, pattern) || !strings.HasSuffix(name, ".md") {
		return "", false
	}
	if !notBefore.IsZero() {
		info, err := entry.Info()
		if err != nil || info.ModTime().Before(notBefore) {
			return "", false
		}
	}
	return filepath.Join(councilDir, name), true
}

// extractCouncilFindings extracts structured findings from a council report.
func extractCouncilFindings(reportPath string, max int) ([]finding, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("read report: %w", err)
	}

	// Look for structured findings: FINDING: ... | FIX: ... | REF: ...
	re := regexp.MustCompile(`(?m)FINDING:\s*(.+?)\s*\|\s*FIX:\s*(.+?)\s*\|\s*REF:\s*(.+?)$`)
	allMatches := re.FindAllSubmatch(data, -1)

	var findings []finding
	for i, m := range allMatches {
		if i >= max {
			break
		}
		findings = append(findings, finding{
			Description: string(m[1]),
			Fix:         string(m[2]),
			Ref:         string(m[3]),
		})
	}

	// Fallback: if no structured findings, extract from "## Shared Findings" section
	if len(findings) == 0 {
		re2 := regexp.MustCompile(`(?m)^\d+\.\s+\*\*(.+?)\*\*\s*[—–-]\s*(.+)$`)
		allMatches2 := re2.FindAllSubmatch(data, -1)
		for i, m := range allMatches2 {
			if i >= max {
				break
			}
			findings = append(findings, finding{
				Description: string(m[1]) + ": " + string(m[2]),
				Fix:         "See council report",
				Ref:         reportPath,
			})
		}
	}

	return findings, nil
}

// --- Epic and completion helpers ---

// extractEpicID finds the most recently created open epic ID via bd CLI.
// bd list returns epics in creation order; we take the LAST match so that
// the epic just created by the plan phase is selected over older ones.
func extractEpicID(bdCommand string) (string, error) {
	command := effectiveBDCommand(bdCommand)

	// Prefer JSON output for prefix-agnostic parsing.
	cmd := exec.Command(command, "list", "--type", "epic", "--status", "open", "--json")
	out, err := cmd.Output()
	if err == nil {
		epicID, parseErr := parseLatestEpicIDFromJSON(out)
		if parseErr == nil {
			return epicID, nil
		}
		VerbosePrintf("Warning: could not parse bd JSON epic list (falling back to text): %v\n", parseErr)
	} else {
		VerbosePrintf("Warning: bd list --json failed (falling back to text): %v\n", err)
	}

	// Fallback for older bd builds that do not support JSON output.
	cmd = exec.Command(command, "list", "--type", "epic", "--status", "open")
	out, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bd list: %w", err)
	}
	return parseLatestEpicIDFromText(string(out))
}

func parseLatestEpicIDFromJSON(data []byte) (string, error) {
	var entries []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", fmt.Errorf("parse bd list JSON: %w", err)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		epicID := strings.TrimSpace(entries[i].ID)
		if epicID != "" {
			return epicID, nil
		}
	}
	return "", fmt.Errorf("no epic found in bd list output")
}

func parseLatestEpicIDFromText(output string) (string, error) {
	// Allow custom prefixes (bd-*, ag-*, etc.) and keep the match anchored
	// to issue-like tokens near the start of each line.
	idPattern := regexp.MustCompile(`^[a-z][a-z0-9]*-[a-z0-9][a-z0-9.]*$`)

	var latest string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		limit := len(fields)
		if limit > 3 {
			limit = 3
		}
		for i := range limit {
			field := fields[i]
			token := strings.Trim(field, "[]()")
			if idPattern.MatchString(token) {
				latest = token
				break
			}
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no epic found in bd list output")
	}
	return latest, nil
}

// detectFastPath checks if an epic is a micro-epic (≤2 issues, no blockers).
func detectFastPath(epicID string, bdCommand string) (bool, error) {
	cmd := exec.Command(effectiveBDCommand(bdCommand), "children", epicID)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("bd children: %w", err)
	}
	return parseFastPath(string(out)), nil
}

// parseFastPath determines if bd children output indicates a micro-epic.
func parseFastPath(output string) bool {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	issueCount := 0
	blockedCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		issueCount++
		if strings.Contains(strings.ToLower(line), "blocked") {
			blockedCount++
		}
	}
	return issueCount <= 2 && blockedCount == 0
}

// checkCrankCompletion checks epic completion via bd children statuses.
// Returns "DONE", "BLOCKED", or "PARTIAL".
func checkCrankCompletion(epicID string, bdCommand string) (string, error) {
	cmd := exec.Command(effectiveBDCommand(bdCommand), "children", epicID)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bd children: %w", err)
	}
	return parseCrankCompletion(string(out)), nil
}

// parseCrankCompletion determines completion status from bd children output.
func parseCrankCompletion(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	total := 0
	closed := 0
	blocked := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		total++
		lower := strings.ToLower(line)
		if strings.Contains(lower, "closed") || strings.Contains(lower, "✓") {
			closed++
		}
		if strings.Contains(lower, "blocked") {
			blocked++
		}
	}

	if total == 0 {
		return "DONE"
	}
	if closed == total {
		return "DONE"
	}
	if blocked > 0 {
		return "BLOCKED"
	}
	return "PARTIAL"
}

// --- Phase summaries ---

// writePhaseSummary writes a fallback summary only if Claude didn't write one.
func writePhaseSummary(cwd string, state *phasedState, phaseNum int) {
	rpiDir := filepath.Join(cwd, ".agents", "rpi")
	path := filepath.Join(rpiDir, fmt.Sprintf("phase-%d-summary.md", phaseNum))

	// If Claude already wrote a summary, keep it (it's richer than our mechanical one)
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Phase %d: Claude-written summary found\n", phaseNum)
		return
	}
	fmt.Printf("Phase %d: no Claude summary found, writing fallback\n", phaseNum)

	if err := os.MkdirAll(rpiDir, 0755); err != nil {
		VerbosePrintf("Warning: could not create rpi dir for summary: %v\n", err)
		return
	}

	summary := generatePhaseSummary(state, phaseNum)
	if summary == "" {
		return
	}

	if err := os.WriteFile(path, []byte(summary), 0644); err != nil {
		VerbosePrintf("Warning: could not write phase summary: %v\n", err)
	}
}

// generatePhaseSummary produces a concise summary of what a phase accomplished.
func generatePhaseSummary(state *phasedState, phaseNum int) string {
	switch phaseNum {
	case 1: // Discovery (research + plan + pre-mortem)
		summary := fmt.Sprintf("Discovery completed for goal: %s\n", state.Goal)
		summary += "Research: see .agents/research/ for findings.\n"
		if state.EpicID != "" {
			summary += fmt.Sprintf("Plan: epic %s", state.EpicID)
			if state.FastPath {
				summary += " (micro-epic, fast path)"
			}
			summary += "\n"
		}
		verdict := state.Verdicts["pre_mortem"]
		if verdict != "" {
			summary += fmt.Sprintf("Pre-mortem verdict: %s\nSee .agents/council/*pre-mortem*.md for details.", verdict)
		}
		return summary
	case 2: // Implementation (crank)
		return fmt.Sprintf("Crank completed for epic %s.\nCheck bd children %s for issue statuses.", state.EpicID, state.EpicID)
	case 3: // Validation (vibe + post-mortem)
		summary := ""
		vibeVerdict := state.Verdicts["vibe"]
		if vibeVerdict != "" {
			summary += fmt.Sprintf("Vibe verdict: %s\nSee .agents/council/*vibe*.md for details.\n", vibeVerdict)
		}
		pmVerdict := state.Verdicts["post_mortem"]
		if pmVerdict != "" {
			summary += fmt.Sprintf("Post-mortem verdict: %s\n", pmVerdict)
		}
		summary += "See .agents/council/*post-mortem*.md and .agents/learnings/ for extracted knowledge."
		return summary
	}
	return ""
}

// handoffDetected checks if a phase wrote a handoff file (context degradation signal).
func handoffDetected(cwd string, phaseNum int) bool {
	path := filepath.Join(cwd, ".agents", "rpi", fmt.Sprintf("phase-%d-handoff.md", phaseNum))
	_, err := os.Stat(path)
	return err == nil
}

// cleanPhaseSummaries removes stale phase summaries and handoffs from a prior run.
func cleanPhaseSummaries(stateDir string) {
	for i := 1; i <= len(phases); i++ {
		path := filepath.Join(stateDir, fmt.Sprintf("phase-%d-summary.md", i))
		os.Remove(path) //nolint:errcheck
		handoffPath := filepath.Join(stateDir, fmt.Sprintf("phase-%d-handoff.md", i))
		os.Remove(handoffPath) //nolint:errcheck
		resultPath := filepath.Join(stateDir, fmt.Sprintf("phase-%d-result.json", i))
		os.Remove(resultPath) //nolint:errcheck
	}
}

// --- Phase result artifacts ---

// phaseResultFileFmt is the filename pattern for per-phase result artifacts.
// Each phase writes "phase-{N}-result.json" to .agents/rpi/.
// Contract: docs/contracts/rpi-phase-result.schema.json
const phaseResultFileFmt = "phase-%d-result.json"

// phaseResult is a structured artifact written after each phase completes or fails.
// Schema: docs/contracts/rpi-phase-result.schema.json
type phaseResult struct {
	SchemaVersion   int               `json:"schema_version"`
	RunID           string            `json:"run_id"`
	Phase           int               `json:"phase"`
	PhaseName       string            `json:"phase_name"`
	Status          string            `json:"status"`
	Retries         int               `json:"retries,omitempty"`
	Error           string            `json:"error,omitempty"`
	Backend         string            `json:"backend,omitempty"`
	Artifacts       map[string]string `json:"artifacts,omitempty"`
	Verdicts        map[string]string `json:"verdicts,omitempty"`
	StartedAt       string            `json:"started_at"`
	CompletedAt     string            `json:"completed_at,omitempty"`
	DurationSeconds float64           `json:"duration_seconds,omitempty"`
}

// writePhaseResult writes a phase-result.json artifact (named phase-{N}-result.json) atomically (write to .tmp, rename).
func writePhaseResult(cwd string, result *phaseResult) error {
	stateDir := filepath.Join(cwd, ".agents", "rpi")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal phase result: %w", err)
	}

	finalPath := filepath.Join(stateDir, fmt.Sprintf(phaseResultFileFmt, result.Phase))
	tmpPath := finalPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write phase result tmp: %w", err)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("rename phase result: %w", err)
	}

	VerbosePrintf("Phase result written to %s\n", finalPath)
	return nil
}

// validatePriorPhaseResult checks that phase-{expectedPhase}-result.json exists
// and has status "completed". Called at the start of phases 2 and 3.
func validatePriorPhaseResult(cwd string, expectedPhase int) error {
	resultPath := filepath.Join(cwd, ".agents", "rpi", fmt.Sprintf(phaseResultFileFmt, expectedPhase))
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return fmt.Errorf("prior phase %d result not found at %s: %w", expectedPhase, resultPath, err)
	}

	var result phaseResult
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("prior phase %d result is malformed: %w", expectedPhase, err)
	}

	if result.Status != "completed" {
		return fmt.Errorf("prior phase %d has status %q (expected %q)", expectedPhase, result.Status, "completed")
	}

	return nil
}

// --- State persistence ---

const phasedStateFile = "phased-state.json"

// rpiRunRegistryDir returns the per-run registry directory path.
// All per-run artifacts (state, heartbeat) are written here so the registry
// survives interruption and supports resume/status lookup.
// Path: .agents/rpi/runs/<run-id>/
func rpiRunRegistryDir(cwd, runID string) string {
	if runID == "" {
		return ""
	}
	return filepath.Join(cwd, ".agents", "rpi", "runs", runID)
}

// savePhasedState writes orchestrator state to disk atomically.
// The state is written to two locations:
//  1. .agents/rpi/phased-state.json (legacy flat path for backward compatibility)
//  2. .agents/rpi/runs/<run-id>/state.json (per-run registry directory)
//
// Both writes use the tmp+rename pattern to prevent corrupt partial writes.
func savePhasedState(cwd string, state *phasedState) error {
	stateDir := filepath.Join(cwd, ".agents", "rpi")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	data = append(data, '\n')

	// Atomic write to flat path (backward compatible).
	flatPath := filepath.Join(stateDir, phasedStateFile)
	if err := writePhasedStateAtomic(flatPath, data); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	// Also write to per-run registry directory when run ID is available.
	if state.RunID != "" {
		runDir := rpiRunRegistryDir(cwd, state.RunID)
		if mkErr := os.MkdirAll(runDir, 0755); mkErr != nil {
			VerbosePrintf("Warning: create run registry dir: %v\n", mkErr)
		} else {
			registryPath := filepath.Join(runDir, phasedStateFile)
			if wErr := writePhasedStateAtomic(registryPath, data); wErr != nil {
				VerbosePrintf("Warning: write run registry state: %v\n", wErr)
			}
		}
	}

	VerbosePrintf("State saved to %s\n", flatPath)
	return nil
}

// writePhasedStateAtomic writes data to path using a tmp-file+rename pattern.
// This ensures readers never observe a partial write.
func writePhasedStateAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".phased-state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create tmp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename tmp: %w", err)
	}
	cleanup = false
	return nil
}

// loadPhasedState reads orchestrator state from disk.
// It first tries the per-run registry directory (most recent run), then falls
// back to the flat .agents/rpi/phased-state.json path for backward compatibility.
func loadPhasedState(cwd string) (*phasedState, error) {
	flatPath := filepath.Join(cwd, ".agents", "rpi", phasedStateFile)

	// Try to find the most recently modified state in any run registry directory.
	// This allows resume when the worktree only has the registry (not the flat file).
	runState, runErr := loadLatestRunRegistryState(cwd)
	if runErr == nil && runState != nil {
		// Prefer registry state only when it is newer than (or the same as) the flat file.
		flatInfo, flatStatErr := os.Stat(flatPath)
		if flatStatErr != nil {
			// Flat file does not exist — use registry state.
			return runState, nil
		}
		registryPath := filepath.Join(rpiRunRegistryDir(cwd, runState.RunID), phasedStateFile)
		registryInfo, regStatErr := os.Stat(registryPath)
		if regStatErr == nil && !registryInfo.ModTime().Before(flatInfo.ModTime()) {
			return runState, nil
		}
	}

	// Fall back to flat path.
	data, err := os.ReadFile(flatPath)
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	return parsePhasedState(data)
}

// loadLatestRunRegistryState scans .agents/rpi/runs/ and returns the state
// from the most recently modified run directory, or nil if none exists.
func loadLatestRunRegistryState(cwd string) (*phasedState, error) {
	runsDir := filepath.Join(cwd, ".agents", "rpi", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, err
	}

	var latestModTime int64
	var latestData []byte

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		statePath := filepath.Join(runsDir, entry.Name(), phasedStateFile)
		info, err := os.Stat(statePath)
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > latestModTime {
			data, readErr := os.ReadFile(statePath)
			if readErr != nil {
				continue
			}
			latestModTime = info.ModTime().UnixNano()
			latestData = data
		}
	}

	if latestData == nil {
		return nil, os.ErrNotExist
	}
	return parsePhasedState(latestData)
}

// parsePhasedState parses JSON bytes into a phasedState with nil-safe maps.
func parsePhasedState(data []byte) (*phasedState, error) {
	var state phasedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	// Ensure maps are never nil after deserialization.
	if state.Verdicts == nil {
		state.Verdicts = make(map[string]string)
	}
	if state.Attempts == nil {
		state.Attempts = make(map[string]int)
	}

	return &state, nil
}

// updateRunHeartbeat writes the current UTC timestamp to
// .agents/rpi/runs/<run-id>/heartbeat.txt atomically.
// It is called during phase execution to signal the run is alive.
// Failures are logged but do not abort the phase.
func updateRunHeartbeat(cwd, runID string) {
	if runID == "" {
		return
	}
	runDir := rpiRunRegistryDir(cwd, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		VerbosePrintf("Warning: create run dir for heartbeat: %v\n", err)
		return
	}
	heartbeatPath := filepath.Join(runDir, "heartbeat.txt")
	ts := time.Now().UTC().Format(time.RFC3339Nano) + "\n"
	if err := writePhasedStateAtomic(heartbeatPath, []byte(ts)); err != nil {
		VerbosePrintf("Warning: update heartbeat: %v\n", err)
	}
}

// readRunHeartbeat returns the last heartbeat timestamp for a run, or zero
// time if the heartbeat file does not exist or cannot be parsed.
func readRunHeartbeat(cwd, runID string) time.Time {
	if runID == "" {
		return time.Time{}
	}
	heartbeatPath := filepath.Join(rpiRunRegistryDir(cwd, runID), "heartbeat.txt")
	data, err := os.ReadFile(heartbeatPath)
	if err != nil {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}
	}
	return ts
}

// --- Ratchet and logging ---

// recordRatchetCheckpoint records a ratchet checkpoint for a phase.
func recordRatchetCheckpoint(step, aoCommand string) {
	cmd := exec.Command(effectiveAOCommand(aoCommand), "ratchet", "record", step)
	if err := cmd.Run(); err != nil {
		VerbosePrintf("Warning: ratchet record %s: %v\n", step, err)
	}
}

// logPhaseTransition appends a log entry to the orchestration log.
func logPhaseTransition(logPath, runID, phase, details string) {
	var entry string
	if runID != "" {
		entry = fmt.Sprintf("[%s] [%s] %s: %s\n", time.Now().Format(time.RFC3339), runID, phase, details)
	} else {
		entry = fmt.Sprintf("[%s] %s: %s\n", time.Now().Format(time.RFC3339), phase, details)
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		VerbosePrintf("Warning: could not write orchestration log: %v\n", err)
		return
	}
	defer f.Close() //nolint:errcheck

	if _, err := f.WriteString(entry); err != nil {
		VerbosePrintf("Warning: could not write log entry: %v\n", err)
		return
	}

	// Mirror transitions into append-only ledger and refresh per-run cache.
	// This keeps mutable status files as cache while preserving provenance.
	maybeAppendRPILedgerTransition(logPath, runID, phase, details)
}

func maybeAppendRPILedgerTransition(logPath, runID, phase, details string) {
	if runID == "" {
		return
	}
	rootDir, ok := deriveRepoRootFromRPIOrchestrationLog(logPath)
	if !ok {
		return
	}

	event := rpiLedgerEvent{
		RunID:  runID,
		Phase:  phase,
		Action: ledgerActionFromDetails(details),
		Details: map[string]any{
			"details": details,
		},
	}

	if _, err := appendRPILedgerEvent(rootDir, event); err != nil {
		VerbosePrintf("Warning: could not append RPI ledger event: %v\n", err)
		return
	}
	if err := materializeRPIRunCache(rootDir, runID); err != nil {
		VerbosePrintf("Warning: could not materialize RPI run cache: %v\n", err)
	}
}

func deriveRepoRootFromRPIOrchestrationLog(logPath string) (string, bool) {
	rpiDir := filepath.Dir(filepath.Clean(logPath))
	if filepath.Base(rpiDir) != "rpi" {
		return "", false
	}
	agentsDir := filepath.Dir(rpiDir)
	if filepath.Base(agentsDir) != ".agents" {
		return "", false
	}
	return filepath.Dir(agentsDir), true
}

var ledgerPrefixActions = []struct {
	prefix string
	action string
}{
	{"started", "started"},
	{"completed", "completed"},
	{"failed:", "failed"},
	{"fatal:", "fatal"},
	{"retry", "retry"},
	{"dry-run", "dry-run"},
	{"handoff", "handoff"},
	{"epic=", "summary"},
}

func ledgerActionFromDetails(details string) string {
	normalized := strings.ToLower(strings.TrimSpace(details))
	if normalized == "" {
		return "event"
	}
	for _, pa := range ledgerPrefixActions {
		if strings.HasPrefix(normalized, pa.prefix) {
			return pa.action
		}
	}
	fields := strings.Fields(normalized)
	return cmp.Or(strings.Trim(fields[0], ":"), "event")
}

// logFailureContext records actionable remediation context when a phase fails.
func logFailureContext(logPath, runID, phase string, err error) {
	logPhaseTransition(logPath, runID, phase, fmt.Sprintf("FAILURE_CONTEXT: %v | action: check .agents/rpi/ for phase artifacts, review .agents/council/ for verdicts", err))
}
