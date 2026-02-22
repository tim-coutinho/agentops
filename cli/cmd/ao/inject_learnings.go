package main

import (
	"bufio"
	"cmp"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

// collectLearnings finds recent learnings from .agents/learnings/
// Implements MemRL Two-Phase retrieval: Phase A (similarity/freshness) + Phase B (utility-weighted)
// With CASS integration: applies confidence decay when --apply-decay is set
func collectLearnings(cwd, query string, limit int) ([]learning, error) {
	files, err := findLearningFiles(cwd)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	now := time.Now()
	queryLower := strings.ToLower(query)
	learnings := make([]learning, 0, len(files))

	for _, file := range files {
		l, ok := processLearningFile(file, queryLower, now)
		if !ok {
			continue
		}
		learnings = append(learnings, l)
	}

	rankLearnings(learnings)

	if len(learnings) > limit {
		learnings = learnings[:limit]
	}
	return learnings, nil
}

// findLearningFiles discovers .md and .jsonl files in the learnings directory.
func findLearningFiles(cwd string) ([]string, error) {
	learningsDir := filepath.Join(cwd, ".agents", "learnings")
	if _, err := os.Stat(learningsDir); os.IsNotExist(err) {
		learningsDir = findAgentsSubdir(cwd, "learnings")
		if learningsDir == "" {
			return nil, nil
		}
	}

	files, err := filepath.Glob(filepath.Join(learningsDir, "*.md"))
	if err != nil {
		return nil, err
	}
	jsonlFiles, _ := filepath.Glob(filepath.Join(learningsDir, "*.jsonl"))
	return append(files, jsonlFiles...), nil
}

// processLearningFile parses, filters, and scores a single learning file.
// Returns the learning and true if it should be included, false otherwise.
func processLearningFile(file, queryLower string, now time.Time) (learning, bool) {
	l, err := parseLearningFile(file)
	if err != nil {
		return l, false
	}
	if l.Superseded {
		VerbosePrintf("Skipping superseded learning: %s\n", l.ID)
		return l, false
	}
	if queryLower != "" && !strings.Contains(strings.ToLower(l.Title+" "+l.Summary), queryLower) {
		return l, false
	}

	applyFreshnessScore(&l, file, now)

	if l.Utility == 0 {
		l.Utility = types.InitialUtility
	}
	if injectApplyDecay {
		l = applyConfidenceDecay(l, file, now)
	}
	return l, true
}

// applyFreshnessScore sets the freshness score on a learning based on file modification time.
func applyFreshnessScore(l *learning, file string, now time.Time) {
	info, _ := os.Stat(file)
	if info == nil {
		l.FreshnessScore = 0.5
		return
	}
	ageWeeks := now.Sub(info.ModTime()).Hours() / (24 * 7)
	l.AgeWeeks = ageWeeks
	l.FreshnessScore = freshnessScore(ageWeeks)
}

// rankLearnings applies composite scoring and sorts by score descending.
func rankLearnings(learnings []learning) {
	items := make([]scorable, len(learnings))
	for i := range learnings {
		items[i] = &learnings[i]
	}
	applyCompositeScoringTo(items, types.DefaultLambda)

	slices.SortFunc(learnings, func(a, b learning) int {
		return cmp.Compare(b.CompositeScore, a.CompositeScore)
	})
}

// applyConfidenceDecay applies time-based confidence decay to a learning.
// Confidence decays at 10%/week for learnings that haven't received recent feedback.
// Formula: confidence *= exp(-weeks_since_last_feedback * ConfidenceDecayRate)
func applyConfidenceDecay(l learning, filePath string, now time.Time) learning {
	if !strings.HasSuffix(filePath, ".jsonl") {
		return l
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return l
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return l
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &data); err != nil {
		return l
	}

	confidence := jsonFloat(data, "confidence", 0.5)
	lastInteraction := jsonTimeField(data, "last_decay_at", "last_reward_at")
	if lastInteraction.IsZero() {
		return l
	}

	weeksSinceInteraction := now.Sub(lastInteraction).Hours() / (24 * 7)
	if weeksSinceInteraction <= 0 {
		return l
	}

	newConfidence := computeDecayedConfidence(confidence, weeksSinceInteraction)
	VerbosePrintf("Applied confidence decay to %s: %.3f -> %.3f (%.1f weeks)\n",
		l.ID, confidence, newConfidence, weeksSinceInteraction)

	writeDecayFields(data, newConfidence, now)
	if newJSON, marshalErr := json.Marshal(data); marshalErr == nil {
		lines[0] = string(newJSON)
		_ = os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
	}

	l.Utility *= newConfidence / confidence
	return l
}

// jsonFloat extracts a float64 from a map, returning defaultVal if missing or non-positive.
func jsonFloat(data map[string]any, key string, defaultVal float64) float64 {
	if c, ok := data[key].(float64); ok && c > 0 {
		return c
	}
	return defaultVal
}

// jsonTimeField tries to parse a time.Time from the first non-empty string field found among keys.
func jsonTimeField(data map[string]any, keys ...string) time.Time {
	for _, k := range keys {
		if v, ok := data[k].(string); ok && v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// computeDecayedConfidence applies exponential decay and clamps to a minimum of 0.1.
func computeDecayedConfidence(confidence, weeks float64) float64 {
	decayFactor := math.Exp(-weeks * types.ConfidenceDecayRate)
	result := confidence * decayFactor
	if result < 0.1 {
		return 0.1
	}
	return result
}

// writeDecayFields updates the data map with new confidence, timestamp, and incremented decay count.
func writeDecayFields(data map[string]any, newConfidence float64, now time.Time) {
	data["confidence"] = newConfidence
	data["last_decay_at"] = now.Format(time.RFC3339)
	decayCount := 0.0
	if dc, ok := data["decay_count"].(float64); ok {
		decayCount = dc
	}
	data["decay_count"] = decayCount + 1
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

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fm, 0
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			return fm, i + 1
		}
		parseFrontMatterLine(line, &fm)
	}
	return fm, 0
}

// parseFrontMatterLine parses a single YAML front matter line into fm fields.
func parseFrontMatterLine(line string, fm *frontMatter) {
	switch {
	case strings.HasPrefix(line, "superseded_by:"), strings.HasPrefix(line, "superseded-by:"):
		fm.SupersededBy = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
	case strings.HasPrefix(line, "utility:"):
		utilityStr := strings.TrimSpace(strings.TrimPrefix(line, "utility:"))
		if utility, err := strconv.ParseFloat(utilityStr, 64); err == nil && utility > 0 {
			fm.Utility = utility
			fm.HasUtility = true
		}
	}
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

// isSuperseded returns true if the front matter indicates a superseded learning.
func isSuperseded(fm frontMatter) bool {
	return fm.SupersededBy != "" && fm.SupersededBy != "null" && fm.SupersededBy != "~"
}

// parseLearningBody extracts title and ID from markdown body lines into l.
func parseLearningBody(lines []string, start int, l *learning) {
	defaultID := filepath.Base(l.Source)
	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "# ") && l.Title == "" {
			l.Title = strings.TrimPrefix(line, "# ")
		} else if (strings.HasPrefix(line, "ID:") || strings.HasPrefix(line, "id:")) && l.ID == defaultID {
			l.ID = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}
}

// parseLearningFile extracts learning info from a file
// Sets Superseded=true if superseded_by field is found
func parseLearningFile(path string) (learning, error) {
	if strings.HasSuffix(path, ".jsonl") {
		return parseLearningJSONL(path)
	}

	l := learning{
		ID:     filepath.Base(path),
		Source: path,
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return l, err
	}

	lines := strings.Split(string(content), "\n")
	fm, contentStart := parseFrontMatter(lines)

	if isSuperseded(fm) {
		l.Superseded = true
		return l, nil
	}
	if fm.HasUtility {
		l.Utility = fm.Utility
	}

	parseLearningBody(lines, contentStart, &l)
	l.Summary = extractSummary(lines, contentStart)

	if l.Title == "" {
		l.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return l, nil
}

// populateLearningFromJSON fills learning fields from a parsed JSON map.
func populateLearningFromJSON(data map[string]any, l *learning) {
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
	if utility, ok := data["utility"].(float64); ok && utility > 0 {
		l.Utility = utility
	}
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
	if !scanner.Scan() {
		return l, nil
	}

	var data map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
		return l, nil
	}

	// F3: Filter superseded learnings - skip if superseded_by is set
	if supersededBy, ok := data["superseded_by"]; ok && supersededBy != nil && supersededBy != "" {
		l.Superseded = true
		return l, nil
	}

	populateLearningFromJSON(data, &l)
	return l, nil
}
