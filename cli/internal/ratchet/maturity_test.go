package ratchet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/boshu2/agentops/cli/internal/types"
)

// writeLearning is a test helper that writes a JSONL learning file.
func writeLearning(t *testing.T, dir, name string, data map[string]interface{}) string {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal learning data: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatalf("write learning file: %v", err)
	}
	return path
}

func TestCheckMaturityTransition_ProvisionalToCandidate(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]interface{}
		wantTransit  bool
		wantNew      types.Maturity
		wantReasonSS string // substring expected in reason
	}{
		{
			name: "promotes when utility and reward_count meet thresholds",
			data: map[string]interface{}{
				"id":           "learn-001",
				"maturity":     "provisional",
				"utility":      0.8,
				"reward_count": 4.0,
			},
			wantTransit: true,
			wantNew:     types.MaturityCandidate,
		},
		{
			name: "promotes at exact thresholds",
			data: map[string]interface{}{
				"id":           "learn-002",
				"maturity":     "provisional",
				"utility":      0.7,
				"reward_count": 3.0,
			},
			wantTransit: true,
			wantNew:     types.MaturityCandidate,
		},
		{
			name: "no promotion when utility too low",
			data: map[string]interface{}{
				"id":           "learn-003",
				"maturity":     "provisional",
				"utility":      0.5,
				"reward_count": 5.0,
			},
			wantTransit:  false,
			wantNew:      types.MaturityProvisional,
			wantReasonSS: "not enough positive feedback",
		},
		{
			name: "no promotion when reward_count too low",
			data: map[string]interface{}{
				"id":           "learn-004",
				"maturity":     "provisional",
				"utility":      0.9,
				"reward_count": 2.0,
			},
			wantTransit:  false,
			wantNew:      types.MaturityProvisional,
			wantReasonSS: "not enough positive feedback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeLearning(t, dir, "test.jsonl", tt.data)

			result, err := CheckMaturityTransition(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Transitioned != tt.wantTransit {
				t.Errorf("Transitioned = %v, want %v", result.Transitioned, tt.wantTransit)
			}
			if result.NewMaturity != tt.wantNew {
				t.Errorf("NewMaturity = %q, want %q", result.NewMaturity, tt.wantNew)
			}
			if tt.wantReasonSS != "" && result.Reason == "" {
				t.Errorf("expected reason containing %q, got empty", tt.wantReasonSS)
			}
		})
	}
}

func TestCheckMaturityTransition_CandidateToEstablished(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		wantTransit bool
		wantNew     types.Maturity
	}{
		{
			name: "promotes when all conditions met",
			data: map[string]interface{}{
				"maturity":      "candidate",
				"utility":       0.8,
				"reward_count":  6.0,
				"helpful_count": 5.0,
				"harmful_count": 1.0,
			},
			wantTransit: true,
			wantNew:     types.MaturityEstablished,
		},
		{
			name: "no promotion when helpful not greater than harmful",
			data: map[string]interface{}{
				"maturity":      "candidate",
				"utility":       0.8,
				"reward_count":  6.0,
				"helpful_count": 2.0,
				"harmful_count": 2.0,
			},
			wantTransit: false,
			wantNew:     types.MaturityCandidate,
		},
		{
			name: "no promotion when reward_count less than 5",
			data: map[string]interface{}{
				"maturity":      "candidate",
				"utility":       0.8,
				"reward_count":  4.0,
				"helpful_count": 3.0,
				"harmful_count": 1.0,
			},
			wantTransit: false,
			wantNew:     types.MaturityCandidate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeLearning(t, dir, "test.jsonl", tt.data)

			result, err := CheckMaturityTransition(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Transitioned != tt.wantTransit {
				t.Errorf("Transitioned = %v, want %v", result.Transitioned, tt.wantTransit)
			}
			if result.NewMaturity != tt.wantNew {
				t.Errorf("NewMaturity = %q, want %q", result.NewMaturity, tt.wantNew)
			}
		})
	}
}

