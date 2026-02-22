package vibecheck

import (
	"sort"
	"strings"
)

// MetricSpirals detects revert-fix-revert cycles in the timeline.
// A spiral is defined as 3+ consecutive fix commits touching the same
// scope/component. Threshold: 0 spirals = good (passed).
func MetricSpirals(events []TimelineEvent) Metric {
	if len(events) < 3 {
		return Metric{
			Name:      "spirals",
			Value:     0,
			Threshold: 0,
			Passed:    true,
		}
	}

	spiralCount := countSpirals(events)

	return Metric{
		Name:      "spirals",
		Value:     float64(spiralCount),
		Threshold: 0,
		Passed:    spiralCount == 0,
	}
}

// countSpirals counts the number of fix-chain spirals (3+ consecutive fix
// commits on the same component).
func countSpirals(events []TimelineEvent) int {
	// Sort oldest first for sequential analysis.
	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	spirals := 0
	consecutive := 0
	lastComponent := ""

	for _, e := range sorted {
		msg := strings.ToLower(strings.TrimSpace(e.Message))
		if !strings.HasPrefix(msg, "fix") {
			// Non-fix commit breaks any chain.
			if consecutive >= 3 {
				spirals++
			}
			consecutive = 0
			lastComponent = ""
			continue
		}

		comp := extractComponent(msg)
		if comp == lastComponent || lastComponent == "" {
			consecutive++
			lastComponent = comp
		} else {
			// Different component resets the chain.
			if consecutive >= 3 {
				spirals++
			}
			consecutive = 1
			lastComponent = comp
		}
	}

	// Flush final chain.
	if consecutive >= 3 {
		spirals++
	}

	return spirals
}

// extractComponent extracts a component/scope from a commit message.
// It handles conventional commit format: "fix(scope): description"
// and falls back to the first meaningful word after "fix:".
func extractComponent(msg string) string {
	// Check for conventional commit scope: fix(scope): ...
	if idx := strings.Index(msg, "("); idx >= 0 {
		if end := strings.Index(msg[idx:], ")"); end >= 0 {
			scope := msg[idx+1 : idx+end]
			if scope != "" {
				return scope
			}
		}
	}

	// Strip "fix:" or "fix " prefix and take first word.
	stripped := strings.TrimPrefix(msg, "fix:")
	stripped = strings.TrimPrefix(stripped, "fix ")
	stripped = strings.TrimSpace(stripped)

	words := strings.Fields(stripped)
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?-")
		if len(w) > 2 {
			return w
		}
	}

	return "unknown"
}
