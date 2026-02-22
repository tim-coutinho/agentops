package vibecheck

import (
	"regexp"
	"strings"
	"time"
)

// successClaimPatterns matches commit messages that claim success.
var successClaimPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bfixed\s+(it|the|this|that|bug|issue|problem)`),
	regexp.MustCompile(`(?i)\bworking\b`),
	regexp.MustCompile(`(?i)\bdone\b`),
	regexp.MustCompile(`(?i)\bcomplete[ds]?\b`),
	regexp.MustCompile(`(?i)\b(bug|issue|problem)\s+resolved\b`),
	regexp.MustCompile(`(?i)\btests?\s+pass(es|ing)?\b`),
	regexp.MustCompile(`(?i)\bgreen\b`),
	regexp.MustCompile(`(?i)\bsuccessful(ly)?\b`),
	regexp.MustCompile(`(?i)\ball\s+tests?\b`),
	regexp.MustCompile(`(?i)\bready\b`),
	regexp.MustCompile(`(?i)\bshould\s+work`),
	regexp.MustCompile(`(?i)\bnow\s+works?\b`),
	regexp.MustCompile(`(?i)\bit\s+works?\b`),
}

// tentativePatterns marks commits as uncertain, excluding them from lies.
var tentativePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\btry\b`),
	regexp.MustCompile(`(?i)\battempt`),
	regexp.MustCompile(`(?i)\bmaybe\b`),
	regexp.MustCompile(`(?i)\bwip\b`),
	regexp.MustCompile(`(?i)\bwork\s*in\s*progress`),
	regexp.MustCompile(`(?i)\bexperiment`),
	regexp.MustCompile(`(?i)\btest(ing)?\b`),
	regexp.MustCompile(`(?i)\bdebug`),
	regexp.MustCompile(`(?i)\binvestigat`),
}

// followUpWindow is the max time after a success claim to look for fix commits.
const followUpWindow = 30 * time.Minute

// claimsSuccess returns true if the message matches any success claim pattern.
func claimsSuccess(msg string) bool {
	for _, p := range successClaimPatterns {
		if p.MatchString(msg) {
			return true
		}
	}
	return false
}

// isTentative returns true if the message contains tentative language.
func isTentative(msg string) bool {
	for _, p := range tentativePatterns {
		if p.MatchString(msg) {
			return true
		}
	}
	return false
}

// isFixMessage returns true if the commit message looks like a fix.
func isFixMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.HasPrefix(lower, "fix") ||
		strings.Contains(lower, "fix:") ||
		strings.Contains(lower, "bugfix") ||
		strings.Contains(lower, "hotfix") ||
		strings.Contains(lower, "patch:")
}

// hasFileOverlap returns true if two file lists share any entry.
func hasFileOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, f := range a {
		set[f] = struct{}{}
	}
	for _, f := range b {
		if _, ok := set[f]; ok {
			return true
		}
	}
	return false
}

// DetectTestsLie detects commits that claim success but are quickly followed
// by fix commits on the same files. This suggests the claim was premature.
func DetectTestsLie(events []TimelineEvent) []Finding {
	if len(events) < 2 {
		return nil
	}

	// Sort oldest first for forward scanning.
	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	sortOldestFirst(sorted)

	var findings []Finding

	for i, ev := range sorted {
		msg := ev.Message
		if !claimsSuccess(msg) || isTentative(msg) {
			continue
		}

		// Look at following commits within the window.
		for j := i + 1; j < len(sorted); j++ {
			next := sorted[j]
			gap := next.Timestamp.Sub(ev.Timestamp)
			if gap > followUpWindow {
				break
			}

			if !isFixMessage(next.Message) {
				continue
			}

			// Check file overlap or both have no file info (fallback).
			overlap := hasFileOverlap(ev.Files, next.Files)
			bothEmpty := len(ev.Files) == 0 && len(next.Files) == 0

			if overlap || bothEmpty {
				findings = append(findings, Finding{
					Severity: "critical",
					Category: "tests-passing-lie",
					Message:  "Claimed success (" + ev.Message + ") but fix followed within " + gap.String() + ": " + next.Message,
				})
				break // One finding per claim commit.
			}
		}
	}

	return findings
}
