package types

import (
	"errors"
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
	for i := range 25 {
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

func TestValidateMemRLPolicyContract_AllErrors(t *testing.T) {
	valid := DefaultMemRLPolicyContract()

	t.Run("schema_version_zero", func(t *testing.T) {
		c := valid
		c.SchemaVersion = 0
		err := ValidateMemRLPolicyContract(c)
		if err == nil {
			t.Error("expected error for schema_version 0")
		}
		if !errors.Is(err, ErrSchemaVersionInvalid) {
			t.Errorf("expected ErrSchemaVersionInvalid, got %v", err)
		}
	})

	t.Run("invalid_default_mode", func(t *testing.T) {
		c := valid
		c.DefaultMode = "invalid"
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for invalid default_mode")
		}
	})

	t.Run("invalid_unknown_failure_class_action", func(t *testing.T) {
		c := valid
		c.UnknownFailureClassAction = "invalid"
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for invalid unknown_failure_class_action")
		}
	})

	t.Run("invalid_missing_metadata_action", func(t *testing.T) {
		c := valid
		c.MissingMetadataAction = "invalid"
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for invalid missing_metadata_action")
		}
	})

	t.Run("empty_tie_break_rules", func(t *testing.T) {
		c := valid
		c.TieBreakRules = nil
		err := ValidateMemRLPolicyContract(c)
		if err == nil {
			t.Error("expected error for empty tie_break_rules")
		}
		if !errors.Is(err, ErrTieBreakRulesEmpty) {
			t.Errorf("expected ErrTieBreakRulesEmpty, got %v", err)
		}
	})

	t.Run("empty_rules", func(t *testing.T) {
		c := valid
		c.Rules = nil
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty rules")
		}
	})

	t.Run("rule_empty_id", func(t *testing.T) {
		c := valid
		c.Rules = []MemRLPolicyRule{{
			RuleID:        "",
			Mode:          MemRLModeObserve,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionRetry,
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty rule_id")
		}
	})

	t.Run("rule_invalid_mode", func(t *testing.T) {
		c := valid
		c.Rules = []MemRLPolicyRule{{
			RuleID:        "test",
			Mode:          "bad",
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionRetry,
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for invalid rule mode")
		}
	})

	t.Run("rule_invalid_action", func(t *testing.T) {
		c := valid
		c.Rules = []MemRLPolicyRule{{
			RuleID:        "test",
			Mode:          MemRLModeObserve,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        "bad",
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for invalid rule action")
		}
	})

	t.Run("rule_invalid_attempt_bucket", func(t *testing.T) {
		c := valid
		c.Rules = []MemRLPolicyRule{{
			RuleID:        "test",
			Mode:          MemRLModeObserve,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: "bad",
			Action:        MemRLActionRetry,
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for invalid attempt_bucket")
		}
	})

	t.Run("rule_unknown_failure_class", func(t *testing.T) {
		c := valid
		c.Rules = []MemRLPolicyRule{{
			RuleID:        "test",
			Mode:          MemRLModeObserve,
			FailureClass:  "nonexistent_class",
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionRetry,
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for unknown failure_class")
		}
	})

	t.Run("empty_rollback_matrix", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = nil
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty rollback_matrix")
		}
	})

	t.Run("rollback_empty_trigger_id", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = []MemRLRollbackTrigger{{TriggerID: ""}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty trigger_id")
		}
	})

	t.Run("rollback_empty_metric", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = []MemRLRollbackTrigger{{
			TriggerID: "test",
			Metric:    "",
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty metric")
		}
	})

	t.Run("rollback_empty_metric_source_command", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = []MemRLRollbackTrigger{{
			TriggerID:           "test",
			Metric:              "score",
			MetricSourceCommand: "",
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty metric_source_command")
		}
	})

	t.Run("rollback_empty_lookback_window", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = []MemRLRollbackTrigger{{
			TriggerID:           "test",
			Metric:              "score",
			MetricSourceCommand: "cmd",
			LookbackWindow:      "",
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty lookback_window")
		}
	})

	t.Run("rollback_empty_threshold", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = []MemRLRollbackTrigger{{
			TriggerID:           "test",
			Metric:              "score",
			MetricSourceCommand: "cmd",
			LookbackWindow:      "7d",
			MinSampleSize:       3,
			Threshold:           "",
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty threshold")
		}
	})

	t.Run("rollback_empty_operator_action", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = []MemRLRollbackTrigger{{
			TriggerID:           "test",
			Metric:              "score",
			MetricSourceCommand: "cmd",
			LookbackWindow:      "7d",
			MinSampleSize:       3,
			Threshold:           ">0.8",
			OperatorAction:      "",
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty operator_action")
		}
	})

	t.Run("rollback_empty_verification_command", func(t *testing.T) {
		c := valid
		c.RollbackMatrix = []MemRLRollbackTrigger{{
			TriggerID:           "test",
			Metric:              "score",
			MetricSourceCommand: "cmd",
			LookbackWindow:      "7d",
			MinSampleSize:       3,
			Threshold:           ">0.8",
			OperatorAction:      "alert",
			VerificationCommand: "",
		}}
		if err := ValidateMemRLPolicyContract(c); err == nil {
			t.Error("expected error for empty verification_command")
		}
	})
}

func TestBucketMemRLAttempt_AllPaths(t *testing.T) {
	tests := []struct {
		name        string
		attempt     int
		maxAttempts int
		want        MemRLAttemptBucket
	}{
		{"zero max attempts", 1, 0, MemRLAttemptBucketOverflow},
		{"negative max attempts", 1, -1, MemRLAttemptBucketOverflow},
		{"attempt 0 initial", 0, 3, MemRLAttemptBucketInitial},
		{"attempt 1 initial", 1, 3, MemRLAttemptBucketInitial},
		{"attempt 2 middle", 2, 3, MemRLAttemptBucketMiddle},
		{"attempt equals max final", 3, 3, MemRLAttemptBucketFinal},
		{"attempt exceeds max overflow", 4, 3, MemRLAttemptBucketOverflow},
		{"max 1, attempt 1 initial", 1, 1, MemRLAttemptBucketInitial},
		{"max 2, attempt 1 initial", 1, 2, MemRLAttemptBucketInitial},
		{"max 2, attempt 2 final", 2, 2, MemRLAttemptBucketFinal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BucketMemRLAttempt(tt.attempt, tt.maxAttempts)
			if got != tt.want {
				t.Errorf("BucketMemRLAttempt(%d, %d) = %q, want %q", tt.attempt, tt.maxAttempts, got, tt.want)
			}
		})
	}
}

