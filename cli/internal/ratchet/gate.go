package ratchet

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BdCLITimeout is the maximum duration to wait for bd CLI commands.
const BdCLITimeout = 5 * time.Second

// ErrBdCLITimeout is returned when bd CLI command times out.
var ErrBdCLITimeout = fmt.Errorf("bd CLI timeout after %s", BdCLITimeout)

// GateChecker validates that prerequisites are met before a step can proceed.
type GateChecker struct {
	locator *Locator
}

// NewGateChecker creates a new gate checker.
func NewGateChecker(startDir string) (*GateChecker, error) {
	locator, err := NewLocator(startDir)
	if err != nil {
		return nil, err
	}
	return &GateChecker{locator: locator}, nil
}

// Check validates that the gate for a step is satisfied.
func (g *GateChecker) Check(step Step) (*GateResult, error) {
	switch step {
	case StepResearch:
		return g.checkResearchGate()
	case StepPreMortem:
		return g.checkPreMortemGate()
	case StepPlan:
		return g.checkPlanGate()
	case StepImplement, StepCrank:
		return g.checkImplementGate()
	case StepVibe:
		return g.checkVibeGate()
	case StepPostMortem:
		return g.checkPostMortemGate()
	default:
		return &GateResult{
			Step:    step,
			Passed:  false,
			Message: fmt.Sprintf("Unknown step: %s", step),
		}, nil
	}
}

// checkResearchGate - No gate (chaos phase, always passes).
func (g *GateChecker) checkResearchGate() (*GateResult, error) {
	return &GateResult{
		Step:    StepResearch,
		Passed:  true,
		Message: "Research has no prerequisites (chaos phase)",
	}, nil
}

// checkPreMortemGate - Requires research artifact to exist.
func (g *GateChecker) checkPreMortemGate() (*GateResult, error) {
	// Look for any research artifact
	patterns := []string{
		"research/*.md",
		"research/**/*.md",
	}

	for _, pattern := range patterns {
		path, loc, err := g.locator.FindFirst(pattern)
		if err == nil {
			return &GateResult{
				Step:     StepPreMortem,
				Passed:   true,
				Message:  fmt.Sprintf("Research artifact found: %s", path),
				Input:    path,
				Location: string(loc),
			}, nil
		}
	}

	return &GateResult{
		Step:    StepPreMortem,
		Passed:  false,
		Message: "No research artifact found. Run /research first.",
	}, nil
}

// checkPlanGate - Requires synthesis or spec v2+ artifact.
func (g *GateChecker) checkPlanGate() (*GateResult, error) {
	// Look for synthesis or spec artifacts
	patterns := []string{
		"synthesis/*.md",
		"specs/*-v2.md",
		"specs/*-v*.md", // Any versioned spec
	}

	for _, pattern := range patterns {
		path, loc, err := g.locator.FindFirst(pattern)
		if err == nil {
			return &GateResult{
				Step:     StepPlan,
				Passed:   true,
				Message:  fmt.Sprintf("Spec/synthesis artifact found: %s", path),
				Input:    path,
				Location: string(loc),
			}, nil
		}
	}

	return &GateResult{
		Step:    StepPlan,
		Passed:  false,
		Message: "No spec or synthesis artifact found. Run /pre-mortem first.",
	}, nil
}

// checkImplementGate - Requires open or in_progress epic via bd CLI.
func (g *GateChecker) checkImplementGate() (*GateResult, error) {
	// Try to find an open epic
	epicID, err := g.findEpic("open")
	if err != nil || epicID == "" {
		// Also try in_progress
		epicID, _ = g.findEpic("in_progress")
	}

	if epicID != "" {
		return &GateResult{
			Step:     StepImplement,
			Passed:   true,
			Message:  fmt.Sprintf("Epic %s exists", epicID),
			Input:    epicID,
			Location: "beads",
		}, nil
	}

	return &GateResult{
		Step:    StepImplement,
		Passed:  false,
		Message: "No open epic found. Run /plan first.",
	}, nil
}

// checkVibeGate - Soft gate (always passes, but warns if no code).
func (g *GateChecker) checkVibeGate() (*GateResult, error) {
	// This is a soft gate - always passes
	// But we check for code changes to provide a meaningful message
	hasChanges := g.checkGitChanges()

	if hasChanges {
		return &GateResult{
			Step:    StepVibe,
			Passed:  true,
			Message: "Code changes detected, ready for validation",
		}, nil
	}

	return &GateResult{
		Step:    StepVibe,
		Passed:  true,
		Message: "Soft gate: always passes (no code changes detected)",
	}, nil
}

// checkPostMortemGate - Requires recently closed epic.
func (g *GateChecker) checkPostMortemGate() (*GateResult, error) {
	// Look for a closed epic
	epicID, err := g.findEpic("closed")
	if err == nil && epicID != "" {
		return &GateResult{
			Step:     StepPostMortem,
			Passed:   true,
			Message:  fmt.Sprintf("Closed epic %s found", epicID),
			Input:    epicID,
			Location: "beads",
		}, nil
	}

	// Also pass if there are completed entries in the chain
	// (user may be running post-mortem on informal work)
	return &GateResult{
		Step:    StepPostMortem,
		Passed:  true,
		Message: "Soft gate: always passes (no closed epic found, informal review OK)",
	}, nil
}

// findEpic uses bd CLI to find an epic with the given status.
func (g *GateChecker) findEpic(status string) (string, error) {
	// Create context with 5s timeout
	ctx, cancel := context.WithTimeout(context.Background(), BdCLITimeout)
	defer cancel()

	// Call bd list --type epic --status <status>
	cmd := exec.CommandContext(ctx, "bd", "list", "--type", "epic", "--status", status)
	out, err := cmd.Output()
	if err != nil {
		// Check if the error was due to context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", ErrBdCLITimeout
		}
		return "", err
	}

	// Parse output - bd list outputs ID as first field
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// First field is the ID
		fields := strings.Fields(line)
		if len(fields) > 0 {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("no epic found with status %s", status)
}

// checkGitChanges returns true if there are uncommitted changes.
func (g *GateChecker) checkGitChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// GetRequiredInput returns the expected input artifact for a step.
func GetRequiredInput(step Step) string {
	switch step {
	case StepResearch:
		return "" // No input required
	case StepPreMortem:
		return ".agents/research/*.md"
	case StepPlan:
		return ".agents/specs/*-v2.md OR .agents/synthesis/*.md"
	case StepImplement, StepCrank:
		return "epic:<epic-id>"
	case StepVibe:
		return "code changes (optional)"
	case StepPostMortem:
		return "closed epic (optional)"
	default:
		return "unknown"
	}
}

// GetExpectedOutput returns the expected output artifact for a step.
func GetExpectedOutput(step Step) string {
	switch step {
	case StepResearch:
		return ".agents/research/<topic>.md"
	case StepPreMortem:
		return ".agents/specs/<topic>-v2.md"
	case StepPlan:
		return "epic:<epic-id>"
	case StepImplement, StepCrank:
		return "issue:<issue-id> (closed)"
	case StepVibe:
		return "validation report"
	case StepPostMortem:
		return ".agents/retros/<date>-<topic>.md"
	default:
		return "unknown"
	}
}
