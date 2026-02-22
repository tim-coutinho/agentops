package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boshu2/agentops/cli/internal/goals"
	"github.com/spf13/cobra"
)

type validateResult struct {
	Valid     bool     `json:"valid"`
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	GoalCount int      `json:"goal_count"`
	Version   int      `json:"version"`
}

var goalsValidateCmd = &cobra.Command{
	Use:     "validate",
	Aliases: []string{"v"},
	Short:   "Validate GOALS.yaml structure and wiring",
	GroupID: "measurement",
	RunE: func(cmd *cobra.Command, args []string) error {
		result := validateResult{}

		gf, err := goals.LoadGoals(goalsFile)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("load: %v", err))
			return outputValidateResult(result)
		}

		result.Version = gf.Version
		result.GoalCount = len(gf.Goals)

		// Structural validation
		if errs := goals.ValidateGoals(gf); len(errs) > 0 {
			for _, e := range errs {
				result.Errors = append(result.Errors, e.Error())
			}
		}

		// Wiring check: every check-*.sh script in scripts/ should be referenced by a goal
		scriptFiles, _ := filepath.Glob("scripts/check-*.sh")
		for _, sf := range scriptFiles {
			base := filepath.Base(sf)
			found := false
			for _, g := range gf.Goals {
				if strings.Contains(g.Check, base) {
					found = true
					break
				}
			}
			if !found {
				result.Warnings = append(result.Warnings, fmt.Sprintf("script %s not wired to any goal", base))
			}
		}

		// Wiring check: every goal's check script file should exist (if it references scripts/)
		for _, g := range gf.Goals {
			if strings.HasPrefix(g.Check, "scripts/") {
				// Extract script path (first word)
				parts := strings.Fields(g.Check)
				if len(parts) > 0 {
					if _, err := os.Stat(parts[0]); os.IsNotExist(err) {
						result.Errors = append(result.Errors, fmt.Sprintf("goal %s: script %s does not exist", g.ID, parts[0]))
					}
				}
			}
		}

		result.Valid = len(result.Errors) == 0
		return outputValidateResult(result)
	},
}

func outputValidateResult(result validateResult) error {
	if goalsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if result.Valid {
		fmt.Printf("VALID: %d goals, version %d\n", result.GoalCount, result.Version)
	} else {
		fmt.Printf("INVALID: %d errors\n", len(result.Errors))
	}

	for _, e := range result.Errors {
		fmt.Printf("  ERROR: %s\n", e)
	}
	for _, w := range result.Warnings {
		fmt.Printf("  WARN: %s\n", w)
	}

	if !result.Valid {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func init() {
	goalsCmd.AddCommand(goalsValidateCmd)
}
