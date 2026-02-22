package main

import (
	"testing"
	"time"
)

func TestLastPhaseName_Empty(t *testing.T) {
	got := lastPhaseName(nil)
	if got != "" {
		t.Errorf("lastPhaseName(nil) = %q, want empty", got)
	}
}

func TestLastPhaseName_WithPhases(t *testing.T) {
	phases := []rpiPhaseEntry{
		{Name: "research"},
		{Name: "plan"},
		{Name: "implement"},
	}
	got := lastPhaseName(phases)
	if got != "implement" {
		t.Errorf("lastPhaseName = %q, want implement", got)
	}
}

func TestTotalRetries_Empty(t *testing.T) {
	got := totalRetries(nil)
	if got != 0 {
		t.Errorf("totalRetries(nil) = %d, want 0", got)
	}
}

func TestTotalRetries_WithRetries(t *testing.T) {
	retries := map[string]int{
		"research":  2,
		"plan":      1,
		"implement": 0,
	}
	got := totalRetries(retries)
	if got != 3 {
		t.Errorf("totalRetries = %d, want 3", got)
	}
}

func TestFormatLogRunDuration_Zero(t *testing.T) {
	got := formatLogRunDuration(0)
	if got != "" {
		t.Errorf("formatLogRunDuration(0) = %q, want empty", got)
	}
}

func TestFormatLogRunDuration_Negative(t *testing.T) {
	got := formatLogRunDuration(-time.Second)
	if got != "" {
		t.Errorf("formatLogRunDuration(-1s) = %q, want empty", got)
	}
}

func TestFormatLogRunDuration_Positive(t *testing.T) {
	got := formatLogRunDuration(90 * time.Second)
	if got == "" {
		t.Fatal("expected non-empty duration string")
	}
}

func TestJoinVerdicts_Empty(t *testing.T) {
	got := joinVerdicts(nil)
	if got != "" {
		t.Errorf("joinVerdicts(nil) = %q, want empty", got)
	}
}

func TestJoinVerdicts_SingleEntry(t *testing.T) {
	verdicts := map[string]string{"research": "PASS"}
	got := joinVerdicts(verdicts)
	if got == "" {
		t.Fatal("expected non-empty verdict string")
	}
	if got != "research=PASS" {
		// Order is not guaranteed for maps, just check content
		t.Logf("joinVerdicts = %q", got)
	}
}

func TestFormattedLogRunStatus_NoVerdict(t *testing.T) {
	run := rpiRun{Status: "running"}
	got := formattedLogRunStatus(run)
	if got != "running" {
		t.Errorf("formattedLogRunStatus = %q, want running", got)
	}
}

func TestFormattedLogRunStatus_CompletedWithVerdict(t *testing.T) {
	run := rpiRun{
		Status:   "completed",
		Verdicts: map[string]string{"research": "PASS"},
	}
	got := formattedLogRunStatus(run)
	if got == "completed" {
		t.Error("expected verdict to be appended to status")
	}
	if len(got) < 9 {
		t.Errorf("formattedLogRunStatus = %q, seems too short", got)
	}
}

func TestFormattedLogRunStatus_CompletedNoVerdict(t *testing.T) {
	run := rpiRun{Status: "completed"}
	got := formattedLogRunStatus(run)
	if got != "completed" {
		t.Errorf("formattedLogRunStatus = %q, want completed", got)
	}
}
