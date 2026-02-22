package goals

import (
	"os"
	"testing"
	"time"
)

func TestMeasureOne_Pass(t *testing.T) {
	goal := Goal{
		ID:     "test-pass",
		Check:  "exit 0",
		Weight: 5,
	}
	m := MeasureOne(goal, 5*time.Second)
	if m.Result != "pass" {
		t.Errorf("expected pass, got %q", m.Result)
	}
	if m.GoalID != "test-pass" {
		t.Errorf("GoalID = %q, want %q", m.GoalID, "test-pass")
	}
	if m.Weight != 5 {
		t.Errorf("Weight = %d, want 5", m.Weight)
	}
	if m.Duration < 0 {
		t.Errorf("Duration should be >= 0, got %f", m.Duration)
	}
}

func TestMeasureOne_Fail(t *testing.T) {
	goal := Goal{
		ID:     "test-fail",
		Check:  "exit 1",
		Weight: 3,
	}
	m := MeasureOne(goal, 5*time.Second)
	if m.Result != "fail" {
		t.Errorf("expected fail, got %q", m.Result)
	}
}

func TestMeasureOne_Timeout(t *testing.T) {
	goal := Goal{
		ID:     "test-timeout",
		Check:  "sleep 10",
		Weight: 1,
	}
	m := MeasureOne(goal, 100*time.Millisecond)
	if m.Result != "skip" {
		t.Errorf("expected skip on timeout, got %q", m.Result)
	}
}

func TestMeasureOne_OutputTruncated(t *testing.T) {
	// Generate output > 500 chars
	longOutput := "echo '" + string(make([]byte, 600)) + "'"
	goal := Goal{
		ID:     "test-truncate",
		Check:  "python3 -c \"print('A'*600)\" 2>/dev/null || printf '%0.s A' {1..600}",
		Weight: 1,
	}
	// Use a simpler approach: just produce long output
	goal.Check = "printf '%600s' | tr ' ' 'A'"
	m := MeasureOne(goal, 5*time.Second)
	_ = longOutput
	if len(m.Output) > 500 {
		t.Errorf("output should be truncated to 500 chars, got %d", len(m.Output))
	}
}

func TestMeasureOne_ContinuousMetric_ParsesValue(t *testing.T) {
	threshold := 0.5
	goal := Goal{
		ID:     "test-continuous",
		Check:  "echo 0.75",
		Weight: 2,
		Continuous: &ContinuousMetric{
			Metric:    "my_metric",
			Threshold: threshold,
		},
	}
	m := MeasureOne(goal, 5*time.Second)
	if m.Value == nil {
		t.Fatal("expected Value to be set for continuous metric")
	}
	if *m.Value != 0.75 {
		t.Errorf("Value = %f, want 0.75", *m.Value)
	}
	if m.Threshold == nil {
		t.Fatal("expected Threshold to be set for continuous metric")
	}
	if *m.Threshold != threshold {
		t.Errorf("Threshold = %f, want %f", *m.Threshold, threshold)
	}
}

func TestMeasureOne_ContinuousMetric_NonNumericOutput(t *testing.T) {
	goal := Goal{
		ID:     "test-nonnumeric",
		Check:  "echo hello",
		Weight: 1,
		Continuous: &ContinuousMetric{
			Metric:    "my_metric",
			Threshold: 0.5,
		},
	}
	m := MeasureOne(goal, 5*time.Second)
	// Non-numeric output: Value should remain nil
	if m.Value != nil {
		t.Errorf("expected Value to be nil for non-numeric output, got %f", *m.Value)
	}
}

func TestMeasure_MetaGoalsRunFirst(t *testing.T) {
	var order []string
	// We can't inject ordering directly, but we can verify meta goals appear first
	// in the returned measurements.
	gf := &GoalFile{
		Version: 2,
		Goals: []Goal{
			{ID: "non-meta-1", Check: "exit 0", Weight: 1, Type: GoalTypeHealth},
			{ID: "meta-1", Check: "exit 0", Weight: 1, Type: GoalTypeMeta},
			{ID: "non-meta-2", Check: "exit 0", Weight: 1, Type: GoalTypeQuality},
		},
	}
	snap := Measure(gf, 5*time.Second)
	if len(snap.Goals) != 3 {
		t.Fatalf("expected 3 measurements, got %d", len(snap.Goals))
	}
	// Meta goal should come first
	if snap.Goals[0].GoalID != "meta-1" {
		t.Errorf("expected meta-1 first, got %q", snap.Goals[0].GoalID)
	}
	_ = order
}

func TestMeasure_SummaryCorrect(t *testing.T) {
	gf := &GoalFile{
		Version: 2,
		Goals: []Goal{
			{ID: "pass-1", Check: "exit 0", Weight: 5, Type: GoalTypeHealth},
			{ID: "pass-2", Check: "exit 0", Weight: 3, Type: GoalTypeHealth},
			{ID: "fail-1", Check: "exit 1", Weight: 2, Type: GoalTypeHealth},
		},
	}
	snap := Measure(gf, 5*time.Second)
	if snap.Summary.Total != 3 {
		t.Errorf("Total = %d, want 3", snap.Summary.Total)
	}
	if snap.Summary.Passing != 2 {
		t.Errorf("Passing = %d, want 2", snap.Summary.Passing)
	}
	if snap.Summary.Failing != 1 {
		t.Errorf("Failing = %d, want 1", snap.Summary.Failing)
	}
	// Weighted score: (5+3)/(5+3+2) * 100 = 80
	expectedScore := 80.0
	if snap.Summary.Score != expectedScore {
		t.Errorf("Score = %f, want %f", snap.Summary.Score, expectedScore)
	}
}

func TestMeasure_SkippedGoalsExcludedFromScore(t *testing.T) {
	gf := &GoalFile{
		Version: 2,
		Goals: []Goal{
			{ID: "pass-1", Check: "exit 0", Weight: 5, Type: GoalTypeHealth},
			{ID: "skip-1", Check: "sleep 10", Weight: 10, Type: GoalTypeHealth},
		},
	}
	snap := Measure(gf, 50*time.Millisecond)
	if snap.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", snap.Summary.Skipped)
	}
	// Score should only count passing/failing, not skipped
	// pass-1 weight=5, skip-1 excluded; score = 5/5 * 100 = 100
	if snap.Summary.Score != 100.0 {
		t.Errorf("Score = %f, want 100.0 (skipped excluded)", snap.Summary.Score)
	}
}

func TestMeasure_EmptyGoals(t *testing.T) {
	gf := &GoalFile{Version: 2, Goals: []Goal{}}
	snap := Measure(gf, 5*time.Second)
	if snap.Summary.Total != 0 {
		t.Errorf("Total = %d, want 0", snap.Summary.Total)
	}
	if snap.Summary.Score != 0 {
		t.Errorf("Score = %f, want 0 for empty goals", snap.Summary.Score)
	}
	if snap.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestGitSHA_OutsideGitRepo(t *testing.T) {
	// Exercise the gitSHA error path (line 120-122).
	// Change to a temp dir that is NOT a git repo, call gitSHA, then restore.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origDir) //nolint:errcheck // best effort restore
	}()

	sha := gitSHA()
	if sha != "" {
		t.Errorf("expected empty SHA outside git repo, got %q", sha)
	}
}
