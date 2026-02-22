package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

// PhaseExecutor abstracts the backend used to run a single phase session.
//
// Selection policy (deterministic, logged):
//
//	stream  — stream-json execution with live parsing and fallback semantics.
//	direct  — plain prompt execution without stream parsing.
//
// The chosen backend is recorded in phasedState.Backend, emitted to stdout, and
// appended to the orchestration log so every run has a traceable selection record.
type PhaseExecutor interface {
	// Name returns the backend identifier ("direct" or "stream").
	Name() string
	// Execute runs the phase prompt and blocks until the session completes.
	Execute(prompt, cwd, runID string, phaseNum int) error
}

type directExecutor struct {
	runtimeCommand string
	phaseTimeout   time.Duration
}

func (d *directExecutor) Name() string { return "direct" }
func (d *directExecutor) Execute(prompt, cwd, runID string, phaseNum int) error {
	return spawnRuntimeDirectImpl(d.runtimeCommand, prompt, cwd, phaseNum, d.phaseTimeout)
}

type streamExecutor struct {
	runtimeCommand       string
	statusPath           string
	allPhases            []PhaseProgress
	phaseTimeout         time.Duration
	stallTimeout         time.Duration
	streamStartupTimeout time.Duration
	stallCheckInterval   time.Duration
}

func (s *streamExecutor) Name() string { return "stream" }
func (s *streamExecutor) Execute(prompt, cwd, runID string, phaseNum int) error {
	err := spawnRuntimePhaseWithStream(s.runtimeCommand, prompt, cwd, runID, phaseNum, s.statusPath, s.allPhases, s.phaseTimeout, s.stallTimeout, s.streamStartupTimeout, s.stallCheckInterval)
	if err == nil {
		return nil
	}
	if !shouldFallbackToDirect(err) {
		return err
	}
	fmt.Printf("Stream backend degraded for phase %d; falling back to direct execution (%v)\n", phaseNum, err)
	if directErr := spawnRuntimeDirectImpl(s.runtimeCommand, prompt, cwd, phaseNum, s.phaseTimeout); directErr != nil {
		return fmt.Errorf("stream execution failed: %w; direct fallback failed: %v", err, directErr)
	}
	return nil
}

func shouldFallbackToDirect(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "stream startup timeout") ||
		strings.Contains(msg, "stream parse error") ||
		(strings.Contains(msg, string(failReasonStall)) && strings.Contains(msg, "no stream activity"))
}

// backendCapabilities probes the runtime environment for executor prerequisites.
// All fields are populated by probeBackendCapabilities.
type backendCapabilities struct {
	// LiveStatusEnabled is true when --live-status flag is set.
	LiveStatusEnabled bool
	// RuntimeMode is one of auto|direct|stream.
	RuntimeMode string
}

// probeBackendCapabilities detects available backends in the current environment.
// It is a pure function (no side effects) to keep selectExecutor testable.
func probeBackendCapabilities(liveStatus bool, runtimeMode string) backendCapabilities {
	return backendCapabilities{
		LiveStatusEnabled: liveStatus,
		RuntimeMode:       normalizeRuntimeMode(runtimeMode),
	}
}

// selectExecutorFromCaps resolves an executor from pre-probed capabilities.
// This is the deterministic core of backend selection — testable without env mutation.
// opts provides timeout/interval values for the executors; use defaultPhasedEngineOptions()
// when calling from tests that do not have a full opts available.
//
// Selection order (first match wins):
//  1. runtime=stream — always stream
//  2. runtime=direct — always direct
//  3. runtime=auto   — stream when live-status enabled, otherwise direct
func selectExecutorFromCaps(caps backendCapabilities, statusPath string, allPhases []PhaseProgress, opts phasedEngineOptions) (PhaseExecutor, string) {
	switch caps.RuntimeMode {
	case "stream":
		return &streamExecutor{
			runtimeCommand:       opts.RuntimeCommand,
			statusPath:           statusPath,
			allPhases:            allPhases,
			phaseTimeout:         opts.PhaseTimeout,
			stallTimeout:         opts.StallTimeout,
			streamStartupTimeout: opts.StreamStartupTimeout,
			stallCheckInterval:   opts.StallCheckInterval,
		}, "runtime=stream"
	case "direct":
		return &directExecutor{
			runtimeCommand: opts.RuntimeCommand,
			phaseTimeout:   opts.PhaseTimeout,
		}, "runtime=direct"
	default: // auto
		if caps.LiveStatusEnabled {
			return &streamExecutor{
				runtimeCommand:       opts.RuntimeCommand,
				statusPath:           statusPath,
				allPhases:            allPhases,
				phaseTimeout:         opts.PhaseTimeout,
				stallTimeout:         opts.StallTimeout,
				streamStartupTimeout: opts.StreamStartupTimeout,
				stallCheckInterval:   opts.StallCheckInterval,
			}, "runtime=auto live-status enabled"
		}
		return &directExecutor{
			runtimeCommand: opts.RuntimeCommand,
			phaseTimeout:   opts.PhaseTimeout,
		}, "runtime=auto live-status disabled"
	}
}