func TestCheckMaturityTransition_CandidateToProvisionalDemotion(t *testing.T) {
	dir := t.TempDir()
	path := writeLearning(t, dir, "test.jsonl", map[string]interface{}{
		"maturity": "candidate",
		"utility":  0.2,
	})

	result, err := CheckMaturityTransition(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Transitioned {
		t.Error("expected transition (demotion)")
	}
	if result.NewMaturity != types.MaturityProvisional {
		t.Errorf("NewMaturity = %q, want %q", result.NewMaturity, types.MaturityProvisional)
	}
	if result.OldMaturity != types.MaturityCandidate {
		t.Errorf("OldMaturity = %q, want %q", result.OldMaturity, types.MaturityCandidate)
	}
}

func TestCheckMaturityTransition_EstablishedToCandidateDemotion(t *testing.T) {
	tests := []struct {
		name        string
		utility     float64
		wantTransit bool
		wantNew     types.Maturity
	}{
		{"demotes when utility below 0.5", 0.4, true, types.MaturityCandidate},
		{"stays established at exactly 0.5", 0.5, false, types.MaturityEstablished},
		{"stays established when utility high", 0.8, false, types.MaturityEstablished},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeLearning(t, dir, "test.jsonl", map[string]interface{}{
				"maturity": "established",
				"utility":  tt.utility,
			})

			result, err := CheckMaturityTransition(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Transitioned != tt.wantTransit {
				t.Errorf("Transitioned = %v, want %v", result.Transitioned, tt.wantTransit)
			}
			if result.NewMaturity != tt.wantNew {
				t.Errorf("NewMaturity = %q, want %q", result.NewMaturity, tt.wantNew)
			}
		})
	}
}

func TestCheckMaturityTransition_AntiPatternPriority(t *testing.T) {
	// Anti-pattern transition takes priority over all other transitions.
	tests := []struct {
		name         string
		data         map[string]interface{}
		wantTransit  bool
		wantNew      types.Maturity
	}{
		{
			name: "provisional becomes anti-pattern",
			data: map[string]interface{}{
				"maturity":      "provisional",
				"utility":       0.1,
				"harmful_count": 6.0,
			},
			wantTransit: true,
			wantNew:     types.MaturityAntiPattern,
		},
		{
			name: "candidate becomes anti-pattern",
			data: map[string]interface{}{
				"maturity":      "candidate",
				"utility":       0.2,
				"harmful_count": 5.0,
			},
			wantTransit: true,
			wantNew:     types.MaturityAntiPattern,
		},
		{
			name: "established becomes anti-pattern",
			data: map[string]interface{}{
				"maturity":      "established",
				"utility":       0.15,
				"harmful_count": 7.0,
			},
			wantTransit: true,
			wantNew:     types.MaturityAntiPattern,
		},
		{
			name: "already anti-pattern stays anti-pattern (no transition)",
			data: map[string]interface{}{
				"maturity":      "anti-pattern",
				"utility":       0.1,
				"harmful_count": 10.0,
			},
			wantTransit: false,
			wantNew:     types.MaturityAntiPattern,
		},
		{
			name: "not anti-pattern when harmful_count below threshold",
			data: map[string]interface{}{
				"maturity":      "provisional",
				"utility":       0.1,
				"harmful_count": 4.0,
			},
			wantTransit: false,
			wantNew:     types.MaturityProvisional,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeLearning(t, dir, "test.jsonl", tt.data)

			result, err := CheckMaturityTransition(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Transitioned != tt.wantTransit {
				t.Errorf("Transitioned = %v, want %v", result.Transitioned, tt.wantTransit)
			}
			if result.NewMaturity != tt.wantNew {
				t.Errorf("NewMaturity = %q, want %q", result.NewMaturity, tt.wantNew)
			}
		})
	}
}

func TestCheckMaturityTransition_AntiPatternRehabilitation(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		wantTransit bool
		wantNew     types.Maturity
	}{
		{
			name: "rehabilitates when utility high and helpful dominant",
			data: map[string]interface{}{
				"maturity":      "anti-pattern",
				"utility":       0.7,
				"helpful_count": 11.0,
				"harmful_count": 5.0,
			},
			wantTransit: true,
			wantNew:     types.MaturityProvisional,
		},
		{
			name: "no rehab when helpful not greater than 2x harmful",
			data: map[string]interface{}{
				"maturity":      "anti-pattern",
				"utility":       0.7,
				"helpful_count": 10.0,
				"harmful_count": 5.0,
			},
			wantTransit: false,
			wantNew:     types.MaturityAntiPattern,
		},
		{
			name: "no rehab when utility below 0.6",
			data: map[string]interface{}{
				"maturity":      "anti-pattern",
				"utility":       0.5,
				"helpful_count": 20.0,
				"harmful_count": 5.0,
			},
			wantTransit: false,
			wantNew:     types.MaturityAntiPattern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeLearning(t, dir, "test.jsonl", tt.data)

			result, err := CheckMaturityTransition(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Transitioned != tt.wantTransit {
				t.Errorf("Transitioned = %v, want %v", result.Transitioned, tt.wantTransit)
			}
			if result.NewMaturity != tt.wantNew {
				t.Errorf("NewMaturity = %q, want %q", result.NewMaturity, tt.wantNew)
			}
		})
	}
}

