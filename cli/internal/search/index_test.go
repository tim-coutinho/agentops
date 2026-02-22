package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildIndex(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	writeFile(t, filepath.Join(dir, "session1.md"), "Mutex pattern for concurrent access\nGo routines and channels")
	writeFile(t, filepath.Join(dir, "session2.md"), "Authentication with OAuth tokens\nMutex lock contention")
	writeFile(t, filepath.Join(dir, "notes.jsonl"), `{"summary":"database migration strategy"}`)
	writeFile(t, filepath.Join(dir, "ignore.txt"), "this should be ignored")

	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	// "mutex" should appear in session1.md and session2.md
	if docs, ok := idx.Terms["mutex"]; !ok || len(docs) != 2 {
		t.Errorf("expected 'mutex' in 2 docs, got %d", len(docs))
	}

	// "authentication" should appear only in session2.md
	if docs, ok := idx.Terms["authentication"]; !ok || len(docs) != 1 {
		t.Errorf("expected 'authentication' in 1 doc, got %d", len(docs))
	}

	// "database" should appear only in notes.jsonl
	if docs, ok := idx.Terms["database"]; !ok || len(docs) != 1 {
		t.Errorf("expected 'database' in 1 doc, got %d", len(docs))
	}

	// "ignored" (from .txt file) should not be in the index
	if _, ok := idx.Terms["ignored"]; ok {
		t.Error("expected .txt file to be ignored")
	}
}

func TestBuildIndexSubdirs(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "learnings")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sub, "deep.md"), "deeply nested content")

	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	if docs, ok := idx.Terms["deeply"]; !ok || len(docs) != 1 {
		t.Errorf("expected 'deeply' in 1 doc, got %v", docs)
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "mutex pattern goroutine")
	writeFile(t, filepath.Join(dir, "b.md"), "mutex lock contention")
	writeFile(t, filepath.Join(dir, "c.md"), "authentication oauth tokens")

	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	// Search for "mutex pattern" — a.md matches both terms (score 2), b.md matches one (score 1)
	results := Search(idx, "mutex pattern", 10)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	if results[0].Score < results[1].Score {
		t.Errorf("expected results sorted by score descending")
	}
	// a.md should be first (matches both "mutex" and "pattern")
	if filepath.Base(results[0].Path) != "a.md" {
		t.Errorf("expected a.md first, got %s", results[0].Path)
	}

	// Search with limit
	results = Search(idx, "mutex", 1)
	if len(results) != 1 {
		t.Errorf("expected 1 result with limit=1, got %d", len(results))
	}

	// Search for non-existent term
	results = Search(idx, "xyznonexistent", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	// Empty query
	results = Search(idx, "", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestSaveAndLoadIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "doc.md"), "mutex pattern authentication")

	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	indexPath := filepath.Join(dir, "index.jsonl")
	if err := SaveIndex(idx, indexPath); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index file not created: %v", err)
	}

	// Load and verify
	loaded, err := LoadIndex(indexPath)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	// Check that loaded index has the same terms
	for term, docs := range idx.Terms {
		loadedDocs, ok := loaded.Terms[term]
		if !ok {
			t.Errorf("loaded index missing term %q", term)
			continue
		}
		if len(loadedDocs) != len(docs) {
			t.Errorf("term %q: expected %d docs, got %d", term, len(docs), len(loadedDocs))
		}
	}

	// Search the loaded index
	results := Search(loaded, "mutex", 10)
	if len(results) != 1 {
		t.Errorf("expected 1 result from loaded index, got %d", len(results))
	}
}

