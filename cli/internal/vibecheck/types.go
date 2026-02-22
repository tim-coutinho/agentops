// Package vibecheck provides types and tools for analyzing git commit timelines
// and producing vibe-check results (metrics, findings, and grades).
package vibecheck

import "time"

// TimelineEvent represents a single commit in the git timeline.
type TimelineEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	SHA          string    `json:"sha"`
	Author       string    `json:"author"`
	Message      string    `json:"message"`
	FilesChanged int       `json:"files_changed"`
	Insertions   int       `json:"insertions"`
	Deletions    int       `json:"deletions"`
	Tags         []string  `json:"tags,omitempty"`
	Files        []string  `json:"files,omitempty"`
}

// VibeCheckResult is the top-level output of a vibe-check analysis.
type VibeCheckResult struct {
	Score    float64            `json:"score"`
	Grade    string             `json:"grade"`
	Events   []TimelineEvent    `json:"events"`
	Metrics  map[string]float64 `json:"metrics"`
	Findings []Finding          `json:"findings,omitempty"`
}

// Severity constants for findings.
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
)

// Health classification constants.
const (
	HealthCritical = "critical"
	HealthWarning  = "warning"
	HealthHealthy  = "healthy"
)

// Finding represents a single observation surfaced during analysis.
type Finding struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// Metric captures a named measurement with a pass/fail threshold.
type Metric struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Passed    bool    `json:"passed"`
}
