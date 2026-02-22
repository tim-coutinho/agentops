package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	cliRPI "github.com/boshu2/agentops/cli/internal/rpi"
)

var (
	phasedFrom         string
	phasedTestFirst    bool
	phasedFastPath     bool
	phasedInteractive  bool
	phasedMaxRetries   int
	phasedPhaseTimeout time.Duration
	phasedStallTimeout time.Duration
	// phasedStreamStartupTimeout bounds how long stream backend can run without
	// receiving its first parsed event before falling back to direct execution.
	phasedStreamStartupTimeout time.Duration
	phasedNoWorktree           bool
	phasedLiveStatus           bool
	phasedSwarmFirst           bool
	phasedAutoCleanStale       bool
	phasedAutoCleanStaleAfter  time.Duration
	phasedRuntimeMode          string
	phasedRuntimeCommand       string
)

// phaseFailureReason classifies why a phase spawn failed.
type phaseFailureReason string

const (
	failReasonTimeout phaseFailureReason = "timeout"
	failReasonStall   phaseFailureReason = "stall"
	failReasonExit    phaseFailureReason = "exit_error"
	failReasonUnknown phaseFailureReason = "unknown"
)

func init() {
	phasedCmd := &cobra.Command{
		Use:   "phased <goal>",
		Short: "Run RPI with fresh runtime session per phase",
		Long: `Orchestrate the full RPI lifecycle using 3 consolidated phases.

Each phase gets its own context window (Ralph Wiggum pattern):
  1. Discovery       — research + plan + pre-mortem (shared context, prompt cache hot)
  2. Implementation  — crank (fresh context for heavy work)
  3. Validation      — vibe + post-mortem (fresh eyes, independent of implementer)

This consolidation cuts cold starts from 6 to 3, keeps prompt cache warm
within each phase, and preserves the key isolation boundary: the implementer
session is never the validator session.

Between phases, the CLI reads filesystem artifacts, constructs prompts
via templates, and spawns the next session. Retry loops for gate failures
are handled within the session (discovery) or across sessions (validation).

Examples:
  ao rpi phased "add user authentication"       # full lifecycle (3 sessions)
  ao rpi phased --from=implementation "add auth" # skip to crank (needs epic)
  ao rpi phased --from=validation                # just vibe + post-mortem
  ao rpi phased --dry-run "add auth"             # show prompts without spawning
  ao rpi phased --fast-path "fix typo"           # force --quick for gates`,
		Args: cobra.MaximumNArgs(1),
		RunE: runRPIPhased,
	}

	phasedCmd.Flags().StringVar(&phasedFrom, "from", "discovery", "Start from phase (discovery, implementation, validation; aliases: research, plan, pre-mortem, crank, vibe, post-mortem)")
	phasedCmd.Flags().BoolVar(&phasedTestFirst, "test-first", false, "Pass --test-first to /crank for spec-first TDD")
	phasedCmd.Flags().BoolVar(&phasedFastPath, "fast-path", false, "Force fast path (--quick for gates)")
	phasedCmd.Flags().BoolVar(&phasedInteractive, "interactive", false, "Enable human gates at research and plan phases")
	phasedCmd.Flags().IntVar(&phasedMaxRetries, "max-retries", 3, "Maximum retry attempts per gate (default: 3)")
	phasedCmd.Flags().DurationVar(&phasedPhaseTimeout, "phase-timeout", 90*time.Minute, "Maximum wall-clock runtime per phase (0 disables timeout)")
	phasedCmd.Flags().DurationVar(&phasedStallTimeout, "stall-timeout", 10*time.Minute, "Maximum time without progress before declaring stall (0 disables)")
	phasedCmd.Flags().DurationVar(&phasedStreamStartupTimeout, "stream-startup-timeout", 45*time.Second, "Maximum time to wait for first stream event before falling back to direct execution (0 disables)")
	phasedCmd.Flags().BoolVar(&phasedNoWorktree, "no-worktree", false, "Disable worktree isolation (run in current directory)")
	phasedCmd.Flags().BoolVar(&phasedLiveStatus, "live-status", false, "Stream phase progress to a live-status.md file")
	phasedCmd.Flags().BoolVar(&phasedSwarmFirst, "swarm-first", true, "Default each phase to swarm/agent-team execution; fall back to direct execution if swarm runtime is unavailable")
	phasedCmd.Flags().BoolVar(&phasedAutoCleanStale, "auto-clean-stale", false, "Run stale-run cleanup before starting phased execution")
	phasedCmd.Flags().DurationVar(&phasedAutoCleanStaleAfter, "auto-clean-stale-after", 24*time.Hour, "Only clean stale runs older than this age when auto-clean is enabled")
	phasedCmd.Flags().StringVar(&phasedRuntimeMode, "runtime", "auto", "Phase runtime mode: auto|direct|stream")
	phasedCmd.Flags().StringVar(&phasedRuntimeCommand, "runtime-cmd", "claude", "Runtime command to execute prompts (must support '-p')")

	rpiCmd.AddCommand(phasedCmd)
}

