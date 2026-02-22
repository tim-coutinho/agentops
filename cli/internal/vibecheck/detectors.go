package vibecheck

import "sort"

// RunDetectors runs all detectors against the given events and returns
// the aggregated findings.
func RunDetectors(events []TimelineEvent) []Finding {
	var findings []Finding
	findings = append(findings, DetectTestsLie(events)...)
	findings = append(findings, DetectContextAmnesia(events)...)
	findings = append(findings, DetectInstructionDrift(events)...)
	findings = append(findings, DetectLoggingOnly(events)...)
	return findings
}

// ClassifyHealth determines the overall health based on findings.
//
// Rules (from the TypeScript reference):
//   - Any "critical" severity finding -> "critical"
//   - Any "warning" severity finding  -> "warning"
//   - Otherwise                       -> "healthy"
func ClassifyHealth(findings []Finding) string {
	hasCritical := false
	hasWarning := false

	for _, f := range findings {
		switch f.Severity {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		}
	}

	if hasCritical {
		return "critical"
	}
	if hasWarning {
		return "warning"
	}
	return "healthy"
}

// sortOldestFirst sorts events by timestamp ascending (oldest first).
func sortOldestFirst(events []TimelineEvent) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
}
