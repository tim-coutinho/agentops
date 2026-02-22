package vibecheck

import (
	"math"
	"sort"
	"time"
)

// MetricFlow measures consistency of work by computing the standard deviation
// of daily commit counts. Lower std dev = better flow. The returned value is
// a 0-100 score: 100 means perfectly even distribution, lower means spiky.
// Threshold: score >= 50 = good (passed).
func MetricFlow(events []TimelineEvent) Metric {
	if len(events) < 2 {
		return Metric{
			Name:      "flow",
			Value:     0,
			Threshold: 50,
			Passed:    false,
		}
	}

	dailyCounts := dailyCommitCounts(events)
	if len(dailyCounts) < 2 {
		return Metric{
			Name:      "flow",
			Value:     100,
			Threshold: 50,
			Passed:    true,
		}
	}

	mean, stddev := meanStddev(dailyCounts)

	// Convert stddev to a 0-100 score.
	// Coefficient of variation (CV) = stddev/mean.
	// Score = max(0, 100 - CV*100).
	// CV of 0 = score 100 (perfect), CV of 1+ = score 0 (very spiky).
	var score float64
	if mean > 0 {
		cv := stddev / mean
		score = math.Max(0, 100-cv*100)
	}

	// Round to 1 decimal.
	score = float64(int(score*10+0.5)) / 10.0

	return Metric{
		Name:      "flow",
		Value:     score,
		Threshold: 50,
		Passed:    score >= 50,
	}
}

// dailyCommitCounts returns a slice of commit counts per day, including
// zero-count days in the range between the first and last commit.
func dailyCommitCounts(events []TimelineEvent) []float64 {
	// Sort oldest first.
	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Build a map of day -> count.
	dayCounts := make(map[string]int)
	for _, e := range sorted {
		day := e.Timestamp.Format(time.DateOnly)
		dayCounts[day]++
	}

	// Generate all days in the range.
	first := sorted[0].Timestamp.Truncate(24 * time.Hour)
	last := sorted[len(sorted)-1].Timestamp.Truncate(24 * time.Hour)

	var counts []float64
	for d := first; !d.After(last); d = d.Add(24 * time.Hour) {
		day := d.Format(time.DateOnly)
		counts = append(counts, float64(dayCounts[day]))
	}

	return counts
}

// meanStddev computes the arithmetic mean and population standard deviation.
func meanStddev(values []float64) (float64, float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / n

	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= n

	return mean, math.Sqrt(variance)
}
