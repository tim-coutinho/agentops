package goals

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndLoadHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	entry1 := HistoryEntry{
		Timestamp:    "2026-01-01T10:00:00Z",
		GoalsPassing: 5,
		GoalsTotal:   8,
		Score:        62.5,
		SnapshotPath: "/tmp/snap1.json",
		GitSHA:       "abc123",
	}
	entry2 := HistoryEntry{
		Timestamp:    "2026-01-02T10:00:00Z",
		GoalsPassing: 7,
		GoalsTotal:   8,
		Score:        87.5,
		SnapshotPath: "/tmp/snap2.json",
		GitSHA:       "def456",
	}

	if err := AppendHistory(entry1, path); err != nil {
		t.Fatalf("AppendHistory entry1: %v", err)
	}
	if err := AppendHistory(entry2, path); err != nil {
		t.Fatalf("AppendHistory entry2: %v", err)
	}

	entries, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].GitSHA != "abc123" {
		t.Errorf("entry[0].GitSHA = %q, want abc123", entries[0].GitSHA)
	}
	if entries[1].Score != 87.5 {
		t.Errorf("entry[1].Score = %f, want 87.5", entries[1].Score)
	}
}

func TestLoadHistory_NonExistentFile(t *testing.T) {
	entries, err := LoadHistory("/nonexistent/history.jsonl")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty slice for missing file, got %d entries", len(entries))
	}
}

func TestLoadHistory_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyPath := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := LoadHistory(emptyPath)
	if err != nil {
		t.Fatalf("LoadHistory empty file: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty file, got %d", len(entries))
	}
}

func TestQueryHistory_FiltersByTime(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: "2026-01-01T10:00:00Z", GoalsPassing: 1},
		{Timestamp: "2026-01-02T10:00:00Z", GoalsPassing: 2},
		{Timestamp: "2026-01-03T10:00:00Z", GoalsPassing: 3},
	}

	since, _ := time.Parse(time.RFC3339, "2026-01-02T00:00:00Z")
	result := QueryHistory(entries, "", since)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries >= 2026-01-02, got %d", len(result))
	}
	if result[0].GoalsPassing != 2 {
		t.Errorf("first result GoalsPassing = %d, want 2", result[0].GoalsPassing)
	}
}

func TestQueryHistory_NoMatches(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: "2025-01-01T10:00:00Z", GoalsPassing: 1},
	}
	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	result := QueryHistory(entries, "", since)
	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

func TestQueryHistory_SkipsMalformedTimestamps(t *testing.T) {
	entries := []HistoryEntry{
		{Timestamp: "not-a-timestamp", GoalsPassing: 99},
		{Timestamp: "2026-01-02T10:00:00Z", GoalsPassing: 5},
	}
	since, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	result := QueryHistory(entries, "", since)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry (skipping malformed), got %d", len(result))
	}
	if result[0].GoalsPassing != 5 {
		t.Errorf("GoalsPassing = %d, want 5", result[0].GoalsPassing)
	}
}

func TestQueryHistory_EmptyEntries(t *testing.T) {
	since := time.Now()
	result := QueryHistory([]HistoryEntry{}, "", since)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestQueryHistory_GoalIDParameterIgnored(t *testing.T) {
	// goalID is currently unused but accepted; ensure no panic
	entries := []HistoryEntry{
		{Timestamp: "2026-01-01T10:00:00Z", GoalsPassing: 1},
	}
	since, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	result := QueryHistory(entries, "some-goal-id", since)
	if len(result) != 1 {
		t.Errorf("goalID filter should be ignored, expected 1 entry, got %d", len(result))
	}
}

func TestAppendHistory_OpenFileError(t *testing.T) {
	tmpDir := t.TempDir()
	readOnly := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnly, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0700) })

	entry := HistoryEntry{
		Timestamp:    "2026-01-01T10:00:00Z",
		GoalsPassing: 1,
		GoalsTotal:   1,
	}
	err := AppendHistory(entry, filepath.Join(readOnly, "history.jsonl"))
	if err == nil {
		t.Error("expected error when appending to file in read-only directory")
	}
}

func TestLoadHistory_PermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	histPath := filepath.Join(tmpDir, "history.jsonl")
	if err := os.WriteFile(histPath, []byte(`{"timestamp":"t1"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(histPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(histPath, 0644) })

	_, err := LoadHistory(histPath)
	if err == nil {
		t.Error("expected error when history file is unreadable")
	}
}

func TestLoadHistory_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	histPath := filepath.Join(tmpDir, "history.jsonl")
	if err := os.WriteFile(histPath, []byte("{bad json\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHistory(histPath)
	if err == nil {
		t.Error("expected error for malformed JSON in history")
	}
}

func TestAppendHistory_GoalsAdded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	entry := HistoryEntry{
		Timestamp:    "2026-01-01T10:00:00Z",
		GoalsPassing: 3,
		GoalsTotal:   5,
		GoalsAdded:   2,
		Score:        60.0,
	}
	if err := AppendHistory(entry, path); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	entries, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].GoalsAdded != 2 {
		t.Errorf("GoalsAdded = %d, want 2", entries[0].GoalsAdded)
	}
}

func TestAppendHistory_ReadOnlyDir(t *testing.T) {
	tmp := t.TempDir()
	readOnly := filepath.Join(tmp, "readonly")
	if err := os.MkdirAll(readOnly, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0700) })

	entry := HistoryEntry{
		Timestamp:    "2026-01-01T00:00:00Z",
		GoalsPassing: 1,
		GoalsTotal:   2,
		Score:        0.5,
	}
	err := AppendHistory(entry, filepath.Join(readOnly, "history.jsonl"))
	if err == nil {
		t.Error("expected error when appending to read-only directory")
	}
}

func TestLoadHistory_UnreadableFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "history.jsonl")

	// Write valid data
	if err := os.WriteFile(path, []byte(`{"timestamp":"2026-01-01T00:00:00Z"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Make file unreadable
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0600) })

	_, err := LoadHistory(path)
	if err == nil {
		t.Error("expected error when loading unreadable history file")
	}
}

func TestLoadHistory_MalformedJSONLine(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "history.jsonl")

	// Write malformed JSON
	if err := os.WriteFile(path, []byte("{invalid json\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHistory(path)
	if err == nil {
		t.Error("expected error when loading malformed JSON history")
	}
}

func TestLoadHistory_EmptyLines(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "history.jsonl")

	// Write valid JSON with empty lines interspersed
	content := `{"timestamp":"2026-01-01T00:00:00Z","goals_passing":1,"goals_total":2,"score":0.5}
` + "\n" + `{"timestamp":"2026-01-02T00:00:00Z","goals_passing":2,"goals_total":2,"score":1.0}
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (skipping empty line), got %d", len(entries))
	}
}

func TestLoadHistory_ScannerError(t *testing.T) {
	// Exercise the scanner.Err() error path (line 65-67).
	// Create a file with a line exceeding the default 64KB scanner buffer.
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	// Write a valid entry first
	validEntry := `{"timestamp":"2026-01-01T10:00:00Z","goal_id":"G1","delta":0.1,"score":0.8}`

	// Create a line that exceeds 64KB (default scanner buffer)
	hugeLine := make([]byte, 70*1024)
	for i := range hugeLine {
		hugeLine[i] = 'x'
	}

	content := validEntry + "\n" + string(hugeLine) + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHistory(path)
	if err == nil {
		t.Error("expected scanner error for line exceeding buffer")
	}
}
