// Package ratchet implements the Brownian Ratchet workflow tracking.
// This file implements CASS (Contextual Agent Session Search) maturity transitions.
package ratchet

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boshu2/agentops/cli/internal/types"
)

// MaturityTransitionResult contains the result of a maturity transition check.
type MaturityTransitionResult struct {
	// LearningID is the identifier of the learning.
	LearningID string `json:"learning_id"`

	// OldMaturity is the maturity before transition.
	OldMaturity types.Maturity `json:"old_maturity"`

	// NewMaturity is the maturity after transition.
	NewMaturity types.Maturity `json:"new_maturity"`

	// Transitioned indicates if a transition occurred.
	Transitioned bool `json:"transitioned"`

	// Reason explains why the transition did or didn't occur.
	Reason string `json:"reason"`

	// Utility is the current utility value.
	Utility float64 `json:"utility"`

	// Confidence is the current confidence value.
	Confidence float64 `json:"confidence"`

	// HelpfulCount is the number of helpful feedback events.
	HelpfulCount int `json:"helpful_count"`

	// HarmfulCount is the number of harmful feedback events.
	HarmfulCount int `json:"harmful_count"`

	// RewardCount is the total number of feedback events.
	RewardCount int `json:"reward_count"`
}

// CheckMaturityTransition evaluates if a learning should transition to a new maturity level.
// Transition rules:
//   - provisional → candidate: utility >= 0.7 AND reward_count >= 3
//   - candidate → established: utility >= 0.7 AND reward_count >= 5 AND helpful_count > harmful_count
//   - any → anti-pattern: utility <= 0.2 AND harmful_count >= 5
//   - established → candidate: utility < 0.5 (demotion)
//   - candidate → provisional: utility < 0.3 (demotion)
func CheckMaturityTransition(learningPath string) (*MaturityTransitionResult, error) {
	data, err := readLearningData(learningPath)
	if err != nil {
		return nil, err
	}

	result := buildMaturityTransitionResult(learningPath, data)
	if applyAntiPatternTransition(result) {
		return result, nil
	}

	applyMaturitySpecificTransition(result)
	return result, nil
}

