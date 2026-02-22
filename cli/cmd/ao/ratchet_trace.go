package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boshu2/agentops/cli/internal/ratchet"
)

func init() {
	traceSubCmd := &cobra.Command{
		Use:     "trace <artifact>",
		GroupID: "search",
		Short:   "Trace provenance backward",
		Long: `Trace an artifact back through the ratchet chain.

Shows the provenance chain from output to input.

Examples:
  ao ratchet trace .agents/retros/2025-01-24-topic.md
  ao ratchet trace epic:ol-0001`,
		Args: cobra.ExactArgs(1),
		RunE: runRatchetTrace,
	}
	ratchetCmd.AddCommand(traceSubCmd)
}

// runRatchetTrace traces provenance for an artifact.
func runRatchetTrace(cmd *cobra.Command, args []string) error {
	artifact := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	chain, err := ratchet.LoadChain(cwd)
	if err != nil {
		return fmt.Errorf("load chain: %w", err)
	}

	// Find all entries that reference this artifact
	type traceEntry struct {
		Step   ratchet.Step `json:"step"`
		Input  string       `json:"input"`
		Output string       `json:"output"`
		Time   string       `json:"time"`
	}

	trace := struct {
		Artifact string       `json:"artifact"`
		Chain    []traceEntry `json:"chain"`
	}{
		Artifact: artifact,
		Chain:    []traceEntry{},
	}

	// Walk backward through chain
	current := artifact
	for i := len(chain.Entries) - 1; i >= 0; i-- {
		entry := chain.Entries[i]
		if entry.Output == current || strings.HasSuffix(entry.Output, current) {
			trace.Chain = append([]traceEntry{{
				Step:   entry.Step,
				Input:  entry.Input,
				Output: entry.Output,
				Time:   entry.Timestamp.Format(time.RFC3339),
			}}, trace.Chain...)
			current = entry.Input
		}
	}

	switch GetOutput() {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(trace)

	default:
		fmt.Printf("Provenance Trace: %s\n", artifact)
		fmt.Println("=" + strings.Repeat("=", len(artifact)+18))

		if len(trace.Chain) == 0 {
			fmt.Println("No provenance chain found")
			return nil
		}

		for i, entry := range trace.Chain {
			if i > 0 {
				fmt.Println("  ↓")
			}
			fmt.Printf("%d. %s\n", i+1, entry.Step)
			if entry.Input != "" {
				fmt.Printf("   Input:  %s\n", entry.Input)
			}
			fmt.Printf("   Output: %s\n", entry.Output)
			fmt.Printf("   Time:   %s\n", entry.Time)
		}
	}

	return nil
}
