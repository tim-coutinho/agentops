package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/boshu2/agentops/cli/internal/types"
)

func TestUpdateJSONLUtility(t *testing.T) {
	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "feedback_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	tests := []struct {
		name           string
		initialContent string
		reward         float64
		alpha          float64
		wantOldUtility float64
		wantNewUtility float64
	}{
		{
			name:           "initial utility (no utility field)",
			initialContent: `{"id":"L001","title":"Test Learning"}`,
			reward:         1.0,
			alpha:          0.1,
			wantOldUtility: 0.5,  // InitialUtility
			wantNewUtility: 0.55, // (1-0.1)*0.5 + 0.1*1.0
		},
		{
			name:           "existing utility positive reward",
			initialContent: `{"id":"L002","title":"Test","utility":0.6}`,
			reward:         1.0,
			alpha:          0.1,
			wantOldUtility: 0.6,
			wantNewUtility: 0.64, // (1-0.1)*0.6 + 0.1*1.0
		},
		{
			name:           "existing utility negative reward",
			initialContent: `{"id":"L003","title":"Test","utility":0.6}`,
			reward:         0.0,
			alpha:          0.1,
			wantOldUtility: 0.6,
			wantNewUtility: 0.54, // (1-0.1)*0.6 + 0.1*0.0
		},
		{
			name:           "partial reward",
			initialContent: `{"id":"L004","title":"Test","utility":0.5}`,
			reward:         0.75,
			alpha:          0.1,
			wantOldUtility: 0.5,
			wantNewUtility: 0.525, // (1-0.1)*0.5 + 0.1*0.75
		},
		{
			name:           "higher alpha faster learning",
			initialContent: `{"id":"L005","title":"Test","utility":0.5}`,
			reward:         1.0,
			alpha:          0.3,
			wantOldUtility: 0.5,
			wantNewUtility: 0.65, // (1-0.3)*0.5 + 0.3*1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test file
			path := filepath.Join(tmpDir, tt.name+".jsonl")
			if err := os.WriteFile(path, []byte(tt.initialContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Update utility
			oldUtility, newUtility, err := updateJSONLUtility(path, tt.reward, tt.alpha)
			if err != nil {
				t.Fatalf("updateJSONLUtility() error = %v", err)
			}

			// Check old utility
			if abs(oldUtility-tt.wantOldUtility) > 0.001 {
				t.Errorf("oldUtility = %v, want %v", oldUtility, tt.wantOldUtility)
			}

			// Check new utility
			if abs(newUtility-tt.wantNewUtility) > 0.001 {
				t.Errorf("newUtility = %v, want %v", newUtility, tt.wantNewUtility)
			}

			// Verify file was updated correctly
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			var data map[string]any
			if err := json.Unmarshal(content, &data); err != nil {
				t.Fatalf("failed to parse updated file: %v", err)
			}

			// Verify utility was written
			utility, ok := data["utility"].(float64)
			if !ok {
				t.Error("utility field not found in updated file")
			}
			if abs(utility-tt.wantNewUtility) > 0.001 {
				t.Errorf("file utility = %v, want %v", utility, tt.wantNewUtility)
			}

			// Verify last_reward was written
			lastReward, ok := data["last_reward"].(float64)
			if !ok {
				t.Error("last_reward field not found")
			}
			if abs(lastReward-tt.reward) > 0.001 {
				t.Errorf("last_reward = %v, want %v", lastReward, tt.reward)
			}

			// Verify reward_count was incremented
			rewardCount, ok := data["reward_count"].(float64)
			if !ok {
				t.Error("reward_count field not found")
			}
			if rewardCount != 1 {
				t.Errorf("reward_count = %v, want 1", rewardCount)
			}
		})
	}
}

func TestCounterDirectionFromFeedback(t *testing.T) {
	tests := []struct {
		name            string
		reward          float64
		explicitHelpful bool
		explicitHarmful bool
		wantHelpful     bool
		wantHarmful     bool
	}{
		{
			name:            "explicit helpful wins",
			reward:          0.0,
			explicitHelpful: true,
			explicitHarmful: false,
			wantHelpful:     true,
			wantHarmful:     false,
		},
		{
			name:            "explicit harmful wins",
			reward:          1.0,
			explicitHelpful: false,
			explicitHarmful: true,
			wantHelpful:     false,
			wantHarmful:     true,
		},
		{
			name:            "high reward implied helpful",
			reward:          0.95,
			explicitHelpful: false,
			explicitHarmful: false,
			wantHelpful:     true,
			wantHarmful:     false,
		},
		{
			name:            "low reward implied harmful",
			reward:          0.05,
			explicitHelpful: false,
			explicitHarmful: false,
			wantHelpful:     false,
			wantHarmful:     true,
		},
		{
			name:            "mid reward neutral",
			reward:          0.5,
			explicitHelpful: false,
			explicitHarmful: false,
			wantHelpful:     false,
			wantHarmful:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHelpful, gotHarmful := counterDirectionFromFeedback(tt.reward, tt.explicitHelpful, tt.explicitHarmful)
			if gotHelpful != tt.wantHelpful || gotHarmful != tt.wantHarmful {
				t.Errorf("counterDirectionFromFeedback(%v, %v, %v) = (%v, %v), want (%v, %v)",
					tt.reward, tt.explicitHelpful, tt.explicitHarmful,
					gotHelpful, gotHarmful, tt.wantHelpful, tt.wantHarmful)
			}
		})
	}
}