func readLearningData(learningPath string) (map[string]any, error) {
	content, err := os.ReadFile(learningPath)
	if err != nil {
		return nil, fmt.Errorf("read learning: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return nil, ErrEmptyLearningFile
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &data); err != nil {
		return nil, fmt.Errorf("parse learning: %w", err)
	}

	return data, nil
}

func buildMaturityTransitionResult(learningPath string, data map[string]any) *MaturityTransitionResult {
	learningID := stringFromData(data, "id", filepath.Base(learningPath), false)
	currentMaturity := types.Maturity(stringFromData(data, "maturity", string(types.MaturityProvisional), true))

	return &MaturityTransitionResult{
		LearningID:   learningID,
		OldMaturity:  currentMaturity,
		NewMaturity:  currentMaturity,
		Transitioned: false,
		Utility:      floatFromData(data, "utility", types.InitialUtility),
		Confidence:   floatFromData(data, "confidence", 0.5),
		HelpfulCount: intFromData(data, "helpful_count"),
		HarmfulCount: intFromData(data, "harmful_count"),
		RewardCount:  intFromData(data, "reward_count"),
	}
}

func applyAntiPatternTransition(result *MaturityTransitionResult) bool {
	if result.Utility > types.MaturityAntiPatternThreshold || result.HarmfulCount < types.MinFeedbackForAntiPattern {
		return false
	}

	result.NewMaturity = types.MaturityAntiPattern
	result.Transitioned = result.OldMaturity != types.MaturityAntiPattern
	result.Reason = fmt.Sprintf("utility %.2f <= %.2f and harmful_count %d >= %d",
		result.Utility, types.MaturityAntiPatternThreshold, result.HarmfulCount, types.MinFeedbackForAntiPattern)
	return true
}

func applyMaturitySpecificTransition(result *MaturityTransitionResult) {
	switch result.OldMaturity {
	case types.MaturityProvisional:
		applyProvisionalTransition(result)
	case types.MaturityCandidate:
		applyCandidateTransition(result)
	case types.MaturityEstablished:
		applyEstablishedTransition(result)
	case types.MaturityAntiPattern:
		applyAntiPatternRehabilitationTransition(result)
	}
}

func applyProvisionalTransition(result *MaturityTransitionResult) {
	if result.Utility >= types.MaturityPromotionThreshold && result.RewardCount >= types.MinFeedbackForPromotion {
		result.NewMaturity = types.MaturityCandidate
		result.Transitioned = true
		result.Reason = fmt.Sprintf("utility %.2f >= %.2f and reward_count %d >= %d",
			result.Utility, types.MaturityPromotionThreshold, result.RewardCount, types.MinFeedbackForPromotion)
		return
	}

	result.Reason = "not enough positive feedback for promotion"
}

func applyCandidateTransition(result *MaturityTransitionResult) {
	if result.Utility >= types.MaturityPromotionThreshold && result.RewardCount >= 5 && result.HelpfulCount > result.HarmfulCount {
		result.NewMaturity = types.MaturityEstablished
		result.Transitioned = true
		result.Reason = fmt.Sprintf("utility %.2f >= %.2f, reward_count %d >= 5, helpful > harmful (%d > %d)",
			result.Utility, types.MaturityPromotionThreshold, result.RewardCount, result.HelpfulCount, result.HarmfulCount)
		return
	}

	if result.Utility < types.MaturityDemotionThreshold {
		result.NewMaturity = types.MaturityProvisional
		result.Transitioned = true
		result.Reason = fmt.Sprintf("utility %.2f < %.2f (demotion)",
			result.Utility, types.MaturityDemotionThreshold)
		return
	}

	result.Reason = "maintaining candidate status"
}

func applyEstablishedTransition(result *MaturityTransitionResult) {
	if result.Utility < 0.5 {
		result.NewMaturity = types.MaturityCandidate
		result.Transitioned = true
		result.Reason = fmt.Sprintf("utility %.2f < 0.5 (demotion from established)",
			result.Utility)
		return
	}

	result.Reason = "maintaining established status"
}

func applyAntiPatternRehabilitationTransition(result *MaturityTransitionResult) {
	if result.Utility >= 0.6 && result.HelpfulCount > result.HarmfulCount*2 {
		result.NewMaturity = types.MaturityProvisional
		result.Transitioned = true
		result.Reason = fmt.Sprintf("utility %.2f >= 0.6 and helpful > 2*harmful (%d > %d) - rehabilitation",
			result.Utility, result.HelpfulCount, result.HarmfulCount*2)
		return
	}

	result.Reason = "maintaining anti-pattern status"
}

func stringFromData(data map[string]any, key, defaultValue string, requireNonEmpty bool) string {
	value, ok := data[key].(string)
	if !ok {
		return defaultValue
	}
	if requireNonEmpty && value == "" {
		return defaultValue
	}
	return value
}

func floatFromData(data map[string]any, key string, defaultValue float64) float64 {
	value, ok := data[key].(float64)
	if !ok {
		return defaultValue
	}
	return value
}

func intFromData(data map[string]any, key string) int {
	value, ok := data[key].(float64)
	if !ok {
		return 0
	}
	return int(value)
}

// ApplyMaturityTransition checks and applies a maturity transition to a learning file.
// Returns the transition result and updates the file if a transition occurred.
func ApplyMaturityTransition(learningPath string) (*MaturityTransitionResult, error) {
	result, err := CheckMaturityTransition(learningPath)
	if err != nil {
		return nil, err
	}

	if !result.Transitioned {
		return result, nil
	}

	updates := map[string]any{
		"maturity":            string(result.NewMaturity),
		"maturity_changed_at": time.Now().Format(time.RFC3339),
		"maturity_reason":     result.Reason,
	}
	if err := updateJSONLFirstLine(learningPath, updates); err != nil {
		return nil, err
	}

	return result, nil
}

// mergeJSONData unmarshals firstLine as JSON, merges updates into it, and returns re-marshaled JSON.
func mergeJSONData(firstLine string, updates map[string]any) ([]byte, error) {
	var data map[string]any
	if err := json.Unmarshal([]byte(firstLine), &data); err != nil {
		return nil, fmt.Errorf("parse learning for update: %w", err)
	}
	for k, v := range updates {
		data[k] = v
	}
	newJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal updated learning: %w", err)
	}
	return newJSON, nil
}

