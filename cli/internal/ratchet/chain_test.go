package ratchet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helper to write a JSONL chain file with metadata and entries.
func writeJSONLChain(t *testing.T, dir string, id string, epicID string, entries []ChainEntry) string {
	t.Helper()
	chainDir := filepath.Join(dir, ".agents", "ao")
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(chainDir, ChainFile)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()

	meta := struct {
		ID      string    `json:"id"`
		Started time.Time `json:"started"`
		EpicID  string    `json:"epic_id,omitempty"`
	}{ID: id, Started: time.Now(), EpicID: epicID}
	line, _ := json.Marshal(meta)
	_, _ = f.Write(append(line, '\n'))

	for _, e := range entries {
		line, _ := json.Marshal(e)
		_, _ = f.Write(append(line, '\n'))
	}
	return path
}

// helper to write a legacy YAML chain file.
func writeLegacyChain(t *testing.T, dir string, content string) string {
	t.Helper()
	legacyDir := filepath.Join(dir, ".agents", "provenance")
	if err := os.MkdirAll(legacyDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(legacyDir, LegacyChainFile)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadChainJSONL(t *testing.T) {
	tmp := t.TempDir()
	entries := []ChainEntry{
		{Step: StepResearch, Timestamp: time.Now(), Output: "r1.md", Locked: true},
		{Step: StepPlan, Timestamp: time.Now(), Output: "p1.md", Locked: false},
	}
	writeJSONLChain(t, tmp, "test-chain-1", "ag-123", entries)

	chain, err := LoadChain(tmp)
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}

	if chain.ID != "test-chain-1" {
		t.Errorf("expected ID='test-chain-1', got %q", chain.ID)
	}
	if chain.EpicID != "ag-123" {
		t.Errorf("expected EpicID='ag-123', got %q", chain.EpicID)
	}
	if len(chain.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(chain.Entries))
	}
	if chain.Entries[0].Step != StepResearch {
		t.Errorf("expected first entry step=research, got %q", chain.Entries[0].Step)
	}
	if chain.Entries[1].Step != StepPlan {
		t.Errorf("expected second entry step=plan, got %q", chain.Entries[1].Step)
	}
}

func TestLoadChainNoAgentsDir(t *testing.T) {
	tmp := t.TempDir()
	chain, err := LoadChain(tmp)
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(chain.Entries) != 0 {
		t.Errorf("expected 0 entries for missing .agents, got %d", len(chain.Entries))
	}
	if chain.ID == "" {
		t.Error("expected auto-generated ID, got empty")
	}
}

func TestLoadChainLegacyYAML(t *testing.T) {
	tmp := t.TempDir()
	// Create .agents dir so findAgentsDir succeeds
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatalf("mkdir .agents: %v", err)
	}

	yaml := `id: legacy-1
started: "2026-01-15T10:00:00Z"
epic_id: ag-old
chain:
  - step: research
    timestamp: "2026-01-15T10:01:00Z"
    output: research.md
    locked: true
  - step: plan
    timestamp: "2026-01-15T10:02:00Z"
    output: plan.md
    locked: false
    skipped: true
    reason: "not needed"
`
	writeLegacyChain(t, tmp, yaml)

	chain, err := LoadChain(tmp)
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}

	if chain.ID != "legacy-1" {
		t.Errorf("expected ID='legacy-1', got %q", chain.ID)
	}
	if chain.EpicID != "ag-old" {
		t.Errorf("expected EpicID='ag-old', got %q", chain.EpicID)
	}
	if len(chain.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(chain.Entries))
	}
	if chain.Entries[1].Skipped != true {
		t.Error("expected second entry Skipped=true")
	}
	if chain.Entries[1].Reason != "not needed" {
		t.Errorf("expected Reason='not needed', got %q", chain.Entries[1].Reason)
	}
}

