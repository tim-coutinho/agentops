package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boshu2/agentops/cli/internal/goals"
	"github.com/spf13/cobra"
)

var goalsMeasureGoalID string

var goalsMeasureCmd = &cobra.Command{
	Use:     "measure",
	Aliases: []string{"m"},
	Short:   "Run goal checks and produce a snapshot",
	GroupID: "measurement",
	RunE: func(cmd *cobra.Command, args []string) error {
		gf, err := goals.LoadGoals(goalsFile)
		if err != nil {
			return fmt.Errorf("loading goals: %w", err)
		}

		if errs := goals.ValidateGoals(gf); len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "validation: %s\n", e)
			}
			return fmt.Errorf("%d validation errors", len(errs))
		}

		timeout := time.Duration(goalsTimeout) * time.Second

		// Filter to single goal if --goal specified
		if goalsMeasureGoalID != "" {
			var filtered []goals.Goal
			for _, g := range gf.Goals {
				if g.ID == goalsMeasureGoalID {
					filtered = append(filtered, g)
				}
			}
			if len(filtered) == 0 {
				return fmt.Errorf("goal %q not found", goalsMeasureGoalID)
			}
			gf.Goals = filtered
		}

		snap := goals.Measure(gf, timeout)

		// Save snapshot
		snapDir := ".agents/ao/goals/baselines"
		path, err := goals.SaveSnapshot(snap, snapDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save snapshot: %v\n", err)
		} else if verbose {
			fmt.Fprintf(os.Stderr, "Snapshot saved: %s\n", path)
		}

		if goalsJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(snap)
		}

		// Table output
		fmt.Printf("%-30s %-6s %8s %6s\n", "GOAL", "RESULT", "DURATION", "WEIGHT")
		fmt.Printf("%-30s %-6s %8s %6s\n", "----", "------", "--------", "------")
		for _, m := range snap.Goals {
			id := m.GoalID
			if len(id) > 30 {
				id = id[:27] + "..."
			}
			fmt.Printf("%-30s %-6s %7.1fs %6d\n", id, m.Result, m.Duration, m.Weight)
		}
		fmt.Println()
		fmt.Printf("Score: %.1f%% (%d/%d passing, %d skipped)\n",
			snap.Summary.Score, snap.Summary.Passing, snap.Summary.Total, snap.Summary.Skipped)

		return nil
	},
}

func init() {
	goalsMeasureCmd.Flags().StringVar(&goalsMeasureGoalID, "goal", "", "Measure a single goal by ID")
	goalsCmd.AddCommand(goalsMeasureCmd)
}
