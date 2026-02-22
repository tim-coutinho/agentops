package types

import (
	"reflect"
	"testing"
)

func TestMemRLMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  MemRLMode
	}{
		{name: "off", input: "off", want: MemRLModeOff},
		{name: "observe", input: "observe", want: MemRLModeObserve},
		{name: "enforce", input: "enforce", want: MemRLModeEnforce},
		{name: "mixed case trimmed", input: " EnFoRcE ", want: MemRLModeEnforce},
		{name: "invalid defaults to off", input: "invalid", want: MemRLModeOff},
		{name: "empty defaults to off", input: "", want: MemRLModeOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMemRLMode(tt.input)
			if got != tt.want {
				t.Fatalf("ParseMemRLMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}

	t.Setenv(MemRLModeEnvVar, "observe")
	if got := GetMemRLMode(); got != MemRLModeObserve {
		t.Fatalf("GetMemRLMode() = %q, want %q", got, MemRLModeObserve)
	}
}

func TestMemRLPolicyContract(t *testing.T) {
	contract := DefaultMemRLPolicyContract()
	if err := ValidateMemRLPolicyContract(contract); err != nil {
		t.Fatalf("ValidateMemRLPolicyContract(default) failed: %v", err)
	}
	if contract.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", contract.SchemaVersion)
	}
	if contract.DefaultMode != MemRLModeOff {
		t.Fatalf("DefaultMode = %q, want %q", contract.DefaultMode, MemRLModeOff)
	}
}

func TestMemRLPolicyTableConformance(t *testing.T) {
	contract := DefaultMemRLPolicyContract()

	for _, rule := range contract.Rules {
		if rule.FailureClass == MemRLFailureClassAny || rule.AttemptBucket == MemRLAttemptBucketAny {
			continue
		}
		input := MemRLPolicyInput{
			Mode:            rule.Mode,
			FailureClass:    rule.FailureClass,
			AttemptBucket:   rule.AttemptBucket,
			MetadataPresent: true,
		}
		got := EvaluateMemRLPolicy(contract, input)
		if got.Action != rule.Action {
			t.Fatalf("rule %s conformance action=%q, want %q", rule.RuleID, got.Action, rule.Action)
		}
		if got.RuleID != rule.RuleID {
			t.Fatalf("rule %s conformance rule_id=%q, want %q", rule.RuleID, got.RuleID, rule.RuleID)
		}
	}
}

func TestMemRLPolicyTable(t *testing.T) {
	TestMemRLPolicyTableConformance(t)
}

func TestMemRLReplay(t *testing.T) {
	input := MemRLPolicyInput{
		Mode:            MemRLModeEnforce,
		FailureClass:    MemRLFailureClassVibeFail,
		Attempt:         2,
		MaxAttempts:     3,
		MetadataPresent: true,
	}

	first := EvaluateDefaultMemRLPolicy(input)
	for i := 0; i < 25; i++ {
		got := EvaluateDefaultMemRLPolicy(input)
		if !reflect.DeepEqual(first, got) {
			t.Fatalf("non-deterministic replay at iteration %d: first=%+v got=%+v", i, first, got)
		}
	}
}

func TestMemRLEvaluatorDeterminism(t *testing.T) {
	TestMemRLReplay(t)
}

func TestMemRLModeOffParity(t *testing.T) {
	offInputRetry := MemRLPolicyInput{
		Mode:            MemRLModeOff,
		FailureClass:    MemRLFailureClassVibeFail,
		Attempt:         1,
		MaxAttempts:     3,
		MetadataPresent: true,
	}
	if got := EvaluateDefaultMemRLPolicy(offInputRetry).Action; got != MemRLActionRetry {
		t.Fatalf("mode=off attempt=1 action=%q, want retry", got)
	}

	offInputEscalate := MemRLPolicyInput{
		Mode:            MemRLModeOff,
		FailureClass:    MemRLFailureClassVibeFail,
		Attempt:         3,
		MaxAttempts:     3,
		MetadataPresent: true,
	}
	if got := EvaluateDefaultMemRLPolicy(offInputEscalate).Action; got != MemRLActionEscalate {
		t.Fatalf("mode=off attempt=max action=%q, want escalate", got)
	}
}

func TestMemRLUnknownFailureClass(t *testing.T) {
	got := EvaluateDefaultMemRLPolicy(MemRLPolicyInput{
		Mode:            MemRLModeEnforce,
		FailureClass:    MemRLFailureClass("new_failure_class"),
		Attempt:         1,
		MaxAttempts:     3,
		MetadataPresent: true,
	})
	if got.Action != MemRLActionEscalate {
		t.Fatalf("unknown failure class action=%q, want escalate", got.Action)
	}
	if got.Reason != "unknown_failure_class" {
		t.Fatalf("unknown failure class reason=%q, want unknown_failure_class", got.Reason)
	}
}

func TestMemRLMissingMetadata(t *testing.T) {
	got := EvaluateDefaultMemRLPolicy(MemRLPolicyInput{
		Mode:            MemRLModeEnforce,
		FailureClass:    "",
		Attempt:         1,
		MaxAttempts:     3,
		MetadataPresent: false,
	})
	if got.Action != MemRLActionEscalate {
		t.Fatalf("missing metadata action=%q, want escalate", got.Action)
	}
	if got.Reason != "missing_metadata" {
		t.Fatalf("missing metadata reason=%q, want missing_metadata", got.Reason)
	}
}

func TestMemRLTieBreak(t *testing.T) {
	contract := DefaultMemRLPolicyContract()
	contract.Rules = []MemRLPolicyRule{
		{
			RuleID:        "z",
			Mode:          MemRLModeEnforce,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionRetry,
			Priority:      1,
		},
		{
			RuleID:        "a",
			Mode:          MemRLModeEnforce,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionEscalate,
			Priority:      1,
		},
	}

	got := EvaluateMemRLPolicy(contract, MemRLPolicyInput{
		Mode:            MemRLModeEnforce,
		FailureClass:    MemRLFailureClassVibeFail,
		AttemptBucket:   MemRLAttemptBucketMiddle,
		MetadataPresent: true,
	})
	if got.RuleID != "a" {
		t.Fatalf("tie-break picked rule_id=%q, want %q", got.RuleID, "a")
	}
	if got.Action != MemRLActionEscalate {
		t.Fatalf("tie-break action=%q, want escalate", got.Action)
	}
}

func TestMemRLRollbackMatrixValidation(t *testing.T) {
	contract := DefaultMemRLPolicyContract()
	if len(contract.RollbackMatrix) == 0 {
		t.Fatal("RollbackMatrix should not be empty")
	}
	if err := ValidateMemRLPolicyContract(contract); err != nil {
		t.Fatalf("default contract should validate: %v", err)
	}

	broken := contract
	broken.RollbackMatrix[0].MinSampleSize = 0
	if err := ValidateMemRLPolicyContract(broken); err == nil {
		t.Fatal("expected validation error when rollback trigger min_sample_size <= 0")
	}
}
