package goals

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Measurement captures the result of running a single goal's check command.
type Measurement struct {
	GoalID    string   `json:"goal_id"`
	Result    string   `json:"result"` // "pass", "fail", "skip", "error"
	Value     *float64 `json:"value,omitempty"`
	Threshold *float64 `json:"threshold,omitempty"`
	Duration  float64  `json:"duration_s"`
	Output    string   `json:"output,omitempty"`
	Weight    int      `json:"weight"`
}

// classifyResult maps command exit status to a result string.
func classifyResult(ctxErr, cmdErr error) string {
	switch {
	case errors.Is(ctxErr, context.DeadlineExceeded):
		return resultSkip
	case cmdErr != nil:
		return resultFail
	default:
		return resultPass
	}
}

// truncateOutput limits output to 500 characters and trims whitespace.
func truncateOutput(raw []byte) string {
	s := string(raw)
	if len(s) > 500 {
		s = s[:500]
	}
	return strings.TrimSpace(s)
}

// applyContinuousMetric parses a numeric value from output for continuous goals.
func applyContinuousMetric(m *Measurement, goal Goal) {
	if goal.Continuous == nil || m.Output == "" {
		return
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(m.Output), 64); err == nil {
		m.Value = &v
		t := goal.Continuous.Threshold
		m.Threshold = &t
	}
}

// MeasureOne runs a single goal's check command and returns a Measurement.
// Exit 0 = pass, non-zero = fail, context deadline exceeded = skip.
// Uses process groups so child processes are killed on timeout.
func MeasureOne(goal Goal, timeout time.Duration) Measurement {
	m := Measurement{GoalID: goal.ID, Weight: goal.Weight}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "bash", "-c", goal.Check)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Kill the entire process group, not just the parent.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 3 * time.Second
	out, err := cmd.CombinedOutput()
	m.Duration = time.Since(start).Seconds()
	m.Output = truncateOutput(out)
	m.Result = classifyResult(ctx.Err(), err)
	applyContinuousMetric(&m, goal)
	return m
}

// Measure runs all goals and returns a Snapshot. Meta-goals run first, then all others.
func Measure(gf *GoalFile, timeout time.Duration) *Snapshot {
	measurements := runGoals(gf.Goals, timeout)
	return &Snapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		GitSHA:    gitSHA(),
		Goals:     measurements,
		Summary:   computeSummary(measurements),
	}
}

// maxParallelGoals limits concurrent goal checks to avoid resource contention.
const maxParallelGoals = 4

// runGoals executes meta-goals first (sequential), then non-meta goals (parallel).
func runGoals(allGoals []Goal, timeout time.Duration) []Measurement {
	// Phase 1: meta-goals run sequentially (they may affect non-meta goals).
	var measurements []Measurement
	for _, g := range allGoals {
		if g.Type == GoalTypeMeta {
			measurements = append(measurements, MeasureOne(g, timeout))
		}
	}

	// Phase 2: non-meta goals run concurrently with a semaphore.
	var nonMeta []Goal
	for _, g := range allGoals {
		if g.Type != GoalTypeMeta {
			nonMeta = append(nonMeta, g)
		}
	}
	if len(nonMeta) == 0 {
		return measurements
	}

	results := make([]Measurement, len(nonMeta))
	sem := make(chan struct{}, maxParallelGoals)
	var wg sync.WaitGroup
	for i, g := range nonMeta {
		wg.Add(1)
		go func(idx int, goal Goal) {
			defer wg.Done()
			sem <- struct{}{}
			results[idx] = MeasureOne(goal, timeout)
			<-sem
		}(i, g)
	}
	wg.Wait()

	return append(measurements, results...)
}

// computeSummary aggregates pass/fail/skip counts and weighted score.
func computeSummary(measurements []Measurement) SnapshotSummary {
	var summary SnapshotSummary
	summary.Total = len(measurements)
	var weightedPass, weightedTotal int
	for _, m := range measurements {
		switch m.Result {
		case "pass":
			summary.Passing++
			weightedPass += m.Weight
			weightedTotal += m.Weight
		case "fail", "error":
			summary.Failing++
			weightedTotal += m.Weight
		case "skip":
			summary.Skipped++
		}
	}
	if weightedTotal > 0 {
		summary.Score = float64(weightedPass) / float64(weightedTotal) * 100
	}
	return summary
}

// gitSHA returns the short git SHA of HEAD, or "" on error.
func gitSHA() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
