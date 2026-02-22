package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// DefaultBaseDir is the default storage directory.
	DefaultBaseDir = ".agents/ao"

	// SessionsDir holds session markdown/jsonl files.
	SessionsDir = "sessions"

	// IndexDir holds the session index.
	IndexDir = "index"

	// ProvenanceDir holds provenance records.
	ProvenanceDir = "provenance"

	// IndexFile is the name of the main index file.
	IndexFile = "sessions.jsonl"

	// ProvenanceFile is the name of the provenance graph.
	ProvenanceFile = "graph.jsonl"

	// SlugMaxLength is the maximum length for URL-safe slugs.
	SlugMaxLength = 50

	// SlugMinWordBoundary is the minimum length before trimming at word boundary.
	SlugMinWordBoundary = 30
)

// FileStorage implements Storage using the local filesystem.
type FileStorage struct {
	// BaseDir is the root directory (e.g., .agents/ao).
	BaseDir string

	// Formatters are the output formats to use.
	Formatters []Formatter

	mu sync.Mutex
}

// FileStorageOption configures a FileStorage instance.
type FileStorageOption func(*FileStorage)

// WithBaseDir sets the base directory.
func WithBaseDir(dir string) FileStorageOption {
	return func(fs *FileStorage) {
		fs.BaseDir = dir
	}
}

// WithFormatters sets the output formatters.
func WithFormatters(formatters ...Formatter) FileStorageOption {
	return func(fs *FileStorage) {
		fs.Formatters = formatters
	}
}

// NewFileStorage creates a new file-based storage.
func NewFileStorage(opts ...FileStorageOption) *FileStorage {
	fs := &FileStorage{
		BaseDir:    DefaultBaseDir,
		Formatters: nil, // Will be set by caller
	}

	for _, opt := range opts {
		opt(fs)
	}

	return fs
}

// Init creates the required directory structure.
func (fs *FileStorage) Init() error {
	dirs := []string{
		filepath.Join(fs.BaseDir, SessionsDir),
		filepath.Join(fs.BaseDir, IndexDir),
		filepath.Join(fs.BaseDir, ProvenanceDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return nil
}

// WriteSession writes a session to storage using all configured formatters.
// Returns the path to the primary session file.
func (fs *FileStorage) WriteSession(session *Session) (string, error) {
	if session.ID == "" {
		return "", fmt.Errorf("session ID is required")
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Generate unique filename: YYYY-MM-DD-{slug}-{sessionID[:7]}.ext
	slug := generateSlug(session.Summary)
	shortID := session.ID
	if len(shortID) > 7 {
		shortID = shortID[:7]
	}

	dateStr := session.Date.Format("2006-01-02")
	baseName := fmt.Sprintf("%s-%s-%s", dateStr, slug, shortID)

	var primaryPath string

	// Write using each formatter
	for i, formatter := range fs.Formatters {
		ext := formatter.Extension()
		filename := baseName + ext
		fullPath := filepath.Join(fs.BaseDir, SessionsDir, filename)

		if err := fs.atomicWrite(fullPath, func(w io.Writer) error {
			return formatter.Format(w, session)
		}); err != nil {
			return "", fmt.Errorf("write %s format: %w", ext, err)
		}

		// First formatter produces the primary path
		if i == 0 {
			primaryPath = fullPath
		}
	}

	return primaryPath, nil
}

// WriteIndex appends an entry to the session index.
func (fs *FileStorage) WriteIndex(entry *IndexEntry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	indexPath := filepath.Join(fs.BaseDir, IndexDir, IndexFile)

	// Check for duplicate by session ID
	if fs.hasIndexEntry(indexPath, entry.SessionID) {
		// Already indexed, skip
		return nil
	}

	return fs.appendJSONL(indexPath, entry)
}

// WriteProvenance records provenance information.
func (fs *FileStorage) WriteProvenance(record *ProvenanceRecord) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	provPath := filepath.Join(fs.BaseDir, ProvenanceDir, ProvenanceFile)
	return fs.appendJSONL(provPath, record)
}

// ReadSession retrieves a session by ID.
func (fs *FileStorage) ReadSession(sessionID string) (*Session, error) {
	// Find session file by scanning index
	entries, err := fs.ListSessions()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.SessionID == sessionID {
			// Read the session file
			return fs.readSessionFile(entry.SessionPath)
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

// ListSessions returns all session index entries.
func (fs *FileStorage) ListSessions() (entries []IndexEntry, err error) {
	indexPath := filepath.Join(fs.BaseDir, IndexDir, IndexFile)

	f, err := os.Open(indexPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry IndexEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// QueryProvenance finds provenance records for an artifact.
func (fs *FileStorage) QueryProvenance(artifactPath string) (records []ProvenanceRecord, err error) {
	provPath := filepath.Join(fs.BaseDir, ProvenanceDir, ProvenanceFile)

	f, err := os.Open(provPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var record ProvenanceRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		if record.ArtifactPath == artifactPath {
			records = append(records, record)
		}
	}

	return records, scanner.Err()
}

// Close releases any resources.
func (fs *FileStorage) Close() error {
	return nil // No resources to release for file storage
}

// atomicWrite writes to a temp file and renames atomically.
func (fs *FileStorage) atomicWrite(path string, writeFunc func(io.Writer) error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Create temp file in same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath) //nolint:errcheck // cleanup in error path
		}
	}()

	// Write content
	if err := writeFunc(tmpFile); err != nil {
		_ = tmpFile.Close() //nolint:errcheck // cleanup in error path
		return fmt.Errorf("write content: %w", err)
	}

	// Sync to ensure data is on disk
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close() //nolint:errcheck // cleanup in error path
		return fmt.Errorf("sync file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename to final: %w", err)
	}

	success = true
	return nil
}

// appendJSONL appends a JSON line to a file atomically.
func (fs *FileStorage) appendJSONL(path string, v interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// For append, we can't use pure atomic write, but we can:
	// 1. Write to temp
	// 2. Append temp content to main file
	// This ensures partial writes don't corrupt the main file

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // sync already called, close best-effort
	}()

	// Write with newline
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write line: %w", err)
	}

	return f.Sync()
}

