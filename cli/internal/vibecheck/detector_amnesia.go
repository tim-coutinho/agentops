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
// fileEdit records a single edit to a file.
type fileEdit struct {
	ts  time.Time
	sha string
	msg string
}

// DetectContextAmnesia finds repeated rapid edits to the same file, indicating the agent lost context.
func DetectContextAmnesia(events []TimelineEvent) []Finding {
	if len(events) < amnesiaMinEdits {
		return nil
	}

	sorted := make([]TimelineEvent, len(events))
	copy(sorted, events)
	sortOldestFirst(sorted)

	fileEdits := buildFileEdits(sorted)

	var findings []Finding
	for file, edits := range fileEdits {
		if f, ok := detectAmnesiaInFile(file, edits); ok {
			findings = append(findings, f)
		}
	}
	return findings
}

// buildFileEdits groups events by file into timestamped edit lists.
func buildFileEdits(events []TimelineEvent) map[string][]fileEdit {
	result := make(map[string][]fileEdit)
	for _, ev := range events {
		for _, f := range ev.Files {
			result[f] = append(result[f], fileEdit{
				ts:  ev.Timestamp,
				sha: ev.SHA,
				msg: ev.Message,
			})
		}
	}
	return result
}

// detectAmnesiaInFile checks if a file was edited amnesiaMinEdits+ times within amnesiaWindow.
func detectAmnesiaInFile(file string, edits []fileEdit) (Finding, bool) {
	if len(edits) < amnesiaMinEdits {
		return Finding{}, false
	}

	sort.Slice(edits, func(i, j int) bool {
		return edits[i].ts.Before(edits[j].ts)
	})

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
			return Finding{
				Severity: SeverityWarning,
				Category: "context-amnesia",
				Message:  file + " modified " + itoa(count) + " times within 1 hour, suggesting lost context",
				File:     file,
			}, true
		}
	}
	return Finding{}, false
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