func TestLoadChainEmptyJSONL(t *testing.T) {
	tmp := t.TempDir()
	chainDir := filepath.Join(tmp, ".agents", "ao")
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Create .agents dir marker
	// Write only metadata line, no entries
	f, err := os.Create(filepath.Join(chainDir, ChainFile))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	meta := struct {
		ID      string    `json:"id"`
		Started time.Time `json:"started"`
	}{ID: "empty-chain", Started: time.Now()}
	line, _ := json.Marshal(meta)
	_, _ = f.Write(append(line, '\n'))
	f.Close()

	chain, err := LoadChain(tmp)
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if chain.ID != "empty-chain" {
		t.Errorf("expected ID='empty-chain', got %q", chain.ID)
	}
	if len(chain.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(chain.Entries))
	}
}

func TestSaveAndReload(t *testing.T) {
	tmp := t.TempDir()
	chainPath := filepath.Join(tmp, ".agents", "ao", ChainFile)

	chain := &Chain{
		ID:      "save-test",
		Started: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		EpicID:  "ag-save",
		Entries: []ChainEntry{
			{Step: StepResearch, Timestamp: time.Now(), Output: "r.md", Locked: true},
			{Step: StepPlan, Timestamp: time.Now(), Output: "p.md", Locked: false, Input: "r.md"},
		},
	}
	chain.SetPath(chainPath)

	if err := chain.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload
	loaded, err := loadJSONLChain(chainPath)
	if err != nil {
		t.Fatalf("loadJSONLChain: %v", err)
	}
	if loaded.ID != "save-test" {
		t.Errorf("expected ID='save-test', got %q", loaded.ID)
	}
	if loaded.EpicID != "ag-save" {
		t.Errorf("expected EpicID='ag-save', got %q", loaded.EpicID)
	}
	if len(loaded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Entries))
	}
	if loaded.Entries[1].Input != "r.md" {
		t.Errorf("expected Input='r.md', got %q", loaded.Entries[1].Input)
	}
}

