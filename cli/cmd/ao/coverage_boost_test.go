package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// rpi_phased.go — normalizeOptsCommands
// ---------------------------------------------------------------------------

func TestRPI_NormalizeOptsCommands(t *testing.T) {
	tests := []struct {
		name       string
		opts       phasedEngineOptions
		wantRT     string // RuntimeMode after normalisation
		wantCmd    string // RuntimeCommand after normalisation
		wantAO     string
		wantBD     string
		wantTmux   string
	}{
		{
			name:     "all empty → defaults",
			opts:     phasedEngineOptions{},
			wantRT:   "auto",
			wantCmd:  "claude",
			wantAO:   "ao",
			wantBD:   "bd",
			wantTmux: "tmux",
		},
		{
			name: "whitespace-only → defaults",
			opts: phasedEngineOptions{
				RuntimeMode:    "  ",
				RuntimeCommand: "  ",
				AOCommand:      "  ",
				BDCommand:      "  ",
				TmuxCommand:    "  ",
			},
			wantRT:   "auto",
			wantCmd:  "claude",
			wantAO:   "ao",
			wantBD:   "bd",
			wantTmux: "tmux",
		},
		{
			name: "custom values preserved",
			opts: phasedEngineOptions{
				RuntimeMode:    "STREAM",
				RuntimeCommand: "/usr/bin/claude",
				AOCommand:      "/opt/ao",
				BDCommand:      "/opt/bd",
				TmuxCommand:    "/opt/tmux",
			},
			wantRT:   "stream",
			wantCmd:  "/usr/bin/claude",
			wantAO:   "/opt/ao",
			wantBD:   "/opt/bd",
			wantTmux: "/opt/tmux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.opts
			normalizeOptsCommands(&opts)
			if opts.RuntimeMode != tt.wantRT {
				t.Errorf("RuntimeMode = %q, want %q", opts.RuntimeMode, tt.wantRT)
			}
			if opts.RuntimeCommand != tt.wantCmd {
				t.Errorf("RuntimeCommand = %q, want %q", opts.RuntimeCommand, tt.wantCmd)
			}
			if opts.AOCommand != tt.wantAO {
				t.Errorf("AOCommand = %q, want %q", opts.AOCommand, tt.wantAO)
			}
			if opts.BDCommand != tt.wantBD {
				t.Errorf("BDCommand = %q, want %q", opts.BDCommand, tt.wantBD)
			}
			if opts.TmuxCommand != tt.wantTmux {
				t.Errorf("TmuxCommand = %q, want %q", opts.TmuxCommand, tt.wantTmux)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rpi_phased.go — applyComplexityFastPath
// ---------------------------------------------------------------------------

func TestRPI_ApplyComplexityFastPath(t *testing.T) {
	tests := []struct {
		name             string
		goal             string
		optsFastPath     bool
		wantStateFP      bool
		wantComplexity   ComplexityLevel
	}{
		{
			name:           "trivial goal → auto fast-path",
			goal:           "fix typo",
			optsFastPath:   false,
			wantStateFP:    true,
			wantComplexity: ComplexityFast,
		},
		{
			name:           "complex goal → no fast-path",
			goal:           "refactor the entire authentication system across all modules",
			optsFastPath:   false,
			wantStateFP:    false,
			wantComplexity: ComplexityFull,
		},
		{
			name:           "trivial but opts already fast → stays as-is",
			goal:           "fix typo",
			optsFastPath:   true,
			wantStateFP:    true, // doesn't toggle it OFF
			wantComplexity: ComplexityFast,
		},
		{
			name:           "standard-length goal",
			goal:           "add user authentication with JWT tokens and refresh",
			optsFastPath:   false,
			wantStateFP:    false,
			wantComplexity: ComplexityStandard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &phasedState{
				Goal:     tt.goal,
				FastPath: tt.optsFastPath,
				Verdicts: make(map[string]string),
				Attempts: make(map[string]int),
			}
			opts := phasedEngineOptions{FastPath: tt.optsFastPath}
			applyComplexityFastPath(state, opts)
			if state.Complexity != tt.wantComplexity {
				t.Errorf("Complexity = %q, want %q", state.Complexity, tt.wantComplexity)
			}
			if state.FastPath != tt.wantStateFP {
				t.Errorf("FastPath = %v, want %v", state.FastPath, tt.wantStateFP)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rpi_phased.go — saveTerminalState
// ---------------------------------------------------------------------------

func TestRPI_SaveTerminalState(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".agents", "rpi")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	state := newTestPhasedState().WithGoal("test").WithRunID("run-term")
	saveTerminalState(tmpDir, state, "failed", "phase 2 error")

	if state.TerminalStatus != "failed" {
		t.Errorf("TerminalStatus = %q, want %q", state.TerminalStatus, "failed")
	}
	if state.TerminalReason != "phase 2 error" {
		t.Errorf("TerminalReason = %q, want %q", state.TerminalReason, "phase 2 error")
	}
	if state.TerminatedAt == "" {
		t.Error("TerminatedAt should be set")
	}
	// Verify it was persisted
	loaded, err := loadPhasedState(tmpDir)
	if err != nil {
		t.Fatalf("loadPhasedState: %v", err)
	}
	if loaded.TerminalStatus != "failed" {
		t.Errorf("loaded TerminalStatus = %q, want %q", loaded.TerminalStatus, "failed")
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_setup.go — mergeExistingStateFields
// ---------------------------------------------------------------------------

func TestRPI_MergeExistingStateFields(t *testing.T) {
	t.Run("merges all fields from existing", func(t *testing.T) {
		state := newTestPhasedState().WithGoal("current goal")
		existing := newTestPhasedState().
			WithEpicID("ag-x1").
			WithFastPath(true).
			WithSwarmFirst(true)
		existing.Verdicts["pre_mortem"] = "PASS"
		existing.Attempts["phase_1"] = 2

		mergeExistingStateFields(state, existing, phasedEngineOptions{}, "non-empty goal")

		if state.EpicID != "ag-x1" {
			t.Errorf("EpicID = %q, want %q", state.EpicID, "ag-x1")
		}
		if !state.FastPath {
			t.Error("FastPath should be true (from existing)")
		}
		if !state.SwarmFirst {
			t.Error("SwarmFirst should be true (from existing)")
		}
		if state.Verdicts["pre_mortem"] != "PASS" {
			t.Errorf("Verdicts[pre_mortem] = %q, want PASS", state.Verdicts["pre_mortem"])
		}
		if state.Attempts["phase_1"] != 2 {
			t.Errorf("Attempts[phase_1] = %d, want 2", state.Attempts["phase_1"])
		}
		// When goal arg is non-empty, state.Goal is NOT touched by mergeExistingStateFields
		if state.Goal != "current goal" {
			t.Errorf("Goal = %q, want %q (should remain unchanged)", state.Goal, "current goal")
		}
	})

	t.Run("empty goal inherits from existing", func(t *testing.T) {
		state := newTestPhasedState()
		existing := newTestPhasedState().WithGoal("old goal").WithEpicID("ag-y1")

		mergeExistingStateFields(state, existing, phasedEngineOptions{}, "")

		if state.Goal != "old goal" {
			t.Errorf("Goal = %q, want %q (inherited from existing)", state.Goal, "old goal")
		}
	})

	t.Run("opts fast-path OR'd with existing", func(t *testing.T) {
		state := newTestPhasedState()
		existing := newTestPhasedState()

		// Existing has FastPath=false, but opts has FastPath=true
		mergeExistingStateFields(state, existing, phasedEngineOptions{FastPath: true}, "goal")
		if !state.FastPath {
			t.Error("FastPath should be true from opts even when existing is false")
		}
	})

	t.Run("nil verdicts/attempts from existing are skipped", func(t *testing.T) {
		state := newTestPhasedState()
		state.Verdicts["vibe"] = "WARN"
		existing := &phasedState{
			EpicID:   "ag-z1",
			Verdicts: nil,
			Attempts: nil,
		}

		mergeExistingStateFields(state, existing, phasedEngineOptions{}, "goal")

		// Original verdicts should remain since existing.Verdicts is nil
		if state.Verdicts["vibe"] != "WARN" {
			t.Errorf("Verdicts should retain original when existing is nil, got: %v", state.Verdicts)
		}
	})
}

// ---------------------------------------------------------------------------
// rpi_phased_setup.go — resolveExistingWorktree
// ---------------------------------------------------------------------------

func TestRPI_ResolveExistingWorktree(t *testing.T) {
	t.Run("NoWorktree returns empty", func(t *testing.T) {
		state := newTestPhasedState()
		existing := newTestPhasedState().WithWorktreePath("/some/path")
		got, err := resolveExistingWorktree(state, existing, phasedEngineOptions{NoWorktree: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("empty existing worktree returns empty", func(t *testing.T) {
		state := newTestPhasedState()
		existing := newTestPhasedState() // no WorktreePath
		got, err := resolveExistingWorktree(state, existing, phasedEngineOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("existing worktree path that exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		state := newTestPhasedState()
		existing := newTestPhasedState().WithWorktreePath(tmpDir).WithRunID("run-wt")

		got, err := resolveExistingWorktree(state, existing, phasedEngineOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != tmpDir {
			t.Errorf("expected %q, got %q", tmpDir, got)
		}
		if state.WorktreePath != tmpDir {
			t.Errorf("state.WorktreePath = %q, want %q", state.WorktreePath, tmpDir)
		}
		if state.RunID != "run-wt" {
			t.Errorf("state.RunID = %q, want %q", state.RunID, "run-wt")
		}
	})

	t.Run("existing worktree path that does not exist returns error", func(t *testing.T) {
		state := newTestPhasedState()
		existing := newTestPhasedState().WithWorktreePath("/nonexistent/path/12345")

		_, err := resolveExistingWorktree(state, existing, phasedEngineOptions{})
		if err == nil {
			t.Fatal("expected error for nonexistent worktree path")
		}
		if !strings.Contains(err.Error(), "no longer exists") {
			t.Errorf("error message %q should mention 'no longer exists'", err.Error())
		}
	})
}

// ---------------------------------------------------------------------------
// rpi_phased_setup.go — ensureStateRunID
// ---------------------------------------------------------------------------

func TestRPI_EnsureStateRunID(t *testing.T) {
	t.Run("generates ID when empty", func(t *testing.T) {
		state := &phasedState{}
		ensureStateRunID(state)
		if state.RunID == "" {
			t.Error("expected RunID to be generated")
		}
		if len(state.RunID) != 8 { // 4 bytes = 8 hex chars
			t.Errorf("RunID length = %d, want 8 hex chars", len(state.RunID))
		}
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		state := &phasedState{RunID: "existing-id"}
		ensureStateRunID(state)
		if state.RunID != "existing-id" {
			t.Errorf("RunID = %q, want %q", state.RunID, "existing-id")
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 20; i++ {
			state := &phasedState{}
			ensureStateRunID(state)
			if ids[state.RunID] {
				t.Errorf("duplicate RunID generated: %s", state.RunID)
			}
			ids[state.RunID] = true
		}
	})
}

// ---------------------------------------------------------------------------
// rpi_phased_setup.go — newPhasedState
// ---------------------------------------------------------------------------

func TestRPI_NewPhasedState(t *testing.T) {
	opts := phasedEngineOptions{
		FastPath:  true,
		TestFirst: true,
		SwarmFirst: false,
	}
	state := newPhasedState(opts, 2, "add auth")

	if state.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", state.SchemaVersion)
	}
	if state.Goal != "add auth" {
		t.Errorf("Goal = %q, want %q", state.Goal, "add auth")
	}
	if state.Phase != 2 {
		t.Errorf("Phase = %d, want 2", state.Phase)
	}
	if state.StartPhase != 2 {
		t.Errorf("StartPhase = %d, want 2", state.StartPhase)
	}
	if state.Cycle != 1 {
		t.Errorf("Cycle = %d, want 1", state.Cycle)
	}
	if !state.FastPath {
		t.Error("FastPath should be true from opts")
	}
	if !state.TestFirst {
		t.Error("TestFirst should be true from opts")
	}
	if state.SwarmFirst {
		t.Error("SwarmFirst should be false from opts")
	}
	if state.Verdicts == nil {
		t.Error("Verdicts map should be initialized")
	}
	if state.Attempts == nil {
		t.Error("Attempts map should be initialized")
	}
	if state.StartedAt == "" {
		t.Error("StartedAt should be set")
	}
	// Verify StartedAt is valid RFC3339
	if _, err := time.Parse(time.RFC3339, state.StartedAt); err != nil {
		t.Errorf("StartedAt %q is not valid RFC3339: %v", state.StartedAt, err)
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — normalizeRuntimeMode
// ---------------------------------------------------------------------------

func TestRPI_NormalizeRuntimeMode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "auto"},
		{"  ", "auto"},
		{"auto", "auto"},
		{"AUTO", "auto"},
		{"Direct", "direct"},
		{"STREAM", "stream"},
		{"  stream  ", "stream"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeRuntimeMode(tt.input)
			if got != tt.want {
				t.Errorf("normalizeRuntimeMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — effectiveRuntimeCommand, effectiveAOCommand, effectiveBDCommand
// ---------------------------------------------------------------------------

func TestRPI_EffectiveRuntimeCommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "claude"},
		{"  ", "claude"},
		{"claude", "claude"},
		{"/usr/local/bin/claude", "/usr/local/bin/claude"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := effectiveRuntimeCommand(tt.input)
			if got != tt.want {
				t.Errorf("effectiveRuntimeCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRPI_EffectiveAOCommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "ao"},
		{"  ", "ao"},
		{"ao", "ao"},
		{"/opt/homebrew/bin/ao", "/opt/homebrew/bin/ao"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := effectiveAOCommand(tt.input)
			if got != tt.want {
				t.Errorf("effectiveAOCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRPI_EffectiveBDCommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "bd"},
		{"  ", "bd"},
		{"bd", "bd"},
		{"/opt/homebrew/bin/bd", "/opt/homebrew/bin/bd"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := effectiveBDCommand(tt.input)
			if got != tt.want {
				t.Errorf("effectiveBDCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — generateRunID
// ---------------------------------------------------------------------------

func TestRPI_GenerateRunID(t *testing.T) {
	id := generateRunID()
	if id == "" {
		t.Fatal("generateRunID returned empty")
	}
	if len(id) != 12 {
		t.Errorf("generateRunID length = %d, want 12 hex chars", len(id))
	}
	// Verify hex characters only
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("generateRunID contains non-hex char: %c in %q", c, id)
			break
		}
	}
	// Verify uniqueness
	id2 := generateRunID()
	if id == id2 {
		t.Errorf("two consecutive calls returned same ID: %q", id)
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — renderPreambleInstructions
// ---------------------------------------------------------------------------

func TestRPI_RenderPreambleInstructions(t *testing.T) {
	data := struct {
		PhaseNum      int
		ContextBudget string
	}{
		PhaseNum:      2,
		ContextBudget: "BUDGET: test budget",
	}

	var buf strings.Builder
	renderPreambleInstructions(&buf, data)
	result := buf.String()

	if !strings.Contains(result, "CONTEXT DISCIPLINE") {
		t.Error("output should contain CONTEXT DISCIPLINE")
	}
	if !strings.Contains(result, "phase 2 of 3") {
		t.Error("output should contain 'phase 2 of 3'")
	}
	if !strings.Contains(result, "BUDGET: test budget") {
		t.Error("output should contain the context budget")
	}
	if !strings.Contains(result, "PHASE SUMMARY CONTRACT") {
		t.Error("output should contain PHASE SUMMARY CONTRACT")
	}
	if !strings.Contains(result, "phase-2-summary.md") {
		t.Error("output should contain phase-specific summary filename")
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — defaultPhasedEngineOptions
// ---------------------------------------------------------------------------

func TestRPI_DefaultPhasedEngineOptions(t *testing.T) {
	opts := defaultPhasedEngineOptions()

	if opts.From != "discovery" {
		t.Errorf("From = %q, want %q", opts.From, "discovery")
	}
	if opts.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", opts.MaxRetries)
	}
	if opts.PhaseTimeout != 90*time.Minute {
		t.Errorf("PhaseTimeout = %v, want 90m", opts.PhaseTimeout)
	}
	if opts.StallTimeout != 10*time.Minute {
		t.Errorf("StallTimeout = %v, want 10m", opts.StallTimeout)
	}
	if opts.StreamStartupTimeout != 45*time.Second {
		t.Errorf("StreamStartupTimeout = %v, want 45s", opts.StreamStartupTimeout)
	}
	if !opts.SwarmFirst {
		t.Error("SwarmFirst should be true by default")
	}
	if opts.AutoCleanStaleAfter != 24*time.Hour {
		t.Errorf("AutoCleanStaleAfter = %v, want 24h", opts.AutoCleanStaleAfter)
	}
	if opts.StallCheckInterval != 30*time.Second {
		t.Errorf("StallCheckInterval = %v, want 30s", opts.StallCheckInterval)
	}
	if opts.RuntimeMode != "auto" {
		t.Errorf("RuntimeMode = %q, want %q", opts.RuntimeMode, "auto")
	}
	if opts.RuntimeCommand != "claude" {
		t.Errorf("RuntimeCommand = %q, want %q", opts.RuntimeCommand, "claude")
	}
	if opts.AOCommand != "ao" {
		t.Errorf("AOCommand = %q, want %q", opts.AOCommand, "ao")
	}
	if opts.BDCommand != "bd" {
		t.Errorf("BDCommand = %q, want %q", opts.BDCommand, "bd")
	}
	if opts.TmuxCommand != "tmux" {
		t.Errorf("TmuxCommand = %q, want %q", opts.TmuxCommand, "tmux")
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — phaseNameToNum (extended cases for case insensitivity)
// ---------------------------------------------------------------------------

func TestRPI_PhaseNameToNum_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		{"Discovery", 1},
		{"DISCOVERY", 1},
		{" discovery ", 1},
		{"Implementation", 2},
		{"CRANK", 2},
		{" Validation ", 3},
		{"VIBE", 3},
		{"POST-MORTEM", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phaseNameToNum(tt.name)
			if got != tt.expected {
				t.Errorf("phaseNameToNum(%q) = %d, want %d", tt.name, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// inject.go — resolveInjectQuery
// ---------------------------------------------------------------------------

func TestInject_ResolveInjectQuery(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"positional arg used", []string{"authentication"}, "authentication"},
		{"empty args returns context flag value", []string{}, ""},
		{"multiple args uses first", []string{"first", "second"}, "first"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore the global
			origCtx := injectContext
			injectContext = ""
			defer func() { injectContext = origCtx }()

			got := resolveInjectQuery(tt.args)
			if got != tt.want {
				t.Errorf("resolveInjectQuery(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}

	t.Run("falls back to context flag", func(t *testing.T) {
		origCtx := injectContext
		injectContext = "from-flag"
		defer func() { injectContext = origCtx }()

		got := resolveInjectQuery(nil)
		if got != "from-flag" {
			t.Errorf("resolveInjectQuery(nil) = %q, want %q", got, "from-flag")
		}
	})
}

// ---------------------------------------------------------------------------
// inject.go — renderKnowledge
// ---------------------------------------------------------------------------

func TestInject_RenderKnowledge(t *testing.T) {
	k := &injectedKnowledge{
		Learnings: []learning{{ID: "L1", Title: "Test", Summary: "Summary"}},
		Timestamp: time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
	}

	t.Run("markdown format", func(t *testing.T) {
		got, err := renderKnowledge(k, "markdown")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "Injected Knowledge") {
			t.Error("expected markdown header")
		}
		if !strings.Contains(got, "L1") {
			t.Error("expected learning ID in output")
		}
	})

	t.Run("json format", func(t *testing.T) {
		got, err := renderKnowledge(k, "json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}
		learnings, ok := parsed["learnings"].([]any)
		if !ok || len(learnings) != 1 {
			t.Errorf("expected 1 learning in JSON output, got: %v", parsed["learnings"])
		}
	})

	t.Run("empty knowledge json", func(t *testing.T) {
		empty := &injectedKnowledge{
			Timestamp: time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
		}
		got, err := renderKnowledge(empty, "json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "timestamp") {
			t.Error("expected timestamp in JSON output")
		}
	})
}

// ---------------------------------------------------------------------------
// inject.go — writeLearningsSection, writePatternsSection, writeSessionsSection, writeConstraintsSection
// ---------------------------------------------------------------------------

func TestInject_WriteLearningsSection(t *testing.T) {
	t.Run("empty slice writes nothing", func(t *testing.T) {
		var sb strings.Builder
		writeLearningsSection(&sb, nil)
		if sb.Len() != 0 {
			t.Errorf("expected empty output, got %q", sb.String())
		}
	})

	t.Run("with learnings", func(t *testing.T) {
		var sb strings.Builder
		writeLearningsSection(&sb, []learning{
			{ID: "L1", Title: "Title One", Summary: ""},
			{ID: "L2", Title: "Title Two", Summary: "Summary text"},
		})
		got := sb.String()
		if !strings.Contains(got, "### Recent Learnings") {
			t.Error("expected section header")
		}
		// L1 has no summary, should use title
		if !strings.Contains(got, "**L1**: Title One") {
			t.Errorf("expected L1 with title, got: %q", got)
		}
		// L2 has summary, should use summary
		if !strings.Contains(got, "**L2**: Summary text") {
			t.Errorf("expected L2 with summary, got: %q", got)
		}
	})
}

func TestInject_WritePatternsSection(t *testing.T) {
	t.Run("empty slice writes nothing", func(t *testing.T) {
		var sb strings.Builder
		writePatternsSection(&sb, nil)
		if sb.Len() != 0 {
			t.Errorf("expected empty output, got %q", sb.String())
		}
	})

	t.Run("with patterns", func(t *testing.T) {
		var sb strings.Builder
		writePatternsSection(&sb, []pattern{
			{Name: "Guard Clause", Description: "Use early returns"},
			{Name: "NoDesc"},
		})
		got := sb.String()
		if !strings.Contains(got, "### Active Patterns") {
			t.Error("expected section header")
		}
		if !strings.Contains(got, "**Guard Clause**: Use early returns") {
			t.Error("expected pattern with description")
		}
		if !strings.Contains(got, "**NoDesc**") {
			t.Error("expected pattern without description")
		}
	})
}

func TestInject_WriteSessionsSection(t *testing.T) {
	t.Run("empty slice writes nothing", func(t *testing.T) {
		var sb strings.Builder
		writeSessionsSection(&sb, nil)
		if sb.Len() != 0 {
			t.Errorf("expected empty output, got %q", sb.String())
		}
	})

	t.Run("with sessions", func(t *testing.T) {
		var sb strings.Builder
		writeSessionsSection(&sb, []session{
			{Date: "2026-02-20", Summary: "Worked on auth"},
		})
		got := sb.String()
		if !strings.Contains(got, "### Recent Sessions") {
			t.Error("expected section header")
		}
		if !strings.Contains(got, "[2026-02-20] Worked on auth") {
			t.Error("expected session entry")
		}
	})
}

func TestInject_WriteConstraintsSection(t *testing.T) {
	t.Run("empty slice writes nothing", func(t *testing.T) {
		var sb strings.Builder
		writeConstraintsSection(&sb, nil)
		if sb.Len() != 0 {
			t.Errorf("expected empty output, got %q", sb.String())
		}
	})

	t.Run("with constraints", func(t *testing.T) {
		var sb strings.Builder
		writeConstraintsSection(&sb, []olConstraint{
			{Pattern: "no-eval", Detection: "eval() found"},
		})
		got := sb.String()
		if !strings.Contains(got, "### Olympus Constraints") {
			t.Error("expected section header")
		}
		if !strings.Contains(got, "[olympus constraint]") {
			t.Error("expected constraint prefix")
		}
	})
}

// ---------------------------------------------------------------------------
// index.go — diffFileSets
// ---------------------------------------------------------------------------

func TestIndex_DiffFileSets(t *testing.T) {
	tests := []struct {
		name         string
		expected     map[string]bool
		existing     map[string]bool
		wantMissing  int
		wantExtra    int
	}{
		{
			name:     "identical sets",
			expected: map[string]bool{"a.md": true, "b.md": true},
			existing: map[string]bool{"a.md": true, "b.md": true},
		},
		{
			name:        "missing files",
			expected:    map[string]bool{"a.md": true, "b.md": true, "c.md": true},
			existing:    map[string]bool{"a.md": true},
			wantMissing: 2,
		},
		{
			name:      "extra files",
			expected:  map[string]bool{"a.md": true},
			existing:  map[string]bool{"a.md": true, "old.md": true},
			wantExtra: 1,
		},
		{
			name:        "both missing and extra",
			expected:    map[string]bool{"a.md": true, "new.md": true},
			existing:    map[string]bool{"a.md": true, "stale.md": true},
			wantMissing: 1,
			wantExtra:   1,
		},
		{
			name:     "empty sets",
			expected: map[string]bool{},
			existing: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing, extra := diffFileSets(tt.expected, tt.existing)
			if len(missing) != tt.wantMissing {
				t.Errorf("missing = %v (len %d), want len %d", missing, len(missing), tt.wantMissing)
			}
			if len(extra) != tt.wantExtra {
				t.Errorf("extra = %v (len %d), want len %d", extra, len(extra), tt.wantExtra)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// index.go — buildExpectedFileSet
// ---------------------------------------------------------------------------

func TestIndex_BuildExpectedFileSet(t *testing.T) {
	entries := []indexEntry{
		{Filename: "a.md"},
		{Filename: "b.md"},
		{Filename: "c.md"},
	}
	got := buildExpectedFileSet(entries)
	if len(got) != 3 {
		t.Errorf("expected 3 entries, got %d", len(got))
	}
	for _, e := range entries {
		if !got[e.Filename] {
			t.Errorf("missing %q in file set", e.Filename)
		}
	}

	t.Run("empty entries", func(t *testing.T) {
		got := buildExpectedFileSet(nil)
		if len(got) != 0 {
			t.Errorf("expected empty set, got %d", len(got))
		}
	})
}

// ---------------------------------------------------------------------------
// index.go — parseIndexTableRows
// ---------------------------------------------------------------------------

func TestIndex_ParseIndexTableRows(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name: "standard table",
			content: `# Index: Learnings

| File | Date | Summary | Tags |
|------|------|---------|------|
| a.md | 2026-01-01 | First | go |
| b.md | 2026-01-02 | Second | rust |
`,
			want: 2,
		},
		{
			name:    "empty content",
			content: "",
			want:    0,
		},
		{
			name: "header only no data rows",
			content: `| File | Date | Summary | Tags |
|------|------|---------|------|
`,
			want: 0,
		},
		{
			name: "skips File header row",
			content: `|------|------|---------|------|
| File | 2026-01-01 | sum | tag |
| real.md | 2026-01-01 | sum | tag |
`,
			// "File" in first column is filtered out by the fname != "File" check
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIndexTableRows([]byte(tt.content))
			if len(got) != tt.want {
				t.Errorf("parseIndexTableRows() returned %d entries, want %d; entries: %v", len(got), tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// index.go — buildIndexDiffMessage
// ---------------------------------------------------------------------------

func TestIndex_BuildIndexDiffMessage(t *testing.T) {
	t.Run("missing only", func(t *testing.T) {
		msg := buildIndexDiffMessage(".agents/learnings", []string{"new.md"}, nil)
		if !strings.Contains(msg, "STALE .agents/learnings") {
			t.Errorf("expected STALE prefix, got: %q", msg)
		}
		if !strings.Contains(msg, "missing=[new.md]") {
			t.Errorf("expected missing list, got: %q", msg)
		}
	})

	t.Run("extra only", func(t *testing.T) {
		msg := buildIndexDiffMessage(".agents/learnings", nil, []string{"old.md"})
		if !strings.Contains(msg, "extra=[old.md]") {
			t.Errorf("expected extra list, got: %q", msg)
		}
	})

	t.Run("both missing and extra", func(t *testing.T) {
		msg := buildIndexDiffMessage(".agents/plans", []string{"a.md", "b.md"}, []string{"c.md"})
		if !strings.Contains(msg, "missing=[a.md, b.md]") {
			t.Errorf("expected sorted missing list, got: %q", msg)
		}
		if !strings.Contains(msg, "extra=[c.md]") {
			t.Errorf("expected extra list, got: %q", msg)
		}
	})

	t.Run("empty lists", func(t *testing.T) {
		msg := buildIndexDiffMessage(".agents/test", nil, nil)
		if !strings.HasPrefix(msg, "STALE .agents/test:") {
			t.Errorf("expected STALE prefix only, got: %q", msg)
		}
	})
}

// ---------------------------------------------------------------------------
// index.go — parseFrontmatter (the YAML index variant)
// ---------------------------------------------------------------------------

func TestIndex_ParseFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int // number of keys in result
		checkFn func(map[string]any) bool
	}{
		{
			name:    "no frontmatter",
			content: "# Title\nContent",
			wantLen: 0,
		},
		{
			name:    "empty frontmatter",
			content: "---\n---\n# Title",
			wantLen: 0,
		},
		{
			name:    "with summary and date",
			content: "---\nsummary: Test summary\ndate: 2026-01-15\n---\n# Title",
			wantLen: 2,
			checkFn: func(fm map[string]any) bool {
				s, ok := fm["summary"].(string)
				return ok && s == "Test summary"
			},
		},
		{
			name:    "with tags array",
			content: "---\ntags:\n  - go\n  - testing\n---\n# Title",
			wantLen: 1,
			checkFn: func(fm map[string]any) bool {
				tags, ok := fm["tags"].([]any)
				return ok && len(tags) == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFrontmatter(tt.content)
			if len(got) != tt.wantLen {
				t.Errorf("parseFrontmatter() returned %d keys, want %d; map: %v", len(got), tt.wantLen, got)
			}
			if tt.checkFn != nil && !tt.checkFn(got) {
				t.Errorf("check function failed, map: %v", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// index.go — extractDateField
// ---------------------------------------------------------------------------

func TestIndex_ExtractDateField(t *testing.T) {
	tests := []struct {
		name     string
		fm       map[string]any
		filename string
		want     string
	}{
		{
			name:     "created_at takes priority",
			fm:       map[string]any{"created_at": "2026-01-20", "date": "2026-01-01"},
			filename: "2026-02-01-file.md",
			want:     "2026-01-20",
		},
		{
			name:     "date field used when no created_at",
			fm:       map[string]any{"date": "2026-01-25"},
			filename: "file.md",
			want:     "2026-01-25",
		},
		{
			name:     "falls back to filename",
			fm:       map[string]any{},
			filename: "2026-03-10-my-file.md",
			want:     "2026-03-10",
		},
		{
			name:     "no date anywhere",
			fm:       map[string]any{},
			filename: "no-date-file.md",
			want:     "unknown",
		},
		{
			name:     "datetime value extracts date portion",
			fm:       map[string]any{"created_at": "2026-01-20T12:30:00Z"},
			filename: "file.md",
			want:     "2026-01-20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDateField(tt.fm, tt.filename)
			if got != tt.want {
				t.Errorf("extractDateField() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// index.go — scanAndSortDir
// ---------------------------------------------------------------------------

func TestIndex_ScanAndSortDir(t *testing.T) {
	t.Run("sorts by date descending", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, ".agents", "learnings")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		createTestFile(t, subDir, "2026-01-01-old.md", "---\nsummary: Old\n---\n# Old")
		createTestFile(t, subDir, "2026-03-01-new.md", "---\nsummary: New\n---\n# New")
		createTestFile(t, subDir, "2026-02-01-mid.md", "---\nsummary: Mid\n---\n# Mid")

		_, entries, ok := scanAndSortDir(tmpDir, ".agents/learnings")
		if !ok {
			t.Fatal("scanAndSortDir returned not-ok")
		}
		if len(entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(entries))
		}
		// First entry should be newest
		if entries[0].Date != "2026-03-01" {
			t.Errorf("first entry date = %q, want 2026-03-01", entries[0].Date)
		}
		if entries[2].Date != "2026-01-01" {
			t.Errorf("last entry date = %q, want 2026-01-01", entries[2].Date)
		}
	})

	t.Run("nonexistent directory returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, _, ok := scanAndSortDir(tmpDir, ".agents/nonexistent")
		if ok {
			t.Error("expected false for nonexistent directory")
		}
	})
}

// ---------------------------------------------------------------------------
// index.go — writeIndex (dry-run mode)
// ---------------------------------------------------------------------------

func TestIndex_WriteIndex_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	entries := []indexEntry{
		{Filename: "a.md", Date: "2026-01-01", Summary: "Test", Tags: "go"},
	}

	err := writeIndex(tmpDir, ".agents/learnings", entries, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	indexPath := filepath.Join(tmpDir, "INDEX.md")
	if _, err := os.Stat(indexPath); err == nil {
		t.Error("INDEX.md should not be written in dry-run mode")
	}
}

// ---------------------------------------------------------------------------
// index.go — extractEntry integration
// ---------------------------------------------------------------------------

func TestIndex_ExtractEntry(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("full frontmatter extraction", func(t *testing.T) {
		content := `---
created_at: 2026-02-15
summary: My test summary
tags: [go, testing]
---
# Title Here
Content.`
		path := filepath.Join(tmpDir, "2026-02-15-test.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		entry := extractEntry(path, "2026-02-15-test.md")
		if entry.Date != "2026-02-15" {
			t.Errorf("Date = %q, want 2026-02-15", entry.Date)
		}
		if entry.Summary != "My test summary" {
			t.Errorf("Summary = %q, want %q", entry.Summary, "My test summary")
		}
		if !strings.Contains(entry.Tags, "go") || !strings.Contains(entry.Tags, "testing") {
			t.Errorf("Tags = %q, want go and testing", entry.Tags)
		}
	})

	t.Run("H1 fallback for summary", func(t *testing.T) {
		content := "---\ncreated_at: 2026-01-01\n---\n# My H1 Title\nContent."
		path := filepath.Join(tmpDir, "h1-test.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		entry := extractEntry(path, "h1-test.md")
		if entry.Summary != "My H1 Title" {
			t.Errorf("Summary = %q, want %q", entry.Summary, "My H1 Title")
		}
	})

	t.Run("filename fallback for summary", func(t *testing.T) {
		content := "---\ncreated_at: 2026-01-01\n---\nJust content, no heading."
		path := filepath.Join(tmpDir, "2026-01-01-no-heading.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		entry := extractEntry(path, "2026-01-01-no-heading.md")
		if entry.Summary != "no-heading" {
			t.Errorf("Summary = %q, want %q", entry.Summary, "no-heading")
		}
	})

	t.Run("nonexistent file uses filename fallbacks", func(t *testing.T) {
		entry := extractEntry(filepath.Join(tmpDir, "nope.md"), "2026-03-01-missing.md")
		if entry.Date != "2026-03-01" {
			t.Errorf("Date = %q, want 2026-03-01", entry.Date)
		}
		if entry.Summary != "missing" {
			t.Errorf("Summary = %q, want %q", entry.Summary, "missing")
		}
	})
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — readPhaseSummaries with truncation
// ---------------------------------------------------------------------------

func TestRPI_ReadPhaseSummaries_Truncation(t *testing.T) {
	tmpDir := t.TempDir()
	rpiDir := filepath.Join(tmpDir, ".agents", "rpi")
	if err := os.MkdirAll(rpiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a phase-1 summary that exceeds 2000 chars
	longContent := strings.Repeat("A", 2500)
	if err := os.WriteFile(filepath.Join(rpiDir, "phase-1-summary.md"), []byte(longContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := readPhaseSummaries(tmpDir, 2)
	if !strings.Contains(result, "...") {
		t.Error("expected truncation marker '...' for long summary")
	}
	// Should contain phase header
	if !strings.Contains(result, "[Phase 1: discovery]") {
		t.Error("expected phase label in summary")
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_context.go — buildPhaseContext with summaries from disk
// ---------------------------------------------------------------------------

func TestRPI_BuildPhaseContext_WithSummaries(t *testing.T) {
	tmpDir := t.TempDir()
	rpiDir := filepath.Join(tmpDir, ".agents", "rpi")
	if err := os.MkdirAll(rpiDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(rpiDir, "phase-1-summary.md"), []byte("Discovery found issues X, Y, Z"), 0644); err != nil {
		t.Fatal(err)
	}

	state := newTestPhasedState().WithGoal("add auth").WithEpicID("ag-5k2")
	state.Verdicts["pre_mortem"] = "PASS"

	ctx := buildPhaseContext(tmpDir, state, 2)
	if !strings.Contains(ctx, "Goal: add auth") {
		t.Error("expected goal in context")
	}
	if !strings.Contains(ctx, "pre-mortem verdict: PASS") {
		t.Error("expected verdict in context")
	}
	if !strings.Contains(ctx, "Discovery found issues X, Y, Z") {
		t.Error("expected phase-1 summary content in context")
	}
	if !strings.HasPrefix(ctx, "--- RPI Context") {
		t.Error("expected RPI Context header prefix")
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_phase_runner.go — writeFinalPhasedReport (smoke test)
// ---------------------------------------------------------------------------

func TestRPI_WriteFinalPhasedReport(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	state := newTestPhasedState().WithGoal("test goal").WithEpicID("ag-xyz").WithRunID("run-final")
	state.Verdicts["pre_mortem"] = "PASS"
	state.Verdicts["vibe"] = "WARN"

	// Should not panic
	writeFinalPhasedReport(state, logPath)

	// Verify log was written
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "complete") {
		t.Errorf("log should contain 'complete', got: %q", content)
	}
	if !strings.Contains(content, "ag-xyz") {
		t.Errorf("log should contain epic ID, got: %q", content)
	}
}

// ---------------------------------------------------------------------------
// rpi_phased_phase_runner.go — logAndFailPhase
// ---------------------------------------------------------------------------

func TestRPI_LogAndFailPhase(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".agents", "rpi")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(tmpDir, "test.log")
	state := newTestPhasedState().WithRunID("run-fail")
	simErr := fmt.Errorf("simulated error")

	returnedErr := logAndFailPhase(state, "validation", logPath, tmpDir, simErr)
	if returnedErr == nil {
		t.Fatal("expected error to be returned")
	}

	// Verify terminal state was set
	if state.TerminalStatus != "failed" {
		t.Errorf("TerminalStatus = %q, want %q", state.TerminalStatus, "failed")
	}
	if !strings.Contains(state.TerminalReason, "validation") {
		t.Errorf("TerminalReason = %q, should contain 'validation'", state.TerminalReason)
	}
	if state.TerminatedAt == "" {
		t.Error("TerminatedAt should be set")
	}

	// Verify log was written
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "FATAL") {
		t.Error("log should contain FATAL entry")
	}
}