func TestUpdateJSONLUtilityImpliedCounters(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "counter.jsonl")
	content := `{"id":"L100","title":"Counter Test","utility":0.5}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	origHelpful := feedbackHelpful
	origHarmful := feedbackHarmful
	feedbackHelpful = false
	feedbackHarmful = false
	t.Cleanup(func() {
		feedbackHelpful = origHelpful
		feedbackHarmful = origHarmful
	})

	if _, _, err := updateJSONLUtility(path, 0.95, 0.1); err != nil {
		t.Fatalf("high-reward update failed: %v", err)
	}
	if _, _, err := updateJSONLUtility(path, 0.05, 0.1); err != nil {
		t.Fatalf("low-reward update failed: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if got := int(data["helpful_count"].(float64)); got != 1 {
		t.Errorf("helpful_count = %d, want 1", got)
	}
	if got := int(data["harmful_count"].(float64)); got != 1 {
		t.Errorf("harmful_count = %d, want 1", got)
	}
}

func TestUpdateMarkdownUtility(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "feedback_md_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	tests := []struct {
		name           string
		initialContent string
		reward         float64
		alpha          float64
		wantOldUtility float64
		wantNewUtility float64
	}{
		{
			name: "no front matter",
			initialContent: `# Test Learning

This is the content.`,
			reward:         1.0,
			alpha:          0.1,
			wantOldUtility: types.InitialUtility,
			wantNewUtility: 0.55,
		},
		{
			name: "existing front matter without utility",
			initialContent: `---
id: L001
---
# Test Learning`,
			reward:         0.0,
			alpha:          0.1,
			wantOldUtility: types.InitialUtility,
			wantNewUtility: 0.45,
		},
		{
			name: "existing front matter with utility",
			initialContent: `---
utility: 0.7
---
# Test Learning`,
			reward:         1.0,
			alpha:          0.1,
			wantOldUtility: 0.7,
			wantNewUtility: 0.73,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".md")
			if err := os.WriteFile(path, []byte(tt.initialContent), 0644); err != nil {
				t.Fatal(err)
			}

			oldUtility, newUtility, err := updateMarkdownUtility(path, tt.reward, tt.alpha)
			if err != nil {
				t.Fatalf("updateMarkdownUtility() error = %v", err)
			}

			if abs(oldUtility-tt.wantOldUtility) > 0.001 {
				t.Errorf("oldUtility = %v, want %v", oldUtility, tt.wantOldUtility)
			}
			if abs(newUtility-tt.wantNewUtility) > 0.001 {
				t.Errorf("newUtility = %v, want %v", newUtility, tt.wantNewUtility)
			}
		})
	}
}

func TestFindLearningFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "find_learning_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	// Create .agents/learnings directory
	learningsDir := filepath.Join(tmpDir, ".agents", "learnings")
	if err := os.MkdirAll(learningsDir, 0755); err != nil {
		t.Fatal(err)
	}
	patternsDir := filepath.Join(tmpDir, ".agents", "patterns")
	if err := os.MkdirAll(patternsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	testFiles := []string{"L001.jsonl", "L002.md", "learning-003.jsonl"}
	for _, name := range testFiles {
		path := filepath.Join(learningsDir, name)
		if err := os.WriteFile(path, []byte(`{"id":"test"}`), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(patternsDir, "retry-backoff.md"), []byte("# Retry Backoff"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		learningID string
		wantFile   string
		wantErr    bool
	}{
		{
			name:       "find by ID (jsonl)",
			learningID: "L001",
			wantFile:   "L001.jsonl",
			wantErr:    false,
		},
		{
			name:       "find by ID (md)",
			learningID: "L002",
			wantFile:   "L002.md",
			wantErr:    false,
		},
		{
			name:       "find by partial match",
			learningID: "003",
			wantFile:   "learning-003.jsonl",
			wantErr:    false,
		},
		{
			name:       "find pattern by name",
			learningID: "retry-backoff",
			wantFile:   "retry-backoff.md",
			wantErr:    false,
		},
		{
			name:       "not found",
			learningID: "nonexistent",
			wantFile:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := findLearningFile(tmpDir, tt.learningID)
			if (err != nil) != tt.wantErr {
				t.Errorf("findLearningFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantFile != "" && filepath.Base(path) != tt.wantFile {
				t.Errorf("findLearningFile() = %v, want %v", filepath.Base(path), tt.wantFile)
			}
		})
	}
}

func TestNeedsUtilityMigration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "no utility field",
			content: `{"id":"L001","title":"Test"}`,
			want:    true,
		},
		{
			name:    "utility is zero",
			content: `{"id":"L002","utility":0}`,
			want:    true,
		},
		{
			name:    "has utility",
			content: `{"id":"L003","utility":0.5}`,
			want:    false,
		},
		{
			name:    "has high utility",
			content: `{"id":"L004","utility":0.9}`,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".jsonl")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := needsUtilityMigration(path)
			if err != nil {
				t.Fatalf("needsUtilityMigration() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("needsUtilityMigration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddUtilityField(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "add_utility_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // test cleanup
	}()

	content := `{"id":"L001","title":"Test Learning"}`
	path := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := addUtilityField(path); err != nil {
		t.Fatalf("addUtilityField() error = %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	utility, ok := result["utility"].(float64)
	if !ok {
		t.Fatal("utility field not added")
	}
	if abs(utility-types.InitialUtility) > 0.001 {
		t.Errorf("utility = %v, want %v", utility, types.InitialUtility)
	}
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
