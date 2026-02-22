package main

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	cliConfig "github.com/boshu2/agentops/cli/internal/config"
	cliRPI "github.com/boshu2/agentops/cli/internal/rpi"
)

// phasedEngineOptions captures all configurable parameters for runPhasedEngine.
// This allows the loop and other callers to invoke the phased engine programmatically
// without depending on global cobra flag variables.
type phasedEngineOptions struct {
	From                 string
	FastPath             bool
	TestFirst            bool
	Interactive          bool
	MaxRetries           int
	PhaseTimeout         time.Duration
	StallTimeout         time.Duration
	StreamStartupTimeout time.Duration
	NoWorktree           bool
	LiveStatus           bool
	SwarmFirst           bool
	AutoCleanStale       bool
	AutoCleanStaleAfter  time.Duration
	StallCheckInterval   time.Duration
	RuntimeMode          string
	RuntimeCommand       string
	AOCommand            string
	BDCommand            string
	TmuxCommand          string
}

// defaultPhasedEngineOptions returns options matching the default cobra flag values.
func defaultPhasedEngineOptions() phasedEngineOptions {
	return phasedEngineOptions{
		From:                 "discovery",
		MaxRetries:           3,
		PhaseTimeout:         90 * time.Minute,
		StallTimeout:         10 * time.Minute,
		StreamStartupTimeout: 45 * time.Second,
		SwarmFirst:           true,
		AutoCleanStaleAfter:  24 * time.Hour,
		StallCheckInterval:   30 * time.Second,
		RuntimeMode:          "auto",
		RuntimeCommand:       "claude",
		AOCommand:            "ao",
		BDCommand:            "bd",
		TmuxCommand:          "tmux",
	}
}

// Phase represents an RPI phase with its index and name.
type phase struct {
	Num  int
	Name string
	Step string // ratchet step name
}

var phases = []phase{
	{1, "discovery", "research"},
	{2, "implementation", "implement"},
	{3, "validation", "validate"},
}

// phasedState persists orchestrator state between phase spawns.
type phasedState struct {
	SchemaVersion   int                 `json:"schema_version"`
	Goal            string              `json:"goal"`
	EpicID          string              `json:"epic_id,omitempty"`
	Phase           int                 `json:"phase"`
	StartPhase      int                 `json:"start_phase"`
	Cycle           int                 `json:"cycle"`
	ParentEpic      string              `json:"parent_epic,omitempty"`
	FastPath        bool                `json:"fast_path"`
	TestFirst       bool                `json:"test_first"`
	SwarmFirst      bool                `json:"swarm_first"`
	Complexity      ComplexityLevel     `json:"complexity,omitempty"` // fast, standard, full
	Verdicts        map[string]string   `json:"verdicts"`
	Attempts        map[string]int      `json:"attempts"`
	StartedAt       string              `json:"started_at"`
	WorktreePath    string              `json:"worktree_path,omitempty"`
	RunID           string              `json:"run_id,omitempty"`
	OrchestratorPID int                 `json:"orchestrator_pid,omitempty"`
	Backend         string              `json:"backend,omitempty"`
	TerminalStatus  string              `json:"terminal_status,omitempty"` // interrupted, failed, stale, completed
	TerminalReason  string              `json:"terminal_reason,omitempty"`
	TerminatedAt    string              `json:"terminated_at,omitempty"`
	Opts            phasedEngineOptions `json:"opts"`
}

// retryContext holds context for retrying a failed gate.
type retryContext struct {
	Attempt  int
	Findings []finding
	Verdict  string
}

// finding represents a structured finding from a council report.
type finding struct {
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Ref         string `json:"ref"`
}

// phaseSummaryInstruction is prepended to each phase prompt so Claude writes a rich summary.
// Placed first so it survives context compaction (early instructions persist longer).
const phaseSummaryInstruction = `PHASE SUMMARY CONTRACT: Before finishing this session, write a concise summary (max 500 tokens) to .agents/rpi/phase-{{.PhaseNum}}-summary.md covering key insights, tradeoffs considered, and risks for subsequent phases. This file is read by the next phase.

`

