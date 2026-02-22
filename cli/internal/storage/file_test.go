package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// jsonlFormatter writes sessions as a single JSONL line (for testing).
type jsonlFormatter struct{}

func (f *jsonlFormatter) Format(w io.Writer, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

func (f *jsonlFormatter) Extension() string { return ".jsonl" }

// errorFormatter always returns an error (for testing error paths).
type errorFormatter struct{}

func (f *errorFormatter) Format(_ io.Writer, _ *Session) error {
	return fmt.Errorf("format error")
}

func (f *errorFormatter) Extension() string { return ".err" }

func TestFileStorage_Init(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))

	if err := fs.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify directories were created
	dirs := []string{
		filepath.Join(baseDir, SessionsDir),
		filepath.Join(baseDir, IndexDir),
		filepath.Join(baseDir, ProvenanceDir),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Init() did not create directory %s", dir)
		}
	}
}

func TestFileStorage_WriteIndex_Dedup(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	entry := &IndexEntry{
		SessionID:   "test-session-123",
		Date:        time.Now(),
		SessionPath: "/path/to/session.md",
		Summary:     "Test session",
	}

	// Write same entry twice
	if err := fs.WriteIndex(entry); err != nil {
		t.Fatalf("WriteIndex() first call error = %v", err)
	}
	if err := fs.WriteIndex(entry); err != nil {
		t.Fatalf("WriteIndex() second call error = %v", err)
	}

	// Verify only one entry exists
	entries, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry after dedup, got %d", len(entries))
	}
}

func TestFileStorage_WriteProvenance(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	record := &ProvenanceRecord{
		ID:           "prov-123",
		ArtifactPath: "/output/session.md",
		ArtifactType: "session",
		SourcePath:   "/input/transcript.jsonl",
		SourceType:   "transcript",
		SessionID:    "session-456",
		CreatedAt:    time.Now(),
	}

	if err := fs.WriteProvenance(record); err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}

	// Query provenance
	records, err := fs.QueryProvenance("/output/session.md")
	if err != nil {
		t.Fatalf("QueryProvenance() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("Expected 1 provenance record, got %d", len(records))
	}
	if records[0].ID != "prov-123" {
		t.Errorf("Expected ID 'prov-123', got %s", records[0].ID)
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "session"},
		{"Hello World", "hello-world"},
		{"Test 123", "test-123"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!@#$%^&*()Characters", "special-characters"},
		{"UPPERCASE", "uppercase"},
		{"A very long slug that exceeds the maximum allowed length for slugs which is fifty characters", "a-very-long-slug-that-exceeds-the-maximum-allowed"},
	}

	for _, tt := range tests {
		got := generateSlug(tt.input)
		if got != tt.want {
			t.Errorf("generateSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFileStorage_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	testPath := filepath.Join(baseDir, "test.txt")
	content := "test content"

	err := fs.atomicWrite(testPath, func(w io.Writer) error {
		_, err := w.Write([]byte(content))
		return err
	})
	if err != nil {
		t.Fatalf("atomicWrite() error = %v", err)
	}

	// Verify content
	data, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected content %q, got %q", content, string(data))
	}

	// Verify no temp files left behind
	files, _ := filepath.Glob(filepath.Join(baseDir, ".tmp-*"))
	if len(files) > 0 {
		t.Errorf("Temp files left behind: %v", files)
	}
}

func TestFileStorage_ListSessions_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// List from empty index
	entries, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected empty/nil entries, got %v", entries)
	}
}

func TestWithFormatters(t *testing.T) {
	f1 := &jsonlFormatter{}
	f2 := &jsonlFormatter{}
	fs := NewFileStorage(WithFormatters(f1, f2))
	if len(fs.Formatters) != 2 {
		t.Errorf("WithFormatters() set %d formatters, want 2", len(fs.Formatters))
	}
}

