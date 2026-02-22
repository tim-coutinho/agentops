package main

import (
	"testing"
	"time"

	"github.com/boshu2/agentops/cli/internal/ratchet"
)

// Additional tests for metrics_nudge.go - buildRPIState with non-empty chains

func TestBuildRPIState_WithLockedResearchStep(t *testing.T) {
	chain := &ratchet.Chain{
		Entries: []ratchet.ChainEntry{
			{
				Step:      ratchet.StepResearch,
				Timestamp: time.Now(),
				Output:    ".agents/research/my-topic.md",
				Locked:    true,
			},
		},
	}
	state := buildRPIState(chain)
	if state.LastStep != string(ratchet.StepResearch) {
		t.Errorf("LastStep = %q, want %q", state.LastStep, string(ratchet.StepResearch))
	}
	if state.Artifact != ".agents/research/my-topic.md" {
		t.Errorf("Artifact = %q, want %q", state.Artifact, ".agents/research/my-topic.md")
	}
	if state.NextStep == "" {
		t.Error("expected non-empty NextStep after research")
	}
	if state.Skill == "" {
		t.Error("expected non-empty Skill after research")
	}
}

func TestBuildRPIState_WithMultipleSteps(t *testing.T) {
	chain := &ratchet.Chain{
		Entries: []ratchet.ChainEntry{
			{
				Step:      ratchet.StepResearch,
				Timestamp: time.Now().Add(-2 * time.Hour),
				Output:    ".agents/research/topic.md",
				Locked:    true,
			},
			{
				Step:      ratchet.StepPreMortem,
				Timestamp: time.Now().Add(-1 * time.Hour),
				Output:    ".agents/pre-mortem/analysis.md",
				Locked:    true,
			},
		},
	}
	state := buildRPIState(chain)
	// Last step should be the last locked/in_progress step in AllSteps() order
	if state.LastStep == "" {
		t.Error("expected non-empty LastStep for chain with entries")
	}
}

func TestBuildRPIState_VerbosePrintf(t *testing.T) {
	t.Log("smoke: VerbosePrintf should not panic when verbose is false")
	oldVerbose := verbose
	verbose = false
	defer func() { verbose = oldVerbose }()
	VerbosePrintf("test message %s %d", "arg", 42)
}

func TestVerbosePrintf_WhenEnabled(t *testing.T) {
	t.Log("smoke: VerbosePrintf should not panic when verbose is true")
	oldVerbose := verbose
	verbose = true
	defer func() { verbose = oldVerbose }()
	// Just call it - the test ensures the true branch is executed
	VerbosePrintf("verbose test output\n")
}
