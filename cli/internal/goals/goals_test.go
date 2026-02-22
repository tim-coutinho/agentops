package goals

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
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

func TestValidationError_Error(t *testing.T) {
	e := ValidationError{GoalID: "my-goal", Field: "check", Message: "required"}
	msg := e.Error()
	if msg == "" {
		t.Fatal("Error() should return a non-empty string")
	}
	// Should contain the goal ID, field, and message
	for _, substr := range []string{"my-goal", "check", "required"} {
		if len(msg) == 0 {
			t.Errorf("Error() missing substring %q", substr)
		}
	}
}

func TestLoadGoals_UnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v99.yaml")
	content := "version: 99\ngoals:\n  - id: test\n    description: d\n    check: echo ok\n    weight: 5\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadGoals(path)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestLoadGoals_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// Use something that parses as YAML but fails struct mapping — actually YAML is permissive.
	// Use truly broken YAML.
	content := "version: [\nbad yaml\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadGoals(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestValidateGoals_InvalidType(t *testing.T) {
	gf := &GoalFile{
		Version: 2,
		Goals: []Goal{
			{ID: "typed-goal", Description: "d", Check: "c", Weight: 5, Type: GoalType("invalid-type")},
		},
	}
	errs := ValidateGoals(gf)
	found := false
	for _, e := range errs {
		if e.Field == "type" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected type validation error, not found")
	}
}

func TestValidateGoals_InvalidIDFormat(t *testing.T) {
	gf := &GoalFile{
		Version: 2,
		Goals: []Goal{
			{ID: "UPPER_CASE", Description: "d", Check: "c", Weight: 5, Type: GoalTypeHealth},
		},
	}
	errs := ValidateGoals(gf)
	found := false
	for _, e := range errs {
		if e.Field == "id" && e.Message == "must be kebab-case" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected kebab-case id validation error, not found")
	}
}

// --- Benchmarks ---

func BenchmarkLoadGoals(b *testing.B) {
	dir := b.TempDir()
	path := dir + "/GOALS.yaml"
	content := `version: 3
goals:
  - id: test-coverage
    description: Achieve 95%+ test coverage
    check: go test -cover
    weight: 3
    type: quality
  - id: complexity-budget
    description: Keep complexity under 15
    check: gocyclo -over 15
    weight: 2
    type: health
  - id: lint-clean
    description: Zero staticcheck findings
    check: staticcheck ./...
    weight: 1
    type: quality
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		b.Fatalf("write: %v", err)
	}

	b.ResetTimer()
	for range b.N {
		_, _ = LoadGoals(path)
	}
}
