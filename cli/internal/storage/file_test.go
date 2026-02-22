package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		ID:       "read-test-session-1",
		Date:     now,
		Summary:  "Readable session",
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
	if !contains(err.Error(), "empty session file") {
		t.Errorf("readSessionFile(empty) error = %v, want 'empty session file'", err)
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

// contains is a test helper for substring matching.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
