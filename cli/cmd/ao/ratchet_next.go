package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/boshu2/agentops/cli/internal/ratchet"
)

// NextResult holds the result of ratchet next command.
type NextResult struct {
	Next         string `json:"next" yaml:"next"`
	Reason       string `json:"reason" yaml:"reason"`
	LastStep     string `json:"last_step" yaml:"last_step"`
	LastArtifact string `json:"last_artifact" yaml:"last_artifact"`
	Skill        string `json:"skill" yaml:"skill"`
	Complete     bool   `json:"complete" yaml:"complete"`
}

// stepSkillMap maps ratchet steps to their corresponding skill commands.
var stepSkillMap = map[string]string{
	"research":    "/research",
	"pre-mortem":  "/pre-mortem",
	"plan":        "/plan",
	"implement":   "/implement or /crank",
	"crank":       "/implement or /crank",
	"vibe":        "/vibe",
	"post-mortem": "/post-mortem",
}

func init() {
	ratchetNextCmd := &cobra.Command{
		Use:     "next",
		Aliases: []string{"n"},
		GroupID: "inspection",
		Short:   "Show next pending RPI step",
		Long: `Show the next pending step in the RPI workflow.

Returns structured output indicating what to do next based on the current
ratchet chain state. Returns "complete" if all steps are locked.

Examples:
  ao ratchet next
  ao ratchet next -o json
  ao ratchet next --epic ol-0001`,
		RunE: runRatchetNext,
	}
	ratchetNextCmd.Flags().StringVar(&ratchetEpicID, "epic", "", "Filter by epic ID")
	ratchetCmd.AddCommand(ratchetNextCmd)
}

// runRatchetNext determines the next pending RPI step.
func runRatchetNext(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	chain, err := ratchet.LoadChain(cwd)
	if err != nil {
		return fmt.Errorf("load chain: %w", err)
	}

	// Filter by epic if requested
	if ratchetEpicID != "" && chain.EpicID != ratchetEpicID {
		return fmt.Errorf("no chain found for epic %s", ratchetEpicID)
	}

	result := computeNextStep(chain)
	return outputNextResult(&result)
}

// computeNextStep analyzes the chain and determines the next step.
func computeNextStep(chain *ratchet.Chain) NextResult {
	allSteps := ratchet.AllSteps()

	// If chain is empty, start at the beginning
	if len(chain.Entries) == 0 {
		return NextResult{
			Next:     string(allSteps[0]),
			Reason:   "no steps completed yet",
			Skill:    stepSkillMap[string(allSteps[0])],
			Complete: false,
		}
	}

	// Find the last locked or skipped step
	var lastStep ratchet.Step
	var lastEntry *ratchet.ChainEntry

	for i := len(chain.Entries) - 1; i >= 0; i-- {
		entry := &chain.Entries[i]
		if entry.Locked || entry.Skipped {
			lastStep = entry.Step
			lastEntry = entry
			break
		}
	}

	// If no locked/skipped steps found, start at the beginning
	if lastEntry == nil {
		return NextResult{
			Next:     string(allSteps[0]),
			Reason:   "no steps locked yet",
			Skill:    stepSkillMap[string(allSteps[0])],
			Complete: false,
		}
	}

	// Find position of last step in AllSteps
	var lastStepIndex = -1
	for i, step := range allSteps {
		if step == lastStep {
			lastStepIndex = i
			break
		}
	}

	// Check if we're complete (last locked step is post-mortem)
	if lastStep == ratchet.StepPostMortem {
		return NextResult{
			Next:         "",
			Reason:       "all steps completed",
			LastStep:     string(lastStep),
			LastArtifact: lastEntry.Output,
			Skill:        "",
			Complete:     true,
		}
	}

	// Handle implement/crank special case
	// Either implement or crank satisfies that position in the chain
	// If we just completed implement, next is vibe (skip crank)
	// If we just completed crank, next is also vibe
	if lastStep == ratchet.StepImplement || lastStep == ratchet.StepCrank {
		// Next step after implement/crank is vibe
		return NextResult{
			Next:         string(ratchet.StepVibe),
			Reason:       fmt.Sprintf("%s locked", lastStep),
			LastStep:     string(lastStep),
			LastArtifact: lastEntry.Output,
			Skill:        stepSkillMap[string(ratchet.StepVibe)],
			Complete:     false,
		}
	}

	// For all other cases, next step is the one after the last locked step
	if lastStepIndex >= 0 && lastStepIndex < len(allSteps)-1 {
		nextStep := allSteps[lastStepIndex+1]

		// Skip crank if we just completed plan (go straight to implement)
		// Actually, according to AllSteps order, after plan comes implement, then crank
		// But the requirement says either implement or crank satisfies that position
		// So we should suggest implement first
		if lastStep == ratchet.StepPlan {
			nextStep = ratchet.StepImplement
		}

		return NextResult{
			Next:         string(nextStep),
			Reason:       fmt.Sprintf("%s locked", lastStep),
			LastStep:     string(lastStep),
			LastArtifact: lastEntry.Output,
			Skill:        stepSkillMap[string(nextStep)],
			Complete:     false,
		}
	}

	// Shouldn't reach here, but handle gracefully
	return NextResult{
		Next:         "",
		Reason:       "unexpected state",
		LastStep:     string(lastStep),
		LastArtifact: lastEntry.Output,
		Skill:        "",
		Complete:     false,
	}
}

// outputNextResult formats and outputs the result based on output format.
func outputNextResult(result *NextResult) error {
	switch GetOutput() {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)

	case "yaml":
		enc := yaml.NewEncoder(os.Stdout)
		return enc.Encode(result)

	default: // table
		if result.Complete {
			fmt.Println("✓ All RPI steps complete")
			if result.LastStep != "" {
				fmt.Printf("\nLast step: %s\n", result.LastStep)
			}
			if result.LastArtifact != "" {
				fmt.Printf("Last artifact: %s\n", result.LastArtifact)
			}
			return nil
		}

		fmt.Printf("Next step: %s\n", result.Next)
		fmt.Printf("Reason: %s\n", result.Reason)
		if result.Skill != "" {
			fmt.Printf("Suggested skill: %s\n", result.Skill)
		}
		if result.LastStep != "" {
			fmt.Printf("\nLast step: %s\n", result.LastStep)
		}
		if result.LastArtifact != "" {
			fmt.Printf("Last artifact: %s\n", result.LastArtifact)
		}
		return nil
	}
}
