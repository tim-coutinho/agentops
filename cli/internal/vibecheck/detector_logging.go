package vibecheck

import "regexp"

// loggingMessagePatterns matches commit messages that suggest logging/debug work.
var loggingMessagePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\badd(ed|ing)?\s+(log|debug|print)`),
	regexp.MustCompile(`(?i)\blog(ging)?\b`),
	regexp.MustCompile(`(?i)\bdebug(ging)?\b`),
	regexp.MustCompile(`(?i)\bprint\s+statement`),
	regexp.MustCompile(`(?i)\btrace\b`),
	regexp.MustCompile(`(?i)\bconsole\b`),
	regexp.MustCompile(`(?i)\btemporary\b`),
	regexp.MustCompile(`(?i)\btemp\b`),
	regexp.MustCompile(`(?i)\bwip\b`),
	regexp.MustCompile(`(?i)\binvestigat`),
	regexp.MustCompile(`(?i)\bdiagnos`),
}

// maxSmallDiffLines is the threshold for a "small" diff (insertions+deletions).
const maxSmallDiffLines = 20

// maxConsecutiveLogging is the threshold for flagging consecutive logging commits.
const maxConsecutiveLogging = 3

// isLoggingMessage returns true if the commit message matches logging patterns.
func isLoggingMessage(msg string) bool {
	for _, p := range loggingMessagePatterns {
		if p.MatchString(msg) {
			return true
		}
	}
	return false
}

// DetectLoggingOnly detects commits where the only changes appear to be
// log/print/debug statements, identified by commit messages containing
// logging keywords combined with small diffs.
func DetectLoggingOnly(events []TimelineEvent) []Finding {
	if len(events) == 0 {
		return nil
	}

	// Sort oldest first.
	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	sortOldestFirst(sorted)

	var findings []Finding

	// Track consecutive logging commits.
	consecutive := 0
	maxConsec := 0

	for _, ev := range sorted {
		diffSize := ev.Insertions + ev.Deletions
		if isLoggingMessage(ev.Message) && diffSize <= maxSmallDiffLines && diffSize > 0 {
			consecutive++
			if consecutive > maxConsec {
				maxConsec = consecutive
			}
		} else {
			consecutive = 0
		}
	}

	if maxConsec >= maxConsecutiveLogging {
		findings = append(findings, Finding{
			Severity: "warning",
			Category: "logging-only",
			Message:  itoa(maxConsec) + " consecutive commits appear to be logging/debug only, suggesting a debug spiral",
		})
	}

	return findings
}
