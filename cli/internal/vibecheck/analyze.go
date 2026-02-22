package vibecheck

import (
	"fmt"
	"time"
)

// AnalyzeOptions configures an Analyze operation.
type AnalyzeOptions struct {
	// RepoPath is the path to the git repository.
	RepoPath string
	// Since specifies the time window (events after this time).
	Since time.Time
}

// Analyze orchestrates the full vibe-check pipeline:
// 1. Parse the timeline from git log
// 2. Compute metrics
// 3. Run detectors
// 4. Compute overall rating
// 5. Return combined result
func Analyze(opts AnalyzeOptions) (*VibeCheckResult, error) {
	// Validate inputs
	if opts.RepoPath == "" {
		return nil, ErrRepoPathRequired
	}

	// Parse timeline from git log
	events, err := ParseTimeline(opts.RepoPath, opts.Since)
	if err != nil {
		return nil, fmt.Errorf("parsing timeline: %w", err)
	}

	// Compute metrics
	metricsMap := ComputeMetrics(events)

	// Compute overall rating
	score, grade := ComputeOverallRating(metricsMap)

	// Run detectors to find issues
	findings := RunDetectors(events)
	if findings == nil {
		findings = []Finding{}
	}

	// Convert metrics map to the VibeCheckResult format
	metricsResult := make(map[string]float64)
	for name, m := range metricsMap {
		metricsResult[name] = m.Value
	}

	// Build and return result
	result := &VibeCheckResult{
		Score:    score,
		Grade:    grade,
		Events:   events,
		Metrics:  metricsResult,
		Findings: findings,
	}

	return result, nil
}
