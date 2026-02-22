package ratchet

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAllSteps(t *testing.T) {
	steps := AllSteps()

	// Verify count
	if len(steps) != 7 {
		t.Fatalf("AllSteps() returned %d steps, want 7", len(steps))
	}

	// Verify order matches workflow sequence
	expected := []Step{
		StepResearch,
		StepPreMortem,
		StepPlan,
		StepImplement,
		StepCrank,
		StepVibe,
		StepPostMortem,
	}

	for i, step := range expected {
		if steps[i] != step {
			t.Errorf("AllSteps()[%d] = %q, want %q", i, steps[i], step)
		}
	}
}

func TestAllStepsReturnsNewSlice(t *testing.T) {
	// Verify AllSteps returns a new slice each time (not a shared reference)
	a := AllSteps()
	b := AllSteps()

	a[0] = "mutated"
	if b[0] == "mutated" {
		t.Error("AllSteps() should return a new slice each call, got shared reference")
	}
}

func TestParseStep(t *testing.T) {
	tests := []struct {
		name string
		input string
		want Step
	}{
		// Canonical names
		{"canonical research", "research", StepResearch},
		{"canonical pre-mortem", "pre-mortem", StepPreMortem},
		{"canonical plan", "plan", StepPlan},
		{"canonical implement", "implement", StepImplement},
		{"canonical crank", "crank", StepCrank},
		{"canonical vibe", "vibe", StepVibe},
		{"canonical post-mortem", "post-mortem", StepPostMortem},

		// Case insensitivity
		{"uppercase RESEARCH", "RESEARCH", StepResearch},
		{"mixed case Plan", "Plan", StepPlan},
		{"all caps VIBE", "VIBE", StepVibe},
		{"mixed Pre-Mortem", "Pre-Mortem", StepPreMortem},

		// Whitespace trimming
		{"leading space", " research", StepResearch},
		{"trailing space", "plan ", StepPlan},
		{"both spaces", " vibe ", StepVibe},
		{"tab whitespace", "\tcrank\t", StepCrank},

		// Aliases without hyphen
		{"premortem no hyphen", "premortem", StepPreMortem},
		{"postmortem no hyphen", "postmortem", StepPostMortem},

		// Aliases with underscore
		{"pre_mortem underscore", "pre_mortem", StepPreMortem},
		{"post_mortem underscore", "post_mortem", StepPostMortem},

		// Semantic aliases
		{"formulate alias", "formulate", StepPlan},
		{"autopilot alias", "autopilot", StepCrank},
		{"validate alias", "validate", StepVibe},
		{"review alias", "review", StepPostMortem},
		{"execute alias", "execute", StepCrank},

		// Unrecognized
		{"empty string", "", ""},
		{"unknown step", "unknown", ""},
		{"partial match", "res", ""},
		{"typo", "reserch", ""},
		{"numeric", "123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStep(tt.input)
			if got != tt.want {
				t.Errorf("ParseStep(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStepIsValid(t *testing.T) {
	tests := []struct {
		name  string
		step  Step
		valid bool
	}{
		{"research is valid", StepResearch, true},
		{"pre-mortem is valid", StepPreMortem, true},
		{"plan is valid", StepPlan, true},
		{"implement is valid", StepImplement, true},
		{"crank is valid", StepCrank, true},
		{"vibe is valid", StepVibe, true},
		{"post-mortem is valid", StepPostMortem, true},
		{"empty is invalid", Step(""), false},
		{"unknown is invalid", Step("bogus"), false},
		{"partial is invalid", Step("res"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.step.IsValid(); got != tt.valid {
				t.Errorf("Step(%q).IsValid() = %v, want %v", tt.step, got, tt.valid)
			}
		})
	}
}

func TestStepConstants(t *testing.T) {
	// Verify step constant values match expected strings
	tests := []struct {
		step Step
		want string
	}{
		{StepResearch, "research"},
		{StepPreMortem, "pre-mortem"},
		{StepPlan, "plan"},
		{StepImplement, "implement"},
		{StepCrank, "crank"},
		{StepVibe, "vibe"},
		{StepPostMortem, "post-mortem"},
	}

	for _, tt := range tests {
		if string(tt.step) != tt.want {
			t.Errorf("Step constant = %q, want %q", tt.step, tt.want)
		}
	}
}

func TestTierString(t *testing.T) {
	tests := []struct {
		tier Tier
		want string
	}{
		{TierObservation, "observation"},
		{TierLearning, "learning"},
		{TierPattern, "pattern"},
		{TierSkill, "skill"},
		{TierCore, "core"},
		{Tier(99), "unknown"},
		{Tier(-1), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.tier.String(); got != tt.want {
				t.Errorf("Tier(%d).String() = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestTierLocation(t *testing.T) {
	tests := []struct {
		tier Tier
		want string
	}{
		{TierObservation, ".agents/candidates/"},
		{TierLearning, ".agents/learnings/"},
		{TierPattern, ".agents/patterns/"},
		{TierSkill, "plugins/*/skills/"},
		{TierCore, "CLAUDE.md"},
		{Tier(99), ""},
		{Tier(-1), ""},
	}

	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			if got := tt.tier.Location(); got != tt.want {
				t.Errorf("Tier(%d).Location() = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestTierConstants(t *testing.T) {
	// Verify tier constant values
	tests := []struct {
		tier Tier
		want int
	}{
		{TierObservation, 0},
		{TierLearning, 1},
		{TierPattern, 2},
		{TierSkill, 3},
		{TierCore, 4},
	}

	for _, tt := range tests {
		if int(tt.tier) != tt.want {
			t.Errorf("Tier %s = %d, want %d", tt.tier, tt.tier, tt.want)
		}
	}
}

func TestTierOrdering(t *testing.T) {
	// Verify tiers are ordered from lowest to highest quality
	if TierObservation >= TierLearning {
		t.Error("TierObservation should be less than TierLearning")
	}
	if TierLearning >= TierPattern {
		t.Error("TierLearning should be less than TierPattern")
	}
	if TierPattern >= TierSkill {
		t.Error("TierPattern should be less than TierSkill")
	}
	if TierSkill >= TierCore {
		t.Error("TierSkill should be less than TierCore")
	}
}

func TestChainEntryJSONRoundTrip(t *testing.T) {
	tier := TierPattern
	entry := ChainEntry{
		Step:       StepResearch,
		Timestamp:  time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Input:      "/path/to/input.md",
		Output:     "/path/to/output.md",
		Locked:     true,
		Skipped:    false,
		Reason:     "",
		Tier:       &tier,
		Location:   ".agents/patterns/",
		Cycle:      2,
		ParentEpic: "ag-abc",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ChainEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Step != entry.Step {
		t.Errorf("Step = %q, want %q", got.Step, entry.Step)
	}
	if !got.Timestamp.Equal(entry.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, entry.Timestamp)
	}
	if got.Input != entry.Input {
		t.Errorf("Input = %q, want %q", got.Input, entry.Input)
	}
	if got.Output != entry.Output {
		t.Errorf("Output = %q, want %q", got.Output, entry.Output)
	}
	if got.Locked != entry.Locked {
		t.Errorf("Locked = %v, want %v", got.Locked, entry.Locked)
	}
	if got.Tier == nil || *got.Tier != tier {
		t.Errorf("Tier = %v, want %v", got.Tier, &tier)
	}
	if got.Location != entry.Location {
		t.Errorf("Location = %q, want %q", got.Location, entry.Location)
	}
	if got.Cycle != entry.Cycle {
		t.Errorf("Cycle = %d, want %d", got.Cycle, entry.Cycle)
	}
	if got.ParentEpic != entry.ParentEpic {
		t.Errorf("ParentEpic = %q, want %q", got.ParentEpic, entry.ParentEpic)
	}
}

func TestChainEntrySkippedJSONFields(t *testing.T) {
	entry := ChainEntry{
		Step:      StepPlan,
		Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Output:    "skipped",
		Locked:    false,
		Skipped:   true,
		Reason:    "not needed for hotfix",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if v, ok := m["skipped"]; !ok || v != true {
		t.Error("expected 'skipped' field = true in JSON")
	}
	if v, ok := m["reason"]; !ok || v != "not needed for hotfix" {
		t.Errorf("expected 'reason' field, got %v", v)
	}
}

func TestChainEntryOmitemptyFields(t *testing.T) {
	// Fields with omitempty should not appear when zero-valued
	entry := ChainEntry{
		Step:      StepImplement,
		Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Output:    "result",
		Locked:    true,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	omitemptyFields := []string{"input", "skipped", "reason", "tier", "location", "cycle", "parent_epic"}
	for _, field := range omitemptyFields {
		if _, ok := m[field]; ok {
			t.Errorf("expected field %q to be omitted when zero-valued", field)
		}
	}

	// These fields should always be present
	requiredFields := []string{"step", "timestamp", "output", "locked"}
	for _, field := range requiredFields {
		if _, ok := m[field]; !ok {
			t.Errorf("expected field %q to be present", field)
		}
	}
}

func TestChainJSONRoundTrip(t *testing.T) {
	chain := Chain{
		ID:      "ag-test",
		Started: time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		Entries: []ChainEntry{
			{
				Step:      StepResearch,
				Timestamp: time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC),
				Output:    "/path/research.md",
				Locked:    true,
			},
			{
				Step:      StepPlan,
				Timestamp: time.Date(2026, 2, 10, 11, 0, 0, 0, time.UTC),
				Input:     "/path/research.md",
				Output:    "/path/plan.md",
				Locked:    true,
			},
		},
		EpicID: "ag-epic",
	}

	data, err := json.Marshal(chain)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Chain
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != chain.ID {
		t.Errorf("ID = %q, want %q", got.ID, chain.ID)
	}
	if !got.Started.Equal(chain.Started) {
		t.Errorf("Started = %v, want %v", got.Started, chain.Started)
	}
	if len(got.Entries) != len(chain.Entries) {
		t.Fatalf("Entries len = %d, want %d", len(got.Entries), len(chain.Entries))
	}
	if got.EpicID != chain.EpicID {
		t.Errorf("EpicID = %q, want %q", got.EpicID, chain.EpicID)
	}
}

func TestChainOmitemptyEpicID(t *testing.T) {
	chain := Chain{
		ID:      "test",
		Started: time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
		Entries: []ChainEntry{},
	}

	data, err := json.Marshal(chain)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if _, ok := m["epic_id"]; ok {
		t.Error("expected 'epic_id' to be omitted when empty")
	}
}

func TestGateResultJSONRoundTrip(t *testing.T) {
	result := GateResult{
		Step:     StepResearch,
		Passed:   true,
		Message:  "research artifact found",
		Input:    "/path/to/research.md",
		Location: ".agents/research/",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got GateResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Step != result.Step {
		t.Errorf("Step = %q, want %q", got.Step, result.Step)
	}
	if got.Passed != result.Passed {
		t.Errorf("Passed = %v, want %v", got.Passed, result.Passed)
	}
	if got.Message != result.Message {
		t.Errorf("Message = %q, want %q", got.Message, result.Message)
	}
	if got.Input != result.Input {
		t.Errorf("Input = %q, want %q", got.Input, result.Input)
	}
	if got.Location != result.Location {
		t.Errorf("Location = %q, want %q", got.Location, result.Location)
	}
}

func TestValidationResultJSONRoundTrip(t *testing.T) {
	tier := TierLearning
	expiryDate := "2026-05-10"
	result := ValidationResult{
		Step:                StepVibe,
		Valid:               false,
		Issues:              []string{"missing tests", "no coverage"},
		Warnings:            []string{"complexity high"},
		Tier:                &tier,
		Lenient:             true,
		LenientExpiryDate:   &expiryDate,
		LenientExpiringSoon: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ValidationResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Step != result.Step {
		t.Errorf("Step = %q, want %q", got.Step, result.Step)
	}
	if got.Valid != result.Valid {
		t.Errorf("Valid = %v, want %v", got.Valid, result.Valid)
	}
	if len(got.Issues) != 2 {
		t.Errorf("Issues len = %d, want 2", len(got.Issues))
	}
	if len(got.Warnings) != 1 {
		t.Errorf("Warnings len = %d, want 1", len(got.Warnings))
	}
	if got.Tier == nil || *got.Tier != tier {
		t.Errorf("Tier = %v, want %v", got.Tier, &tier)
	}
	if got.Lenient != true {
		t.Error("Lenient should be true")
	}
	if got.LenientExpiryDate == nil || *got.LenientExpiryDate != expiryDate {
		t.Errorf("LenientExpiryDate = %v, want %q", got.LenientExpiryDate, expiryDate)
	}
	if got.LenientExpiringSoon != true {
		t.Error("LenientExpiringSoon should be true")
	}
}

func TestValidationResultOmitemptyFields(t *testing.T) {
	result := ValidationResult{
		Step:  StepPlan,
		Valid: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	omitemptyFields := []string{"issues", "warnings", "tier", "lenient_expiry_date", "lenient_expiring_soon"}
	for _, field := range omitemptyFields {
		if _, ok := m[field]; ok {
			t.Errorf("expected field %q to be omitted when zero-valued", field)
		}
	}
}

func TestFindResultJSONRoundTrip(t *testing.T) {
	result := FindResult{
		Pattern: "research/*.md",
		Matches: []FindMatch{
			{Path: "/a/research/topic.md", Location: "crew", Priority: 0},
			{Path: "/b/research/topic.md", Location: "rig", Priority: 1},
		},
		Warnings: []string{"duplicate found across locations"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got FindResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Pattern != result.Pattern {
		t.Errorf("Pattern = %q, want %q", got.Pattern, result.Pattern)
	}
	if len(got.Matches) != 2 {
		t.Fatalf("Matches len = %d, want 2", len(got.Matches))
	}
	if got.Matches[0].Path != "/a/research/topic.md" {
		t.Errorf("Matches[0].Path = %q, want %q", got.Matches[0].Path, "/a/research/topic.md")
	}
	if got.Matches[0].Location != "crew" {
		t.Errorf("Matches[0].Location = %q, want %q", got.Matches[0].Location, "crew")
	}
	if got.Matches[0].Priority != 0 {
		t.Errorf("Matches[0].Priority = %d, want 0", got.Matches[0].Priority)
	}
	if got.Matches[1].Priority != 1 {
		t.Errorf("Matches[1].Priority = %d, want 1", got.Matches[1].Priority)
	}
	if len(got.Warnings) != 1 {
		t.Errorf("Warnings len = %d, want 1", len(got.Warnings))
	}
}

func TestFindMatchJSONFields(t *testing.T) {
	match := FindMatch{
		Path:     "/test/path.md",
		Location: "town",
		Priority: 2,
	}

	data, err := json.Marshal(match)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if v, ok := m["path"]; !ok || v != "/test/path.md" {
		t.Errorf("path field = %v", v)
	}
	if v, ok := m["location"]; !ok || v != "town" {
		t.Errorf("location field = %v", v)
	}
	if v, ok := m["priority"]; !ok || int(v.(float64)) != 2 {
		t.Errorf("priority field = %v", v)
	}
}

func TestValidateOptionsDefaults(t *testing.T) {
	opts := ValidateOptions{}

	if opts.Lenient != false {
		t.Error("default Lenient should be false")
	}
	if opts.LenientExpiryDate != nil {
		t.Error("default LenientExpiryDate should be nil")
	}
}

func TestParseStepAliasesCompleteness(t *testing.T) {
	t.Helper()

	// Every canonical step should parse to itself
	for _, step := range AllSteps() {
		got := ParseStep(string(step))
		if got != step {
			t.Errorf("ParseStep(%q) = %q, want %q (canonical self-lookup failed)", step, got, step)
		}
	}
}

// TestParseStepPhasedModeAliases verifies that the phased-mode canonical phase
// names are accepted as ratchet step aliases so that hooks and tools can use
// them directly without knowing the underlying ratchet step name.
func TestParseStepPhasedModeAliases(t *testing.T) {
	tests := []struct {
		alias    string
		wantStep Step
	}{
		// Phase-canonical names → ratchet steps
		{"discovery", StepResearch},
		{"validation", StepVibe},
		// Existing aliases must still work
		{"validate", StepVibe},
		{"implement", StepImplement},
		{"research", StepResearch},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			got := ParseStep(tt.alias)
			if got == "" {
				t.Errorf("ParseStep(%q) returned empty — phased-mode alias not registered", tt.alias)
				return
			}
			if got != tt.wantStep {
				t.Errorf("ParseStep(%q) = %q, want %q", tt.alias, got, tt.wantStep)
			}
		})
	}
}

func TestParseStepCaseAndWhitespaceCombined(t *testing.T) {
	// Combine case variation with whitespace
	got := ParseStep("  RESEARCH  ")
	if got != StepResearch {
		t.Errorf("ParseStep with upper+whitespace = %q, want %q", got, StepResearch)
	}

	got = ParseStep("\tPre-Mortem\t")
	if got != StepPreMortem {
		t.Errorf("ParseStep with mixed case+tabs = %q, want %q", got, StepPreMortem)
	}
}

func TestTierNilPointerInChainEntry(t *testing.T) {
	// ChainEntry with nil Tier should serialize without tier field
	entry := ChainEntry{
		Step:      StepResearch,
		Timestamp: time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Output:    "test",
		Locked:    true,
		Tier:      nil,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if _, ok := m["tier"]; ok {
		t.Error("expected 'tier' to be omitted when nil")
	}
}
