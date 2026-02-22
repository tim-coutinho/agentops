package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// collectSessionFiles gathers .jsonl and .md files from the sessions directory,
// sorted by modification time (newest first).
func collectSessionFiles(sessionsDir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(sessionsDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	mdFiles, _ := filepath.Glob(filepath.Join(sessionsDir, "*.md"))
	files = append(files, mdFiles...)

	slices.SortFunc(files, func(a, b string) int {
		infoA, _ := os.Stat(a)
		infoB, _ := os.Stat(b)
		if infoA == nil || infoB == nil {
			return 0
		}
		return infoB.ModTime().Compare(infoA.ModTime())
	})
	return files, nil
}

// collectRecentSessions finds recent session summaries
func collectRecentSessions(cwd, query string, limit int) ([]session, error) {
	sessionsDir := filepath.Join(cwd, ".agents", "ao", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := collectSessionFiles(sessionsDir)
	if err != nil {
		return nil, err
	}

	sessions := make([]session, 0, len(files))
	queryLower := strings.ToLower(query)

	for _, file := range files {
		if len(sessions) >= limit {
			break
		}

		s, err := parseSessionFile(file)
		if err != nil || s.Summary == "" {
			continue
		}

		if query != "" && !strings.Contains(strings.ToLower(s.Summary), queryLower) {
			continue
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}

// parseJSONLSessionSummary reads the first line of a JSONL file and returns
// the truncated "summary" field value (empty string if absent or unparseable).
func parseJSONLSessionSummary(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only session load, close error non-fatal
	}()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		var data map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &data); err == nil {
			if summary, ok := data["summary"].(string); ok {
				return truncateText(summary, 150), nil
			}
		}
	}
	return "", nil
}

// parseMarkdownSessionSummary extracts the first content paragraph from a
// markdown file, skipping headings and frontmatter delimiters.
func parseMarkdownSessionSummary(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			return truncateText(line, 150), nil
		}
	}
	return "", nil
}

// parseSessionFile extracts session summary from a file
func parseSessionFile(path string) (session, error) {
	s := session{}

	info, err := os.Stat(path)
	if err != nil {
		return s, err
	}
	s.Date = info.ModTime().Format("2006-01-02")

	if strings.HasSuffix(path, ".jsonl") {
		s.Summary, err = parseJSONLSessionSummary(path)
	} else {
		s.Summary, err = parseMarkdownSessionSummary(path)
	}

	return s, err
}