// contextDisciplineInstruction is prepended to every phase prompt to prevent compaction.
// CONTEXT DISCIPLINE: This constant exists so the CLI can enforce context-aware behavior.
const contextDisciplineInstruction = `CONTEXT DISCIPLINE: You are running inside ao rpi phased (phase {{.PhaseNum}} of 3). Each phase gets a FRESH context window. Stay disciplined:
- Do NOT accumulate large file contents in context. Read files with the Read tool JIT and extract only what you need.
- Do NOT explore broadly when narrow exploration suffices. Be surgical.
- Write findings, plans, and results to DISK (files in .agents/), not just in conversation.
- If you are delegating to workers or spawning agents, do NOT accumulate their full output. Read their result files from disk.
- If you notice context degradation (forgetting earlier instructions, repeating yourself, losing track of the goal), IMMEDIATELY write a handoff to .agents/rpi/phase-{{.PhaseNum}}-handoff.md with: (1) what you accomplished, (2) what remains, (3) key context. Then finish cleanly.
{{.ContextBudget}}
`

// phaseContextBudgets provides phase-specific context guidance.
var phaseContextBudgets = map[int]string{
	1: "BUDGET: This session runs research + plan + pre-mortem. Research: limit to ~15 file reads, write findings to .agents/research/. Plan: write to .agents/plans/, focus on issue creation. Pre-mortem: invoke /council, read the verdict, done. If pre-mortem FAILs, re-plan and re-run pre-mortem within this session (max 3 attempts).",
	2: "BUDGET (CRITICAL): Crank is the highest-risk phase for context. /crank spawns workers internally. Do NOT re-read worker output into your context. Trust /crank to manage its waves. Read only the completion status.",
	3: "BUDGET: This session runs vibe + post-mortem. Vibe: invoke /council on recent changes, read the verdict. Post-mortem: invoke /council + /retro, read output files, write summary. Minimal context for both.",
}

// phasePrompts defines Go templates for each phase's Claude invocation.
// Phase 1 (discovery) chains research + plan + pre-mortem in a single session
// for prompt cache reuse. Phase 2 (implementation) gets a fresh context window.
// Phase 3 (validation) chains vibe + post-mortem with fresh eyes.
var phasePrompts = map[int]string{
	// Discovery: research → plan → pre-mortem (all in one session)
	1: `{{if .SwarmFirst}}SWARM-FIRST EXECUTION CONTRACT:
- Default to /swarm for each step in this phase (research, plan, pre-mortem) using a lead + worker team pattern.
- If /swarm runtime is unavailable, execute the direct commands below in this same session.
- Keep worker outputs on disk and consume thin summaries only.

{{end}}Run these skills IN SEQUENCE. Do not skip any step.

STEP 1 — Research:
{{if .SwarmFirst}}Prefer: execute this step via /swarm with research-focused workers.
Fallback direct command:
{{end}}/research "{{.Goal}}"{{if not .Interactive}} --auto{{end}}

STEP 2 — Plan:
After research completes, run:
{{if .SwarmFirst}}Prefer: execute this step via /swarm with planning/decomposition workers.
Fallback direct command:
{{end}}/plan "{{.Goal}}"{{if not .Interactive}} --auto{{end}}

STEP 3 — Pre-mortem:
After plan completes, run:
{{if .SwarmFirst}}Prefer: execute this step via /swarm (including council/critique workers when available).
Fallback direct command:
{{end}}/pre-mortem{{if .FastPath}} --quick{{end}}

If pre-mortem returns FAIL, re-run /plan with the findings and then /pre-mortem again. Max 3 total attempts. If still FAIL after 3 attempts, stop and report.
	If pre-mortem returns PASS or WARN, proceed.`,

	// Implementation: crank (single skill, fresh context)
	2: `{{if .SwarmFirst}}SWARM-FIRST EXECUTION CONTRACT:
- Run implementation with swarm-managed waves by default (lead + worker teams).
- Prefer crank paths that delegate to /swarm for wave execution.

{{end}}/crank {{.EpicID}}{{if .TestFirst}} --test-first{{end}}`,

	// Validation: vibe → post-mortem (both in one session, fresh eyes)
	3: `{{if .SwarmFirst}}SWARM-FIRST EXECUTION CONTRACT:
- Use swarm/team execution for validation and retrospective steps where available.
- Keep validator and implementer contexts isolated; do not reuse implementation worker context.

{{end}}Run these skills IN SEQUENCE. Do not skip any step.

STEP 1 — Vibe:
{{if .SwarmFirst}}Prefer: execute vibe using /swarm-driven validation workers.
Fallback direct command:
{{end}}/vibe{{if .FastPath}} --quick{{end}} recent

If vibe returns FAIL, STOP and report the findings. Do NOT proceed to post-mortem.
If vibe returns PASS or WARN, proceed.

STEP 2 — Post-mortem:
{{if .SwarmFirst}}Prefer: execute post-mortem using /swarm-driven retro workers.
Fallback direct command:
{{end}}/post-mortem{{if .FastPath}} --quick{{end}} {{.EpicID}}`,
}

