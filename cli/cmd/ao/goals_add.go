package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/boshu2/agentops/cli/internal/goals"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	goalsAddWeight      int
	goalsAddType        string
	goalsAddDescription string
)

var goalsAddCmd = &cobra.Command{
	Use:     "add <id> <check-command>",
	Aliases: []string{"a"},
	Short:   "Add a new goal to GOALS.yaml",
	GroupID: "management",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		check := args[1]

		// Validate kebab-case ID (reuse pattern from internal/goals).
		if !goals.KebabRe.MatchString(id) {
			return fmt.Errorf("goal ID must be kebab-case: %q", id)
		}

		// Load existing goals to check for duplicates.
		gf, err := goals.LoadGoals(goalsFile)
		if err != nil {
			return fmt.Errorf("loading goals: %w", err)
		}

		for _, g := range gf.Goals {
			if g.ID == id {
				return fmt.Errorf("goal %q already exists", id)
			}
		}

		// Validate check command runs successfully (unless --dry-run).
		if !dryRun {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(goalsTimeout)*time.Second)
			defer cancel()
			testCmd := exec.CommandContext(ctx, "bash", "-c", check)
			if out, err := testCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("check command failed (exit non-zero):\n%s", string(out))
			}
		}

		// Validate type.
		goalType := goals.GoalType(goalsAddType)
		if goalsAddType != "" && !goals.ValidTypes[goalType] {
			return fmt.Errorf("invalid type %q (valid: health, architecture, quality, meta)", goalsAddType)
		}
		if goalsAddType == "" {
			goalType = goals.GoalTypeHealth
		}

		desc := goalsAddDescription
		if desc == "" {
			desc = id // Fallback to ID.
		}

		// Build new goal.
		newGoal := goals.Goal{
			ID:          id,
			Description: desc,
			Check:       check,
			Weight:      goalsAddWeight,
			Type:        goalType,
		}

		// Append to GoalFile and write back.
		gf.Goals = append(gf.Goals, newGoal)
		data, err := yaml.Marshal(gf)
		if err != nil {
			return fmt.Errorf("marshaling goals: %w", err)
		}
		if err := os.WriteFile(goalsFile, data, 0o644); err != nil {
			return fmt.Errorf("writing goals: %w", err)
		}

		fmt.Printf("Added goal %q (type: %s, weight: %d)\n", id, goalType, goalsAddWeight)
		return nil
	},
}

func init() {
	goalsAddCmd.Flags().IntVar(&goalsAddWeight, "weight", 5, "Goal weight (1-10)")
	goalsAddCmd.Flags().StringVar(&goalsAddType, "type", "", "Goal type (health, architecture, quality, meta)")
	goalsAddCmd.Flags().StringVar(&goalsAddDescription, "description", "", "Goal description")
	goalsCmd.AddCommand(goalsAddCmd)
}