// selectExecutor resolves the executor backend based on flags and environment.
// The selection policy, chosen backend, and reason are logged to logPath for
// observability. Pass an empty logPath to skip log writing (e.g., in tests).
//
// Selection order: runtime override (stream/direct) > auto (live-status=>stream, else direct).
func selectExecutor(statusPath string, allPhases []PhaseProgress) PhaseExecutor {
	return selectExecutorWithLog(statusPath, allPhases, "", "", false, defaultPhasedEngineOptions())
}

// selectExecutorWithLog is the log-aware variant used by runRPIPhasedWithOpts.
// logPath and runID are used to append the selection record to the orchestration log.
// liveStatus must be provided explicitly so the function does not read package globals.
// opts provides timeout/interval values embedded into the returned executor.
func selectExecutorWithLog(statusPath string, allPhases []PhaseProgress, logPath, runID string, liveStatus bool, opts phasedEngineOptions) PhaseExecutor {
	caps := probeBackendCapabilities(liveStatus, opts.RuntimeMode)
	executor, reason := selectExecutorFromCaps(caps, statusPath, allPhases, opts)
	msg := fmt.Sprintf("backend=%s reason=%q", executor.Name(), reason)
	fmt.Printf("Executor backend: %s (%s)\n", executor.Name(), reason)
	if logPath != "" {
		logPhaseTransition(logPath, runID, "backend-selection", msg)
	}
	return executor
}

// Exit codes for phased orchestration.
const (
	ExitGateFail  = 10 // Council gate returned FAIL
	ExitUserAbort = 20 // User cancelled the session
	ExitCLIError  = 30 // Runtime CLI error (not found, config issue)
)

// spawnClaudePhase exists for compatibility with legacy direct-spawn tests.
// Production code should use a PhaseExecutor (selectExecutorFromCaps) instead.
func spawnClaudePhase(prompt, cwd, runID string, phaseNum int) error {
	_ = runID
	return spawnDirectFn(prompt, cwd, phaseNum)
}

// spawnRuntimeDirectImpl runs <runtimeCommand> -p directly.
// phaseTimeout controls the maximum runtime; pass 0 to disable the timeout.
func spawnRuntimeDirectImpl(runtimeCommand, prompt, cwd string, phaseNum int, phaseTimeout time.Duration) error {
	command := effectiveRuntimeCommand(runtimeCommand)

	ctx := context.Background()
	cancel := func() {}
	if phaseTimeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), phaseTimeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, command, "-p", prompt)
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = cleanEnvNoClaude()
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("phase %d timed out after %s (set --phase-timeout to increase)", phaseNum, phaseTimeout)
	}
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("%s exited with code %d: %w", command, exitErr.ExitCode(), err)
	}
	return fmt.Errorf("%s execution failed: %w", command, err)
}

// spawnClaudeDirectImpl is the legacy wrapper pinned to the default runtime.
func spawnClaudeDirectImpl(prompt, cwd string, phaseNum int, phaseTimeout time.Duration) error {
	return spawnRuntimeDirectImpl("claude", prompt, cwd, phaseNum, phaseTimeout)
}

// spawnRuntimePhaseWithStream spawns a runtime session using --output-format stream-json
// and feeds stdout through ParseStreamEvents for live progress tracking.
// An onUpdate callback calls WriteLiveStatus after every parsed event so that
// external watchers (e.g. ao status) can tail the status file.
// Stderr is passed through to os.Stderr for real-time error visibility.
// phaseTimeout, stallTimeout, streamStartupTimeout, and checkInterval are passed
// explicitly so the function does not read package-level globals.
// streamWatchdogState holds the shared atomic state used by watchdog goroutines
// and the stream event callback to coordinate stall/startup detection.
type streamWatchdogState struct {
	eventCount       atomic.Int64
	lastActivityUnix atomic.Int64
}

