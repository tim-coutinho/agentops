package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeForDedup(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		wantSame bool
	}{
		{
			name:     "exact duplicates",
			a:        "Lead-only commit eliminates merge conflicts",
			b:        "Lead-only commit eliminates merge conflicts",
			wantSame: true,
		},
		{
			name:     "case difference",
			a:        "Lead-Only Commit Pattern",
			b:        "lead-only commit pattern",
			wantSame: true,
		},
		{
			name:     "whitespace difference",
			a:        "  Lead-only  commit   pattern  ",
			b:        "Lead-only commit pattern",
			wantSame: true,
		},
		{
			name:     "trailing ellipsis stripped",
			a:        "Workers write files but never commit...",
			b:        "Workers write files but never commit",
			wantSame: true,
		},
		{
			name:     "distinct content with same 80-char prefix",
			a:        "Topological wave decomposition extracts parallelism from dependency graphs by grouping leaves — this is about WAVES",
			b:        "Topological wave decomposition extracts parallelism from dependency graphs by grouping leaves — this is about SORTING",
			wantSame: false,
		},
		{
			name:     "short similar strings are distinct",
			a:        "Use content hashing for dedup",
			b:        "Use content hashing for dedup detection",
			wantSame: false,
		},
		{
			name:     "completely different",
			a:        "Workers should never commit",
			b:        "Wave sizing follows dependency graph",
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyA := normalizeForDedup(tt.a)
			keyB := normalizeForDedup(tt.b)
			gotSame := keyA == keyB
			if gotSame != tt.wantSame {
				t.Errorf("normalizeForDedup(%q) == normalizeForDedup(%q): got %v, want %v\n  keyA=%s\n  keyB=%s",
					tt.a, tt.b, gotSame, tt.wantSame, keyA, keyB)
			}
		})
	}
}

func TestDedupSimilar(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		wantCount int
	}{
		{
			name:      "nil input",
			input:     nil,
			wantCount: 0,
		},
		{
			name:      "empty input",
			input:     []string{},
			wantCount: 0,
		},
		{
			name:      "no duplicates",
			input:     []string{"alpha", "beta", "gamma"},
			wantCount: 3,
		},
		{
			name:      "exact duplicates removed",
			input:     []string{"alpha", "beta", "alpha", "gamma", "beta"},
			wantCount: 3,
		},
		{
			name: "case-insensitive dedup",
			input: []string{
				"Lead-only commit pattern",
				"lead-only commit pattern",
				"LEAD-ONLY COMMIT PATTERN",
			},
			wantCount: 1,
		},
		{
			name: "preserves distinct long strings",
			input: []string{
				"Topological wave decomposition extracts parallelism from dependency graphs by grouping leaves — approach A",
				"Topological wave decomposition extracts parallelism from dependency graphs by grouping leaves — approach B",
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedupSimilar(tt.input)
			if len(result) != tt.wantCount {
				t.Errorf("dedupSimilar() returned %d items, want %d. Items: %v", len(result), tt.wantCount, result)
			}
		})
	}
}