func TestSaveNoPathError(t *testing.T) {
	chain := &Chain{ID: "no-path"}
	err := chain.Save()
	if err == nil {
		t.Fatal("expected error when path is empty")
	}
	if err.Error() != "chain has no path set" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAppendToEmptyFile(t *testing.T) {
	tmp := t.TempDir()
	chainPath := filepath.Join(tmp, "chain.jsonl")

	chain := &Chain{
		ID:      "append-test",
		Started: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Entries: []ChainEntry{},
	}
	chain.SetPath(chainPath)

	entry := ChainEntry{
		Step:      StepResearch,
		Timestamp: time.Now(),
		Output:    "research.md",
		Locked:    true,
	}

	if err := chain.Append(entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if len(chain.Entries) != 1 {
		t.Errorf("expected 1 in-memory entry, got %d", len(chain.Entries))
	}

	// Reload from disk and verify metadata was written
	loaded, err := loadJSONLChain(chainPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if loaded.ID != "append-test" {
		t.Errorf("expected ID='append-test', got %q", loaded.ID)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry on disk, got %d", len(loaded.Entries))
	}
}

func TestAppendToExistingFile(t *testing.T) {
	tmp := t.TempDir()
	entries := []ChainEntry{
		{Step: StepResearch, Timestamp: time.Now(), Output: "r.md", Locked: true},
	}
	chainPath := writeJSONLChain(t, tmp, "append-existing", "", entries)

	chain, err := loadJSONLChain(chainPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	newEntry := ChainEntry{
		Step:      StepPlan,
		Timestamp: time.Now(),
		Output:    "plan.md",
		Locked:    false,
	}
	if err := chain.Append(newEntry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if len(chain.Entries) != 2 {
		t.Errorf("expected 2 in-memory entries, got %d", len(chain.Entries))
	}

	// Reload and verify append was persisted
	loaded, err := loadJSONLChain(chainPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	// Appended to existing file, so metadata + original + new = 3 lines total
	// But metadata is only from original write; loaded should have original + appended
	if len(loaded.Entries) != 2 {
		t.Errorf("expected 2 entries on disk, got %d", len(loaded.Entries))
	}
}

func TestAppendNoPathError(t *testing.T) {
	chain := &Chain{ID: "no-path"}
	err := chain.Append(ChainEntry{Step: StepResearch})
	if err == nil {
		t.Fatal("expected error when path is empty")
	}
	if err.Error() != "chain has no path set" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetLatest(t *testing.T) {
	chain := &Chain{
		Entries: []ChainEntry{
			{Step: StepResearch, Output: "r1.md"},
			{Step: StepPlan, Output: "p1.md"},
			{Step: StepResearch, Output: "r2.md"},
		},
	}

	tests := []struct {
		name     string
		step     Step
		wantNil  bool
		wantOut  string
	}{
		{
			name:    "returns latest of multiple entries",
			step:    StepResearch,
			wantOut: "r2.md",
		},
		{
			name:    "returns only entry",
			step:    StepPlan,
			wantOut: "p1.md",
		},
		{
			name:    "returns nil for missing step",
			step:    StepVibe,
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := chain.GetLatest(tc.step)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil entry")
			}
			if got.Output != tc.wantOut {
				t.Errorf("expected Output=%q, got %q", tc.wantOut, got.Output)
			}
		})
	}
}

func TestIsLocked(t *testing.T) {
	chain := &Chain{
		Entries: []ChainEntry{
			{Step: StepResearch, Locked: true},
			{Step: StepPlan, Locked: false},
		},
	}

	tests := []struct {
		name string
		step Step
		want bool
	}{
		{"locked step", StepResearch, true},
		{"unlocked step", StepPlan, false},
		{"missing step", StepVibe, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := chain.IsLocked(tc.step); got != tc.want {
				t.Errorf("IsLocked(%q) = %v, want %v", tc.step, got, tc.want)
			}
		})
	}
}

func TestGetStatus(t *testing.T) {
	chain := &Chain{
		Entries: []ChainEntry{
			{Step: StepResearch, Locked: true},
			{Step: StepPlan, Locked: false},
			{Step: StepImplement, Skipped: true},
		},
	}

	tests := []struct {
		name string
		step Step
		want StepStatus
	}{
		{"locked step", StepResearch, StatusLocked},
		{"in-progress step", StepPlan, StatusInProgress},
		{"skipped step", StepImplement, StatusSkipped},
		{"pending step", StepVibe, StatusPending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := chain.GetStatus(tc.step); got != tc.want {
				t.Errorf("GetStatus(%q) = %q, want %q", tc.step, got, tc.want)
			}
		})
	}
}

func TestGetStatusSkippedTakesPriority(t *testing.T) {
	// When both Skipped and Locked are true, Skipped wins
	chain := &Chain{
		Entries: []ChainEntry{
			{Step: StepResearch, Locked: true, Skipped: true},
		},
	}
	if got := chain.GetStatus(StepResearch); got != StatusSkipped {
		t.Errorf("expected StatusSkipped when both Skipped and Locked, got %q", got)
	}
}

func TestGetAllStatus(t *testing.T) {
	chain := &Chain{
		Entries: []ChainEntry{
			{Step: StepResearch, Locked: true},
			{Step: StepPlan, Locked: false},
		},
	}

	status := chain.GetAllStatus()

	// Should have entries for all steps
	allSteps := AllSteps()
	if len(status) != len(allSteps) {
		t.Errorf("expected %d statuses, got %d", len(allSteps), len(status))
	}

	if status[StepResearch] != StatusLocked {
		t.Errorf("expected research=locked, got %q", status[StepResearch])
	}
	if status[StepPlan] != StatusInProgress {
		t.Errorf("expected plan=in_progress, got %q", status[StepPlan])
	}
	if status[StepVibe] != StatusPending {
		t.Errorf("expected vibe=pending, got %q", status[StepVibe])
	}
}

func TestPathAndSetPath(t *testing.T) {
	chain := &Chain{ID: "path-test"}
	if chain.Path() != "" {
		t.Errorf("expected empty path, got %q", chain.Path())
	}

	chain.SetPath("/tmp/test/chain.jsonl")
	if chain.Path() != "/tmp/test/chain.jsonl" {
		t.Errorf("expected '/tmp/test/chain.jsonl', got %q", chain.Path())
	}
}

func TestMigrateChain(t *testing.T) {
	tmp := t.TempDir()
	// Create .agents dir
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatalf("mkdir .agents: %v", err)
	}

	yaml := `id: migrate-1
started: "2026-01-15T10:00:00Z"
chain:
  - step: research
    timestamp: "2026-01-15T10:01:00Z"
    output: research.md
    locked: true
`
	writeLegacyChain(t, tmp, yaml)

	if err := MigrateChain(tmp); err != nil {
		t.Fatalf("MigrateChain: %v", err)
	}

	// Verify new JSONL file was created
	newPath := filepath.Join(tmp, ".agents", "ao", ChainFile)
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Fatal("expected new chain file to exist after migration")
	}

	// Reload and verify
	loaded, err := loadJSONLChain(newPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if loaded.ID != "migrate-1" {
		t.Errorf("expected ID='migrate-1', got %q", loaded.ID)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}
}

func TestMigrateChainNoAgentsDir(t *testing.T) {
	tmp := t.TempDir()
	err := MigrateChain(tmp)
	if err == nil {
		t.Fatal("expected error when no .agents dir")
	}
}

func TestMigrateChainNoLegacyFile(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err := MigrateChain(tmp)
	if err == nil {
		t.Fatal("expected error when no legacy file")
	}
}

func TestLoadJSONLSkipsMalformedLines(t *testing.T) {
	tmp := t.TempDir()
	chainDir := filepath.Join(tmp, ".agents", "ao")
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(chainDir, ChainFile)

	// Write metadata + valid entry + malformed line + valid entry
	meta := `{"id":"malformed-test","started":"2026-02-10T12:00:00Z"}`
	e1 := `{"step":"research","timestamp":"2026-02-10T12:00:00Z","output":"r.md","locked":true}`
	bad := `{this is not valid json`
	e2 := `{"step":"plan","timestamp":"2026-02-10T12:01:00Z","output":"p.md","locked":false}`
	content := meta + "\n" + e1 + "\n" + bad + "\n" + e2 + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	chain, err := loadJSONLChain(path)
	if err != nil {
		t.Fatalf("loadJSONLChain: %v", err)
	}

	// Should have 2 valid entries, skipping the malformed line
	if len(chain.Entries) != 2 {
		t.Errorf("expected 2 entries (skipping malformed), got %d", len(chain.Entries))
	}
}

func TestLoadJSONLSkipsEmptyLines(t *testing.T) {
	tmp := t.TempDir()
	chainDir := filepath.Join(tmp, ".agents", "ao")
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(chainDir, ChainFile)

	meta := `{"id":"empty-lines","started":"2026-02-10T12:00:00Z"}`
	e1 := `{"step":"research","timestamp":"2026-02-10T12:00:00Z","output":"r.md","locked":true}`
	content := meta + "\n\n" + e1 + "\n\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	chain, err := loadJSONLChain(path)
	if err != nil {
		t.Fatalf("loadJSONLChain: %v", err)
	}

	if chain.ID != "empty-lines" {
		t.Errorf("expected ID='empty-lines', got %q", chain.ID)
	}
	if len(chain.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(chain.Entries))
	}
}

func TestLoadJSONLBadMetadata(t *testing.T) {
	tmp := t.TempDir()
	chainDir := filepath.Join(tmp, ".agents", "ao")
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(chainDir, ChainFile)

	content := "{bad metadata\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := loadJSONLChain(path)
	if err == nil {
		t.Fatal("expected error for bad metadata")
	}
}

func TestFindAgentsDirWalksUp(t *testing.T) {
	tmp := t.TempDir()
	// Create .agents at root, then start from a nested dir
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	nested := filepath.Join(tmp, "a", "b", "c")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, err := findAgentsDir(nested)
	if err != nil {
		t.Fatalf("findAgentsDir: %v", err)
	}
	want := filepath.Join(tmp, ".agents")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestFindAgentsDirNotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := findAgentsDir(tmp)
	if err == nil {
		t.Fatal("expected error when no .agents dir exists")
	}
}

func TestLegacyChainEmptyStarted(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Legacy chain with no started field and no timestamps
	yaml := `id: no-times
chain:
  - step: research
    output: r.md
    locked: true
`
	writeLegacyChain(t, tmp, yaml)

	chain, err := LoadChain(tmp)
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}

	if chain.Started.IsZero() {
		t.Error("expected non-zero Started when legacy has empty started")
	}
	if chain.Entries[0].Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp when legacy entry has empty timestamp")
	}
}

