package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

func TestComputePlanChecksum(t *testing.T) {
	// Create temp file
	tmpDir, err := os.MkdirTemp("", "plans_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	content := "# Plan Content\n\nThis is test content."
	path := filepath.Join(tmpDir, "test-plan.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Test checksum computation
	t.Run("valid file", func(t *testing.T) {
		checksum, err := computePlanChecksum(path)
		if err != nil {
			t.Errorf("computePlanChecksum() error = %v", err)
		}
		if len(checksum) != 16 { // 8 bytes = 16 hex chars
			t.Errorf("checksum length = %d, want 16", len(checksum))
		}
	})

	// Test same content = same checksum
	t.Run("deterministic", func(t *testing.T) {
		cs1, _ := computePlanChecksum(path)
		cs2, _ := computePlanChecksum(path)
		if cs1 != cs2 {
			t.Errorf("checksums differ for same file: %s vs %s", cs1, cs2)
		}
	})

	// Test nonexistent file
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := computePlanChecksum(filepath.Join(tmpDir, "nonexistent.md"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestCreatePlanEntry(t *testing.T) {
	modTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	entry := createPlanEntry(
		"/path/to/plan.md",
		modTime,
		"/project/path",
		"my-plan",
		"ol-123",
		"abc123",
	)

	if entry.Path != "/path/to/plan.md" {
		t.Errorf("Path = %q, want %q", entry.Path, "/path/to/plan.md")
	}
	if entry.CreatedAt != modTime {
		t.Errorf("CreatedAt = %v, want %v", entry.CreatedAt, modTime)
	}
	if entry.ProjectPath != "/project/path" {
		t.Errorf("ProjectPath = %q, want %q", entry.ProjectPath, "/project/path")
	}
	if entry.PlanName != "my-plan" {
		t.Errorf("PlanName = %q, want %q", entry.PlanName, "my-plan")
	}
	if entry.BeadsID != "ol-123" {
		t.Errorf("BeadsID = %q, want %q", entry.BeadsID, "ol-123")
	}
	if entry.Checksum != "abc123" {
		t.Errorf("Checksum = %q, want %q", entry.Checksum, "abc123")
	}
	if entry.Status != types.PlanStatusActive {
		t.Errorf("Status = %v, want %v", entry.Status, types.PlanStatusActive)
	}
}

func TestBuildBeadsIDIndex(t *testing.T) {
	entries := []types.PlanManifestEntry{
		{Path: "/a.md", BeadsID: "ol-001"},
		{Path: "/b.md", BeadsID: "ol-002"},
		{Path: "/c.md", BeadsID: ""}, // No beads ID
		{Path: "/d.md", BeadsID: "ol-003"},
	}

	index := buildBeadsIDIndex(entries)

	if len(index) != 3 {
		t.Errorf("index length = %d, want 3", len(index))
	}
	if index["ol-001"] != 0 {
		t.Errorf("index[ol-001] = %d, want 0", index["ol-001"])
	}
	if index["ol-002"] != 1 {
		t.Errorf("index[ol-002] = %d, want 1", index["ol-002"])
	}
	if index["ol-003"] != 3 {
		t.Errorf("index[ol-003] = %d, want 3", index["ol-003"])
	}
	if _, ok := index[""]; ok {
		t.Error("empty beads ID should not be indexed")
	}
}

func TestSyncEpicStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      types.PlanStatus
		beadsStatus string
		wantChanged bool
		wantStatus  types.PlanStatus
	}{
		{
			name:        "active to completed",
			status:      types.PlanStatusActive,
			beadsStatus: "closed",
			wantChanged: true,
			wantStatus:  types.PlanStatusCompleted,
		},
		{
			name:        "completed to active",
			status:      types.PlanStatusCompleted,
			beadsStatus: "open",
			wantChanged: true,
			wantStatus:  types.PlanStatusActive,
		},
		{
			name:        "no change active",
			status:      types.PlanStatusActive,
			beadsStatus: "open",
			wantChanged: false,
			wantStatus:  types.PlanStatusActive,
		},
		{
			name:        "no change completed",
			status:      types.PlanStatusCompleted,
			beadsStatus: "closed",
			wantChanged: false,
			wantStatus:  types.PlanStatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []types.PlanManifestEntry{
				{Status: tt.status},
			}
			changed := syncEpicStatus(entries, 0, tt.beadsStatus)
			if changed != tt.wantChanged {
				t.Errorf("syncEpicStatus() changed = %v, want %v", changed, tt.wantChanged)
			}
			if entries[0].Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", entries[0].Status, tt.wantStatus)
			}
		})
	}
}

func TestCountUnlinkedEntries(t *testing.T) {
	entries := []types.PlanManifestEntry{
		{PlanName: "plan-1", BeadsID: "ol-001"},
		{PlanName: "plan-2", BeadsID: ""},
		{PlanName: "plan-3", BeadsID: "ol-003"},
		{PlanName: "plan-4", BeadsID: ""},
	}

	// Note: countUnlinkedEntries also calls VerbosePrintf, which is fine in tests
	count := countUnlinkedEntries(entries)
	if count != 2 {
		t.Errorf("countUnlinkedEntries() = %d, want 2", count)
	}
}

func TestAppendManifestEntry(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.jsonl")

	entry := types.PlanManifestEntry{
		Path:     "/plan1.md",
		PlanName: "plan1",
		Status:   types.PlanStatusActive,
	}

	t.Run("creates new file", func(t *testing.T) {
		if err := appendManifestEntry(manifestPath, entry); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content, _ := os.ReadFile(manifestPath)
		if len(content) == 0 {
			t.Error("expected non-empty file")
		}
	})

	t.Run("appends to existing", func(t *testing.T) {
		entry2 := types.PlanManifestEntry{
			Path:     "/plan2.md",
			PlanName: "plan2",
			Status:   types.PlanStatusActive,
		}
		if err := appendManifestEntry(manifestPath, entry2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		entries, err := loadManifest(manifestPath)
		if err != nil {
			t.Fatalf("loadManifest error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries after append, got %d", len(entries))
		}
	})
}

func TestLoadManifest(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := loadManifest(filepath.Join(tmpDir, "nope.jsonl"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		emptyPath := filepath.Join(tmpDir, "empty.jsonl")
		if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		entries, err := loadManifest(emptyPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("got %d entries from empty file, want 0", len(entries))
		}
	})

	t.Run("skips invalid lines", func(t *testing.T) {
		mixedPath := filepath.Join(tmpDir, "mixed.jsonl")
		entry := types.PlanManifestEntry{Path: "/valid.md", PlanName: "valid"}
		line, _ := json.Marshal(entry)
		content := string(line) + "\nnot json\n" + string(line) + "\n"
		if err := os.WriteFile(mixedPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		entries, err := loadManifest(mixedPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("got %d entries, want 2 (invalid line skipped)", len(entries))
		}
	})
}

func TestSaveManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "save.jsonl")

	entries := []types.PlanManifestEntry{
		{Path: "/a.md", PlanName: "a", Status: types.PlanStatusActive},
		{Path: "/b.md", PlanName: "b", Status: types.PlanStatusCompleted},
	}

	if err := saveManifest(manifestPath, entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify round-trip
	loaded, err := loadManifest(manifestPath)
	if err != nil {
		t.Fatalf("loadManifest error: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("got %d entries after save/load, want 2", len(loaded))
	}
	if loaded[0].PlanName != "a" {
		t.Errorf("first entry name = %q, want %q", loaded[0].PlanName, "a")
	}
}

func TestBuildBeadsStatusIndex(t *testing.T) {
	epics := []beadsEpic{
		{ID: "ol-001", Status: "open"},
		{ID: "ol-002", Status: "closed"},
		{ID: "ol-003", Status: "open"},
	}

	index := buildBeadsStatusIndex(epics)

	if len(index) != 3 {
		t.Errorf("index length = %d, want 3", len(index))
	}
	if index["ol-001"] != "open" {
		t.Errorf("index[ol-001] = %q, want %q", index["ol-001"], "open")
	}
	if index["ol-002"] != "closed" {
		t.Errorf("index[ol-002] = %q, want %q", index["ol-002"], "closed")
	}
}

func TestDetectStatusDrifts(t *testing.T) {
	byBeadsID := map[string]*types.PlanManifestEntry{
		"ol-001": {PlanName: "plan-1", BeadsID: "ol-001", Status: types.PlanStatusActive},
		"ol-002": {PlanName: "plan-2", BeadsID: "ol-002", Status: types.PlanStatusCompleted},
		"ol-003": {PlanName: "plan-3", BeadsID: "ol-003", Status: types.PlanStatusActive},
	}

	beadsIndex := map[string]string{
		"ol-001": "open", // matches
		"ol-002": "open", // mismatch: manifest=completed, beads=open
		// ol-003 missing from beads
	}

	drifts := detectStatusDrifts(byBeadsID, beadsIndex)

	// Should find 2 drifts: status_mismatch for ol-002, missing_beads for ol-003
	if len(drifts) != 2 {
		t.Errorf("detectStatusDrifts() found %d drifts, want 2", len(drifts))
	}

	// Check for expected drift types
	foundMismatch := false
	foundMissing := false
	for _, d := range drifts {
		if d.Type == "status_mismatch" && d.BeadsID == "ol-002" {
			foundMismatch = true
		}
		if d.Type == "missing_beads" && d.BeadsID == "ol-003" {
			foundMissing = true
		}
	}
	if !foundMismatch {
		t.Error("expected to find status_mismatch for ol-002")
	}
	if !foundMissing {
		t.Error("expected to find missing_beads for ol-003")
	}
}

func TestDetectOrphanedEntries(t *testing.T) {
	entries := []types.PlanManifestEntry{
		{PlanName: "linked-1", BeadsID: "ol-001"},
		{PlanName: "orphan-1", BeadsID: ""},
		{PlanName: "linked-2", BeadsID: "ol-002"},
		{PlanName: "orphan-2", BeadsID: ""},
	}

	drifts := detectOrphanedEntries(entries)

	if len(drifts) != 2 {
		t.Errorf("detectOrphanedEntries() found %d drifts, want 2", len(drifts))
	}

	for _, d := range drifts {
		if d.Type != "orphaned" {
			t.Errorf("drift type = %q, want 'orphaned'", d.Type)
		}
	}
}