func TestFindPendingTranscripts(t *testing.T) {
	// Create temp directory with test transcript files
	tmpDir, err := os.MkdirTemp("", "batch_forge_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test JSONL files (must be > 100 bytes to pass the size filter)
	for _, name := range []string{"session1.jsonl", "session2.jsonl"} {
		content := []byte(`{"role":"user","content":"hello world, this is a test message with enough content"}` + "\n" +
			`{"role":"assistant","content":"hi there, this is a sufficiently long response to exceed the 100 byte minimum"}` + "\n")
		if err := os.WriteFile(filepath.Join(tmpDir, name), content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a file too small to be a transcript
	if err := os.WriteFile(filepath.Join(tmpDir, "tiny.jsonl"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a non-JSONL file
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a subagents directory that should be skipped
	subDir := filepath.Join(tmpDir, "subagents")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "sub.jsonl"), []byte(`{"role":"user","content":"skip me please"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	candidates, err := findPendingTranscripts(tmpDir)
	if err != nil {
		t.Fatalf("findPendingTranscripts: %v", err)
	}

	if len(candidates) != 2 {
		t.Errorf("got %d candidates, want 2", len(candidates))
		for _, c := range candidates {
			t.Logf("  candidate: %s (size=%d)", c.path, c.size)
		}
	}

	// Verify sorted by mod time (oldest first)
	if len(candidates) >= 2 {
		if candidates[0].modTime.After(candidates[1].modTime) {
			t.Error("candidates not sorted by modification time (oldest first)")
		}
	}
}

func TestFindPendingTranscriptsEmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "batch_forge_empty")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	candidates, err := findPendingTranscripts(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("got %d candidates, want 0", len(candidates))
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tt := range tests {
		got := humanSize(tt.bytes)
		if got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestTranscriptCandidateFields(t *testing.T) {
	now := time.Now()
	c := transcriptCandidate{
		path:    "/tmp/test.jsonl",
		modTime: now,
		size:    1234,
	}

	if c.path != "/tmp/test.jsonl" {
		t.Error("path mismatch")
	}
	if c.size != 1234 {
		t.Error("size mismatch")
	}
	if !c.modTime.Equal(now) {
		t.Error("modTime mismatch")
	}
}

func TestLoadForgedIndex(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "forged.jsonl")

	// Test empty index (file doesn't exist)
	forgedSet, err := loadForgedIndex(indexPath)
	if err != nil {
		t.Fatalf("loadForgedIndex failed: %v", err)
	}
	if len(forgedSet) != 0 {
		t.Errorf("expected empty set, got %d entries", len(forgedSet))
	}

	// Write some test records
	records := []ForgedRecord{
		{Path: "/path/to/session1.jsonl", ForgedAt: time.Now(), Session: "session-1"},
		{Path: "/path/to/session2.jsonl", ForgedAt: time.Now(), Session: "session-2"},
		{Path: "/path/to/session3.jsonl", ForgedAt: time.Now(), Session: "session-3"},
	}

	f, err := os.Create(indexPath)
	if err != nil {
		t.Fatalf("create index file: %v", err)
	}
	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal record: %v", err)
		}
		_, _ = f.Write(append(data, '\n'))
	}
	f.Close()

	// Load index
	forgedSet, err = loadForgedIndex(indexPath)
	if err != nil {
		t.Fatalf("loadForgedIndex failed: %v", err)
	}

	// Verify all paths are in set
	if len(forgedSet) != 3 {
		t.Errorf("expected 3 entries, got %d", len(forgedSet))
	}
	for _, record := range records {
		if !forgedSet[record.Path] {
			t.Errorf("expected path %s to be in set", record.Path)
		}
	}
}

func TestAppendForgedRecord(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "forged.jsonl")

	record1 := ForgedRecord{
		Path:     "/path/to/session1.jsonl",
		ForgedAt: time.Now(),
		Session:  "session-1",
	}

	// Append first record
	if err := appendForgedRecord(indexPath, record1); err != nil {
		t.Fatalf("appendForgedRecord failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("expected index file to exist")
	}

	// Append second record
	record2 := ForgedRecord{
		Path:     "/path/to/session2.jsonl",
		ForgedAt: time.Now(),
		Session:  "session-2",
	}
	if err := appendForgedRecord(indexPath, record2); err != nil {
		t.Fatalf("appendForgedRecord failed on second write: %v", err)
	}

	// Load and verify
	forgedSet, err := loadForgedIndex(indexPath)
	if err != nil {
		t.Fatalf("loadForgedIndex failed: %v", err)
	}

	if len(forgedSet) != 2 {
		t.Errorf("expected 2 entries, got %d", len(forgedSet))
	}
	if !forgedSet[record1.Path] {
		t.Errorf("expected path %s to be in set", record1.Path)
	}
	if !forgedSet[record2.Path] {
		t.Errorf("expected path %s to be in set", record2.Path)
	}
}

func TestBatchForgeSkipsAlreadyForged(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "forged.jsonl")

	// Create forged index with one entry
	record := ForgedRecord{
		Path:     "/already/forged.jsonl",
		ForgedAt: time.Now(),
		Session:  "session-old",
	}
	if err := appendForgedRecord(indexPath, record); err != nil {
		t.Fatalf("appendForgedRecord failed: %v", err)
	}

	// Load index
	forgedSet, err := loadForgedIndex(indexPath)
	if err != nil {
		t.Fatalf("loadForgedIndex failed: %v", err)
	}

	// Simulate filtering transcripts
	candidates := []transcriptCandidate{
		{path: "/already/forged.jsonl", modTime: time.Now(), size: 1000},
		{path: "/new/transcript.jsonl", modTime: time.Now(), size: 2000},
	}

	var unforged []transcriptCandidate
	for _, c := range candidates {
		if !forgedSet[c.path] {
			unforged = append(unforged, c)
		}
	}

	// Verify only new transcript remains
	if len(unforged) != 1 {
		t.Errorf("expected 1 unforged transcript, got %d", len(unforged))
	}
	if unforged[0].path != "/new/transcript.jsonl" {
		t.Errorf("expected /new/transcript.jsonl, got %s", unforged[0].path)
	}
}

func TestBatchForgeMaxFlag(t *testing.T) {
	// Simulate --max flag limiting transcripts
	candidates := []transcriptCandidate{
		{path: "/transcript1.jsonl", modTime: time.Now(), size: 1000},
		{path: "/transcript2.jsonl", modTime: time.Now(), size: 1000},
		{path: "/transcript3.jsonl", modTime: time.Now(), size: 1000},
		{path: "/transcript4.jsonl", modTime: time.Now(), size: 1000},
		{path: "/transcript5.jsonl", modTime: time.Now(), size: 1000},
	}

	maxLimit := 3
	var limited []transcriptCandidate
	if maxLimit > 0 && len(candidates) > maxLimit {
		limited = candidates[:maxLimit]
	} else {
		limited = candidates
	}

	if len(limited) != maxLimit {
		t.Errorf("expected %d transcripts after limit, got %d", maxLimit, len(limited))
	}

	// Verify we got the first 3
	for i := range maxLimit {
		if limited[i].path != candidates[i].path {
			t.Errorf("expected %s at position %d, got %s", candidates[i].path, i, limited[i].path)
		}
	}
}

func TestBatchForgeResult(t *testing.T) {
	// Test JSON marshaling of BatchForgeResult
	result := BatchForgeResult{
		Forged:    10,
		Skipped:   3,
		Failed:    1,
		Extracted: 8,
		Paths:     []string{"/path1.jsonl", "/path2.jsonl"},
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Unmarshal and verify
	var decoded BatchForgeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Forged != result.Forged {
		t.Errorf("expected Forged=%d, got %d", result.Forged, decoded.Forged)
	}
	if decoded.Skipped != result.Skipped {
		t.Errorf("expected Skipped=%d, got %d", result.Skipped, decoded.Skipped)
	}
	if decoded.Failed != result.Failed {
		t.Errorf("expected Failed=%d, got %d", result.Failed, decoded.Failed)
	}
	if decoded.Extracted != result.Extracted {
		t.Errorf("expected Extracted=%d, got %d", result.Extracted, decoded.Extracted)
	}
	if len(decoded.Paths) != len(result.Paths) {
		t.Errorf("expected %d paths, got %d", len(result.Paths), len(decoded.Paths))
	}
}
