package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boshu2/agentops/cli/internal/types"
)

var (
	feedbackReward  float64
	feedbackAlpha   float64
	feedbackHelpful bool
	feedbackHarmful bool
)

const (
	impliedHelpfulRewardThreshold = 0.8
	impliedHarmfulRewardThreshold = 0.2
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback <learning-id>",
	Short: "Record reward feedback for a learning",
	Long: `Record reward feedback for a learning to update its utility value.

This implements the MemRL EMA update rule:
  u_{t+1} = (1 - α) × u_t + α × r

Where:
  u_t = current utility value (default: 0.5)
  α   = learning rate (default: 0.1)
  r   = reward signal (0.0 = failure, 1.0 = success)

The utility value affects retrieval ranking in Two-Phase retrieval:
  Score = z_norm(freshness) + λ × z_norm(utility)

CASS Integration:
  - --helpful and --harmful are shortcuts for --reward 1.0 and --reward 0.0
  - Tracks helpful_count and harmful_count for maturity transitions
  - Repeated harmful feedback can promote to anti-pattern status

Examples:
  ao feedback L001 --helpful        # Learning was helpful (same as --reward 1.0)
  ao feedback L001 --harmful        # Learning was harmful (same as --reward 0.0)
  ao feedback L001 --reward 1.0     # Learning was helpful (success)
  ao feedback L001 --reward 0.0     # Learning was not helpful (failure)
  ao feedback L001 --reward 0.75    # Partial success
  ao feedback L001 --reward 1.0 --alpha 0.2   # Faster learning rate`,
	Args: cobra.ExactArgs(1),
	RunE: runFeedback,
}

func init() {
	rootCmd.AddCommand(feedbackCmd)
	feedbackCmd.Flags().Float64Var(&feedbackReward, "reward", -1, "Reward value (0.0 to 1.0)")
	feedbackCmd.Flags().Float64Var(&feedbackAlpha, "alpha", types.DefaultAlpha, "EMA learning rate")
	feedbackCmd.Flags().BoolVar(&feedbackHelpful, "helpful", false, "Mark as helpful (shortcut for --reward 1.0)")
	feedbackCmd.Flags().BoolVar(&feedbackHarmful, "harmful", false, "Mark as harmful (shortcut for --reward 0.0)")
	// Note: reward is no longer required since --helpful/--harmful can be used instead
}

