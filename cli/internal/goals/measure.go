package goals

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
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

// MeasureOne runs a single goal's check command and returns a Measurement.
// Exit 0 = pass, non-zero = fail, context deadline exceeded = skip.
func MeasureOne(goal Goal, timeout time.Duration) Measurement {
	m := Measurement{
		GoalID: goal.ID,
		Weight: goal.Weight,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "bash", "-c", goal.Check)
	out, err := cmd.CombinedOutput()
	m.Duration = time.Since(start).Seconds()

	// Truncate output to 500 chars.
	output := string(out)
	if len(output) > 500 {
		output = output[:500]
	}
	m.Output = strings.TrimSpace(output)

	switch {
	case ctx.Err() == context.DeadlineExceeded:
		m.Result = "skip"
	case err != nil:
		m.Result = "fail"
	default:
		m.Result = "pass"
	}

	// For continuous metrics, try to parse a numeric value from output.
	if goal.Continuous != nil && m.Output != "" {
		if v, parseErr := strconv.ParseFloat(strings.TrimSpace(m.Output), 64); parseErr == nil {
			m.Value = &v
			t := goal.Continuous.Threshold
			m.Threshold = &t
		}
	}

	return m
}

// Measure runs all goals and returns a Snapshot. Meta-goals run first, then all others.
func Measure(gf *GoalFile, timeout time.Duration) *Snapshot {
	var measurements []Measurement

	// Run meta-goals first.
	for _, g := range gf.Goals {
		if g.Type == GoalTypeMeta {
			measurements = append(measurements, MeasureOne(g, timeout))
		}
	}

	// Run non-meta goals.
	for _, g := range gf.Goals {
		if g.Type != GoalTypeMeta {
			measurements = append(measurements, MeasureOne(g, timeout))
		}
	}

	// Compute summary.
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
			// Skipped goals excluded from weighted score.
		}
	}

	if weightedTotal > 0 {
		summary.Score = float64(weightedPass) / float64(weightedTotal) * 100
	}

	sha := gitSHA()

	return &Snapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		GitSHA:    sha,
		Goals:     measurements,
		Summary:   summary,
	}
}

// gitSHA returns the short git SHA of HEAD, or "" on error.
func gitSHA() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