func TestFileStorage_Close(t *testing.T) {
	fs := NewFileStorage()
	if err := fs.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestFileStorage_GetPaths(t *testing.T) {
	baseDir := "/tmp/test-ao"
	fs := NewFileStorage(WithBaseDir(baseDir))

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"GetBaseDir", fs.GetBaseDir(), baseDir},
		{"GetSessionsDir", fs.GetSessionsDir(), filepath.Join(baseDir, SessionsDir)},
		{"GetIndexPath", fs.GetIndexPath(), filepath.Join(baseDir, IndexDir, IndexFile)},
		{"GetProvenancePath", fs.GetProvenancePath(), filepath.Join(baseDir, ProvenanceDir, ProvenanceFile)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestFileStorage_WriteSession(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		session   *Session
		wantErr   bool
		errSubstr string
	}{
		{
			name: "basic session",
			session: &Session{
				ID:      "abc12345678",
				Date:    now,
				Summary: "Test session write",
			},
		},
		{
			name: "short session ID",
			session: &Session{
				ID:      "short",
				Date:    now,
				Summary: "Short ID session",
			},
		},
		{
			name:      "empty session ID",
			session:   &Session{ID: "", Date: now},
			wantErr:   true,
			errSubstr: "session ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			baseDir := filepath.Join(tmpDir, ".agents/ao")

			fs := NewFileStorage(
				WithBaseDir(baseDir),
				WithFormatters(&jsonlFormatter{}),
			)
			if err := fs.Init(); err != nil {
				t.Fatal(err)
			}

			path, err := fs.WriteSession(tt.session)
			if tt.wantErr {
				if err == nil {
					t.Fatal("WriteSession() expected error, got nil")
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("WriteSession() error = %v, want substr %q", err, tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteSession() error = %v", err)
			}
			if path == "" {
				t.Fatal("WriteSession() returned empty path")
			}

			// Verify file exists and contains valid JSON
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", path, err)
			}
			var parsed Session
			if err := json.Unmarshal(data[:len(data)-1], &parsed); err != nil { // trim trailing newline
				t.Fatalf("Unmarshal session error = %v", err)
			}
			if parsed.ID != tt.session.ID {
				t.Errorf("Written session ID = %q, want %q", parsed.ID, tt.session.ID)
			}
		})
	}
}

func TestFileStorage_WriteSession_FormatterError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(
		WithBaseDir(baseDir),
		WithFormatters(&errorFormatter{}),
	)
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	session := &Session{
		ID:   "test-err",
		Date: time.Now(),
	}

	_, err := fs.WriteSession(session)
	if err == nil {
		t.Fatal("WriteSession() with error formatter expected error, got nil")
	}
}

func TestFileStorage_ReadSession(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(
		WithBaseDir(baseDir),
		WithFormatters(&jsonlFormatter{}),
	)
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	session := &Session{
		ID:        "read-test-session-1",
		Date:      now,
		Summary:   "Readable session",
		Knowledge: []string{"learned something"},
	}

	// Write session and index it
	path, err := fs.WriteSession(session)
	if err != nil {
		t.Fatalf("WriteSession() error = %v", err)
	}

	entry := &IndexEntry{
		SessionID:   session.ID,
		Date:        now,
		SessionPath: path,
		Summary:     session.Summary,
	}
	if err := fs.WriteIndex(entry); err != nil {
		t.Fatalf("WriteIndex() error = %v", err)
	}

	tests := []struct {
		name      string
		sessionID string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "existing session",
			sessionID: "read-test-session-1",
		},
		{
			name:      "nonexistent session",
			sessionID: "does-not-exist",
			wantErr:   true,
			errSubstr: "session not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fs.ReadSession(tt.sessionID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ReadSession() expected error, got nil")
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ReadSession() error = %v, want substr %q", err, tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadSession() error = %v", err)
			}
			if got.ID != session.ID {
				t.Errorf("ReadSession() ID = %q, want %q", got.ID, session.ID)
			}
			if got.Summary != session.Summary {
				t.Errorf("ReadSession() Summary = %q, want %q", got.Summary, session.Summary)
			}
		})
	}
}