func TestUpdateIndex(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc.md")
	writeFile(t, docPath, "original content alpha")

	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	// Verify original terms
	if _, ok := idx.Terms["alpha"]; !ok {
		t.Error("expected 'alpha' in index")
	}

	// Update file content
	writeFile(t, docPath, "updated content beta")

	if err := UpdateIndex(idx, docPath); err != nil {
		t.Fatalf("UpdateIndex: %v", err)
	}

	// "alpha" should be gone (or at least not point to docPath)
	if docs, ok := idx.Terms["alpha"]; ok && docs[docPath] {
		t.Error("expected 'alpha' to be removed for docPath after update")
	}

	// "beta" should be present
	if docs, ok := idx.Terms["beta"]; !ok || !docs[docPath] {
		t.Error("expected 'beta' to be present after update")
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple words", "Hello World", []string{"hello", "world"}},
		{"with punctuation", "mutex.Lock() called!", []string{"mutex", "lock", "called"}},
		{"preserves hyphens", "pre-commit hook", []string{"pre-commit", "hook"}},
		{"preserves underscores", "file_path value", []string{"file_path", "value"}},
		{"filters short", "a I go is", []string{"go", "is"}},
		{"mixed case", "GoRoutine MUTEX", []string{"goroutine", "mutex"}},
		{"deduplicates", "foo bar foo", []string{"foo", "bar"}},
		{"empty string", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("tokenize(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadIndexMissing(t *testing.T) {
	_, err := LoadIndex("/nonexistent/path/index.jsonl")
	if err == nil {
		t.Error("expected error for missing index file")
	}
}

func TestSearchEmptyIndex(t *testing.T) {
	idx := NewIndex()
	results := Search(idx, "anything", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty index, got %d", len(results))
	}
}

func TestBuildIndex_UnreadableEntry(t *testing.T) {
	dir := t.TempDir()

	// Create a readable file so the index is non-empty
	writeFile(t, filepath.Join(dir, "good.md"), "readable content here")

	// Create a subdirectory with no read permission to trigger filepath.Walk err
	badDir := filepath.Join(dir, "unreadable")
	if err := os.MkdirAll(badDir, 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(badDir, "secret.md"), "hidden")
	if err := os.Chmod(badDir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(badDir, 0700) // restore for cleanup
	})

	// BuildIndex should succeed; unreadable entries are skipped
	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex should not fail on unreadable entries: %v", err)
	}
	if _, ok := idx.Terms["readable"]; !ok {
		t.Error("expected 'readable' in index from good.md")
	}
}

func TestBuildIndex_NonexistentDir(t *testing.T) {
	// filepath.Walk calls the callback with the Lstat error for the root.
	// Our callback returns nil (skip), so Walk returns nil too.
	// BuildIndex should return an empty index without error.
	idx, err := BuildIndex("/nonexistent/dir/does/not/exist")
	if err != nil {
		t.Fatalf("BuildIndex on nonexistent dir should not error (skips), got: %v", err)
	}
	if len(idx.Terms) != 0 {
		t.Errorf("expected empty index for nonexistent dir, got %d terms", len(idx.Terms))
	}
}

func TestSaveIndex_EmptyDocsSkipped(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex()

	// Manually add a term with empty docs map
	idx.Terms["orphan"] = make(map[string]bool)
	idx.Terms["real"] = map[string]bool{"/some/path.md": true}

	indexPath := filepath.Join(dir, "index.jsonl")
	if err := SaveIndex(idx, indexPath); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// Load and verify orphan was skipped
	loaded, err := LoadIndex(indexPath)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if _, ok := loaded.Terms["orphan"]; ok {
		t.Error("orphan term with empty docs should not be saved")
	}
	if _, ok := loaded.Terms["real"]; !ok {
		t.Error("real term should be saved")
	}
}

func TestSaveIndex_ReadOnlyDir(t *testing.T) {
	dir := t.TempDir()
	// Make directory read-only
	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(readOnlyDir, 0700)
	})

	idx := NewIndex()
	idx.Terms["test"] = map[string]bool{"/a.md": true}

	err := SaveIndex(idx, filepath.Join(readOnlyDir, "subdir", "index.jsonl"))
	if err == nil {
		t.Error("expected error when saving to read-only directory")
	}
}

func TestLoadIndex_EmptyAndMalformedLines(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.jsonl")

	// Write index with empty lines and malformed JSON mixed in
	content := `{"term":"good","paths":["/a.md"]}

{"not valid json
{"term":"also_good","paths":["/b.md"]}
`
	writeFile(t, indexPath, content)

	loaded, err := LoadIndex(indexPath)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	// Should have both valid terms, skipping empty and malformed
	if _, ok := loaded.Terms["good"]; !ok {
		t.Error("expected 'good' term in loaded index")
	}
	if _, ok := loaded.Terms["also_good"]; !ok {
		t.Error("expected 'also_good' term in loaded index")
	}
}