// runPhasedEngine runs the full phased RPI lifecycle for goal in cwd.
// It is the programmatic entry point used by both the phased cobra command
// and the loop command, ensuring both share the same runtime contracts.
func runPhasedEngine(cwd, goal string, opts phasedEngineOptions) (retErr error) {
	// Temporarily change working directory so runRPIPhasedWithOpts's os.Getwd() call
	// and all path resolution operate in the requested cwd.
	origDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	if cwd != "" && cwd != origDir {
		if err := os.Chdir(cwd); err != nil {
			return fmt.Errorf("chdir to %s: %w", cwd, err)
		}
		defer func() {
			_ = os.Chdir(origDir)
		}()
	}

	args := []string{goal}
	if goal == "" {
		args = nil
	}
	return runRPIPhasedWithOpts(opts, args)
}

// runRPIPhased is the cobra RunE handler for `ao rpi phased`.
// It reads options from package-level cobra flag variables and delegates to runRPIPhasedWithOpts.
func runRPIPhased(cmd *cobra.Command, args []string) error {
	opts := phasedEngineOptions{
		From:                 phasedFrom,
		FastPath:             phasedFastPath,
		TestFirst:            phasedTestFirst,
		Interactive:          phasedInteractive,
		MaxRetries:           phasedMaxRetries,
		PhaseTimeout:         phasedPhaseTimeout,
		StallTimeout:         phasedStallTimeout,
		StreamStartupTimeout: phasedStreamStartupTimeout,
		NoWorktree:           phasedNoWorktree,
		LiveStatus:           phasedLiveStatus,
		SwarmFirst:           phasedSwarmFirst,
		AutoCleanStale:       phasedAutoCleanStale,
		AutoCleanStaleAfter:  phasedAutoCleanStaleAfter,
		StallCheckInterval:   stallCheckInterval,
		RuntimeMode:          phasedRuntimeMode,
		RuntimeCommand:       phasedRuntimeCommand,
	}

	// Apply config-based worktree mode if the --no-worktree flag was not explicitly set.
	if !cmd.Flags().Changed("no-worktree") {
		opts.NoWorktree = resolveWorktreeModeFromConfig(opts.NoWorktree)
	}
	toolchain, err := resolveRPIToolchain(
		cliRPI.Toolchain{
			RuntimeMode:    phasedRuntimeMode,
			RuntimeCommand: phasedRuntimeCommand,
		},
		rpiToolchainFlagSet{
			RuntimeMode:    cmd.Flags().Changed("runtime"),
			RuntimeCommand: cmd.Flags().Changed("runtime-cmd"),
		},
	)
	if err != nil {
		return err
	}
	opts.RuntimeMode = toolchain.RuntimeMode
	opts.RuntimeCommand = toolchain.RuntimeCommand
	opts.AOCommand = toolchain.AOCommand
	opts.BDCommand = toolchain.BDCommand
	opts.TmuxCommand = toolchain.TmuxCommand
	if cmd.Flags().Changed("auto-clean-stale-after") {
		opts.AutoCleanStale = true
	}

	return runRPIPhasedWithOpts(opts, args)
}

// normalizeOptsCommands resolves all runtime/tool commands to their effective values.
func normalizeOptsCommands(opts *phasedEngineOptions) {
	opts.RuntimeMode = normalizeRuntimeMode(opts.RuntimeMode)
	opts.RuntimeCommand = effectiveRuntimeCommand(opts.RuntimeCommand)
	opts.AOCommand = effectiveAOCommand(opts.AOCommand)
	opts.BDCommand = effectiveBDCommand(opts.BDCommand)
	opts.TmuxCommand = effectiveTmuxCommand(opts.TmuxCommand)
}

// applyComplexityFastPath classifies goal complexity and activates the fast path
// (skips council validation) for trivial goals.
func applyComplexityFastPath(state *phasedState, opts phasedEngineOptions) {
	complexity := classifyComplexity(state.Goal)
	state.Complexity = complexity
	fmt.Printf("RPI mode: rpi-phased (complexity: %s)\n", complexity)
	if complexity == ComplexityFast && !opts.FastPath {
		state.FastPath = true
		fmt.Println("Complexity: fast — skipping validation phase (phase 3)")
	}
}