func TestFileStorage_ReadSessionFile_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))

	// Create a .md file (unsupported for reading)
	mdPath := filepath.Join(tmpDir, "session.md")
	if err := os.WriteFile(mdPath, []byte("# Session"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := fs.readSessionFile(mdPath)
	if err == nil {
		t.Fatal("readSessionFile(.md) expected error, got nil")
	}
	if !contains(err.Error(), "unsupported format") {
		t.Errorf("readSessionFile(.md) error = %v, want 'unsupported format'", err)
	}
}

func TestFileStorage_ReadSessionFile_EmptyJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))

	// Create an empty .jsonl file
	emptyPath := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(emptyPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := fs.readSessionFile(emptyPath)
	if err == nil {
		t.Fatal("readSessionFile(empty) expected error, got nil")
	}
	if !errors.Is(err, ErrEmptySessionFile) {
		t.Errorf("readSessionFile(empty) error = %v, want ErrEmptySessionFile", err)
	}
}

func TestWriteSession_SentinelErrors(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(
		WithBaseDir(baseDir),
		WithFormatters(&jsonlFormatter{}),
	)
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	_, err := fs.WriteSession(&Session{ID: ""})
	if err == nil {
		t.Fatal("expected error for empty session ID")
	}
	if !errors.Is(err, ErrSessionIDRequired) {
		t.Errorf("expected ErrSessionIDRequired, got %v", err)
	}
}

func TestFileStorage_QueryProvenance_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Write a record
	record := &ProvenanceRecord{
		ID:           "prov-abc",
		ArtifactPath: "/output/a.md",
		ArtifactType: "session",
		SourcePath:   "/input/t.jsonl",
		SourceType:   "transcript",
		CreatedAt:    time.Now(),
	}
	if err := fs.WriteProvenance(record); err != nil {
		t.Fatal(err)
	}

	// Query for a different artifact
	records, err := fs.QueryProvenance("/output/nonexistent.md")
	if err != nil {
		t.Fatalf("QueryProvenance() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Expected 0 records for non-matching path, got %d", len(records))
	}
}

func TestFileStorage_WriteSession_MultipleFormatters(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	f1 := &jsonlFormatter{}
	f2 := &jsonlFormatter{} // second formatter with same ext (won't conflict, diff content irrelevant)
	fs := NewFileStorage(
		WithBaseDir(baseDir),
		WithFormatters(f1, f2),
	)
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	session := &Session{
		ID:      "multi-fmt-test",
		Date:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Summary: "Multi formatter",
	}

	path, err := fs.WriteSession(session)
	if err != nil {
		t.Fatalf("WriteSession() error = %v", err)
	}

	// Primary path should be from the first formatter
	if !contains(path, ".jsonl") {
		t.Errorf("Primary path %q should have .jsonl extension", path)
	}
}

func TestFileStorage_ListSessions_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Write a valid entry then a malformed line
	entry := &IndexEntry{
		SessionID:   "valid-session",
		Date:        time.Now(),
		SessionPath: "/path/to/session.jsonl",
		Summary:     "Valid",
	}
	if err := fs.WriteIndex(entry); err != nil {
		t.Fatal(err)
	}

	// Append malformed line directly
	indexPath := fs.GetIndexPath()
	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("not-valid-json\n"))
	f.Close()

	// ListSessions should skip malformed lines
	entries, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (skipping malformed), got %d", len(entries))
	}
}

