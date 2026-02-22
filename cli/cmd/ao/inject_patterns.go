package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

// collectPatterns finds patterns from .agents/patterns/
func collectPatterns(cwd, query string, limit int) ([]pattern, error) {
	patternsDir := filepath.Join(cwd, ".agents", "patterns")
	if _, err := os.Stat(patternsDir); os.IsNotExist(err) {
		// Try rig root
		patternsDir = findAgentsSubdir(cwd, "patterns")
		if patternsDir == "" {
			return nil, nil
		}
	}

	files, err := filepath.Glob(filepath.Join(patternsDir, "*.md"))
	if err != nil {
		return nil, err
	}

	var patterns []pattern
	queryLower := strings.ToLower(query)
	now := time.Now()

	for _, file := range files {
		p, err := parsePatternFile(file)
		if err != nil {
			continue
		}

		info, _ := os.Stat(file)
		if info != nil {
			ageHours := now.Sub(info.ModTime()).Hours()
			ageWeeks := ageHours / (24 * 7)
			p.AgeWeeks = ageWeeks
			p.FreshnessScore = freshnessScore(ageWeeks)
		} else {
			p.FreshnessScore = 0.5
		}
		if p.Utility == 0 {
			p.Utility = types.InitialUtility
		}

		// Filter by query
		if query != "" {
			content := strings.ToLower(p.Name + " " + p.Description)
			if !strings.Contains(content, queryLower) {
				continue
			}
		}

		patterns = append(patterns, p)
	}

	items := make([]scorable, len(patterns))
	for i := range patterns {
		items[i] = &patterns[i]
	}
	applyCompositeScoringTo(items, types.DefaultLambda)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].CompositeScore > patterns[j].CompositeScore
	})
	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

// parsePatternFile extracts pattern info from a markdown file
func parsePatternFile(path string) (pattern, error) {
	p := pattern{
		Name:     strings.TrimSuffix(filepath.Base(path), ".md"),
		FilePath: path,
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}

	lines := strings.Split(string(content), "\n")
	contentStart := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "---" {
				contentStart = i + 1
				break
			}
			if strings.HasPrefix(line, "utility:") {
				utilityStr := strings.TrimSpace(strings.TrimPrefix(line, "utility:"))
				if utility, parseErr := strconv.ParseFloat(utilityStr, 64); parseErr == nil && utility > 0 {
					p.Utility = utility
				}
			}
		}
	}

	for i := contentStart; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if line == "" {
			continue
		}

		line = strings.TrimSpace(line)

		// Extract name from title
		if strings.HasPrefix(line, "# ") {
			p.Name = strings.TrimPrefix(line, "# ")
			continue
		}

		// First paragraph as description
		if p.Description == "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") && line != "" {
			desc := line
			for j := i + 1; j < len(lines) && j < i+2; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" || strings.HasPrefix(nextLine, "#") {
					break
				}
				desc += " " + nextLine
			}
			p.Description = truncateText(desc, 150)
			break
		}
	}

	return p, nil
}

