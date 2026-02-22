package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boshu2/agentops/cli/internal/goals"
	"github.com/spf13/cobra"
)

var goalsExportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"e"},
	Short:   "Export latest snapshot as JSON (for CI)",
	GroupID: "analysis",
	RunE: func(cmd *cobra.Command, args []string) error {
		snapDir := ".agents/ao/goals/baselines"

		snap, err := goals.LoadLatestSnapshot(snapDir)
		if err != nil {
			// No snapshots — measure fresh
			gf, loadErr := goals.LoadGoals(goalsFile)
			if loadErr != nil {
				return fmt.Errorf("loading goals: %w", loadErr)
			}
			timeout := time.Duration(goalsTimeout) * time.Second
			snap = goals.Measure(gf, timeout)
			if _, saveErr := goals.SaveSnapshot(snap, snapDir); saveErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save snapshot: %v\n", saveErr)
			}
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(snap)
	},
}

func init() {
	goalsCmd.AddCommand(goalsExportCmd)
}
