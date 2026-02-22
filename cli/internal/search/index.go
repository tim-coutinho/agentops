// Package search provides an inverted index for fast keyword-based search
// across AgentOps session and knowledge files.
package search

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Index is an in-memory inverted index mapping lowercase terms to the
// set of document paths that contain them.
type Index struct {
	// Terms maps each lowercase term to a set of document paths.
	Terms map[string]map[string]bool `json:"-"`
}

// IndexEntry is the JSONL-serialised form: one line per term.
type IndexEntry struct {
	Term  string   `json:"term"`
	Paths []string `json:"paths"`
}

// IndexResult is returned by Search.
type IndexResult struct {
	Path  string
	Score int // number of query terms matched
}

// NewIndex creates an empty index.
func NewIndex() *Index {
	return &Index{Terms: make(map[string]map[string]bool)}
}

// BuildIndex scans all .md and .jsonl files under dir (recursively) and
// builds an inverted index from their content.
func BuildIndex(dir string) (*Index, error) {
	idx := NewIndex()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".jsonl" {
			return nil
		}
		if err := indexFile(idx, path); err != nil {
			// Non-fatal: skip files we cannot read
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", dir, err)
	}

	return idx, nil
}

// UpdateIndex adds or re-indexes a single file in the index.
// It first removes any existing entries for the path, then re-scans.
func UpdateIndex(idx *Index, path string) error {
	// Remove old entries for this path
	for _, docs := range idx.Terms {
		delete(docs, path)
	}
	return indexFile(idx, path)
}

// Search finds documents matching the query and returns up to limit results
// sorted by descending score (number of matching query terms).
func Search(idx *Index, query string, limit int) []IndexResult {
	queryTerms := tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	// Count how many query terms each document matches
	scores := make(map[string]int)
	for _, term := range queryTerms {
		if docs, ok := idx.Terms[term]; ok {
			for doc := range docs {
				scores[doc]++
			}
		}
	}

	if len(scores) == 0 {
		return nil
	}

	results := make([]IndexResult, 0, len(scores))
	for path, score := range scores {
		results = append(results, IndexResult{Path: path, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Path < results[j].Path
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// SaveIndex writes the index to a JSONL file (one line per term).
func SaveIndex(idx *Index, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create index file: %w", err)
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // best-effort close
	}()

	w := bufio.NewWriter(f)

	// Sort terms for deterministic output
	terms := make([]string, 0, len(idx.Terms))
	for term := range idx.Terms {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	for _, term := range terms {
		docs := idx.Terms[term]
		if len(docs) == 0 {
			continue
		}
		paths := make([]string, 0, len(docs))
		for p := range docs {
			paths = append(paths, p)
		}
		sort.Strings(paths)

		entry := IndexEntry{Term: term, Paths: paths}
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal term %q: %w", term, err)
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		if _, err := w.WriteString("\n"); err != nil {
			return err
		}
	}

	return w.Flush()
}

// LoadIndex reads an index from a JSONL file.
func LoadIndex(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only, close best-effort
	}()

	idx := NewIndex()
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry IndexEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		docs := make(map[string]bool, len(entry.Paths))
		for _, p := range entry.Paths {
			docs[p] = true
		}
		idx.Terms[entry.Term] = docs
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	return idx, nil
}

// indexFile reads a file and adds its terms to the index.
func indexFile(idx *Index, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only, close best-effort
	}()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	seen := make(map[string]bool) // dedupe terms per file

	for scanner.Scan() {
		line := scanner.Text()
		terms := tokenize(line)
		for _, term := range terms {
			if seen[term] {
				continue
			}
			seen[term] = true
			if idx.Terms[term] == nil {
				idx.Terms[term] = make(map[string]bool)
			}
			idx.Terms[term][path] = true
		}
	}

	return scanner.Err()
}

// tokenize splits text into lowercase word tokens.
// Strips punctuation and filters out very short (< 2 char) tokens.
func tokenize(text string) []string {
	lower := strings.ToLower(text)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_'
	})

	// Filter short tokens and deduplicate within this call
	result := make([]string, 0, len(words))
	seen := make(map[string]bool, len(words))
	for _, w := range words {
		if len(w) < 2 || seen[w] {
			continue
		}
		seen[w] = true
		result = append(result, w)
	}
	return result
}