func TestGetLatestMultipleUpdates(t *testing.T) {
	// Simulate a step being recorded multiple times (re-run scenario)
	chain := &Chain{
		Entries: []ChainEntry{
			{Step: StepResearch, Output: "v1.md", Locked: false},
			{Step: StepResearch, Output: "v2.md", Locked: false},
			{Step: StepResearch, Output: "v3.md", Locked: true},
		},
	}

	got := chain.GetLatest(StepResearch)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Output != "v3.md" {
		t.Errorf("expected Output='v3.md', got %q", got.Output)
	}
	if !got.Locked {
		t.Error("expected latest to be locked")
	}
}

func TestGetLatestEmptyChain(t *testing.T) {
	chain := &Chain{Entries: []ChainEntry{}}
	if got := chain.GetLatest(StepResearch); got != nil {
		t.Errorf("expected nil for empty chain, got %+v", got)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	// Path includes non-existent directories
	chainPath := filepath.Join(tmp, "deep", "nested", "dir", ChainFile)

	chain := &Chain{
		ID:      "dir-create",
		Started: time.Now(),
		Entries: []ChainEntry{},
	}
	chain.SetPath(chainPath)

	if err := chain.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(chainPath); os.IsNotExist(err) {
		t.Fatal("expected chain file to exist after Save")
	}
}

func TestAppendMultipleEntries(t *testing.T) {
	tmp := t.TempDir()
	chainPath := filepath.Join(tmp, "chain.jsonl")

	chain := &Chain{
		ID:      "multi-append",
		Started: time.Now(),
		Entries: []ChainEntry{},
	}
	chain.SetPath(chainPath)

	steps := []Step{StepResearch, StepPlan, StepImplement, StepVibe}
	for _, s := range steps {
		entry := ChainEntry{
			Step:      s,
			Timestamp: time.Now(),
			Output:    string(s) + ".md",
			Locked:    true,
		}
		if err := chain.Append(entry); err != nil {
			t.Fatalf("Append(%s): %v", s, err)
		}
	}

	if len(chain.Entries) != 4 {
		t.Errorf("expected 4 in-memory entries, got %d", len(chain.Entries))
	}

	// Reload and verify
	loaded, err := loadJSONLChain(chainPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(loaded.Entries) != 4 {
		t.Errorf("expected 4 entries on disk, got %d", len(loaded.Entries))
	}
}

func TestChainSaveNoPath(t *testing.T) {
	chain := &Chain{
		ID:      "no-path",
		Started: time.Now(),
	}
	// path is empty
	err := chain.Save()
	if err == nil {
		t.Error("expected error when saving chain with no path")
	}
	if !strings.Contains(err.Error(), "no path set") {
		t.Errorf("expected 'no path set' error, got: %v", err)
	}
}

func TestChainAppendNoPath(t *testing.T) {
	chain := &Chain{
		ID:      "no-path",
		Started: time.Now(),
	}
	err := chain.Append(ChainEntry{Step: StepResearch})
	if err == nil {
		t.Error("expected error when appending to chain with no path")
	}
	if !strings.Contains(err.Error(), "no path set") {
		t.Errorf("expected 'no path set' error, got: %v", err)
	}
}

func TestChainSaveReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	readOnly := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnly, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0700) })

	chain := &Chain{
		ID:      "readonly-test",
		Started: time.Now(),
	}
	chain.SetPath(filepath.Join(readOnly, "sub", "chain.jsonl"))
	err := chain.Save()
	if err == nil {
		t.Error("expected error when saving to read-only directory")
	}
}

