package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/boshu2/agentops/cli/internal/goals"
)

func init() {
	migrateCmd := &cobra.Command{
		Use:     "migrate",
		Short:   "Migrate GOALS.yaml to latest version",
		Aliases: []string{"mg"},
		GroupID: "management",
		Long: `Migrate a version 1 GOALS.yaml to version 2 format.

Adds default values for fields introduced in v2:
  - Sets version to 2
  - Adds mission field if missing
  - Sets goal type to "health" for goals without a type

The original file is backed up to GOALS.yaml.v1.bak before overwriting.

Examples:
  ao goals migrate
  ao goals migrate --file custom-goals.yaml`,
		RunE: runGoalsMigrate,
	}
	goalsCmd.AddCommand(migrateCmd)
}

func runGoalsMigrate(cmd *cobra.Command, args []string) error {
	path := goalsFile

	// Read and parse the file (LoadGoals now accepts v1)
	gf, err := goals.LoadGoals(path)
	if err != nil {
		return fmt.Errorf("load goals: %w", err)
	}

	if gf.Version >= 2 {
		fmt.Printf("%s is already version %d — no migration needed.\n", path, gf.Version)
		return nil
	}

	// Backup original
	backupPath := path + ".v1.bak"
	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read original for backup: %w", err)
	}
	if err := os.WriteFile(backupPath, original, 0o644); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	fmt.Printf("Backed up original to %s\n", backupPath)

	// Apply migration
	goals.MigrateV1ToV2(gf)

	// Write migrated file
	out, err := yaml.Marshal(gf)
	if err != nil {
		return fmt.Errorf("marshal migrated goals: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write migrated goals: %w", err)
	}

	fmt.Printf("Migrated %s from version 1 to version 2.\n", path)
	return nil
}
