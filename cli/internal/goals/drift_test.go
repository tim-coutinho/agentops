package goals

import (
	"testing"
)

func makeSnap(goals []Measurement) *Snapshot {
	return &Snapshot{Goals: goals}
}

func TestComputeDrift_Improved(t *testing.T) {
	baseline := makeSnap([]Measurement{
		{GoalID: "goal-a", Result: "fail", Weight: 5},
	})
	current := makeSnap([]Measurement{
		{GoalID: "goal-a", Result: "pass", Weight: 5},
	})
	results := ComputeDrift(baseline, current)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Delta != "improved" {
		t.Errorf("Delta = %q, want improved", results[0].Delta)
	}
	if results[0].Before != "fail" {
		t.Errorf("Before = %q, want fail", results[0].Before)
	}
	if results[0].After != "pass" {
		t.Errorf("After = %q, want pass", results[0].After)
	}
}

func TestComputeDrift_Regressed(t *testing.T) {
	baseline := makeSnap([]Measurement{
		{GoalID: "goal-b", Result: "pass", Weight: 3},
	})
	current := makeSnap([]Measurement{
		{GoalID: "goal-b", Result: "fail", Weight: 3},
	})
	results := ComputeDrift(baseline, current)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Delta != "regressed" {
		t.Errorf("Delta = %q, want regressed", results[0].Delta)
	}
}

func TestComputeDrift_Unchanged(t *testing.T) {
	baseline := makeSnap([]Measurement{
		{GoalID: "goal-c", Result: "pass", Weight: 2},
	})
	current := makeSnap([]Measurement{
		{GoalID: "goal-c", Result: "pass", Weight: 2},
	})
	results := ComputeDrift(baseline, current)
	if results[0].Delta != "unchanged" {
		t.Errorf("Delta = %q, want unchanged", results[0].Delta)
	}
}

func TestComputeDrift_NewGoal(t *testing.T) {
	baseline := makeSnap([]Measurement{})
	current := makeSnap([]Measurement{
		{GoalID: "new-goal", Result: "pass", Weight: 4},
	})
	results := ComputeDrift(baseline, current)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Before != "new" {
		t.Errorf("Before = %q, want new", results[0].Before)
	}
	if results[0].Delta != "unchanged" {
		t.Errorf("Delta = %q, want unchanged for new goal", results[0].Delta)
	}
}

func TestComputeDrift_SortOrder(t *testing.T) {
	// Regressions should come first, then improvements, then unchanged
	baseline := makeSnap([]Measurement{
		{GoalID: "unchanged-1", Result: "pass", Weight: 9},
		{GoalID: "improved-1", Result: "fail", Weight: 7},
		{GoalID: "regressed-1", Result: "pass", Weight: 5},
	})
	current := makeSnap([]Measurement{
		{GoalID: "unchanged-1", Result: "pass", Weight: 9},
		{GoalID: "improved-1", Result: "pass", Weight: 7},
		{GoalID: "regressed-1", Result: "fail", Weight: 5},
	})
	results := ComputeDrift(baseline, current)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Delta != "regressed" {
		t.Errorf("first result should be regressed, got %q", results[0].Delta)
	}
	if results[1].Delta != "improved" {
		t.Errorf("second result should be improved, got %q", results[1].Delta)
	}
	if results[2].Delta != "unchanged" {
		t.Errorf("third result should be unchanged, got %q", results[2].Delta)
	}
}

func TestComputeDrift_SortByWeightWithinCategory(t *testing.T) {
	baseline := makeSnap([]Measurement{
		{GoalID: "regressed-low", Result: "pass", Weight: 2},
		{GoalID: "regressed-high", Result: "pass", Weight: 8},
	})
	current := makeSnap([]Measurement{
		{GoalID: "regressed-low", Result: "fail", Weight: 2},
		{GoalID: "regressed-high", Result: "fail", Weight: 8},
	})
	results := ComputeDrift(baseline, current)
	if results[0].GoalID != "regressed-high" {
		t.Errorf("higher weight regression should sort first, got %q", results[0].GoalID)
	}
}

func TestComputeDrift_ValueDelta(t *testing.T) {
	baseVal := 1.0
	curVal := 3.0
	baseline := makeSnap([]Measurement{
		{GoalID: "metric-goal", Result: "pass", Weight: 1, Value: &baseVal},
	})
	current := makeSnap([]Measurement{
		{GoalID: "metric-goal", Result: "pass", Weight: 1, Value: &curVal},
	})
	results := ComputeDrift(baseline, current)
	if results[0].ValueDelta == nil {
		t.Fatal("expected ValueDelta to be set")
	}
	if *results[0].ValueDelta != 2.0 {
		t.Errorf("ValueDelta = %f, want 2.0", *results[0].ValueDelta)
	}
}

func TestComputeDrift_EmptySnapshots(t *testing.T) {
	baseline := makeSnap([]Measurement{})
	current := makeSnap([]Measurement{})
	results := ComputeDrift(baseline, current)
	if results != nil && len(results) != 0 {
		t.Errorf("expected empty results for empty snapshots, got %d", len(results))
	}
}

func TestDeltaRank(t *testing.T) {
	cases := []struct {
		delta string
		want  int
	}{
		{"regressed", 0},
		{"improved", 1},
		{"unchanged", 2},
		{"unknown", 2}, // default
		{"", 2},
	}
	for _, tc := range cases {
		got := deltaRank(tc.delta)
		if got != tc.want {
			t.Errorf("deltaRank(%q) = %d, want %d", tc.delta, got, tc.want)
		}
	}
}
