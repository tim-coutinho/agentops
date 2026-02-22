package vibecheck

import "fmt"

// ComputeMetrics runs all five metric calculations on the given events
// and returns a map keyed by metric name.
func ComputeMetrics(events []TimelineEvent) map[string]Metric {
	return map[string]Metric{
		"velocity": MetricVelocity(events),
		"rework":   MetricRework(events),
		"trust":    MetricTrust(events),
		"spirals":  MetricSpirals(events),
		"flow":     MetricFlow(events),
	}
}

// ComputeOverallRating produces an aggregate score (0-100) and a letter grade
// from the computed metrics. Each metric contributes equally.
//
// Scoring per metric:
//   - passed = 20 points (full share of 100/5)
//   - not passed = scaled partial credit based on how close the value is to threshold
//
// Grade thresholds:
//   - A: 80-100
//   - B: 60-79
//   - C: 40-59
//   - D: 20-39
//   - F: 0-19
func ComputeOverallRating(metrics map[string]Metric) (float64, string) {
	if len(metrics) == 0 {
		return 0, "F"
	}

	total := 0.0
	count := 0
	for _, m := range metrics {
		count++
		if m.Passed {
			total += 20
		} else {
			// Partial credit: ratio of value to threshold (clamped to [0, 20]).
			partial := metricPartialCredit(m)
			total += partial
		}
	}

	// Normalize if we somehow have a different number of metrics.
	if count != 5 {
		total = total / float64(count) * 5
	}

	// Clamp to [0, 100].
	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}

	grade := scoreToGrade(total)
	return total, grade
}

// metricPartialCredit computes partial credit (0-20) for a metric that did
// not pass its threshold.
func metricPartialCredit(m Metric) float64 {
	if m.Threshold == 0 {
		// Threshold is 0 (e.g., spirals where 0 is perfect).
		// If not passed, value > 0, so no partial credit.
		return 0
	}

	switch m.Name {
	case "rework":
		// Lower is better: partial = (threshold - value) / threshold * 20
		if m.Value >= m.Threshold {
			ratio := (100 - m.Value) / (100 - m.Threshold)
			if ratio < 0 {
				return 0
			}
			return ratio * 20
		}
		return 20

	case "velocity", "trust", "flow":
		// Higher is better: partial = value / threshold * 20
		if m.Threshold > 0 {
			ratio := m.Value / m.Threshold
			if ratio > 1 {
				ratio = 1
			}
			return ratio * 20
		}
		return 0

	default:
		return 0
	}
}

// scoreToGrade converts a 0-100 score to a letter grade.
func scoreToGrade(score float64) string {
	switch {
	case score >= 80:
		return "A"
	case score >= 60:
		return "B"
	case score >= 40:
		return "C"
	case score >= 20:
		return "D"
	default:
		return "F"
	}
}

// FormatMetricsSummary produces a human-readable summary of metrics.
func FormatMetricsSummary(metrics map[string]Metric, score float64, grade string) string {
	order := []string{"velocity", "rework", "trust", "spirals", "flow"}
	result := fmt.Sprintf("Overall: %.0f/100 (%s)\n\n", score, grade)

	for _, name := range order {
		m, ok := metrics[name]
		if !ok {
			continue
		}
		status := "PASS"
		if !m.Passed {
			status = "FAIL"
		}
		result += fmt.Sprintf("  %-10s %6.1f  (threshold: %.1f)  [%s]\n",
			m.Name, m.Value, m.Threshold, status)
	}

	return result
}