// updateJSONLFirstLine reads a JSONL file, merges fields into the first
// JSON line, and writes the file back.
func updateJSONLFirstLine(path string, updates map[string]any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read learning for update: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return ErrEmptyFile
	}

	newJSON, err := mergeJSONData(lines[0], updates)
	if err != nil {
		return err
	}

	lines[0] = string(newJSON)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		return fmt.Errorf("write updated learning: %w", err)
	}
	return nil
}

// ScanForMaturityTransitions scans a learnings directory and returns all pending transitions.
func ScanForMaturityTransitions(learningsDir string) ([]*MaturityTransitionResult, error) {
	files, err := filepath.Glob(filepath.Join(learningsDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("glob learnings: %w", err)
	}

	var results []*MaturityTransitionResult

	for _, file := range files {
		result, err := CheckMaturityTransition(file)
		if err != nil {
			continue // Skip files that can't be parsed
		}

		// Only include learnings that would transition
		if result.Transitioned {
			results = append(results, result)
		}
	}

	return results, nil
}

// filterLearningsByMaturity returns JSONL file paths whose first line has
// the specified maturity value.
func filterLearningsByMaturity(learningsDir string, target types.Maturity) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(learningsDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("glob learnings: %w", err)
	}

	var result []string
	for _, file := range files {
		if readFirstLineMaturity(file) == string(target) {
			result = append(result, file)
		}
	}
	return result, nil
}

// readFirstLineMaturity reads the first JSON line of a JSONL file and returns
// its "maturity" field value, or "" on any error.
func readFirstLineMaturity(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() {
		_ = f.Close() //nolint:errcheck // read-only
	}()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
		return ""
	}
	maturity, _ := data["maturity"].(string)
	return maturity
}

// GetAntiPatterns returns all learnings marked as anti-patterns.
func GetAntiPatterns(learningsDir string) ([]string, error) {
	return filterLearningsByMaturity(learningsDir, types.MaturityAntiPattern)
}

// GetEstablishedLearnings returns all learnings with established maturity.
func GetEstablishedLearnings(learningsDir string) ([]string, error) {
	return filterLearningsByMaturity(learningsDir, types.MaturityEstablished)
}

// MaturityDistribution represents the count of learnings at each maturity level.
type MaturityDistribution struct {
	Provisional int `json:"provisional"`
	Candidate   int `json:"candidate"`
	Established int `json:"established"`
	AntiPattern int `json:"anti_pattern"`
	Unknown     int `json:"unknown"`
	Total       int `json:"total"`
}

// GetMaturityDistribution returns the distribution of learnings across maturity levels.
// Counts both .jsonl files (with full metadata) and .md files (classified as provisional).
func GetMaturityDistribution(learningsDir string) (*MaturityDistribution, error) {
	jsonlFiles, err := filepath.Glob(filepath.Join(learningsDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("glob learnings: %w", err)
	}

	dist := &MaturityDistribution{}
	for _, file := range jsonlFiles {
		classifyLearningFile(file, dist)
	}

	// Count .md learnings (no JSONL metadata → provisional by default)
	mdFiles, err := filepath.Glob(filepath.Join(learningsDir, "*.md"))
	if err != nil {
		return dist, nil
	}
	for range mdFiles {
		dist.Provisional++
		dist.Total++
	}

	return dist, nil
}

// classifyLearningFile reads the first line of a JSONL file and updates the distribution.
func classifyLearningFile(file string, dist *MaturityDistribution) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck // read-only

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return
	}

	var data map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
		dist.Unknown++
		dist.Total++
		return
	}

	maturity, ok := data["maturity"].(string)
	if !ok || maturity == "" {
		maturity = string(types.MaturityProvisional)
	}

	incrementMaturity(dist, types.Maturity(maturity))
	dist.Total++
}

// incrementMaturity increments the appropriate maturity counter in the distribution.
func incrementMaturity(dist *MaturityDistribution, m types.Maturity) {
	switch m {
	case types.MaturityProvisional:
		dist.Provisional++
	case types.MaturityCandidate:
		dist.Candidate++
	case types.MaturityEstablished:
		dist.Established++
	case types.MaturityAntiPattern:
		dist.AntiPattern++
	default:
		dist.Unknown++
	}
}