func TestLoadChainNonexistent(t *testing.T) {
	chain, err := LoadChain("/nonexistent/path")
	if err != nil {
		t.Fatalf("LoadChain should not error for nonexistent dir: %v", err)
	}
	if chain == nil {
		t.Fatal("expected non-nil chain for nonexistent path")
	}
	if len(chain.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(chain.Entries))
	}
}

func TestChainAppendReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	readOnly := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnly, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0700) })

	chain := &Chain{
		ID:      "append-readonly",
		Started: time.Now(),
	}
	chain.SetPath(filepath.Join(readOnly, "sub", "chain.jsonl"))
	err := chain.Append(ChainEntry{Step: StepResearch})
	if err == nil {
		t.Error("expected error when appending to chain in read-only directory")
	}
}

func TestLoadJSONLChainFileNotExist(t *testing.T) {
	_, err := loadJSONLChain("/nonexistent/chain.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent JSONL chain file")
	}
}

func TestLegacyChainBadYAML(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatal(err)
	}
	// Write invalid YAML to legacy chain
	writeLegacyChain(t, tmp, ":::invalid yaml[[[")

	// LoadChain should fall through to creating a new chain
	chain, err := LoadChain(tmp)
	if err != nil {
		t.Fatalf("LoadChain should not error for bad legacy YAML: %v", err)
	}
	// Should have created a new chain (not legacy), with no entries
	if len(chain.Entries) != 0 {
		t.Errorf("expected 0 entries for bad legacy YAML, got %d", len(chain.Entries))
	}
}

