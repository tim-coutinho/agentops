package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boshu2/agentops/cli/internal/goals"
	"github.com/spf13/cobra"
)

var goalsMetaCmd = &cobra.Command{
	Use:     "meta",
	Short:   "Run and report meta-goals only",
	GroupID: "management",
	RunE: func(cmd *cobra.Command, args []string) error {
		gf, err := goals.LoadGoals(goalsFile)
		if err != nil {
			return fmt.Errorf("loading goals: %w", err)
		}

		// Filter to meta-goals only.
		var metaGoals []goals.Goal
		for _, g := range gf.Goals {
			if g.Type == goals.GoalTypeMeta {
				metaGoals = append(metaGoals, g)
			}
		}

		if len(metaGoals) == 0 {
			fmt.Println("No meta-goals found (type: meta)")
			return nil
		}

		// Build a filtered GoalFile.
		metaGF := &goals.GoalFile{
			Version: gf.Version,
			Mission: gf.Mission,
			Goals:   metaGoals,
		}

		timeout := time.Duration(goalsTimeout) * time.Second
		snap := goals.Measure(metaGF, timeout)

		if goalsJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(snap)
		}

		// Table output.
		fmt.Printf("Meta-Goals: %d total\n\n", len(metaGoals))
		fmt.Printf("%-30s %-6s %8s\n", "GOAL", "RESULT", "DURATION")
		fmt.Printf("%-30s %-6s %8s\n", "----", "------", "--------")
		for _, m := range snap.Goals {
			id := m.GoalID
			if len(id) > 30 {
				id = id[:27] + "..."
			}
			fmt.Printf("%-30s %-6s %7.1fs\n", id, m.Result, m.Duration)
		}
		fmt.Println()

		if snap.Summary.Failing > 0 {
			fmt.Printf("META-HEALTH: DEGRADED (%d/%d failing)\n", snap.Summary.Failing, snap.Summary.Total)
			return fmt.Errorf("meta-goal failures detected")
		}

		fmt.Printf("META-HEALTH: OK (%d/%d passing)\n", snap.Summary.Passing, snap.Summary.Total)
		return nil
	},
}

func init() {
	goalsCmd.AddCommand(goalsMetaCmd)
}