func TestSaveIndex_NestedDirCreation(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex()
	idx.Terms["hello"] = map[string]bool{"/doc.md": true}

	// Save to a deeply nested path that doesn't exist yet
	deepPath := filepath.Join(dir, "a", "b", "c", "index.jsonl")
	if err := SaveIndex(idx, deepPath); err != nil {
		t.Fatalf("SaveIndex to deep path: %v", err)
	}

	// Verify it was created and is loadable
	loaded, err := LoadIndex(deepPath)
	if err != nil {
		t.Fatalf("LoadIndex deep path: %v", err)
	}
	if _, ok := loaded.Terms["hello"]; !ok {
		t.Error("expected 'hello' term in deeply nested index")
	}
}

func TestSearch_LimitZeroReturnsAll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "keyword alpha")
	writeFile(t, filepath.Join(dir, "b.md"), "keyword beta")
	writeFile(t, filepath.Join(dir, "c.md"), "keyword gamma")

	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	// limit=0 means no limit
	results := Search(idx, "keyword", 0)
	if len(results) != 3 {
		t.Errorf("expected 3 results with limit=0, got %d", len(results))
	}
}

func TestSearch_TieBreakByPath(t *testing.T) {
	idx := NewIndex()
	// Two docs with same score
	idx.Terms["shared"] = map[string]bool{"/b.md": true, "/a.md": true}

	results := Search(idx, "shared", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Should be sorted alphabetically by path when scores tie
	if results[0].Path != "/a.md" {
		t.Errorf("expected /a.md first (tie-break by path), got %s", results[0].Path)
	}
}

func TestSaveIndex_CreateFileError(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex()
	idx.Terms["test"] = map[string]bool{"/a.md": true}

	// Create a directory where the file should go, but make it read-only
	targetDir := filepath.Join(dir, "indexdir")
	if err := os.MkdirAll(targetDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(targetDir, 0700) })

	err := SaveIndex(idx, filepath.Join(targetDir, "index.jsonl"))
	if err == nil {
		t.Error("expected error when directory is read-only for file creation")
	}
}

func TestIndexFile_UnreadableFile(t *testing.T) {
	dir := t.TempDir()

	// Create a file then make it unreadable
	mdFile := filepath.Join(dir, "secret.md")
	writeFile(t, mdFile, "secret content")
	if err := os.Chmod(mdFile, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(mdFile, 0644) })

	idx := NewIndex()
	err := indexFile(idx, mdFile)
	if err == nil {
		t.Error("expected error when indexing unreadable file")
	}
}

func TestBuildIndex_SkipsUnreadableFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a readable file and an unreadable file
	writeFile(t, filepath.Join(dir, "readable.md"), "visible content")
	secretFile := filepath.Join(dir, "secret.md")
	writeFile(t, secretFile, "hidden content")
	if err := os.Chmod(secretFile, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(secretFile, 0644) })

	// BuildIndex should succeed, skipping unreadable files
	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if _, ok := idx.Terms["visible"]; !ok {
		t.Error("expected 'visible' in index from readable.md")
	}
	if _, ok := idx.Terms["hidden"]; ok {
		t.Error("expected 'hidden' NOT in index from unreadable file")
	}
}

func TestBuildIndex_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	idx, err := BuildIndex(dir)
	if err != nil {
		t.Fatalf("BuildIndex empty dir: %v", err)
	}
	if len(idx.Terms) != 0 {
		t.Errorf("expected 0 terms for empty dir, got %d", len(idx.Terms))
	}
}

func TestUpdateIndex_UnreadableFile(t *testing.T) {
	idx := NewIndex()

	// Try to update with a nonexistent file
	err := UpdateIndex(idx, "/nonexistent/file.md")
	if err == nil {
		t.Error("expected error when updating with nonexistent file")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
