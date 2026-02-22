package main

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

// collectLearnings finds recent learnings from .agents/learnings/
// Implements MemRL Two-Phase retrieval: Phase A (similarity/freshness) + Phase B (utility-weighted)
// With CASS integration: applies confidence decay when --apply-decay is set
func collectLearnings(cwd, query string, limit int) ([]learning, error) {
	learningsDir := filepath.Join(cwd, ".agents", "learnings")
	if _, err := os.Stat(learningsDir); os.IsNotExist(err) {
		// Try rig root
		learningsDir = findAgentsSubdir(cwd, "learnings")
		if learningsDir == "" {
			return nil, nil // No learnings directory
		}
	}

	files, err := filepath.Glob(filepath.Join(learningsDir, "*.md"))
	if err != nil {
		return nil, err
	}

	// Also check .jsonl files
	jsonlFiles, _ := filepath.Glob(filepath.Join(learningsDir, "*.jsonl"))
	files = append(files, jsonlFiles...)

	var learnings []learning
	queryLower := strings.ToLower(query)
	now := time.Now()

	for _, file := range files {
		l, err := parseLearningFile(file)
		if err != nil {
			continue
		}

		// F3: Skip superseded learnings (superseded_by field set)
		if l.Superseded {
			VerbosePrintf("Skipping superseded learning: %s\n", l.ID)
			continue
		}

		// Filter by query if provided
		if query != "" {
			content := strings.ToLower(l.Title + " " + l.Summary)
			if !strings.Contains(content, queryLower) {
				continue
			}
		}

		// Calculate freshness score: exp(-ageWeeks * decayRate)
		// decayRate = 0.17/week (literature default)
		info, _ := os.Stat(file)
		if info != nil {
			ageHours := now.Sub(info.ModTime()).Hours()
			ageWeeks := ageHours / (24 * 7)
			l.AgeWeeks = ageWeeks
			l.FreshnessScore = freshnessScore(ageWeeks)
		} else {
			l.FreshnessScore = 0.5 // Default for missing stat
		}

		// Set default utility if not set (for markdown files)
		if l.Utility == 0 {
			l.Utility = types.InitialUtility
		}

		// Apply confidence decay if requested (CASS feature)
		if injectApplyDecay {
			l = applyConfidenceDecay(l, file, now)
		}

		learnings = append(learnings, l)
	}

	// Phase B: Calculate composite scores with z-normalization
	// Score = z_norm(freshness) + λ × z_norm(utility)
	applyCompositeScoring(learnings, types.DefaultLambda)

	// Sort by composite score (highest first) - Two-Phase retrieval
	sort.Slice(learnings, func(i, j int) bool {
		return learnings[i].CompositeScore > learnings[j].CompositeScore
	})

	// Limit results
	if len(learnings) > limit {
		learnings = learnings[:limit]
	}

	return learnings, nil
}

// applyConfidenceDecay applies time-based confidence decay to a learning.
// Confidence decays at 10%/week for learnings that haven't received recent feedback.
// Formula: confidence *= exp(-weeks_since_last_feedback * ConfidenceDecayRate)
func applyConfidenceDecay(l learning, filePath string, now time.Time) learning {
	// Read the file to get last_decay_at and confidence
	content, err := os.ReadFile(filePath)
	if err != nil {
		return l
	}

	// Parse to extract CASS fields
	if strings.HasSuffix(filePath, ".jsonl") {
		lines := strings.Split(string(content), "\n")
		if len(lines) == 0 {
			return l
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(lines[0]), &data); err != nil {
			return l
		}

		// Get confidence (default to 0.5)
		confidence := 0.5
		if c, ok := data["confidence"].(float64); ok && c > 0 {
			confidence = c
		}

		// Get last_decay_at or last_reward_at
		var lastInteraction time.Time
		if lda, ok := data["last_decay_at"].(string); ok && lda != "" {
			lastInteraction, _ = time.Parse(time.RFC3339, lda)
		} else if lra, ok := data["last_reward_at"].(string); ok && lra != "" {
			lastInteraction, _ = time.Parse(time.RFC3339, lra)
		}

		// Calculate decay
		if !lastInteraction.IsZero() {
			weeksSinceInteraction := now.Sub(lastInteraction).Hours() / (24 * 7)
			if weeksSinceInteraction > 0 {
				// Apply decay: confidence *= exp(-weeks * decayRate)
				decayFactor := math.Exp(-weeksSinceInteraction * types.ConfidenceDecayRate)
				newConfidence := confidence * decayFactor

				// Clamp to minimum of 0.1
				if newConfidence < 0.1 {
					newConfidence = 0.1
				}

				VerbosePrintf("Applied confidence decay to %s: %.3f -> %.3f (%.1f weeks)\n",
					l.ID, confidence, newConfidence, weeksSinceInteraction)

				// Write back decayed confidence to file
				data["confidence"] = newConfidence
				data["last_decay_at"] = now.Format(time.RFC3339)

				// Increment decay_count (default 0)
				decayCount := 0.0
				if dc, ok := data["decay_count"].(float64); ok {
					decayCount = dc
				}
				data["decay_count"] = decayCount + 1

				newJSON, marshalErr := json.Marshal(data)
				if marshalErr == nil {
					lines[0] = string(newJSON)
					_ = os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
				}

				// Update the learning's composite score weight
				l.Utility = l.Utility * (newConfidence / confidence)
			}
		}
	}

	return l
}

