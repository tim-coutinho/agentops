package goals

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

func TestLoadGoals_V2(t *testing.T) {
	gf, err := LoadGoals(testdataPath("valid_v2.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf.Version != 2 {
		t.Errorf("version = %d, want 2", gf.Version)
	}
	if len(gf.Goals) != 2 {
		t.Fatalf("got %d goals, want 2", len(gf.Goals))
	}

	g := gf.Goals[0]
	if g.ID != "test-coverage" {
		t.Errorf("goal[0].ID = %q, want %q", g.ID, "test-coverage")
	}
	if g.Description == "" {
		t.Error("goal[0].Description is empty")
	}
	if g.Check == "" {
		t.Error("goal[0].Check is empty")
	}
	if g.Weight != 5 {
		t.Errorf("goal[0].Weight = %d, want 5", g.Weight)
	}
	// v2 goals should default Type to "health"
	if g.Type != GoalTypeHealth {
		t.Errorf("goal[0].Type = %q, want %q", g.Type, GoalTypeHealth)
	}
	if gf.Goals[1].Type != GoalTypeHealth {
		t.Errorf("goal[1].Type = %q, want %q", gf.Goals[1].Type, GoalTypeHealth)
	}
}

func TestLoadGoals_V3(t *testing.T) {
	gf, err := LoadGoals(testdataPath("valid_v3.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf.Version != 3 {
		t.Errorf("version = %d, want 3", gf.Version)
	}
	if gf.Mission != "Ship reliable software" {
		t.Errorf("mission = %q, want %q", gf.Mission, "Ship reliable software")
	}
	if len(gf.Goals) != 2 {
		t.Fatalf("got %d goals, want 2", len(gf.Goals))
	}

	g := gf.Goals[0]
	if g.Type != GoalTypeHealth {
		t.Errorf("goal[0].Type = %q, want %q", g.Type, GoalTypeHealth)
	}
	if g.Continuous == nil {
		t.Fatal("goal[0].Continuous is nil")
	}
	if g.Continuous.Metric != "api_latency_p99" {
		t.Errorf("goal[0].Continuous.Metric = %q, want %q", g.Continuous.Metric, "api_latency_p99")
	}
	if g.Continuous.Threshold != 0.2 {
		t.Errorf("goal[0].Continuous.Threshold = %f, want 0.2", g.Continuous.Threshold)
	}
	if len(g.Tags) != 2 {
		t.Errorf("goal[0].Tags len = %d, want 2", len(g.Tags))
	}

	g1 := gf.Goals[1]
	if g1.Type != GoalTypeArchitecture {
		t.Errorf("goal[1].Type = %q, want %q", g1.Type, GoalTypeArchitecture)
	}
}

func TestLoadGoals_FileNotFound(t *testing.T) {
	_, err := LoadGoals(testdataPath("nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestValidateGoals_Valid(t *testing.T) {
	gf, err := LoadGoals(testdataPath("valid_v3.yaml"))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	errs := ValidateGoals(gf)
	if len(errs) != 0 {
		t.Errorf("expected 0 validation errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateGoals_MissingFields(t *testing.T) {
	gf := &GoalFile{
		Version: 2,
		Goals: []Goal{
			{}, // all fields missing
		},
	}
	errs := ValidateGoals(gf)
	// Expect errors for: id (required), description (required), check (required), weight (must be 1-10)
	fields := map[string]bool{}
	for _, e := range errs {
		fields[e.Field] = true
	}
	for _, f := range []string{"id", "description", "check", "weight"} {
		if !fields[f] {
			t.Errorf("expected validation error for field %q, not found", f)
		}
	}
}

func TestValidateGoals_DuplicateIDs(t *testing.T) {
	gf := &GoalFile{
		Version: 2,
		Goals: []Goal{
			{ID: "dup-goal", Description: "first", Check: "echo 1", Weight: 5, Type: GoalTypeHealth},
			{ID: "dup-goal", Description: "second", Check: "echo 2", Weight: 5, Type: GoalTypeHealth},
		},
	}
	errs := ValidateGoals(gf)
	found := false
	for _, e := range errs {
		if e.Field == "id" && e.Message == "duplicate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected duplicate id validation error, not found")
	}
}

func TestValidateGoals_InvalidWeight(t *testing.T) {
	cases := []struct {
		name   string
		weight int
	}{
		{"zero", 0},
		{"eleven", 11},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gf := &GoalFile{
				Version: 2,
				Goals: []Goal{
					{ID: "bad-weight", Description: "d", Check: "c", Weight: tc.weight, Type: GoalTypeHealth},
				},
			}
			errs := ValidateGoals(gf)
			found := false
			for _, e := range errs {
				if e.Field == "weight" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected weight validation error for weight=%d, not found", tc.weight)
			}
		})
	}
}