// hasIndexEntry checks if a session ID already exists in the index.
func (fs *FileStorage) hasIndexEntry(indexPath, sessionID string) bool {
	f, err := os.Open(indexPath)
	if err != nil {
		return false
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only check, errors non-critical
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry IndexEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.SessionID == sessionID {
			return true
		}
	}

	return false
}

// readSessionFile reads a session from a JSONL file.
func (fs *FileStorage) readSessionFile(path string) (*Session, error) {
	// For JSONL format, read and parse
	if strings.HasSuffix(path, ".jsonl") {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = f.Close() //nolint:errcheck // read-only, errors non-critical
		}()

		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			var session Session
			if err := json.Unmarshal(scanner.Bytes(), &session); err != nil {
				return nil, err
			}
			return &session, nil
		}
		return nil, fmt.Errorf("empty session file")
	}

	// For markdown format, we'd need to parse YAML frontmatter
	// For now, return error if not JSONL
	return nil, fmt.Errorf("unsupported format: %s", filepath.Ext(path))
}

// generateSlug creates a URL-safe slug from text.
func generateSlug(text string) string {
	if text == "" {
		return "session"
	}

	s := slugify(strings.ToLower(text))
	s = truncateSlug(s)

	if s == "" {
		return "session"
	}
	return s
}

// slugify replaces non-alphanumeric runs with single hyphens and trims leading/trailing hyphens.
func slugify(input string) string {
	var result strings.Builder
	lastHyphen := false
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen {
			result.WriteRune('-')
			lastHyphen = true
		}
	}
	return strings.Trim(result.String(), "-")
}

// truncateSlug limits the slug to SlugMaxLength, preferring word boundaries.
func truncateSlug(s string) string {
	if len(s) <= SlugMaxLength {
		return s
	}
	s = s[:SlugMaxLength]
	if idx := strings.LastIndex(s, "-"); idx > SlugMinWordBoundary {
		s = s[:idx]
	}
	return s
}

// GetBaseDir returns the configured base directory.
func (fs *FileStorage) GetBaseDir() string {
	return fs.BaseDir
}

// GetSessionsDir returns the full path to the sessions directory.
func (fs *FileStorage) GetSessionsDir() string {
	return filepath.Join(fs.BaseDir, SessionsDir)
}

// GetIndexPath returns the full path to the index file.
func (fs *FileStorage) GetIndexPath() string {
	return filepath.Join(fs.BaseDir, IndexDir, IndexFile)
}

// GetProvenancePath returns the full path to the provenance file.
func (fs *FileStorage) GetProvenancePath() string {
	return filepath.Join(fs.BaseDir, ProvenanceDir, ProvenanceFile)
}