// retryPrompts defines templates for retry invocations with feedback context.
// Phase 1 retries are handled WITHIN the session (the prompt instructs Claude to retry).
// Phase 3 (validation) FAIL triggers a fresh phase 2 (implementation) session.
var retryPrompts = map[int]string{
	// Vibe FAIL → re-crank with feedback (spawns fresh implementation session)
	3: `/crank {{.EpicID}}{{if .TestFirst}} --test-first{{end}}` + "\n\n" +
		`Vibe FAIL (attempt {{.RetryAttempt}}/{{.MaxRetries}}). Address these findings:` + "\n" +
		`{{range .Findings}}FINDING: {{.Description}} | FIX: {{.Fix}} | REF: {{.Ref}}` + "\n" + `{{end}}`,
}

// resolveWorktreeModeFromConfig checks the agentops config for rpi.worktree_mode
// and returns the effective NoWorktree value.
func resolveWorktreeModeFromConfig(flagDefault bool) bool {
	cfg, err := cliConfig.Load(nil)
	if err != nil {
		return flagDefault
	}
	switch cfg.RPI.WorktreeMode {
	case "never":
		return true
	case "always":
		return false
	default: // "auto" or empty
		return flagDefault
	}
}

func normalizeRuntimeMode(mode string) string {
	return cmp.Or(strings.ToLower(strings.TrimSpace(mode)), "auto")
}

func effectiveRuntimeCommand(command string) string {
	return cmp.Or(strings.TrimSpace(command), "claude")
}

func effectiveAOCommand(command string) string {
	return cmp.Or(strings.TrimSpace(command), "ao")
}

func effectiveBDCommand(command string) string {
	return cmp.Or(strings.TrimSpace(command), "bd")
}

func effectiveTmuxCommand(command string) string {
	return cmp.Or(strings.TrimSpace(command), "tmux")
}

func validateRuntimeMode(mode string) error {
	switch normalizeRuntimeMode(mode) {
	case "auto", "direct", "stream":
		return nil
	default:
		return fmt.Errorf("invalid runtime %q (valid: auto|direct|stream)", mode)
	}
}

