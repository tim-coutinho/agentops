package goals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SnapshotSummary holds aggregate counts and weighted score.
type SnapshotSummary struct {
	Total   int     `json:"total"`
	Passing int     `json:"passing"`
	Failing int     `json:"failing"`
	Skipped int     `json:"skipped"`
	Score   float64 `json:"score"`
}

// Snapshot captures a point-in-time measurement of all goals.
type Snapshot struct {
	Timestamp string          `json:"timestamp"`
	GitSHA    string          `json:"git_sha"`
	Goals     []Measurement   `json:"goals"`
	Summary   SnapshotSummary `json:"summary"`
}

// SaveSnapshot writes a snapshot to disk as indented JSON.
// Returns the path of the written file.
func SaveSnapshot(s *Snapshot, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating snapshot dir: %w", err)
	}

	ts := time.Now().UTC().Format("2006-01-02T15-04-05")
	filename := filepath.Join(dir, ts+".json")

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling snapshot: %w", err)
	}

	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return "", fmt.Errorf("writing snapshot: %w", err)
	}

	return filename, nil
}

// LoadSnapshot reads a snapshot from a JSON file.
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing snapshot %s: %w", path, err)
	}

	return &s, nil
}

// LoadLatestSnapshot finds the most recent snapshot in dir by filename
// (timestamps sort lexicographically).
func LoadLatestSnapshot(dir string) (*Snapshot, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var jsonFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			jsonFiles = append(jsonFiles, e.Name())
		}
	}

	if len(jsonFiles) == 0 {
		return nil, fmt.Errorf("no snapshots found in %s", dir)
	}

	sort.Strings(jsonFiles)
	latest := filepath.Join(dir, jsonFiles[len(jsonFiles)-1])

	return LoadSnapshot(latest)
}