func TestMigrateChainBadLegacyYAML(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatal(err)
	}
	writeLegacyChain(t, tmp, ":::bad yaml content[[[")

	err := MigrateChain(tmp)
	if err == nil {
		t.Error("expected error when migrating bad YAML chain")
	}
	if !strings.Contains(err.Error(), "load legacy chain") {
		t.Errorf("expected 'load legacy chain' error, got: %v", err)
	}
}

func TestChain_Save_ReadOnlyDir(t *testing.T) {
	// Exercise the open file error path in Save.
	tmp := t.TempDir()
	readOnlyDir := filepath.Join(tmp, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0700) })

	chain := &Chain{
		ID:      "test-chain",
		path:    filepath.Join(readOnlyDir, "chain.jsonl"),
		Started: time.Now(),
	}

	err := chain.Save()
	if err == nil {
		t.Fatal("expected error when saving to read-only directory")
	}
	if !strings.Contains(err.Error(), "open chain file") {
		t.Errorf("expected 'open chain file' error, got: %v", err)
	}
}

func TestChain_Append_ReadOnlyDir(t *testing.T) {
	// Exercise the open file error path in Append.
	tmp := t.TempDir()
	readOnlyDir := filepath.Join(tmp, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0700) })

	chain := &Chain{
		ID:      "test-chain",
		path:    filepath.Join(readOnlyDir, "chain.jsonl"),
		Started: time.Now(),
	}

	err := chain.Append(ChainEntry{
		Step: StepResearch,
	})
	if err == nil {
		t.Fatal("expected error when appending to chain in read-only directory")
	}
	if !strings.Contains(err.Error(), "open chain file") {
		t.Errorf("expected 'open chain file' error, got: %v", err)
	}
}

func TestChain_Save_NoPath(t *testing.T) {
	chain := &Chain{ID: "test-chain", Started: time.Now()}
	err := chain.Save()
	if err == nil {
		t.Fatal("expected error when saving chain with no path")
	}
	if !strings.Contains(err.Error(), "no path set") {
		t.Errorf("expected 'no path set' error, got: %v", err)
	}
}

func TestChain_Append_NoPath(t *testing.T) {
	chain := &Chain{ID: "test-chain", Started: time.Now()}
	err := chain.Append(ChainEntry{Step: StepResearch})
	if err == nil {
		t.Fatal("expected error when appending to chain with no path")
	}
	if !strings.Contains(err.Error(), "no path set") {
		t.Errorf("expected 'no path set' error, got: %v", err)
	}
}

func TestLoadJSONLChain_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()
	chainPath := filepath.Join(tmpDir, "chain.jsonl")

	// First line is valid chain metadata, second line exceeds scanner buffer
	meta := `{"id":"test","started":"2026-01-01T00:00:00Z"}`
	hugeLine := make([]byte, 1100*1024) // 1.1MB exceeds default 1MB scanner buffer
	for i := range hugeLine {
		hugeLine[i] = 'x'
	}
	content := meta + "\n" + string(hugeLine) + "\n"
	if err := os.WriteFile(chainPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadJSONLChain(chainPath)
	if err == nil {
		t.Error("expected scanner error for huge line")
	}
	if !strings.Contains(err.Error(), "read chain") {
		t.Errorf("expected 'read chain' error, got: %v", err)
	}
}