func TestFileStorage_QueryProvenance_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Query provenance when file does not exist (IsNotExist path)
	records, err := fs.QueryProvenance("/nonexistent")
	if err != nil {
		t.Fatalf("QueryProvenance() on missing file should return nil, got error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestFileStorage_QueryProvenance_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Write a valid provenance record followed by a malformed line
	record := &ProvenanceRecord{
		ID:           "prov-ok",
		ArtifactPath: "/output/ok.md",
		ArtifactType: "session",
	}
	if err := fs.WriteProvenance(record); err != nil {
		t.Fatal(err)
	}

	// Append malformed line
	provPath := fs.GetProvenancePath()
	f, err := os.OpenFile(provPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("not valid json\n"))
	f.Close()

	// Query should skip malformed and return the valid one
	records, err := fs.QueryProvenance("/output/ok.md")
	if err != nil {
		t.Fatalf("QueryProvenance() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record (skipping malformed), got %d", len(records))
	}
}

func TestFileStorage_ReadSession_ListError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Make index directory unreadable to force ListSessions error
	indexDir := filepath.Join(baseDir, IndexDir)

	// Write a valid index file first
	entry := &IndexEntry{
		SessionID:   "test-session",
		SessionPath: "/path/to/session.jsonl",
		Summary:     "Test",
	}
	if err := fs.WriteIndex(entry); err != nil {
		t.Fatal(err)
	}

	// Make the file unreadable
	indexPath := filepath.Join(indexDir, IndexFile)
	if err := os.Chmod(indexPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(indexPath, 0644)
	})

	_, err := fs.ReadSession("test-session")
	if err == nil {
		t.Error("expected error when index file is unreadable")
	}
}

func TestFileStorage_AtomicWrite_WriteFuncError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	testPath := filepath.Join(baseDir, "error-test.txt")

	err := fs.atomicWrite(testPath, func(w io.Writer) error {
		return fmt.Errorf("deliberate write error")
	})
	if err == nil {
		t.Fatal("expected error from writeFunc")
	}
	if !contains(err.Error(), "write content") {
		t.Errorf("expected 'write content' in error, got %v", err)
	}

	// Final file should not exist
	if _, statErr := os.Stat(testPath); !os.IsNotExist(statErr) {
		t.Error("expected no final file after writeFunc error")
	}

	// No temp files should be left behind
	files, _ := filepath.Glob(filepath.Join(baseDir, ".tmp-*"))
	if len(files) > 0 {
		t.Errorf("temp files left behind after error: %v", files)
	}
}

func TestFileStorage_ReadSessionFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))

	// Create a .jsonl file with invalid JSON
	badPath := filepath.Join(tmpDir, "bad.jsonl")
	if err := os.WriteFile(badPath, []byte("not valid json\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := fs.readSessionFile(badPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON in session file")
	}
}

func TestFileStorage_ListSessions_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	// Don't init - so index dir doesn't have sessions.jsonl
	fs := NewFileStorage(WithBaseDir(baseDir))
	// Manually create the dirs but not the index file
	if err := os.MkdirAll(filepath.Join(baseDir, IndexDir), 0700); err != nil {
		t.Fatal(err)
	}

	// Should return nil, nil when file doesn't exist
	entries, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v, want nil", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}
}

func TestFileStorage_HasIndexEntry_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// hasIndexEntry on non-existent file should return false
	got := fs.hasIndexEntry("/nonexistent/index.jsonl", "any-id")
	if got {
		t.Error("expected false for non-existent index file")
	}
}

func TestFileStorage_WriteSession_NoFormatters(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")

	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	session := &Session{
		ID:      "no-formatters",
		Summary: "No formatters set",
	}

	// With no formatters, should return empty path and no error
	path, err := fs.WriteSession(session)
	if err != nil {
		t.Fatalf("WriteSession() error = %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path with no formatters, got %s", path)
	}
}

func TestGenerateSlug_AllSpecialChars(t *testing.T) {
	// Input that results in empty slug after filtering
	got := generateSlug("!@#$%^&*()")
	if got != "session" {
		t.Errorf("generateSlug of all special chars = %q, want 'session'", got)
	}
}

func TestFileStorage_Init_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	readOnly := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnly, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0700) })

	baseDir := filepath.Join(readOnly, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	err := fs.Init()
	if err == nil {
		t.Error("expected error when Init in read-only directory")
	}
}

