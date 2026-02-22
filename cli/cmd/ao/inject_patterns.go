package main

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

// enrichPatternFreshness sets age, freshness, and default utility on a pattern
// based on the file's modification time.
func enrichPatternFreshness(p *pattern, file string, now time.Time) {
	info, _ := os.Stat(file)
	if info != nil {
		ageHours := now.Sub(info.ModTime()).Hours()
		p.AgeWeeks = ageHours / (24 * 7)
		p.FreshnessScore = freshnessScore(p.AgeWeeks)
	} else {
		p.FreshnessScore = 0.5
	}
	if p.Utility == 0 {
		p.Utility = types.InitialUtility
	}
}

// patternMatchesQuery returns true if the pattern name or description contains
// the query (case-insensitive). An empty query matches everything.
func patternMatchesQuery(p pattern, queryLower string) bool {
	if queryLower == "" {
		return true
	}
	content := strings.ToLower(p.Name + " " + p.Description)
	return strings.Contains(content, queryLower)
}

// collectPatterns finds patterns from .agents/patterns/
func collectPatterns(cwd, query string, limit int) ([]pattern, error) {
	patternsDir := filepath.Join(cwd, ".agents", "patterns")
	if _, err := os.Stat(patternsDir); os.IsNotExist(err) {
		patternsDir = findAgentsSubdir(cwd, "patterns")
		if patternsDir == "" {
			return nil, nil
		}
	}

	files, err := filepath.Glob(filepath.Join(patternsDir, "*.md"))
	if err != nil {
		return nil, err
	}

	patterns := make([]pattern, 0, len(files))
	queryLower := strings.ToLower(query)
	now := time.Now()

	for _, file := range files {
		p, err := parsePatternFile(file)
		if err != nil {
			continue
		}
		enrichPatternFreshness(&p, file, now)

		if !patternMatchesQuery(p, queryLower) {
			continue
		}
		patterns = append(patterns, p)
	}

	items := make([]scorable, len(patterns))
	for i := range patterns {
		items[i] = &patterns[i]
	}
	applyCompositeScoringTo(items, types.DefaultLambda)
	slices.SortFunc(patterns, func(a, b pattern) int {
		return cmp.Compare(b.CompositeScore, a.CompositeScore)
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
	contentStart, utility := parseFrontmatterBlock(lines)
	if utility > 0 {
		p.Utility = utility
	}
	name, description := extractPatternNameAndDescription(lines, contentStart)
	if name != "" {
		p.Name = name
	}
	if description != "" {
		p.Description = description
	}

	return p, nil
}

// parseFrontmatterBlock scans YAML frontmatter and returns content start index and utility value.
func parseFrontmatterBlock(lines []string) (contentStart int, utility float64) {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return 0, 0
	}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			return i + 1, utility
		}
		if strings.HasPrefix(line, "utility:") {
			utilityStr := strings.TrimSpace(strings.TrimPrefix(line, "utility:"))
			if u, parseErr := strconv.ParseFloat(utilityStr, 64); parseErr == nil && u > 0 {
				utility = u
			}
		}
	}
	return 0, utility
}

// assembleDescriptionFrom builds a description by joining the line at index i
// with up to one following continuation line.
func assembleDescriptionFrom(lines []string, i int) string {
	desc := strings.TrimSpace(lines[i])
	for j := i + 1; j < len(lines) && j < i+2; j++ {
		nextLine := strings.TrimSpace(lines[j])
		if nextLine == "" || strings.HasPrefix(nextLine, "#") {
			break
		}
		desc += " " + nextLine
	}
	return truncateText(desc, 150)
}

// isContentLine returns true if the trimmed line is a non-empty body line
// (not a heading or frontmatter delimiter).
func isContentLine(line string) bool {
	return line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---")
}

// extractPatternNameAndDescription scans content lines for title and description.
func extractPatternNameAndDescription(lines []string, contentStart int) (name, description string) {
	for i := contentStart; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			name = strings.TrimPrefix(line, "# ")
			continue
		}
		if description == "" && isContentLine(line) {
			description = assembleDescriptionFrom(lines, i)
			break
		}
	}
	return name, description
}