func spawnRuntimePhaseWithStream(runtimeCommand, prompt, cwd, runID string, phaseNum int, statusPath string, allPhases []PhaseProgress, phaseTimeout, stallTimeout, streamStartupTimeout, checkInterval time.Duration) error {
	command := effectiveRuntimeCommand(runtimeCommand)

	ctx, cancel := buildStreamPhaseContext(phaseTimeout)
	defer cancel()

	effectiveCheckInterval := normalizeCheckInterval(checkInterval)
	startedAt := time.Now()

	watchdog := &streamWatchdogState{}
	watchdog.lastActivityUnix.Store(startedAt.UnixNano())

	stallCtx, stallCancel := context.WithCancelCause(ctx)
	defer stallCancel(nil)

	startStreamWatchdogs(stallCtx, stallCancel, watchdog, startedAt, effectiveCheckInterval, stallTimeout, streamStartupTimeout)

	cmd := exec.CommandContext(stallCtx, command, "-p", prompt, "--output-format", "stream-json", "--verbose")
	cmd.Dir = cwd
	cmd.Stderr = os.Stderr
	cmd.Env = cleanEnvNoClaude()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", command, err)
	}

	onUpdate := buildStreamUpdateCallback(watchdog, allPhases, phaseNum, statusPath)
	_, parseErr := ParseStreamEvents(stdout, onUpdate)
	waitErr := cmd.Wait()

	return classifyStreamResult(ctx, stallCtx, command, phaseNum, phaseTimeout, waitErr, parseErr, watchdog.eventCount.Load())
}

// buildStreamPhaseContext creates a context with optional timeout for a stream phase.
func buildStreamPhaseContext(phaseTimeout time.Duration) (context.Context, context.CancelFunc) {
	if phaseTimeout > 0 {
		return context.WithTimeout(context.Background(), phaseTimeout)
	}
	return context.Background(), func() {}
}

// normalizeCheckInterval returns checkInterval or a 1s default.
func normalizeCheckInterval(checkInterval time.Duration) time.Duration {
	if checkInterval <= 0 {
		return 1 * time.Second
	}
	return checkInterval
}

// startStreamWatchdogs launches startup and stall watchdog goroutines.
func startStreamWatchdogs(stallCtx context.Context, stallCancel context.CancelCauseFunc, watchdog *streamWatchdogState, startedAt time.Time, checkInterval, stallTimeout, startupTimeout time.Duration) {
	if startupTimeout > 0 {
		startupCheckInterval := checkInterval
		if startupCheckInterval > 5*time.Second {
			startupCheckInterval = 5 * time.Second
		}
		go runStartupWatchdog(stallCtx, stallCancel, &watchdog.eventCount, startedAt, startupCheckInterval, startupTimeout)
	}
	if stallTimeout > 0 {
		go runStallWatchdog(stallCtx, stallCancel, &watchdog.lastActivityUnix, checkInterval, stallTimeout)
	}
}

// runStartupWatchdog cancels the context if no stream events arrive within the timeout.
func runStartupWatchdog(ctx context.Context, cancel context.CancelCauseFunc, eventCount *atomic.Int64, startedAt time.Time, checkInterval, startupTimeout time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if eventCount.Load() > 0 {
				return
			}
			if time.Since(startedAt) > startupTimeout {
				cancel(fmt.Errorf("stream startup timeout: no events received after %s", startupTimeout))
				return
			}
		}
	}
}

// runStallWatchdog cancels the context if no stream activity occurs within the timeout.
func runStallWatchdog(ctx context.Context, cancel context.CancelCauseFunc, lastActivityUnix *atomic.Int64, checkInterval, stallTimeout time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			last := time.Unix(0, lastActivityUnix.Load())
			if time.Since(last) > stallTimeout {
				cancel(fmt.Errorf("stall detected: no stream activity for %s", stallTimeout))
				return
			}
		}
	}
}

// buildStreamUpdateCallback creates the onUpdate callback for ParseStreamEvents
// that records activity and merges progress into allPhases.
func buildStreamUpdateCallback(watchdog *streamWatchdogState, allPhases []PhaseProgress, phaseNum int, statusPath string) func(PhaseProgress) {
	phaseIdx := phaseNum - 1
	return func(p PhaseProgress) {
		watchdog.eventCount.Add(1)
		watchdog.lastActivityUnix.Store(time.Now().UnixNano())

		if phaseIdx >= 0 && phaseIdx < len(allPhases) {
			mergePhaseProgress(&allPhases[phaseIdx], p)
		}
		if writeErr := WriteLiveStatus(statusPath, allPhases, phaseIdx); writeErr != nil {
			VerbosePrintf("Warning: could not write live status: %v\n", writeErr)
		}
	}
}