func runFeedback(cmd *cobra.Command, args []string) error {
	learningID := args[0]

	// Handle --helpful and --harmful shortcuts
	if feedbackHelpful && feedbackHarmful {
		return fmt.Errorf("cannot use both --helpful and --harmful")
	}
	if feedbackHelpful {
		feedbackReward = 1.0
	} else if feedbackHarmful {
		feedbackReward = 0.0
	}

	// Validate that some feedback was provided
	if feedbackReward < 0 {
		return fmt.Errorf("must provide --reward, --helpful, or --harmful")
	}

	// Validate reward range
	if feedbackReward > 1 {
		return fmt.Errorf("reward must be between 0.0 and 1.0, got: %f", feedbackReward)
	}

	// Validate alpha range
	if feedbackAlpha <= 0 || feedbackAlpha > 1 {
		return fmt.Errorf("alpha must be between 0 and 1 (exclusive 0), got: %f", feedbackAlpha)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Find the learning file
	learningPath, err := findLearningFile(cwd, learningID)
	if err != nil {
		return fmt.Errorf("find learning: %w", err)
	}

	if GetDryRun() {
		fmt.Printf("[dry-run] Would update utility for %s:\n", learningID)
		fmt.Printf("  Reward: %.2f\n", feedbackReward)
		fmt.Printf("  Alpha: %.2f\n", feedbackAlpha)
		return nil
	}

	// Update utility
	oldUtility, newUtility, err := updateLearningUtility(learningPath, feedbackReward, feedbackAlpha)
	if err != nil {
		return fmt.Errorf("update utility: %w", err)
	}

	// Determine feedback type for display
	feedbackType := "custom"
	if feedbackHelpful {
		feedbackType = "helpful"
	} else if feedbackHarmful {
		feedbackType = "harmful"
	}

	switch GetOutput() {
	case "json":
		result := map[string]interface{}{
			"learning_id":   learningID,
			"path":          learningPath,
			"old_utility":   oldUtility,
			"new_utility":   newUtility,
			"reward":        feedbackReward,
			"feedback_type": feedbackType,
			"alpha":         feedbackAlpha,
			"updated_at":    time.Now().Format(time.RFC3339),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)

	default:
		fmt.Printf("Updated utility for %s\n", learningID)
		fmt.Printf("  Previous: %.3f\n", oldUtility)
		fmt.Printf("  Feedback: %s (reward=%.2f)\n", feedbackType, feedbackReward)
		fmt.Printf("  New:      %.3f\n", newUtility)
	}

	return nil
}

// findLearningFile locates a learning file by ID.
func findLearningFile(baseDir, learningID string) (string, error) {
	searchDirs := []string{
		filepath.Join(baseDir, ".agents", "learnings"),
		filepath.Join(baseDir, ".agents", "patterns"),
	}

	// Try direct match with common extensions
	extensions := []string{".jsonl", ".md", ".json"}
	for _, dirPath := range searchDirs {
		for _, ext := range extensions {
			path := filepath.Join(dirPath, learningID+ext)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	// Try without extension if ID already has one
	for _, dirPath := range searchDirs {
		path := filepath.Join(dirPath, learningID)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Search for files containing the ID
	for _, dirPath := range searchDirs {
		files, err := filepath.Glob(filepath.Join(dirPath, "*"+learningID+"*"))
		if err != nil {
			return "", err
		}
		if len(files) > 0 {
			return files[0], nil
		}
	}

	// Walk up to rig root looking for .agents/learnings and .agents/patterns
	dir := baseDir
	for {
		candidates := []string{
			filepath.Join(dir, ".agents", "learnings"),
			filepath.Join(dir, ".agents", "patterns"),
		}
		for _, candidate := range candidates {
			isCurrent := false
			for _, current := range searchDirs {
				if candidate == current {
					isCurrent = true
					break
				}
			}
			if isCurrent {
				continue
			}
			for _, ext := range extensions {
				path := filepath.Join(candidate, learningID+ext)
				if _, err := os.Stat(path); err == nil {
					return path, nil
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("learning not found: %s", learningID)
}

// updateLearningUtility applies the EMA update rule and writes back.
func updateLearningUtility(path string, reward, alpha float64) (oldUtility, newUtility float64, err error) {
	if strings.HasSuffix(path, ".jsonl") {
		return updateJSONLUtility(path, reward, alpha)
	}
	return updateMarkdownUtility(path, reward, alpha)
}

// updateJSONLUtility updates utility in a JSONL file.
// Also tracks helpful_count and harmful_count for CASS maturity transitions.
func updateJSONLUtility(path string, reward, alpha float64) (oldUtility, newUtility float64, err error) {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}

	// Parse first line as JSON
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return 0, 0, fmt.Errorf("empty JSONL file")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &data); err != nil {
		return 0, 0, fmt.Errorf("parse JSONL: %w", err)
	}

	// Get current utility (default to InitialUtility)
	oldUtility = types.InitialUtility
	if u, ok := data["utility"].(float64); ok && u > 0 {
		oldUtility = u
	}

	// Apply EMA update: u_{t+1} = (1 - α) × u_t + α × r
	newUtility = (1-alpha)*oldUtility + alpha*reward

	// Update fields
	data["utility"] = newUtility
	data["last_reward"] = reward
	rewardCount := 0
	if rc, ok := data["reward_count"].(float64); ok {
		rewardCount = int(rc)
	}
	data["reward_count"] = rewardCount + 1
	data["last_reward_at"] = time.Now().Format(time.RFC3339)

	// CASS: Track helpful_count and harmful_count.
	// Explicit CLI labels take precedence; otherwise infer coarse signal from reward bands.
	incrementHelpful, incrementHarmful := counterDirectionFromFeedback(reward, feedbackHelpful, feedbackHarmful)
	if incrementHelpful {
		helpfulCount := 0
		if hc, ok := data["helpful_count"].(float64); ok {
			helpfulCount = int(hc)
		}
		data["helpful_count"] = helpfulCount + 1
	} else if incrementHarmful {
		harmfulCount := 0
		if hc, ok := data["harmful_count"].(float64); ok {
			harmfulCount = int(hc)
		}
		data["harmful_count"] = harmfulCount + 1
	}

	// Update confidence based on feedback count
	// Confidence increases with more feedback: 1 - e^(-rewardCount/5)
	newRewardCount := rewardCount + 1
	confidence := 1.0 - (1.0 / (1.0 + float64(newRewardCount)/5.0))
	data["confidence"] = confidence
	data["last_decay_at"] = time.Now().Format(time.RFC3339)

	// Write back
	newJSON, err := json.Marshal(data)
	if err != nil {
		return 0, 0, err
	}

	// Replace first line, keep the rest
	lines[0] = string(newJSON)
	return oldUtility, newUtility, os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func counterDirectionFromFeedback(reward float64, explicitHelpful, explicitHarmful bool) (helpful bool, harmful bool) {
	if explicitHelpful {
		return true, false
	}
	if explicitHarmful {
		return false, true
	}
	if reward >= impliedHelpfulRewardThreshold {
		return true, false
	}
	if reward <= impliedHarmfulRewardThreshold {
		return false, true
	}
	return false, false
}

// updateMarkdownUtility updates utility in a markdown file with front matter.
func updateMarkdownUtility(path string, reward, alpha float64) (oldUtility, newUtility float64, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}

	text := string(content)
	lines := strings.Split(text, "\n")

	// Check for front matter
	hasFrontMatter := len(lines) > 0 && strings.TrimSpace(lines[0]) == "---"

	oldUtility = types.InitialUtility

	if hasFrontMatter {
		// Find end of front matter
		endIdx := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				endIdx = i
				break
			}
			// Parse existing utility
			if strings.HasPrefix(lines[i], "utility:") {
				_, _ = fmt.Sscanf(lines[i], "utility: %f", &oldUtility) //nolint:errcheck // best effort parse
			}
		}

		if endIdx == -1 {
			return 0, 0, fmt.Errorf("malformed front matter: no closing ---")
		}

		// Apply EMA update
		newUtility = (1-alpha)*oldUtility + alpha*reward

		// Update or add fields in front matter
		updatedFM := updateFrontMatterFields(lines[1:endIdx], map[string]string{
			"utility":        fmt.Sprintf("%.4f", newUtility),
			"last_reward":    fmt.Sprintf("%.2f", reward),
			"reward_count":   incrementRewardCount(lines[1:endIdx]),
			"last_reward_at": time.Now().Format(time.RFC3339),
		})

		// Reconstruct file
		var sb strings.Builder
		sb.WriteString("---\n")
		for _, line := range updatedFM {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString("---\n")
		for i := endIdx + 1; i < len(lines); i++ {
			sb.WriteString(lines[i])
			if i < len(lines)-1 {
				sb.WriteString("\n")
			}
		}

		return oldUtility, newUtility, os.WriteFile(path, []byte(sb.String()), 0644)
	}

	// No front matter - add it
	newUtility = (1-alpha)*oldUtility + alpha*reward

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("utility: %.4f\n", newUtility))
	sb.WriteString(fmt.Sprintf("last_reward: %.2f\n", reward))
	sb.WriteString("reward_count: 1\n")
	sb.WriteString(fmt.Sprintf("last_reward_at: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("---\n")
	sb.WriteString(text)

	return oldUtility, newUtility, os.WriteFile(path, []byte(sb.String()), 0644)
}

// updateFrontMatterFields updates or adds fields in front matter lines.
func updateFrontMatterFields(lines []string, fields map[string]string) []string {
	result := make([]string, 0, len(lines)+len(fields))
	seen := make(map[string]bool)

	for _, line := range lines {
		updated := false
		for key, value := range fields {
			if strings.HasPrefix(line, key+":") {
				result = append(result, fmt.Sprintf("%s: %s", key, value))
				seen[key] = true
				updated = true
				break
			}
		}
		if !updated {
			result = append(result, line)
		}
	}

	// Add missing fields
	for key, value := range fields {
		if !seen[key] {
			result = append(result, fmt.Sprintf("%s: %s", key, value))
		}
	}

	return result
}

// incrementRewardCount parses and increments reward_count from front matter.
func incrementRewardCount(lines []string) string {
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "reward_count:") {
			_, _ = fmt.Sscanf(line, "reward_count: %d", &count) //nolint:errcheck // best effort parse
			break
		}
	}
	return fmt.Sprintf("%d", count+1)
}

// migrateCmd adds utility field to learnings without it.
var migrateCmd = &cobra.Command{
	Use:   "migrate memrl",
	Short: "Migrate learnings to include utility field",
	Long: `Migrate existing learnings to include MemRL utility field.

Scans .agents/learnings/ and adds utility: 0.5 to entries without it.
This prepares learnings for Two-Phase retrieval.

Examples:
  ao migrate memrl
  ao migrate memrl --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	if args[0] != "memrl" {
		return fmt.Errorf("unknown migration: %s (supported: memrl)", args[0])
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	learningsDir := filepath.Join(cwd, ".agents", "learnings")
	if _, err := os.Stat(learningsDir); os.IsNotExist(err) {
		fmt.Println("No learnings directory found.")
		return nil
	}

	files, err := filepath.Glob(filepath.Join(learningsDir, "*.jsonl"))
	if err != nil {
		return err
	}

	migrated := 0
	skipped := 0

	for _, file := range files {
		needsMigration, err := needsUtilityMigration(file)
		if err != nil {
			VerbosePrintf("Warning: check %s: %v\n", filepath.Base(file), err)
			continue
		}

		if !needsMigration {
			skipped++
			continue
		}

		if GetDryRun() {
			fmt.Printf("[dry-run] Would migrate: %s\n", filepath.Base(file))
			migrated++
			continue
		}

		if err := addUtilityField(file); err != nil {
			VerbosePrintf("Warning: migrate %s: %v\n", filepath.Base(file), err)
			continue
		}
		migrated++
	}

	fmt.Printf("Migration complete: %d migrated, %d skipped (already have utility)\n", migrated, skipped)
	return nil
}

// needsUtilityMigration checks if a JSONL file needs utility field added.
func needsUtilityMigration(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close() //nolint:errcheck // read-only file

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			return false, err
		}
		// If utility field exists and is non-zero, no migration needed
		if u, ok := data["utility"].(float64); ok && u > 0 {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// addUtilityField adds utility: 0.5 to a JSONL file.
func addUtilityField(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &data); err != nil {
		return err
	}

	data["utility"] = types.InitialUtility

	newJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	lines[0] = string(newJSON)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}
