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

// isSmallLoggingCommit returns true if the event looks like a small
// log/print/debug-only commit.
func isSmallLoggingCommit(ev TimelineEvent) bool {
	diffSize := ev.Insertions + ev.Deletions
	return isLoggingMessage(ev.Message) && diffSize > 0 && diffSize <= maxSmallDiffLines
}

// maxConsecutiveRun returns the longest run of consecutive true values
// from the predicate applied to each event.
func maxConsecutiveRun(events []TimelineEvent, pred func(TimelineEvent) bool) int {
	maxRun, run := 0, 0
	for _, ev := range events {
		if pred(ev) {
			run++
			if run > maxRun {
				maxRun = run
			}
		} else {
			run = 0
		}
	}
	return maxRun
}

// DetectLoggingOnly detects commits where the only changes appear to be
// log/print/debug statements, identified by commit messages containing
// logging keywords combined with small diffs.
func DetectLoggingOnly(events []TimelineEvent) []Finding {
	if len(events) == 0 {
		return nil
	}

	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	sortOldestFirst(sorted)

	maxConsec := maxConsecutiveRun(sorted, isSmallLoggingCommit)

	if maxConsec >= maxConsecutiveLogging {
		return []Finding{{
			Severity: SeverityWarning,
			Category: "logging-only",
			Message:  itoa(maxConsec) + " consecutive commits appear to be logging/debug only, suggesting a debug spiral",
		}}
	}

	return nil
}