// mergePhaseProgress updates dst with non-zero fields from src.
func mergePhaseProgress(dst *PhaseProgress, src PhaseProgress) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.CurrentAction != "" {
		dst.CurrentAction = src.CurrentAction
	}
	if src.RetryCount != 0 {
		dst.RetryCount = src.RetryCount
	}
	if src.LastError != "" {
		dst.LastError = src.LastError
	}
}

// classifyStreamResult examines the context, wait error, and parse error to
// produce the appropriate error for a completed stream-json phase.
func classifyStreamResult(ctx, stallCtx context.Context, command string, phaseNum int, phaseTimeout time.Duration, waitErr, parseErr error, eventCount int64) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("phase %d (%s) timed out after %s (set --phase-timeout to increase)", phaseNum, failReasonTimeout, phaseTimeout)
	}
	if cause := context.Cause(stallCtx); cause != nil && stallCtx.Err() != nil && ctx.Err() == nil {
		return fmt.Errorf("phase %d (%s): %w", phaseNum, failReasonStall, cause)
	}
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return fmt.Errorf("%s exited with code %d (%s): %w", command, exitErr.ExitCode(), failReasonExit, waitErr)
		}
		return fmt.Errorf("%s execution failed (%s): %w", command, failReasonUnknown, waitErr)
	}
	if parseErr != nil {
		return fmt.Errorf("stream parse error: %w", parseErr)
	}
	if eventCount == 0 {
		return fmt.Errorf("stream startup timeout: stream completed without parseable events")
	}
	return nil
}

// spawnClaudePhaseWithStream is the legacy wrapper pinned to the default runtime.
func spawnClaudePhaseWithStream(prompt, cwd, runID string, phaseNum int, statusPath string, allPhases []PhaseProgress, phaseTimeout, stallTimeout, streamStartupTimeout, checkInterval time.Duration) error {
	return spawnRuntimePhaseWithStream("claude", prompt, cwd, runID, phaseNum, statusPath, allPhases, phaseTimeout, stallTimeout, streamStartupTimeout, checkInterval)
}

func updateLivePhaseStatus(statusPath string, allPhases []PhaseProgress, phaseNum int, action string, retries int, lastErr string) {
	phaseIdx := phaseNum - 1
	if phaseIdx < 0 || phaseIdx >= len(allPhases) {
		return
	}

	p := allPhases[phaseIdx]
	if action != "" {
		p.CurrentAction = summarizeStatusAction(action)
	}
	p.RetryCount = retries
	p.LastError = summarizeStatusAction(lastErr)
	p.LastUpdate = time.Now()
	allPhases[phaseIdx] = p

	if err := WriteLiveStatus(statusPath, allPhases, phaseIdx); err != nil {
		VerbosePrintf("Warning: could not write live status: %v\n", err)
	}
}

// buildAllPhases constructs a []PhaseProgress with Name fields populated
// from the global phases slice, used as the initial state for live status tracking.
func buildAllPhases(phaseDefs []phase) []PhaseProgress {
	all := make([]PhaseProgress, len(phaseDefs))
	for i, p := range phaseDefs {
		all[i] = PhaseProgress{Name: p.Name, CurrentAction: "pending"}
	}
	return all
}

// cleanEnvNoClaude builds a clean env without CLAUDECODE to avoid nesting guard.
func cleanEnvNoClaude() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "CLAUDECODE=") || strings.HasPrefix(e, "CLAUDE_CODE_") {
			continue
		}
		env = append(env, e)
	}
	return env
}

// --- Runtime process helpers ---

// lookPath is the function used to resolve binary paths. Package-level for testability.
var lookPath = exec.LookPath

// spawnClaudeDirectGlobal is the package-level wrapper for spawnClaudeDirectImpl that
// reads phasedPhaseTimeout from the global (for spawnClaudePhase / spawnDirectFn fallback paths).
func spawnClaudeDirectGlobal(prompt, cwd string, phaseNum int) error {
	return spawnClaudeDirectImpl(prompt, cwd, phaseNum, phasedPhaseTimeout)
}

// spawnDirectFn is the function used to spawn claude directly. Package-level for testability.
var spawnDirectFn = spawnClaudeDirectGlobal

// stallCheckInterval controls how frequently the stall watchdog goroutine fires in
// spawnClaudePhaseWithStream. Overridable in tests to exercise stall detection quickly.
var stallCheckInterval = 30 * time.Second
