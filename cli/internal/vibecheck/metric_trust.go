package vibecheck

import (
	"strings"
)

// MetricTrust computes the ratio of test-related commits to code commits.
// Higher is better. A test commit is one whose message references tests.
// Threshold: >0.3 = good (passed).
func MetricTrust(events []TimelineEvent) Metric {
	if len(events) == 0 {
		return Metric{
			Name:      "trust",
			Value:     0,
			Threshold: 0.3,
			Passed:    false,
		}
	}

	testCommits := 0
	codeCommits := 0

	for _, e := range events {
		msg := strings.ToLower(strings.TrimSpace(e.Message))
		if isTestCommit(msg) {
			testCommits++
		} else {
			codeCommits++
		}
	}

	if codeCommits == 0 {
		// All commits are test commits; trust is perfect.
		return Metric{
			Name:      "trust",
			Value:     1.0,
			Threshold: 0.3,
			Passed:    true,
		}
	}

	ratio := float64(testCommits) / float64(codeCommits)
	// Round to 2 decimal places.
	ratio = float64(int(ratio*100+0.5)) / 100.0

	return Metric{
		Name:      "trust",
		Value:     ratio,
		Threshold: 0.3,
		Passed:    ratio > 0.3,
	}
}

// isTestCommit returns true if the commit message suggests test-related work.
func isTestCommit(msg string) bool {
	prefixes := []string{"test:", "test(", "tests:", "tests("}
	for _, p := range prefixes {
		if strings.HasPrefix(msg, p) {
			return true
		}
	}
	keywords := []string{"add test", "update test", "fix test", "write test", "testing"}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