func TestCheckMaturityTransition_DefaultValues(t *testing.T) {
	// When fields are missing, defaults should be applied.
	dir := t.TempDir()
	path := writeLearning(t, dir, "test.jsonl", map[string]interface{}{
		"id": "bare-learning",
	})

	result, err := CheckMaturityTransition(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LearningID != "bare-learning" {
		t.Errorf("LearningID = %q, want %q", result.LearningID, "bare-learning")
	}
	if result.OldMaturity != types.MaturityProvisional {
		t.Errorf("OldMaturity = %q, want %q (default)", result.OldMaturity, types.MaturityProvisional)
	}
	if result.Utility != types.InitialUtility {
		t.Errorf("Utility = %f, want %f (InitialUtility)", result.Utility, types.InitialUtility)
	}
	if result.Confidence != 0.5 {
		t.Errorf("Confidence = %f, want 0.5 (default)", result.Confidence)
	}
	if result.HelpfulCount != 0 {
		t.Errorf("HelpfulCount = %d, want 0", result.HelpfulCount)
	}
	if result.HarmfulCount != 0 {
		t.Errorf("HarmfulCount = %d, want 0", result.HarmfulCount)
	}
	if result.RewardCount != 0 {
		t.Errorf("RewardCount = %d, want 0", result.RewardCount)
	}
}

func TestCheckMaturityTransition_IDFromFilename(t *testing.T) {
	// When no "id" field, LearningID should be derived from filename.
	dir := t.TempDir()
	path := writeLearning(t, dir, "my-learning.jsonl", map[string]interface{}{
		"maturity": "provisional",
	})

	result, err := CheckMaturityTransition(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LearningID != "my-learning.jsonl" {
		t.Errorf("LearningID = %q, want %q", result.LearningID, "my-learning.jsonl")
	}
}

func TestCheckMaturityTransition_Errors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr string
	}{
		{
			name: "nonexistent file",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nonexistent.jsonl")
			},
			wantErr: "read learning",
		},
		{
			name: "empty file",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "empty.jsonl")
				if err := os.WriteFile(path, []byte(""), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return path
			},
			wantErr: "empty learning file",
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "bad.jsonl")
				if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return path
			},
			wantErr: "parse learning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			_, err := CheckMaturityTransition(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("error = %q, want substring %q", got, tt.wantErr)
			}
		})
	}
}

func TestApplyMaturityTransition_WritesFile(t *testing.T) {
	dir := t.TempDir()
	path := writeLearning(t, dir, "test.jsonl", map[string]interface{}{
		"id":           "apply-test",
		"maturity":     "provisional",
		"utility":      0.8,
		"reward_count": 4.0,
	})

	result, err := ApplyMaturityTransition(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Transitioned {
		t.Fatal("expected transition")
	}
	if result.NewMaturity != types.MaturityCandidate {
		t.Errorf("NewMaturity = %q, want %q", result.NewMaturity, types.MaturityCandidate)
	}

	// Verify the file was updated.
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	var updated map[string]interface{}
	if err := json.Unmarshal(content, &updated); err != nil {
		t.Fatalf("parse updated file: %v", err)
	}
	if m, ok := updated["maturity"].(string); !ok || m != "candidate" {
		t.Errorf("file maturity = %q, want %q", m, "candidate")
	}
	if _, ok := updated["maturity_changed_at"].(string); !ok {
		t.Error("expected maturity_changed_at timestamp in updated file")
	}
	if _, ok := updated["maturity_reason"].(string); !ok {
		t.Error("expected maturity_reason in updated file")
	}
}

func TestApplyMaturityTransition_NoTransitionDoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	data := map[string]interface{}{
		"id":       "no-change",
		"maturity": "provisional",
		"utility":  0.3,
	}
	path := writeLearning(t, dir, "test.jsonl", data)

	// Get original content for comparison.
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	result, err := ApplyMaturityTransition(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Transitioned {
		t.Fatal("expected no transition")
	}

	// File should be unchanged.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(original) != string(after) {
		t.Error("file was modified despite no transition")
	}
}

