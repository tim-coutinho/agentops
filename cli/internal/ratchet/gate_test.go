package ratchet

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBdCLITimeout(t *testing.T) {
	// Verify the timeout constant is set correctly
	if BdCLITimeout != 5*time.Second {
		t.Errorf("expected BdCLITimeout to be 5s, got %v", BdCLITimeout)
	}

	// Verify error message is correct
	expectedMsg := "bd CLI timeout after 5s"
	if ErrBdCLITimeout.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrBdCLITimeout.Error())
	}
}

func TestGateCheckerWithMissingBd(t *testing.T) {
	// This test verifies that findEpic handles command errors gracefully.
	// When bd is not installed or not in PATH, we should get an error but not hang.
	tmpDir := t.TempDir()
	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		// NewGateChecker may fail if the directory structure is not set up,
		// which is expected for this test
		t.Skip("GateChecker requires specific directory structure")
	}

	// Call findEpic - it will fail but should not hang
	start := time.Now()
	_, err = checker.findEpic("open")
	elapsed := time.Since(start)

	// The command should return quickly (within timeout) even if bd is not found
	if elapsed > BdCLITimeout+time.Second {
		t.Errorf("findEpic took too long (%v), expected to complete within timeout", elapsed)
	}

	// We expect an error (bd not found or no epic found), but not a timeout
	// unless the command is actually hanging
	if err == ErrBdCLITimeout {
		t.Error("unexpected timeout error - bd command should fail fast if not installed")
	}
}

func TestGetRequiredInput(t *testing.T) {
	cases := []struct {
		step Step
		want string
	}{
		{StepResearch, ""},
		{StepPreMortem, ".agents/research/*.md"},
		{StepPlan, ".agents/specs/*-v2.md OR .agents/synthesis/*.md"},
		{StepImplement, "epic:<epic-id>"},
		{StepCrank, "epic:<epic-id>"},
		{StepVibe, "code changes (optional)"},
		{StepPostMortem, "closed epic (optional)"},
		{Step("unknown"), "unknown"},
	}

	for _, tc := range cases {
		got := GetRequiredInput(tc.step)
		if got != tc.want {
			t.Errorf("GetRequiredInput(%q) = %q, want %q", tc.step, got, tc.want)
		}
	}
}

func TestGetExpectedOutput(t *testing.T) {
	cases := []struct {
		step Step
		want string
	}{
		{StepResearch, ".agents/research/<topic>.md"},
		{StepPreMortem, ".agents/specs/<topic>-v2.md"},
		{StepPlan, "epic:<epic-id>"},
		{StepImplement, "issue:<issue-id> (closed)"},
		{StepCrank, "issue:<issue-id> (closed)"},
		{StepVibe, "validation report"},
		{StepPostMortem, ".agents/retros/<date>-<topic>.md"},
		{Step("unknown"), "unknown"},
	}

	for _, tc := range cases {
		got := GetExpectedOutput(tc.step)
		if got != tc.want {
			t.Errorf("GetExpectedOutput(%q) = %q, want %q", tc.step, got, tc.want)
		}
	}
}

func TestGateChecker_CheckResearch(t *testing.T) {
	tmpDir := t.TempDir()
	// Create .agents dir so the locator can initialize
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepResearch)
	if err != nil {
		t.Fatalf("Check(Research): %v", err)
	}
	if !result.Passed {
		t.Error("Research gate should always pass (chaos phase)")
	}
	if result.Step != StepResearch {
		t.Errorf("Step = %q, want %q", result.Step, StepResearch)
	}
}

func TestGateChecker_CheckPreMortem_NoArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepPreMortem)
	if err != nil {
		t.Fatalf("Check(PreMortem): %v", err)
	}
	// The locator searches upward, so it may find artifacts from the parent project.
	// We just verify the result is well-formed.
	if result.Step != StepPreMortem {
		t.Errorf("Step = %q, want %q", result.Step, StepPreMortem)
	}
}

func TestGateChecker_CheckPreMortem_WithArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".agents")
	researchDir := filepath.Join(agentsDir, "research")
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(researchDir, "topic.md"), []byte("# Research\n"), 0644); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepPreMortem)
	if err != nil {
		t.Fatalf("Check(PreMortem): %v", err)
	}
	if !result.Passed {
		t.Error("PreMortem gate should pass with research artifact")
	}
}

func TestGateChecker_CheckPlan_NoArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepPlan)
	if err != nil {
		t.Fatalf("Check(Plan): %v", err)
	}
	// The locator searches upward, so it may find artifacts from the parent project.
	// We just verify the result is well-formed.
	if result.Step != StepPlan {
		t.Errorf("Step = %q, want %q", result.Step, StepPlan)
	}
}

