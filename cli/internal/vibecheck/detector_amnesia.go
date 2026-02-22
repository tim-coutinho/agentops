package vibecheck

import (
	"sort"
	"time"
)

// amnesiaWindow is the time window for detecting repeated work on the same file.
const amnesiaWindow = 60 * time.Minute

// amnesiaMinEdits is the minimum number of times a file must be modified
// within the window to trigger a finding.
const amnesiaMinEdits = 3

// DetectContextAmnesia detects files that are modified repeatedly within a
// short timeframe, suggesting the agent lost context and re-did work.
func DetectContextAmnesia(events []TimelineEvent) []Finding {
	if len(events) < amnesiaMinEdits {
		return nil
	}

	// Sort oldest first.
	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	sortOldestFirst(sorted)

	// Build a map of file -> list of commit timestamps+SHAs.
	type fileEdit struct {
		ts  time.Time
		sha string
		msg string
	}
	fileEdits := make(map[string][]fileEdit)

	for _, ev := range sorted {
		for _, f := range ev.Files {
			fileEdits[f] = append(fileEdits[f], fileEdit{
				ts:  ev.Timestamp,
				sha: ev.SHA,
				msg: ev.Message,
			})
		}
	}

	var findings []Finding

	for file, edits := range fileEdits {
		if len(edits) < amnesiaMinEdits {
			continue
		}

		// Sort edits by time.
		sort.Slice(edits, func(i, j int) bool {
			return edits[i].ts.Before(edits[j].ts)
		})

		// Sliding window: check for amnesiaMinEdits edits within amnesiaWindow.
		for i := 0; i <= len(edits)-amnesiaMinEdits; i++ {
			windowEnd := edits[i].ts.Add(amnesiaWindow)
			count := 0
			for j := i; j < len(edits); j++ {
				if edits[j].ts.After(windowEnd) {
					break
				}
				count++
			}
			if count >= amnesiaMinEdits {
				findings = append(findings, Finding{
					Severity: "warning",
					Category: "context-amnesia",
					Message:  file + " modified " + itoa(count) + " times within 1 hour, suggesting lost context",
					File:     file,
				})
				break // One finding per file.
			}
		}
	}

	return findings
}

// itoa converts an int to a string without importing strconv in this file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
