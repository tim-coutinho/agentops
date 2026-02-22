package vibecheck

import "slices"

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
		case SeverityCritical:
			hasCritical = true
		case SeverityWarning:
			hasWarning = true
		}
	}

	if hasCritical {
		return HealthCritical
	}
	if hasWarning {
		return HealthWarning
	}
	return HealthHealthy
}

// sortOldestFirst sorts events by timestamp ascending (oldest first).
func sortOldestFirst(events []TimelineEvent) {
	slices.SortFunc(events, func(a, b TimelineEvent) int {
		return a.Timestamp.Compare(b.Timestamp)
	})
}