// saveTerminalState writes a terminal status/reason to state and persists it.
func saveTerminalState(spawnCwd string, state *phasedState, status, reason string) {
	state.TerminalStatus = status
	state.TerminalReason = reason
	state.TerminatedAt = time.Now().Format(time.RFC3339)
	if err := savePhasedState(spawnCwd, state); err != nil {
		VerbosePrintf("Warning: could not persist %s terminal state: %v\n", status, err)
	}
}

// preflightOpts normalizes, validates, and checks runtime availability for opts.
func preflightOpts(opts *phasedEngineOptions) error {
	normalizeOptsCommands(opts)
	if err := validateRuntimeMode(opts.RuntimeMode); err != nil {
		return err
	}
	return preflightRuntimeAvailability(opts.RuntimeCommand)
}

// initExecutorAndPersist selects the executor backend for the run and persists
// the initial state with the backend name.
func initExecutorAndPersist(spawnCwd, logPath, statusPath string, allPhases []PhaseProgress, state *phasedState, opts phasedEngineOptions) PhaseExecutor {
	executor := selectExecutorWithLog(statusPath, allPhases, logPath, state.RunID, opts.LiveStatus, opts)
	state.Backend = executor.Name()
	if err := savePhasedState(spawnCwd, state); err != nil {
		VerbosePrintf("Warning: could not persist startup state: %v\n", err)
	}
	updateRunHeartbeat(spawnCwd, state.RunID)
	return executor
}

// runRPIPhasedWithOpts is the core implementation of the phased RPI lifecycle.
// All configuration is read from opts; no package-level globals are read after
// this point (except test-injection points: lookPath and spawnDirectFn).
// initPhasedState resolves the goal and start phase from args, creates the
// phasedState, applies complexity fast-path, and resumes from prior state if needed.
func initPhasedState(cwd string, opts phasedEngineOptions, args []string) (*phasedState, int, string, error) {
	goal, startPhase, err := resolveGoalAndStartPhase(opts, args, cwd)
	if err != nil {
		return nil, 0, "", err
	}
	state := newPhasedState(opts, startPhase, goal)
	applyComplexityFastPath(state, opts)

	spawnCwd, err := resumePhasedStateIfNeeded(cwd, opts, startPhase, goal, state)
	if err != nil {
		return nil, 0, "", err
	}
	return state, startPhase, spawnCwd, nil
}

func runRPIPhasedWithOpts(opts phasedEngineOptions, args []string) (retErr error) {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	if err := preflightOpts(&opts); err != nil {
		return err
	}
	maybeAutoCleanStale(opts, cwd)

	originalCwd := cwd
	state, startPhase, spawnCwd, err := initPhasedState(cwd, opts, args)
	if err != nil {
		return err
	}

	cleanupSuccess := false
	var logPath string
	spawnCwd, cleanupWorktree, err := setupWorktreeLifecycle(spawnCwd, originalCwd, opts, state)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := cleanupWorktree(cleanupSuccess, logPath); cleanupErr != nil && retErr == nil {
			retErr = cleanupErr
		}
	}()

	ensureStateRunID(state)
	state.OrchestratorPID = os.Getpid()

	_, runLogPath, statusPath, allPhases, err := initializeRunArtifacts(spawnCwd, startPhase, state, opts)
	if err != nil {
		return err
	}
	logPath = runLogPath

	logPhaseTransition(logPath, state.RunID, "start", fmt.Sprintf("goal=%q from=%s complexity=%s fast_path=%v", state.Goal, opts.From, state.Complexity, state.FastPath))

	executor := initExecutorAndPersist(spawnCwd, logPath, statusPath, allPhases, state, opts)

	if err := runPhaseLoop(cwd, spawnCwd, state, startPhase, opts, statusPath, allPhases, logPath, executor); err != nil {
		saveTerminalState(spawnCwd, state, "failed", err.Error())
		return err
	}

	saveTerminalState(spawnCwd, state, "completed", "all phases completed")

	// All phases completed — mark worktree for merge+cleanup.
	cleanupSuccess = true

	writeFinalPhasedReport(state, logPath)

	return nil
}

// maybeAutoCleanStale runs stale-run cleanup before starting if the option is enabled.
// Errors are non-fatal and logged via VerbosePrintf.
func maybeAutoCleanStale(opts phasedEngineOptions, cwd string) {
	if !opts.AutoCleanStale {
		return
	}
	minAge := opts.AutoCleanStaleAfter
	if minAge <= 0 {
		minAge = 24 * time.Hour
	}
	fmt.Printf("Auto-cleaning stale runs older than %s before starting\n", minAge)
	if err := executeRPICleanup(cwd, "", true, false, false, GetDryRun(), minAge); err != nil {
		VerbosePrintf("Warning: auto-clean stale runs failed: %v\n", err)
	}
}
