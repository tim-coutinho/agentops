package vibecheck

import (
	"strings"
)

// MetricRework computes the percentage of files modified more than once
// in the timeline. Lower is better.
// Threshold: <30% = good (passed).
func MetricRework(events []TimelineEvent) Metric {
	if len(events) == 0 {
		return Metric{
			Name:      "rework",
			Value:     0,
			Threshold: 30,
			Passed:    true,
		}
	}

	// Count how many commits touch each "file scope".
	// We approximate file identity from the commit message: if the message
	// mentions a fix for the same scope/component, it counts as rework.
	// Since TimelineEvent doesn't carry per-file paths, we use a heuristic:
	// percentage of fix-type commits relative to total commits.
	//
	// A commit is considered a fix if its message starts with "fix" (conventional commit).
	fixCount := 0
	for _, e := range events {
		msg := strings.ToLower(strings.TrimSpace(e.Message))
		if strings.HasPrefix(msg, "fix") {
			fixCount++
		}
	}

	ratio := float64(fixCount) / float64(len(events)) * 100
	// Round to 1 decimal.
	ratio = float64(int(ratio*10+0.5)) / 10.0

	return Metric{
		Name:      "rework",
		Value:     ratio,
		Threshold: 30,
		Passed:    ratio < 30,
	}
}