func TestEvaluateMemRLPolicy_InvalidMode(t *testing.T) {
	contract := DefaultMemRLPolicyContract()
	input := MemRLPolicyInput{
		Mode:            MemRLMode("invalid_mode"),
		FailureClass:    MemRLFailureClassVibeFail,
		AttemptBucket:   MemRLAttemptBucketInitial,
		MetadataPresent: true,
	}

	decision := EvaluateMemRLPolicy(contract, input)
	// Invalid mode should fall back to contract.DefaultMode
	if decision.Mode != contract.DefaultMode {
		t.Errorf("expected mode %q (default), got %q", contract.DefaultMode, decision.Mode)
	}
}

func TestEvaluateMemRLPolicy_EmptyBucketComputed(t *testing.T) {
	contract := DefaultMemRLPolicyContract()
	input := MemRLPolicyInput{
		Mode:            MemRLModeObserve,
		FailureClass:    MemRLFailureClassVibeFail,
		AttemptBucket:   "", // empty, should be computed from attempt
		Attempt:         1,
		MaxAttempts:     3,
		MetadataPresent: true,
	}

	decision := EvaluateMemRLPolicy(contract, input)
	// Bucket should be computed from attempt 1 -> initial
	if decision.AttemptBucket != MemRLAttemptBucketInitial {
		t.Errorf("expected computed bucket %q, got %q", MemRLAttemptBucketInitial, decision.AttemptBucket)
	}
}

func TestEvaluateMemRLPolicy_MetadataInferred(t *testing.T) {
	contract := DefaultMemRLPolicyContract()
	input := MemRLPolicyInput{
		Mode:            MemRLModeObserve,
		FailureClass:    MemRLFailureClassVibeFail,
		AttemptBucket:   MemRLAttemptBucketInitial,
		MetadataPresent: false, // should be inferred to true
	}

	decision := EvaluateMemRLPolicy(contract, input)
	// MetadataPresent should be inferred to true since FailureClass and bucket are set
	if !decision.MetadataPresent {
		t.Error("expected MetadataPresent to be inferred as true")
	}
}

func TestEvaluateMemRLPolicy_NoMatchingRule(t *testing.T) {
	// Create a minimal contract with no rules
	contract := DefaultMemRLPolicyContract()
	contract.Rules = nil // remove all rules

	input := MemRLPolicyInput{
		Mode:            MemRLModeObserve,
		FailureClass:    MemRLFailureClassVibeFail,
		AttemptBucket:   MemRLAttemptBucketInitial,
		MetadataPresent: true,
	}

	decision := EvaluateMemRLPolicy(contract, input)
	if decision.RuleID != "default.no_matching_rule" {
		t.Errorf("expected rule_id 'default.no_matching_rule', got %q", decision.RuleID)
	}
}

func TestEvaluateMemRLPolicy_PriorityTiebreaker(t *testing.T) {
	// Two rules with same specificity (both wildcards) but different priorities.
	// Higher priority should win.
	contract := DefaultMemRLPolicyContract()
	contract.Rules = []MemRLPolicyRule{
		{
			RuleID:        "low-priority",
			Mode:          MemRLModeObserve,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionRetry,
			Priority:      10,
		},
		{
			RuleID:        "high-priority",
			Mode:          MemRLModeObserve,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionEscalate,
			Priority:      50,
		},
	}

	input := MemRLPolicyInput{
		Mode:            MemRLModeObserve,
		FailureClass:    MemRLFailureClassVibeFail,
		AttemptBucket:   MemRLAttemptBucketInitial,
		MetadataPresent: true,
	}

	decision := EvaluateMemRLPolicy(contract, input)
	if decision.RuleID != "high-priority" {
		t.Errorf("expected high-priority rule to win, got %q", decision.RuleID)
	}
	if decision.Action != MemRLActionEscalate {
		t.Errorf("expected skip action, got %q", decision.Action)
	}
}

// --- Benchmarks ---

func BenchmarkEvaluateMemRLPolicy(b *testing.B) {
	contract := DefaultMemRLPolicyContract()
	input := MemRLPolicyInput{
		Mode:         MemRLModeEnforce,
		FailureClass: MemRLFailureClassVibeFail,
		Attempt:      2,
		MaxAttempts:  5,
	}

	b.ResetTimer()
	for range b.N {
		EvaluateMemRLPolicy(contract, input)
	}
}

func BenchmarkValidateMemRLPolicyContract(b *testing.B) {
	contract := DefaultMemRLPolicyContract()
	b.ResetTimer()
	for range b.N {
		_ = ValidateMemRLPolicyContract(contract)
	}
}

func BenchmarkBucketMemRLAttempt(b *testing.B) {
	b.ResetTimer()
	for i := range b.N {
		BucketMemRLAttempt(i%10+1, 10)
	}
}
