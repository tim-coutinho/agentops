package goals

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"time"
)

// HistoryEntry records aggregate goal status at a point in time.
type HistoryEntry struct {
	Timestamp    string  `json:"timestamp"`
	GoalsPassing int     `json:"goals_passing"`
	GoalsTotal   int     `json:"goals_total"`
	GoalsAdded   int     `json:"goals_added,omitempty"`
	Score        float64 `json:"score"`
	SnapshotPath string  `json:"snapshot_path"`
	GitSHA       string  `json:"git_sha"`
}

// AppendHistory appends a single history entry as a JSON line to the given file.
// Creates the file if it does not exist.
func AppendHistory(entry HistoryEntry, path string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// openOrEmpty opens a file for reading, returning (nil, nil) if it does not exist.
func openOrEmpty(path string) (*os.File, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return f, err
}

// parseHistoryLines parses JSONL lines from a scanner into history entries.
func parseHistoryLines(scanner *bufio.Scanner) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e HistoryEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []HistoryEntry{}
	}
	return entries, nil
}

// LoadHistory reads all history entries from a JSON-lines file.
// Returns an empty slice (not an error) if the file does not exist.
func LoadHistory(path string) ([]HistoryEntry, error) {
	f, err := openOrEmpty(path)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return []HistoryEntry{}, nil
	}
	defer f.Close()

	return parseHistoryLines(bufio.NewScanner(f))
}

// QueryHistory filters history entries to those with Timestamp >= since.
// The goalID parameter is accepted for future per-goal filtering but currently
// has no effect (history entries are aggregate, not per-goal).
func QueryHistory(entries []HistoryEntry, goalID string, since time.Time) []HistoryEntry {
	var result []HistoryEntry
	for _, e := range entries {
		t, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		if !t.Before(since) {
			result = append(result, e)
		}
	}
	if result == nil {
		result = []HistoryEntry{}
	}
	return result
}
