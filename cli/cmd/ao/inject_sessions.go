package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// collectRecentSessions finds recent session summaries
func collectRecentSessions(cwd, query string, limit int) ([]session, error) {
	sessionsDir := filepath.Join(cwd, ".agents", "ao", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := filepath.Glob(filepath.Join(sessionsDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	// Also include markdown summaries
	mdFiles, _ := filepath.Glob(filepath.Join(sessionsDir, "*.md"))
	files = append(files, mdFiles...)

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		if infoI == nil || infoJ == nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	var sessions []session
	queryLower := strings.ToLower(query)

	for _, file := range files {
		if len(sessions) >= limit {
			break
		}

		s, err := parseSessionFile(file)
		if err != nil || s.Summary == "" {
			continue
		}

		// Filter by query
		if query != "" && !strings.Contains(strings.ToLower(s.Summary), queryLower) {
			continue
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
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
		f, err := os.Open(path)
		if err != nil {
			return s, err
		}
		defer func() {
			_ = f.Close() //nolint:errcheck // read-only session load, close error non-fatal
		}()

		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			var data map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &data); err == nil {
				if summary, ok := data["summary"].(string); ok {
					s.Summary = truncateText(summary, 150)
				}
			}
		}
	} else {
		// Markdown - extract first paragraph
		content, err := os.ReadFile(path)
		if err != nil {
			return s, err
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
				s.Summary = truncateText(line, 150)
				break
			}
		}
	}

	return s, nil
}