// frontMatter holds parsed YAML front matter fields
type frontMatter struct {
	SupersededBy string
	Utility      float64
	HasUtility   bool
}

// parseFrontMatter extracts YAML front matter from markdown content
func parseFrontMatter(lines []string) (frontMatter, int) {
	var fm frontMatter
	endLine := 0

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fm, 0
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			endLine = i + 1
			break
		}
		if strings.HasPrefix(line, "superseded_by:") || strings.HasPrefix(line, "superseded-by:") {
			fm.SupersededBy = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
		if strings.HasPrefix(line, "utility:") {
			utilityStr := strings.TrimSpace(strings.TrimPrefix(line, "utility:"))
			if utility, err := strconv.ParseFloat(utilityStr, 64); err == nil && utility > 0 {
				fm.Utility = utility
				fm.HasUtility = true
			}
		}
	}
	return fm, endLine
}

// extractSummary finds the first paragraph after headings
func extractSummary(lines []string, startIdx int) string {
	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "---") {
			continue
		}
		// Take first paragraph (up to 3 lines)
		summary := line
		for j := i + 1; j < len(lines) && j < i+3; j++ {
			nextLine := strings.TrimSpace(lines[j])
			if nextLine == "" || strings.HasPrefix(nextLine, "#") {
				break
			}
			summary += " " + nextLine
		}
		return truncateText(summary, 200)
	}
	return ""
}

// parseLearningFile extracts learning info from a file
// Sets Superseded=true if superseded_by field is found
func parseLearningFile(path string) (learning, error) {
	l := learning{
		ID:     filepath.Base(path),
		Source: path,
	}

	if strings.HasSuffix(path, ".jsonl") {
		return parseLearningJSONL(path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return l, err
	}

	lines := strings.Split(string(content), "\n")

	// Parse front matter
	fm, contentStart := parseFrontMatter(lines)
	if fm.SupersededBy != "" && fm.SupersededBy != "null" && fm.SupersededBy != "~" {
		l.Superseded = true
		return l, nil
	}
	if fm.HasUtility {
		l.Utility = fm.Utility
	}

	// Parse body content
	for i := contentStart; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if strings.HasPrefix(line, "# ") && l.Title == "" {
			l.Title = strings.TrimPrefix(line, "# ")
		} else if (strings.HasPrefix(line, "ID:") || strings.HasPrefix(line, "id:")) && l.ID == filepath.Base(path) {
			l.ID = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}

	l.Summary = extractSummary(lines, contentStart)

	if l.Title == "" {
		l.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return l, nil
}

// parseLearningJSONL extracts learning from JSONL file
// Returns empty learning (with Superseded=true) if superseded_by field is set
func parseLearningJSONL(path string) (learning, error) {
	l := learning{
		ID:      filepath.Base(path),
		Source:  path,
		Utility: types.InitialUtility, // Default to 0.5
	}

	f, err := os.Open(path)
	if err != nil {
		return l, err
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only learning load, close error non-fatal
	}()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		var data map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &data); err == nil {
			// F3: Filter superseded learnings - skip if superseded_by is set
			if supersededBy, ok := data["superseded_by"]; ok && supersededBy != nil && supersededBy != "" {
				l.Superseded = true
				return l, nil // Return early, will be filtered out
			}

			if id, ok := data["id"].(string); ok {
				l.ID = id
			}
			if title, ok := data["title"].(string); ok {
				l.Title = title
			}
			if summary, ok := data["summary"].(string); ok {
				l.Summary = truncateText(summary, 200)
			}
			if content, ok := data["content"].(string); ok && l.Summary == "" {
				l.Summary = truncateText(content, 200)
			}
			// Parse MemRL utility value
			if utility, ok := data["utility"].(float64); ok && utility > 0 {
				l.Utility = utility
			}
		}
	}

	return l, nil
}