func TestLoadLegacyYAMLChain_NonExistentFile(t *testing.T) {
	_, err := loadLegacyYAMLChain("/nonexistent/chain.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadLegacyYAMLChain_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "chain.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml: content"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadLegacyYAMLChain(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestMigrateChain_SaveError(t *testing.T) {
	// Exercise the chain.Save() error path in MigrateChain (line 404-406).
	// Create a valid legacy chain, but make the target directory read-only
	// so Save() fails.
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".agents"), 0700); err != nil {
		t.Fatalf("mkdir .agents: %v", err)
	}

	yaml := `id: migrate-err
started: "2026-01-15T10:00:00Z"
chain:
  - step: research
    timestamp: "2026-01-15T10:01:00Z"
    output: research.md
    locked: true
`
	writeLegacyChain(t, tmp, yaml)

	// Create .agents/ao as a read-only directory so Save() can't create the file
	aoDir := filepath.Join(tmp, ".agents", "ao")
	if err := os.MkdirAll(aoDir, 0500); err != nil {
		t.Fatalf("mkdir ao: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(aoDir, 0700) })

	err := MigrateChain(tmp)
	if err == nil {
		t.Fatal("expected error when Save fails due to read-only directory")
	}
	if !strings.Contains(err.Error(), "save migrated chain") {
		t.Errorf("expected 'save migrated chain' error, got: %v", err)
	}
}

func TestLoadJSONLChain_CloseErrorExposed(t *testing.T) {
	tmpDir := t.TempDir()
	chainPath := filepath.Join(tmpDir, "chain.jsonl")

	// Write valid chain data
	content := `{"id":"test","started":"2026-01-01T00:00:00Z"}` + "\n" +
		`{"step":"research","output":"test.md","timestamp":"2026-01-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(chainPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	// Load should succeed
	chain, err := loadJSONLChain(chainPath)
	if err != nil {
		t.Fatalf("loadJSONLChain: %v", err)
	}
	if chain.ID != "test" {
		t.Errorf("chain ID = %q, want test", chain.ID)
	}
	if len(chain.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(chain.Entries))
	}
}

// --- Benchmarks ---

func benchWriteChainFile(b *testing.B, dir string, numEntries int) {
	b.Helper()
	chainDir := filepath.Join(dir, ".agents", "ao")
	if err := os.MkdirAll(chainDir, 0755); err != nil {
		b.Fatalf("mkdirall: %v", err)
	}

	meta := struct {
		ID      string    `json:"id"`
		Started time.Time `json:"started"`
	}{ID: "bench", Started: time.Now()}
	metaJSON, _ := json.Marshal(meta)
	var lines []string
	lines = append(lines, string(metaJSON))

	for i := 0; i < numEntries; i++ {
		entry := ChainEntry{
			Step:      StepImplement,
			Timestamp: time.Now(),
			Output:    "/some/output/path.md",
			Locked:    true,
		}
		entryJSON, _ := json.Marshal(entry)
		lines = append(lines, string(entryJSON))
	}

	if err := os.WriteFile(filepath.Join(chainDir, ChainFile), []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		b.Fatalf("write chain: %v", err)
	}
}

func BenchmarkLoadChain(b *testing.B) {
	tmpDir := b.TempDir()
	benchWriteChainFile(b, tmpDir, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadChain(tmpDir)
	}
}

func BenchmarkChainAppend(b *testing.B) {
	tmpDir := b.TempDir()
	benchWriteChainFile(b, tmpDir, 5)

	chain, err := LoadChain(tmpDir)
	if err != nil {
		b.Fatalf("load: %v", err)
	}

	entry := ChainEntry{
		Step:      StepImplement,
		Timestamp: time.Now(),
		Output:    "/some/output/path.md",
		Locked:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chain.Append(entry)
	}
}