func TestFileStorage_AppendJSONL_OpenError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Create a directory where the file should be -- os.OpenFile will fail
	provPath := filepath.Join(baseDir, ProvenanceDir, ProvenanceFile)
	_ = os.Remove(provPath) // remove if exists
	if err := os.MkdirAll(provPath, 0700); err != nil {
		t.Fatal(err)
	}

	record := &ProvenanceRecord{
		ID:           "error-test",
		ArtifactPath: "/output/error.md",
	}
	err := fs.WriteProvenance(record)
	if err == nil {
		t.Error("expected error when provenance file path is a directory")
	}
}

func TestFileStorage_AppendJSONL_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Make provenance dir read-only
	provDir := filepath.Join(baseDir, ProvenanceDir)
	if err := os.Chmod(provDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(provDir, 0700) })

	record := &ProvenanceRecord{
		ID:           "readonly-test",
		ArtifactPath: "/output/readonly.md",
	}
	err := fs.WriteProvenance(record)
	if err == nil {
		t.Error("expected error when provenance dir is read-only")
	}
}

func TestFileStorage_AtomicWrite_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Make the sessions dir read-only so CreateTemp fails
	sessDir := filepath.Join(baseDir, SessionsDir)
	if err := os.Chmod(sessDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sessDir, 0700) })

	testPath := filepath.Join(sessDir, "test.txt")
	err := fs.atomicWrite(testPath, func(w io.Writer) error {
		_, err := w.Write([]byte("data"))
		return err
	})
	if err == nil {
		t.Error("expected error from atomicWrite in read-only dir")
	}
	if !contains(err.Error(), "create temp file") {
		t.Errorf("expected 'create temp file' error, got: %v", err)
	}
}

func TestFileStorage_HasIndexEntry_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Write a valid entry then malformed line to index
	entry := &IndexEntry{
		SessionID:   "find-me",
		SessionPath: "/path/to/session.jsonl",
		Summary:     "Find this one",
	}
	if err := fs.WriteIndex(entry); err != nil {
		t.Fatal(err)
	}

	// Append malformed line
	indexPath := fs.GetIndexPath()
	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("{bad json\n"))
	f.Close()

	// hasIndexEntry should still find the valid entry even with malformed lines
	if !fs.hasIndexEntry(indexPath, "find-me") {
		t.Error("expected hasIndexEntry to find valid entry despite malformed lines")
	}

	// Should not find a nonexistent entry
	if fs.hasIndexEntry(indexPath, "not-here") {
		t.Error("expected hasIndexEntry to return false for nonexistent entry")
	}
}

func TestFileStorage_ListSessions_PermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Write a valid entry first
	entry := &IndexEntry{
		SessionID:   "perm-test",
		SessionPath: "/path/session.jsonl",
	}
	if err := fs.WriteIndex(entry); err != nil {
		t.Fatal(err)
	}

	// Make index file unreadable (not-IsNotExist error path)
	indexPath := fs.GetIndexPath()
	if err := os.Chmod(indexPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(indexPath, 0644) })

	_, err := fs.ListSessions()
	if err == nil {
		t.Error("expected error when index file is unreadable")
	}
}

func TestFileStorage_QueryProvenance_PermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Write a valid record
	record := &ProvenanceRecord{
		ID:           "perm-test",
		ArtifactPath: "/output/perm.md",
	}
	if err := fs.WriteProvenance(record); err != nil {
		t.Fatal(err)
	}

	// Make provenance file unreadable
	provPath := fs.GetProvenancePath()
	if err := os.Chmod(provPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(provPath, 0644) })

	_, err := fs.QueryProvenance("/output/perm.md")
	if err == nil {
		t.Error("expected error when provenance file is unreadable")
	}
}

func TestFileStorage_WriteIndex_AppendError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))
	if err := fs.Init(); err != nil {
		t.Fatal(err)
	}

	// Make index dir read-only so appendJSONL cannot create/write the file
	indexDir := filepath.Join(baseDir, IndexDir)
	if err := os.Chmod(indexDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(indexDir, 0700) })

	entry := &IndexEntry{
		SessionID:   "write-fail",
		SessionPath: "/path/to/session.jsonl",
	}
	err := fs.WriteIndex(entry)
	if err == nil {
		t.Error("expected error when index dir is read-only")
	}
}

