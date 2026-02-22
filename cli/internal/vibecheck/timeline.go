package vibecheck

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ParseTimeline runs git log in repoPath for commits since the given time
// and returns a slice of TimelineEvents sorted newest-first.
func ParseTimeline(repoPath string, since time.Time) ([]TimelineEvent, error) {
	sinceStr := since.Format(time.RFC3339)

	// Use a delimiter unlikely to appear in commit messages.
	const delim = "|||"
	format := "%H" + delim + "%aI" + delim + "%an" + delim + "%s"

	cmd := exec.Command("git", "log",
		"--format="+format,
		"--numstat",
		"--since="+sinceStr,
	)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git log failed: %w: %s", err, stderr.String())
	}

	return parseGitLog(stdout.String(), delim)
}

// parseGitLog parses the combined --format + --numstat output from git log.
//
// The format alternates between a header line (fields separated by delim) and
// zero or more numstat lines (tab-separated: insertions, deletions, filename).
// Commits are separated by blank lines.
func parseGitLog(raw string, delim string) ([]TimelineEvent, error) {
	scanner := bufio.NewScanner(strings.NewReader(raw))

	var events []TimelineEvent
	var current *TimelineEvent

	for scanner.Scan() {
		line := scanner.Text()

		// Blank line: separator between commits (or trailing).
		if strings.TrimSpace(line) == "" {
			if current != nil {
				events = append(events, *current)
				current = nil
			}
			continue
		}

		// Try to parse as a header line.
		if parts := strings.SplitN(line, delim, 4); len(parts) == 4 {
			// Flush any pending event without a trailing blank line.
			if current != nil {
				events = append(events, *current)
			}

			ts, err := time.Parse(time.RFC3339, parts[1])
			if err != nil {
				return nil, fmt.Errorf("parsing timestamp %q: %w", parts[1], err)
			}

			current = &TimelineEvent{
				SHA:       parts[0],
				Timestamp: ts,
				Author:    parts[2],
				Message:   parts[3],
			}
			continue
		}

		// Otherwise treat as a numstat line: <insertions>\t<deletions>\t<file>
		if current != nil {
			fields := strings.SplitN(line, "\t", 3)
			if len(fields) == 3 {
				ins, _ := strconv.Atoi(fields[0])
				del, _ := strconv.Atoi(fields[1])
				current.Insertions += ins
				current.Deletions += del
				current.FilesChanged++
				current.Files = append(current.Files, fields[2])
			}
		}
	}

	// Flush last event if the output didn't end with a blank line.
	if current != nil {
		events = append(events, *current)
	}

	// Sort newest first.
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	return events, nil
}
