package goals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadSnapshot(t *testing.T) {
	dir := t.TempDir()
	snap := &Snapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		GitSHA:    "abc123",
		Goals: []Measurement{
			{GoalID: "test-goal", Result: "pass", Weight: 5, Duration: 0.1},
		},
		Summary: SnapshotSummary{
			Total: 1, Passing: 1, Score: 100.0,
		},
	}

	path, err := SaveSnapshot(snap, dir)
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path from SaveSnapshot")
	}

	loaded, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if loaded.GitSHA != snap.GitSHA {
		t.Errorf("GitSHA = %q, want %q", loaded.GitSHA, snap.GitSHA)
	}
	if len(loaded.Goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(loaded.Goals))
	}
	if loaded.Goals[0].GoalID != "test-goal" {
		t.Errorf("GoalID = %q, want test-goal", loaded.Goals[0].GoalID)
	}
	if loaded.Summary.Score != 100.0 {
		t.Errorf("Score = %f, want 100.0", loaded.Summary.Score)
	}
}

func TestSaveSnapshot_CreatesDir(t *testing.T) {
	base := t.TempDir()
	newDir := filepath.Join(base, "nested", "snapshots")
	snap := &Snapshot{Timestamp: "2026-01-01T00:00:00Z"}

	path, err := SaveSnapshot(snap, newDir)
	if err != nil {
		t.Fatalf("SaveSnapshot with new dir: %v", err)
	}
	if _, err := os.Stat(newDir); err != nil {
		t.Errorf("directory not created: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("snapshot file not created: %v", err)
	}
}

func TestLoadSnapshot_NotFound(t *testing.T) {
	_, err := LoadSnapshot("/nonexistent/path/snap.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadSnapshot_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not valid json}"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSnapshot(bad)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadLatestSnapshot(t *testing.T) {
	dir := t.TempDir()

	// Write two snapshots with different timestamps in the filename
	for i, ts := range []string{"2026-01-01T10-00-00", "2026-01-02T10-00-00"} {
		snap := &Snapshot{
			Timestamp: fmt.Sprintf("2026-01-0%dT10:00:00Z", i+1),
			GitSHA:    fmt.Sprintf("sha%d", i),
		}
		data, _ := json.MarshalIndent(snap, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, ts+".json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	latest, err := LoadLatestSnapshot(dir)
	if err != nil {
		t.Fatalf("LoadLatestSnapshot: %v", err)
	}
	// Latest should be the second one (lexicographically larger filename)
	if latest.GitSHA != "sha1" {
		t.Errorf("GitSHA = %q, want sha1 (latest snapshot)", latest.GitSHA)
	}
}

func TestLoadLatestSnapshot_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadLatestSnapshot(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestLoadLatestSnapshot_DirNotFound(t *testing.T) {
	_, err := LoadLatestSnapshot("/nonexistent/dir")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLoadLatestSnapshot_IgnoresNonJSON(t *testing.T) {
	dir := t.TempDir()
	// Write only a non-JSON file
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLatestSnapshot(dir)
	if err == nil {
		t.Fatal("expected error when no JSON files present")
	}
}
