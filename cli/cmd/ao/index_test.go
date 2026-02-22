package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func createTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("create test file %s: %v", name, err)
	}
}

func TestIndexGenerate(t *testing.T) {
	dir := t.TempDir()

	createTestFile(t, dir, "2026-01-15-first.md", `---
summary: First learning entry
tags: [alpha, beta]
---
# First
Content here.
`)

	createTestFile(t, dir, "2026-02-10-second.md", `---
summary: Second learning entry
tags:
  - gamma
  - delta
---
# Second
More content.
`)

	entries, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Write INDEX.md
	err = writeIndex(dir, ".agents/learnings", entries, false)
	if err != nil {
		t.Fatalf("writeIndex: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "INDEX.md"))
	if err != nil {
		t.Fatalf("read INDEX.md: %v", err)
	}

	s := string(content)
	today := time.Now().Format("2006-01-02")

	if !strings.Contains(s, "# Index: Learnings") {
		t.Error("missing header")
	}
	if !strings.Contains(s, today) {
		t.Error("missing today's date in header")
	}
	if !strings.Contains(s, "| 2026-01-15-first.md |") {
		t.Error("missing first file entry")
	}
	if !strings.Contains(s, "| 2026-02-10-second.md |") {
		t.Error("missing second file entry")
	}
	if !strings.Contains(s, "| File | Date | Summary | Tags |") {
		t.Error("missing table header")
	}
	if !strings.Contains(s, "2 entries") {
		t.Error("missing entry count")
	}
}

func TestIndexCheck(t *testing.T) {
	dir := t.TempDir()

	createTestFile(t, dir, "2026-01-01-test.md", `---
summary: Test entry
tags: [test]
---
# Test
`)

	entries, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory: %v", err)
	}

	err = writeIndex(dir, ".agents/learnings", entries, false)
	if err != nil {
		t.Fatalf("writeIndex: %v", err)
	}

	// Check should report current
	isStale, msg := checkIndex(dir, ".agents/learnings", entries)
	if isStale {
		t.Errorf("expected current, got stale: %s", msg)
	}
}

func TestIndexCheckStale(t *testing.T) {
	dir := t.TempDir()

	createTestFile(t, dir, "2026-01-01-test.md", `---
summary: Test entry
tags: [test]
---
# Test
`)

	entries, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory: %v", err)
	}

	err = writeIndex(dir, ".agents/learnings", entries, false)
	if err != nil {
		t.Fatalf("writeIndex: %v", err)
	}

	// Add a new file after INDEX.md was written
	createTestFile(t, dir, "2026-02-01-new.md", `---
summary: New entry
---
# New
`)

	// Re-scan to get updated entries
	newEntries, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory: %v", err)
	}

	isStale, msg := checkIndex(dir, ".agents/learnings", newEntries)
	if !isStale {
		t.Error("expected stale, got current")
	}
	if !strings.Contains(msg, "missing") {
		t.Errorf("expected 'missing' in message, got: %s", msg)
	}
}

func TestIndexMalformedFrontmatter(t *testing.T) {
	dir := t.TempDir()

	createTestFile(t, dir, "2026-03-01-broken.md", `---
this is: [not valid: yaml: {{
---
# Broken File
Some content here.
`)

	entries, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Filename != "2026-03-01-broken.md" {
		t.Errorf("expected filename 2026-03-01-broken.md, got %s", e.Filename)
	}
	// Should fall back to filename date
	if e.Date != "2026-03-01" {
		t.Errorf("expected date 2026-03-01, got %s", e.Date)
	}
	// Summary should fall back to H1
	if e.Summary != "Broken File" {
		t.Errorf("expected summary 'Broken File', got '%s'", e.Summary)
	}
}

func TestIndexBothDateFields(t *testing.T) {
	dir := t.TempDir()

	// File with created_at
	createTestFile(t, dir, "file-with-created-at.md", `---
created_at: 2026-01-20
summary: Has created_at
---
# Created At File
`)

	// File with date
	createTestFile(t, dir, "file-with-date.md", `---
date: 2026-01-25
summary: Has date field
---
# Date File
`)

	// File with both (created_at should win)
	createTestFile(t, dir, "file-with-both.md", `---
created_at: 2026-01-30
date: 2026-01-01
summary: Has both fields
---
# Both Fields
`)

	entries, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Build a map for easy lookup
	byFile := make(map[string]indexEntry)
	for _, e := range entries {
		byFile[e.Filename] = e
	}

	if e := byFile["file-with-created-at.md"]; e.Date != "2026-01-20" {
		t.Errorf("created_at file: expected date 2026-01-20, got %s", e.Date)
	}
	if e := byFile["file-with-date.md"]; e.Date != "2026-01-25" {
		t.Errorf("date file: expected date 2026-01-25, got %s", e.Date)
	}
	if e := byFile["file-with-both.md"]; e.Date != "2026-01-30" {
		t.Errorf("both fields file: expected created_at=2026-01-30, got %s", e.Date)
	}
}