func TestApplyMaturityTransition_ErrorOnMissingFile(t *testing.T) {
	_, err := ApplyMaturityTransition(filepath.Join(t.TempDir(), "missing.jsonl"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestScanForMaturityTransitions(t *testing.T) {
	dir := t.TempDir()

	// File that would transition (provisional -> candidate).
	writeLearning(t, dir, "promote.jsonl", map[string]interface{}{
		"id":           "promote",
		"maturity":     "provisional",
		"utility":      0.8,
		"reward_count": 4.0,
	})

	// File that would NOT transition (stays provisional).
	writeLearning(t, dir, "stay.jsonl", map[string]interface{}{
		"id":       "stay",
		"maturity": "provisional",
		"utility":  0.4,
	})

	// File that would transition (established -> candidate demotion).
	writeLearning(t, dir, "demote.jsonl", map[string]interface{}{
		"id":       "demote",
		"maturity": "established",
		"utility":  0.3,
	})

	// Invalid file (should be skipped).
	if err := os.WriteFile(filepath.Join(dir, "bad.jsonl"), []byte("not json"), 0644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	results, err := ScanForMaturityTransitions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Verify all returned results are transitions.
	for _, r := range results {
		if !r.Transitioned {
			t.Errorf("result for %q should have Transitioned=true", r.LearningID)
		}
	}
}

func TestScanForMaturityTransitions_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := ScanForMaturityTransitions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestGetAntiPatterns(t *testing.T) {
	dir := t.TempDir()

	writeLearning(t, dir, "anti1.jsonl", map[string]interface{}{
		"maturity": "anti-pattern",
	})
	writeLearning(t, dir, "anti2.jsonl", map[string]interface{}{
		"maturity": "anti-pattern",
	})
	writeLearning(t, dir, "prov.jsonl", map[string]interface{}{
		"maturity": "provisional",
	})
	writeLearning(t, dir, "est.jsonl", map[string]interface{}{
		"maturity": "established",
	})

	// Invalid file should be skipped.
	if err := os.WriteFile(filepath.Join(dir, "bad.jsonl"), []byte("{bad"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	results, err := GetAntiPatterns(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d anti-patterns, want 2", len(results))
	}
}

func TestGetAntiPatterns_EmptyDir(t *testing.T) {
	results, err := GetAntiPatterns(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestGetEstablishedLearnings(t *testing.T) {
	dir := t.TempDir()

	writeLearning(t, dir, "est1.jsonl", map[string]interface{}{
		"maturity": "established",
	})
	writeLearning(t, dir, "est2.jsonl", map[string]interface{}{
		"maturity": "established",
	})
	writeLearning(t, dir, "cand.jsonl", map[string]interface{}{
		"maturity": "candidate",
	})
	writeLearning(t, dir, "prov.jsonl", map[string]interface{}{
		"maturity": "provisional",
	})

	results, err := GetEstablishedLearnings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d established learnings, want 2", len(results))
	}
}

func TestGetEstablishedLearnings_EmptyDir(t *testing.T) {
	results, err := GetEstablishedLearnings(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestGetMaturityDistribution(t *testing.T) {
	dir := t.TempDir()

	writeLearning(t, dir, "p1.jsonl", map[string]interface{}{"maturity": "provisional"})
	writeLearning(t, dir, "p2.jsonl", map[string]interface{}{"maturity": "provisional"})
	writeLearning(t, dir, "c1.jsonl", map[string]interface{}{"maturity": "candidate"})
	writeLearning(t, dir, "e1.jsonl", map[string]interface{}{"maturity": "established"})
	writeLearning(t, dir, "e2.jsonl", map[string]interface{}{"maturity": "established"})
	writeLearning(t, dir, "e3.jsonl", map[string]interface{}{"maturity": "established"})
	writeLearning(t, dir, "a1.jsonl", map[string]interface{}{"maturity": "anti-pattern"})
	writeLearning(t, dir, "u1.jsonl", map[string]interface{}{"maturity": "bogus-value"})

	// Invalid JSON counts as unknown.
	if err := os.WriteFile(filepath.Join(dir, "bad.jsonl"), []byte("not json"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	dist, err := GetMaturityDistribution(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dist.Provisional != 2 {
		t.Errorf("Provisional = %d, want 2", dist.Provisional)
	}
	if dist.Candidate != 1 {
		t.Errorf("Candidate = %d, want 1", dist.Candidate)
	}
	if dist.Established != 3 {
		t.Errorf("Established = %d, want 3", dist.Established)
	}
	if dist.AntiPattern != 1 {
		t.Errorf("AntiPattern = %d, want 1", dist.AntiPattern)
	}
	if dist.Unknown != 2 {
		t.Errorf("Unknown = %d, want 2 (bogus + bad json)", dist.Unknown)
	}
	if dist.Total != 9 {
		t.Errorf("Total = %d, want 9", dist.Total)
	}
}

func TestGetMaturityDistribution_DefaultsToProvisional(t *testing.T) {
	// A learning with no maturity field should count as provisional.
	dir := t.TempDir()
	writeLearning(t, dir, "nomat.jsonl", map[string]interface{}{
		"id": "no-maturity",
	})

	dist, err := GetMaturityDistribution(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dist.Provisional != 1 {
		t.Errorf("Provisional = %d, want 1 (default for missing maturity)", dist.Provisional)
	}
	if dist.Total != 1 {
		t.Errorf("Total = %d, want 1", dist.Total)
	}
}

func TestGetMaturityDistribution_EmptyDir(t *testing.T) {
	dist, err := GetMaturityDistribution(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dist.Total != 0 {
		t.Errorf("Total = %d, want 0", dist.Total)
	}
}

func TestStringFromData_RequireNonEmptyFallback(t *testing.T) {
	data := map[string]interface{}{
		"field": "",
	}
	// requireNonEmpty=true with empty value should return default
	got := stringFromData(data, "field", "default-val", true)
	if got != "default-val" {
		t.Errorf("expected default-val for empty string with requireNonEmpty, got %q", got)
	}

	// requireNonEmpty=false with empty value should return ""
	got = stringFromData(data, "field", "default-val", false)
	if got != "" {
		t.Errorf("expected empty string with requireNonEmpty=false, got %q", got)
	}

	// Non-string value should return default
	data2 := map[string]interface{}{
		"field": 42,
	}
	got = stringFromData(data2, "field", "default-val", false)
	if got != "default-val" {
		t.Errorf("expected default-val for non-string value, got %q", got)
	}
}

func TestApplyMaturityTransition_ReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeLearning(t, dir, "readonly.jsonl", map[string]interface{}{
		"id":           "readonly-test",
		"maturity":     "provisional",
		"utility":      0.8,
		"reward_count": 4.0,
	})
	// Make file read-only so write-back fails
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	_, err := ApplyMaturityTransition(path)
	if err == nil {
		t.Error("expected error when writing to read-only learning file")
	}
}

func TestGetEstablishedLearnings_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	// Write a valid established file
	writeLearning(t, dir, "est.jsonl", map[string]interface{}{
		"maturity": "established",
	})
	// Write an unreadable file
	unreadable := filepath.Join(dir, "unreadable.jsonl")
	if err := os.WriteFile(unreadable, []byte(`{"maturity":"established"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0644) })

	results, err := GetEstablishedLearnings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only find the readable one
	if len(results) != 1 {
		t.Errorf("expected 1 established learning (skipping unreadable), got %d", len(results))
	}
}

func TestGetAntiPatterns_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	writeLearning(t, dir, "anti.jsonl", map[string]interface{}{
		"maturity": "anti-pattern",
	})
	unreadable := filepath.Join(dir, "unreadable.jsonl")
	if err := os.WriteFile(unreadable, []byte(`{"maturity":"anti-pattern"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0644) })

	results, err := GetAntiPatterns(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 anti-pattern (skipping unreadable), got %d", len(results))
	}
}

func TestGetMaturityDistribution_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	writeLearning(t, dir, "prov.jsonl", map[string]interface{}{
		"maturity": "provisional",
	})
	unreadable := filepath.Join(dir, "unreadable.jsonl")
	if err := os.WriteFile(unreadable, []byte(`{"maturity":"provisional"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0644) })

	dist, err := GetMaturityDistribution(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the readable file should be counted
	if dist.Total != 1 {
		t.Errorf("Total = %d, want 1 (skipping unreadable)", dist.Total)
	}
}

func TestGetMaturityDistribution_EmptyMaturityField(t *testing.T) {
	dir := t.TempDir()
	writeLearning(t, dir, "empty-mat.jsonl", map[string]interface{}{
		"id":       "test",
		"maturity": "",
	})

	dist, err := GetMaturityDistribution(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty maturity should default to provisional
	if dist.Provisional != 1 {
		t.Errorf("Provisional = %d, want 1 (default for empty maturity)", dist.Provisional)
	}
}

// contains is a test helper for substring matching.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
