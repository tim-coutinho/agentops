package main

import (
	"strings"
	"testing"
)

// TestBuildRetryPrompt_ContextDiscipline_RetryInstructions verifies that the retry prompt
// contains the retry-specific context discipline and phase summary instructions
// (retryContextDisciplineInstruction and retryPhaseSummaryInstruction).
func TestBuildRetryPrompt_ContextDiscipline_RetryInstructions(t *testing.T) {
	state := &phasedState{
		Goal:   "implement feature X",
		EpicID: "ep-001",
		Opts:   phasedEngineOptions{MaxRetries: 3},
	}
	retryCtx := &retryContext{
		Attempt: 1,
		Verdict: "FAIL",
		Findings: []finding{
			{Description: "test failed", Fix: "fix the test", Ref: "ref-1"},
		},
	}

	// Phase 3 has a retry template, so buildRetryPrompt will use it.
	got, err := buildRetryPrompt("", 3, state, retryCtx)
	if err != nil {
		t.Fatalf("buildRetryPrompt returned error: %v", err)
	}

	// Verify the retry context discipline instruction is present.
	if !strings.Contains(got, retryContextDisciplineInstruction) {
		t.Errorf("retry prompt does not contain retryContextDisciplineInstruction\ngot:\n%s", got)
	}

	// Verify the retry phase summary instruction is present.
	if !strings.Contains(got, retryPhaseSummaryInstruction) {
		t.Errorf("retry prompt does not contain retryPhaseSummaryInstruction\ngot:\n%s", got)
	}
}

// TestBuildRetryPrompt_ContextDiscipline_KeyPhrases verifies specific key phrases
// from the retry context discipline and phase summary instructions appear in the prompt.
func TestBuildRetryPrompt_ContextDiscipline_KeyPhrases(t *testing.T) {
	state := &phasedState{
		Goal:   "add context discipline to retry prompts",
		EpicID: "ep-002",
		Opts:   phasedEngineOptions{MaxRetries: 3},
	}
	retryCtx := &retryContext{
		Attempt:  2,
		Verdict:  "FAIL",
		Findings: []finding{},
	}

	got, err := buildRetryPrompt("", 3, state, retryCtx)
	if err != nil {
		t.Fatalf("buildRetryPrompt returned error: %v", err)
	}

	keyPhrases := []string{
		"summarize what was accomplished in prior phases",
		"Do not repeat work that already succeeded",
		"Include a brief summary of prior phase outcomes",
		"focus on the specific failure",
	}

	for _, phrase := range keyPhrases {
		if !strings.Contains(got, phrase) {
			t.Errorf("retry prompt missing key phrase %q\ngot:\n%s", phrase, got)
		}
	}
}

// TestRetryInstructionConstants verifies the retry instruction constants
// have non-empty, meaningful content.
func TestRetryInstructionConstants(t *testing.T) {
	if retryContextDisciplineInstruction == "" {
		t.Error("retryContextDisciplineInstruction must not be empty")
	}
	if retryPhaseSummaryInstruction == "" {
		t.Error("retryPhaseSummaryInstruction must not be empty")
	}

	// Discipline instruction should reference "prior phases" and "retry"
	if !strings.Contains(retryContextDisciplineInstruction, "prior phases") {
		t.Error("retryContextDisciplineInstruction should reference 'prior phases'")
	}
	if !strings.Contains(retryContextDisciplineInstruction, "retry") {
		t.Error("retryContextDisciplineInstruction should reference 'retry'")
	}

	// Phase summary instruction should reference "prior phase outcomes"
	if !strings.Contains(retryPhaseSummaryInstruction, "prior phase outcomes") {
		t.Error("retryPhaseSummaryInstruction should reference 'prior phase outcomes'")
	}
}