func TestGateChecker_CheckPlan_WithSynthesis(t *testing.T) {
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".agents")
	synthesisDir := filepath.Join(agentsDir, "synthesis")
	if err := os.MkdirAll(synthesisDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(synthesisDir, "analysis.md"), []byte("# Synthesis\n"), 0644); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepPlan)
	if err != nil {
		t.Fatalf("Check(Plan): %v", err)
	}
	if !result.Passed {
		t.Error("Plan gate should pass with synthesis artifact")
	}
}

func TestGateChecker_CheckVibe(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepVibe)
	if err != nil {
		t.Fatalf("Check(Vibe): %v", err)
	}
	if !result.Passed {
		t.Error("Vibe gate should always pass (soft gate)")
	}
}

func TestGateChecker_CheckPostMortem(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepPostMortem)
	if err != nil {
		t.Fatalf("Check(PostMortem): %v", err)
	}
	if !result.Passed {
		t.Error("PostMortem gate should pass (soft gate)")
	}
}

func TestGateChecker_CheckImplement(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(StepImplement)
	if err != nil {
		t.Fatalf("Check(Implement): %v", err)
	}
	// Should fail because bd is not available
	if result.Passed {
		t.Log("Implement gate passed (bd CLI found and returned an epic)")
	}
}

func TestGateChecker_CheckUnknownStep(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Skip("GateChecker requires specific directory structure")
	}

	result, err := checker.Check(Step("nonexistent"))
	if err != nil {
		t.Fatalf("Check(unknown): %v", err)
	}
	if result.Passed {
		t.Error("Unknown step gate should fail")
	}
}

func TestNewGateChecker_InvalidDir(t *testing.T) {
	_, err := NewGateChecker("/nonexistent/path/that/does/not/exist")
	// Should not panic, but may return an error depending on locator behavior
	_ = err
}

// restrictSearchOrder temporarily overrides SearchOrder to only crew-local search,
// preventing tests from finding artifacts in the host's ~/gt/.agents/ or parent rigs.
func restrictSearchOrder(t *testing.T) {
	t.Helper()
	orig := SearchOrder
	SearchOrder = []LocationType{LocationCrew}
	t.Cleanup(func() { SearchOrder = orig })
}

func TestGateChecker_CheckPreMortem_NoArtifact_CrewOnly(t *testing.T) {
	restrictSearchOrder(t)

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Fatalf("NewGateChecker: %v", err)
	}

	result, err := checker.Check(StepPreMortem)
	if err != nil {
		t.Fatalf("Check(PreMortem): %v", err)
	}
	if result.Passed {
		t.Error("PreMortem gate should fail with no research artifact in crew-only mode")
	}
	if result.Step != StepPreMortem {
		t.Errorf("Step = %q, want %q", result.Step, StepPreMortem)
	}
}

func TestGateChecker_CheckPlan_NoArtifact_CrewOnly(t *testing.T) {
	restrictSearchOrder(t)

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Fatalf("NewGateChecker: %v", err)
	}

	result, err := checker.Check(StepPlan)
	if err != nil {
		t.Fatalf("Check(Plan): %v", err)
	}
	if result.Passed {
		t.Error("Plan gate should fail with no synthesis/spec artifact in crew-only mode")
	}
	if result.Step != StepPlan {
		t.Errorf("Step = %q, want %q", result.Step, StepPlan)
	}
}

func TestGateChecker_CheckImplement_CrewOnly(t *testing.T) {
	restrictSearchOrder(t)

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Fatalf("NewGateChecker: %v", err)
	}

	result, err := checker.Check(StepImplement)
	if err != nil {
		t.Fatalf("Check(Implement): %v", err)
	}
	// bd CLI may or may not be installed, but implement gate should not pass
	// in a bare temp directory without any epic
	if result.Step != StepImplement {
		t.Errorf("Step = %q, want %q", result.Step, StepImplement)
	}
}

func TestGateChecker_CheckVibe_NoChanges_CrewOnly(t *testing.T) {
	restrictSearchOrder(t)

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Fatalf("NewGateChecker: %v", err)
	}

	result, err := checker.Check(StepVibe)
	if err != nil {
		t.Fatalf("Check(Vibe): %v", err)
	}
	// Vibe is a soft gate -- always passes
	if !result.Passed {
		t.Error("Vibe gate should always pass")
	}
	// In a temp dir with no git, checkGitChanges returns false,
	// so message should indicate "no code changes detected"
	if result.Message == "" {
		t.Error("expected non-empty message from vibe gate")
	}
}

func TestGateChecker_CheckPostMortem_CrewOnly(t *testing.T) {
	restrictSearchOrder(t)

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}

	checker, err := NewGateChecker(tmpDir)
	if err != nil {
		t.Fatalf("NewGateChecker: %v", err)
	}

	result, err := checker.Check(StepPostMortem)
	if err != nil {
		t.Fatalf("Check(PostMortem): %v", err)
	}
	// PostMortem is a soft gate -- always passes
	if !result.Passed {
		t.Error("PostMortem gate should always pass (soft gate)")
	}
}