func TestFileStorage_ReadSessionFile_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	fs := NewFileStorage(WithBaseDir(baseDir))

	_, err := fs.readSessionFile(filepath.Join(tmpDir, "nonexistent.jsonl"))
	if err == nil {
		t.Error("expected error for nonexistent session file")
	}
}

func TestAtomicWrite_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0700) })

	fs := NewFileStorage(WithBaseDir(tmpDir))
	err := fs.atomicWrite(filepath.Join(readOnlyDir, "sub", "file.json"), func(w io.Writer) error {
		_, e := w.Write([]byte("test"))
		return e
	})
	if err == nil {
		t.Error("expected error when directory is read-only")
	}
}

func TestAtomicWrite_WriteFuncError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	expectedErr := fmt.Errorf("write func error")
	err := fs.atomicWrite(filepath.Join(tmpDir, "test.json"), func(w io.Writer) error {
		return expectedErr
	})
	if err == nil {
		t.Error("expected error from writeFunc")
	}
	if !strings.Contains(err.Error(), "write content") {
		t.Errorf("expected 'write content' error, got: %v", err)
	}
}

func TestAppendJSONL_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0700) })

	fs := NewFileStorage(WithBaseDir(tmpDir))
	err := fs.appendJSONL(filepath.Join(readOnlyDir, "sub", "file.jsonl"), map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error when directory is read-only for appendJSONL")
	}
}

func TestListSessions_PermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	indexDir := filepath.Join(baseDir, IndexDir)
	if err := os.MkdirAll(indexDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create index file and make it unreadable
	indexPath := filepath.Join(indexDir, IndexFile)
	if err := os.WriteFile(indexPath, []byte(`{"session_id":"test"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(indexPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(indexPath, 0600) })

	fs := NewFileStorage(WithBaseDir(baseDir))
	_, err := fs.ListSessions()
	if err == nil {
		t.Error("expected error when index file is unreadable")
	}
}

func TestQueryProvenance_PermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".agents/ao")
	provDir := filepath.Join(baseDir, ProvenanceDir)
	if err := os.MkdirAll(provDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create provenance file and make it unreadable
	provPath := filepath.Join(provDir, ProvenanceFile)
	if err := os.WriteFile(provPath, []byte(`{"artifact_path":"test"}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(provPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(provPath, 0600) })

	fs := NewFileStorage(WithBaseDir(baseDir))
	_, err := fs.QueryProvenance("test")
	if err == nil {
		t.Error("expected error when provenance file is unreadable")
	}
}

func TestAppendJSONL_OpenFileError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	// Create the directory but make it readable (so MkdirAll succeeds) but not writable
	targetDir := filepath.Join(tmpDir, "append-target")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(targetDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(targetDir, 0700) })

	err := fs.appendJSONL(filepath.Join(targetDir, "test.jsonl"), map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error when directory is not writable for appendJSONL")
	}
	if !strings.Contains(err.Error(), "open file") {
		t.Errorf("expected 'open file' error, got: %v", err)
	}
}

func TestAppendJSONL_MarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	// Channels cannot be marshaled to JSON
	unmarshalable := make(chan int)
	err := fs.appendJSONL(filepath.Join(tmpDir, "test.jsonl"), unmarshalable)
	if err == nil {
		t.Error("expected error when marshaling unmarshalable value")
	}
	if !strings.Contains(err.Error(), "marshal json") {
		t.Errorf("expected 'marshal json' error, got: %v", err)
	}
}

func TestAppendJSONL_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	// Replace the target file path with a directory so OpenFile fails with EISDIR
	targetPath := filepath.Join(tmpDir, "blocked.jsonl")
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		t.Fatal(err)
	}

	err := fs.appendJSONL(targetPath, map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error when target path is a directory")
	}
}