// renderPreambleInstructions renders the context-discipline and summary-contract
// instruction templates into the prompt builder. These are placed first so they
// survive context compaction.
func renderPreambleInstructions(prompt *strings.Builder, data any) {
	disciplineTmpl, err := template.New("discipline").Parse(contextDisciplineInstruction)
	if err == nil {
		if execErr := disciplineTmpl.Execute(prompt, data); execErr != nil {
			VerbosePrintf("Warning: could not render context discipline instruction: %v\n", execErr)
		}
	}
	summaryTmpl, err := template.New("summary").Parse(phaseSummaryInstruction)
	if err == nil {
		if execErr := summaryTmpl.Execute(prompt, data); execErr != nil {
			VerbosePrintf("Warning: could not render summary instruction: %v\n", execErr)
		}
	}
}

// buildPromptForPhase constructs the Claude invocation prompt for a phase.
func buildPromptForPhase(cwd string, phaseNum int, state *phasedState, _ *retryContext) (string, error) {
	tmplStr, ok := phasePrompts[phaseNum]
	if !ok {
		return "", fmt.Errorf("no prompt template for phase %d", phaseNum)
	}

	tmpl, err := template.New("phase").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := struct {
		Goal          string
		EpicID        string
		FastPath      bool
		TestFirst     bool
		SwarmFirst    bool
		Interactive   bool
		PhaseNum      int
		ContextBudget string
	}{
		Goal:          state.Goal,
		EpicID:        state.EpicID,
		FastPath:      state.FastPath,
		TestFirst:     state.TestFirst,
		SwarmFirst:    state.SwarmFirst,
		Interactive:   state.Opts.Interactive,
		PhaseNum:      phaseNum,
		ContextBudget: phaseContextBudgets[phaseNum],
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	var prompt strings.Builder
	renderPreambleInstructions(&prompt, data)

	// Cross-phase context for phases 2+ (goal, verdicts, prior summaries)
	if phaseNum >= 2 {
		ctx := buildPhaseContext(cwd, state, phaseNum)
		if ctx != "" {
			prompt.WriteString(ctx)
			prompt.WriteString("\n\n")
		}
	}

	prompt.WriteString(buf.String())
	return prompt.String(), nil
}

// buildPhaseContext constructs a context block from goal, verdicts, and prior phase summaries.
func buildPhaseContext(cwd string, state *phasedState, phaseNum int) string {
	var parts []string

	// Always include the goal
	if state.Goal != "" {
		parts = append(parts, fmt.Sprintf("Goal: %s", state.Goal))
	}

	// Include prior verdicts
	for key, verdict := range state.Verdicts {
		parts = append(parts, fmt.Sprintf("%s verdict: %s", strings.ReplaceAll(key, "_", "-"), verdict))
	}

	// Include prior phase summaries (read from disk)
	if cwd != "" {
		summaries := readPhaseSummaries(cwd, phaseNum)
		if summaries != "" {
			parts = append(parts, summaries)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "--- RPI Context (from prior phases) ---\n" + strings.Join(parts, "\n")
}

// readPhaseSummaries reads all phase summary files prior to the given phase.
func readPhaseSummaries(cwd string, currentPhase int) string {
	var summaries []string
	rpiDir := filepath.Join(cwd, ".agents", "rpi")

	for i := 1; i < currentPhase; i++ {
		path := filepath.Join(rpiDir, fmt.Sprintf("phase-%d-summary.md", i))
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		// Cap each summary to prevent context bloat
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		phaseName := "unknown"
		if i > 0 && i <= len(phases) {
			phaseName = phases[i-1].Name
		}
		summaries = append(summaries, fmt.Sprintf("[Phase %d: %s]\n%s", i, phaseName, content))
	}

	if len(summaries) == 0 {
		return ""
	}
	return strings.Join(summaries, "\n\n")
}

// buildRetryPrompt constructs a retry prompt with feedback context.
func buildRetryPrompt(cwd string, phaseNum int, state *phasedState, retryCtx *retryContext) (string, error) {
	tmplStr, ok := retryPrompts[phaseNum]
	if !ok {
		// No retry template — fall back to normal prompt
		return buildPromptForPhase(cwd, phaseNum, state, retryCtx)
	}

	tmpl, err := template.New("retry").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse retry template: %w", err)
	}

	data := struct {
		Goal          string
		EpicID        string
		FastPath      bool
		TestFirst     bool
		RetryAttempt  int
		MaxRetries    int
		Findings      []finding
		PhaseNum      int
		ContextBudget string
	}{
		Goal:          state.Goal,
		EpicID:        state.EpicID,
		FastPath:      state.FastPath,
		TestFirst:     state.TestFirst,
		RetryAttempt:  retryCtx.Attempt,
		MaxRetries:    state.Opts.MaxRetries,
		Findings:      retryCtx.Findings,
		PhaseNum:      phaseNum,
		ContextBudget: phaseContextBudgets[phaseNum],
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute retry template: %w", err)
	}

	skillInvocation := buf.String()

	// Build prompt: context discipline and summary contract first (survive compaction),
	// then the retry skill invocation.
	var prompt strings.Builder

	// 1. Context discipline instruction (first — survives compaction)
	disciplineTmpl, err := template.New("discipline").Parse(contextDisciplineInstruction)
	if err == nil {
		if err := disciplineTmpl.Execute(&prompt, data); err != nil {
			VerbosePrintf("Warning: could not render context discipline instruction: %v\n", err)
		}
	}

	// 2. Summary instruction
	summaryTmpl, err := template.New("summary").Parse(phaseSummaryInstruction)
	if err == nil {
		if err := summaryTmpl.Execute(&prompt, data); err != nil {
			VerbosePrintf("Warning: could not render summary instruction: %v\n", err)
		}
	}

	// 3. Retry skill invocation (last — the actual command with findings)
	prompt.WriteString(skillInvocation)

	return prompt.String(), nil
}

// phaseNameToNum converts a phase name to a consolidated phase number (1-3).
func phaseNameToNum(name string) int {
	normalized := strings.ToLower(strings.TrimSpace(name))
	aliases := map[string]int{
		// Canonical 3-phase names
		"discovery":      1,
		"implementation": 2,
		"validation":     3,
		// Backward-compatible aliases (old 6-phase names map to consolidated phases)
		"research":    1,
		"plan":        1,
		"pre-mortem":  1,
		"premortem":   1,
		"pre_mortem":  1,
		"crank":       2,
		"implement":   2,
		"vibe":        3,
		"validate":    3,
		"post-mortem": 3,
		"postmortem":  3,
		"post_mortem": 3,
	}
	return aliases[normalized]
}

// worktreeTimeout is the timeout for git worktree operations (matches Olympus DefaultTimeout).
const worktreeTimeout = 30 * time.Second

// generateRunID returns a 12-char lowercase hex string from crypto/rand.
func generateRunID() string {
	return cliRPI.GenerateRunID()
}

// getCurrentBranch returns the current branch name, or error if detached HEAD.
func getCurrentBranch(repoRoot string) (string, error) {
	return cliRPI.GetCurrentBranch(repoRoot, worktreeTimeout)
}

// createWorktree creates a sibling git worktree for isolated RPI execution.
// Path: ../<repo-basename>-rpi-<runID>/
func createWorktree(cwd string) (worktreePath, runID string, err error) {
	return cliRPI.CreateWorktree(cwd, worktreeTimeout, VerbosePrintf)
}

// mergeWorktree merges the RPI worktree branch back into the original branch.
// Retries the pre-merge dirty check with backoff to handle the race where
// another parallel run is mid-merge (repo momentarily dirty).
func mergeWorktree(repoRoot, worktreePath, runID string) error {
	return cliRPI.MergeWorktree(repoRoot, worktreePath, runID, worktreeTimeout, VerbosePrintf)
}

// removeWorktree removes a worktree directory and any legacy branch marker.
// Modeled on Olympus internal/git/worktree.go Remove().
func removeWorktree(repoRoot, worktreePath, runID string) error {
	return cliRPI.RemoveWorktree(repoRoot, worktreePath, runID, worktreeTimeout)
}
