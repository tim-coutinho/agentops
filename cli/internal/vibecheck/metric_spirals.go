package vibecheck

import (
	"slices"
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

// spiralTracker tracks consecutive fix commits on the same component.
type spiralTracker struct {
	spirals       int
	consecutive   int
	lastComponent string
}

// flush closes out the current chain, counting it as a spiral if >= 3.
func (st *spiralTracker) flush() {
	if st.consecutive >= 3 {
		st.spirals++
	}
	st.consecutive = 0
	st.lastComponent = ""
}

// feedFix processes a fix commit for the given component.
func (st *spiralTracker) feedFix(comp string) {
	if comp == st.lastComponent || st.lastComponent == "" {
		st.consecutive++
		st.lastComponent = comp
	} else {
		st.flush()
		st.consecutive = 1
		st.lastComponent = comp
	}
}

// countSpirals counts the number of fix-chain spirals (3+ consecutive fix
// commits on the same component).
func countSpirals(events []TimelineEvent) int {
	// Sort oldest first for sequential analysis.
	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	slices.SortFunc(sorted, func(a, b TimelineEvent) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	var st spiralTracker
	for _, e := range sorted {
		msg := strings.ToLower(strings.TrimSpace(e.Message))
		if !strings.HasPrefix(msg, "fix") {
			st.flush()
			continue
		}
		st.feedFix(extractComponent(msg))
	}
	st.flush()

	return st.spirals
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