func TestAtomicWrite_WriteFuncErrorCleansTempFile(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	targetPath := filepath.Join(tmpDir, "atomic-test.json")

	// writeFunc that returns an error -- verify temp file cleanup
	writeErr := fmt.Errorf("simulated write failure")
	err := fs.atomicWrite(targetPath, func(w io.Writer) error {
		return writeErr
	})
	if err == nil {
		t.Fatal("expected error from failing writeFunc")
	}

	// Temp file should have been cleaned up
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("temp file %q should have been cleaned up", e.Name())
		}
	}
}

func TestAtomicWrite_ReadOnlyParentDir(t *testing.T) {
	// Exercise the MkdirAll error path: make the parent dir read-only
	// so that the subdirectory can't be created.
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	roDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(roDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0755) })

	// Path whose parent requires creating a subdirectory inside roDir
	targetPath := filepath.Join(roDir, "subdir", "file.json")
	err := fs.atomicWrite(targetPath, func(w io.Writer) error {
		_, e := w.Write([]byte("data"))
		return e
	})
	if err == nil {
		t.Error("expected error when parent directory is read-only")
	}
}

func TestListSessions_OpenError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	// Create the index directory with the index file as a directory
	// to force an open error (not "not exists", but actual open error)
	indexDir := filepath.Join(tmpDir, IndexDir)
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a directory at the index file path
	indexFilePath := filepath.Join(indexDir, IndexFile)
	if err := os.MkdirAll(indexFilePath, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := fs.ListSessions()
	if err == nil {
		t.Error("expected error when index file path is a directory")
	}
}

func TestQueryProvenance_OpenError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir))

	// Create provenance directory with provenance file as a directory
	provDir := filepath.Join(tmpDir, ProvenanceDir)
	if err := os.MkdirAll(provDir, 0755); err != nil {
		t.Fatal(err)
	}
	provFilePath := filepath.Join(provDir, ProvenanceFile)
	if err := os.MkdirAll(provFilePath, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := fs.QueryProvenance("some/artifact")
	if err == nil {
		t.Error("expected error when provenance file path is a directory")
	}
}

// contains is a test helper for substring matching.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Benchmarks ---

func benchSession(id string) *Session {
	return &Session{
		ID:      id,
		Date:    time.Now(),
		Summary: "Benchmark session for performance testing",
		Decisions: []string{
			"Decided to use context.WithCancel for graceful shutdown",
			"Chose JSONL over SQLite for simplicity",
		},
		Knowledge: []string{
			"Go maps are not ordered, so iteration order is non-deterministic",
			"bufio.Scanner has a default 64KB buffer limit",
		},
		FilesChanged: []string{
			"internal/pool/pool.go",
			"internal/storage/file.go",
		},
	}
}

func BenchmarkWriteSession(b *testing.B) {
	tmpDir := b.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir), WithFormatters(&jsonlFormatter{}))
	if err := fs.Init(); err != nil {
		b.Fatalf("Init: %v", err)
	}

	b.ResetTimer()
	for i := range b.N {
		s := benchSession(fmt.Sprintf("bench-%d", i))
		_, _ = fs.WriteSession(s)
	}
}

func BenchmarkListSessions(b *testing.B) {
	tmpDir := b.TempDir()
	fs := NewFileStorage(WithBaseDir(tmpDir), WithFormatters(&jsonlFormatter{}))
	if err := fs.Init(); err != nil {
		b.Fatalf("Init: %v", err)
	}

	// Seed 20 sessions
	for i := range 20 {
		s := benchSession(fmt.Sprintf("bench-list-%d", i))
		if _, err := fs.WriteSession(s); err != nil {
			b.Fatalf("setup WriteSession: %v", err)
		}
	}

	b.ResetTimer()
	for range b.N {
		_, _ = fs.ListSessions()
	}
}

func BenchmarkGenerateSlug(b *testing.B) {
	text := "This is a long summary about implementing the knowledge flywheel correctly"
	b.ResetTimer()
	for range b.N {
		generateSlug(text)
	}
}
