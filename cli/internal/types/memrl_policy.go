package types

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// MemRLMode controls policy evaluation behavior.
type MemRLMode string

const (
	// MemRLModeOff preserves legacy behavior and records no policy enforcement.
	MemRLModeOff MemRLMode = "off"

	// MemRLModeObserve evaluates policy decisions for auditability but does not enforce them.
	MemRLModeObserve MemRLMode = "observe"

	// MemRLModeEnforce evaluates and enforces policy decisions.
	MemRLModeEnforce MemRLMode = "enforce"
)

// MemRLModeEnvVar configures memrl policy mode.
const MemRLModeEnvVar = "MEMRL_MODE"

// ParseMemRLMode parses mode input and falls back to off for unknown values.
func ParseMemRLMode(raw string) MemRLMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(MemRLModeObserve):
		return MemRLModeObserve
	case string(MemRLModeEnforce):
		return MemRLModeEnforce
	default:
		return MemRLModeOff
	}
}

// GetMemRLMode returns the configured mode from env with deterministic fallback.
func GetMemRLMode() MemRLMode {
	return ParseMemRLMode(os.Getenv(MemRLModeEnvVar))
}

func isValidMemRLMode(mode MemRLMode) bool {
	return mode == MemRLModeOff || mode == MemRLModeObserve || mode == MemRLModeEnforce
}

// MemRLAction is a policy outcome.
type MemRLAction string

const (
	// MemRLActionRetry means continue retry flow.
	MemRLActionRetry MemRLAction = "retry"

	// MemRLActionEscalate means stop retry flow and escalate.
	MemRLActionEscalate MemRLAction = "escalate"
)

func isValidMemRLAction(action MemRLAction) bool {
	return action == MemRLActionRetry || action == MemRLActionEscalate
}

// MemRLFailureClass categorizes policy-relevant failures.
type MemRLFailureClass string

const (
	// MemRLFailureClassAny is a wildcard used for fallback rules.
	MemRLFailureClassAny MemRLFailureClass = "*"

	// MemRLFailureClassPreMortemFail maps to pre-mortem gate FAIL.
	MemRLFailureClassPreMortemFail MemRLFailureClass = "pre_mortem_fail"

	// MemRLFailureClassCrankBlocked maps to crank BLOCKED status.
	MemRLFailureClassCrankBlocked MemRLFailureClass = "crank_blocked"

	// MemRLFailureClassCrankPartial maps to crank PARTIAL status.
	MemRLFailureClassCrankPartial MemRLFailureClass = "crank_partial"

	// MemRLFailureClassVibeFail maps to vibe FAIL status.
	MemRLFailureClassVibeFail MemRLFailureClass = "vibe_fail"

	// MemRLFailureClassPhaseTimeout maps to timeout failures.
	MemRLFailureClassPhaseTimeout MemRLFailureClass = "phase_timeout"

	// MemRLFailureClassPhaseStall maps to stall failures.
	MemRLFailureClassPhaseStall MemRLFailureClass = "phase_stall"

	// MemRLFailureClassPhaseExitError maps to non-zero phase exit errors.
	MemRLFailureClassPhaseExitError MemRLFailureClass = "phase_exit_error"
)

var defaultMemRLFailureClasses = []MemRLFailureClass{
	MemRLFailureClassPreMortemFail,
	MemRLFailureClassCrankBlocked,
	MemRLFailureClassCrankPartial,
	MemRLFailureClassVibeFail,
	MemRLFailureClassPhaseTimeout,
	MemRLFailureClassPhaseStall,
	MemRLFailureClassPhaseExitError,
}

// IsKnownMemRLFailureClass checks whether failure class is in the canonical set.
func IsKnownMemRLFailureClass(fc MemRLFailureClass) bool {
	for _, known := range defaultMemRLFailureClasses {
		if fc == known {
			return true
		}
	}
	return false
}

// MemRLAttemptBucket groups attempts into deterministic buckets.
type MemRLAttemptBucket string

const (
	// MemRLAttemptBucketAny is a wildcard used for fallback rules.
	MemRLAttemptBucketAny MemRLAttemptBucket = "*"

	// MemRLAttemptBucketInitial is first-attempt behavior.
	MemRLAttemptBucketInitial MemRLAttemptBucket = "initial"

	// MemRLAttemptBucketMiddle is non-terminal retry behavior.
	MemRLAttemptBucketMiddle MemRLAttemptBucket = "middle"

	// MemRLAttemptBucketFinal is terminal configured retry behavior.
	MemRLAttemptBucketFinal MemRLAttemptBucket = "final"

	// MemRLAttemptBucketOverflow is attempts beyond configured max.
	MemRLAttemptBucketOverflow MemRLAttemptBucket = "overflow"
)

// BucketMemRLAttempt deterministically maps attempt counters into buckets.
func BucketMemRLAttempt(attempt, maxAttempts int) MemRLAttemptBucket {
	if maxAttempts <= 0 {
		return MemRLAttemptBucketOverflow
	}
	if attempt <= 1 {
		return MemRLAttemptBucketInitial
	}
	if attempt < maxAttempts {
		return MemRLAttemptBucketMiddle
	}
	if attempt == maxAttempts {
		return MemRLAttemptBucketFinal
	}
	return MemRLAttemptBucketOverflow
}

func isValidAttemptBucket(bucket MemRLAttemptBucket) bool {
	return bucket == MemRLAttemptBucketInitial ||
		bucket == MemRLAttemptBucketMiddle ||
		bucket == MemRLAttemptBucketFinal ||
		bucket == MemRLAttemptBucketOverflow ||
		bucket == MemRLAttemptBucketAny
}

// MemRLPolicyRule maps mode x failure class x attempt bucket to an action.
type MemRLPolicyRule struct {
	RuleID        string             `json:"rule_id"`
	Mode          MemRLMode          `json:"memrl_mode"`
	FailureClass  MemRLFailureClass  `json:"failure_class"`
	AttemptBucket MemRLAttemptBucket `json:"attempt_bucket"`
	Action        MemRLAction        `json:"action"`
	Priority      int                `json:"priority"`
}

// MemRLRollbackTrigger defines deterministic rollback guardrails.
type MemRLRollbackTrigger struct {
	TriggerID           string `json:"trigger_id"`
	Metric              string `json:"metric"`
	MetricSourceCommand string `json:"metric_source_command"`
	LookbackWindow      string `json:"lookback_window"`
	MinSampleSize       int    `json:"min_sample_size"`
	Threshold           string `json:"threshold"`
	OperatorAction      string `json:"operator_action"`
	VerificationCommand string `json:"verification_command"`
}

// MemRLPolicyContract is the canonical policy package exported for consumers.
type MemRLPolicyContract struct {
	SchemaVersion             int                    `json:"schema_version"`
	DefaultMode               MemRLMode              `json:"default_mode"`
	UnknownFailureClassAction MemRLAction            `json:"unknown_failure_class_action"`
	MissingMetadataAction     MemRLAction            `json:"missing_metadata_action"`
	TieBreakRules             []string               `json:"tie_break_rules"`
	Rules                     []MemRLPolicyRule      `json:"rules"`
	RollbackMatrix            []MemRLRollbackTrigger `json:"rollback_matrix"`
}

// MemRLPolicyInput is the evaluator input contract.
type MemRLPolicyInput struct {
	Mode            MemRLMode          `json:"memrl_mode"`
	FailureClass    MemRLFailureClass  `json:"failure_class"`
	AttemptBucket   MemRLAttemptBucket `json:"attempt_bucket"`
	Attempt         int                `json:"attempt"`
	MaxAttempts     int                `json:"max_attempts"`
	MetadataPresent bool               `json:"metadata_present"`
}

// MemRLPolicyDecision is the deterministic evaluator output.
type MemRLPolicyDecision struct {
	Mode            MemRLMode          `json:"memrl_mode"`
	FailureClass    MemRLFailureClass  `json:"failure_class"`
	AttemptBucket   MemRLAttemptBucket `json:"attempt_bucket"`
	Action          MemRLAction        `json:"action"`
	RuleID          string             `json:"rule_id"`
	Reason          string             `json:"reason"`
	MetadataPresent bool               `json:"metadata_present"`
}

// DefaultMemRLPolicyContract returns the canonical deterministic policy package.
func DefaultMemRLPolicyContract() MemRLPolicyContract {
	return MemRLPolicyContract{
		SchemaVersion:             1,
		DefaultMode:               MemRLModeOff,
		UnknownFailureClassAction: MemRLActionEscalate,
		MissingMetadataAction:     MemRLActionEscalate,
		TieBreakRules: []string{
			"specificity: exact failure_class and exact attempt_bucket before wildcard matches",
			"priority: higher numeric priority wins within same specificity",
			"rule_id: lexical ascending as final deterministic tie-break",
		},
		Rules:          buildDefaultMemRLRules(),
		RollbackMatrix: defaultMemRLRollbackMatrix(),
	}
}

func buildDefaultMemRLRules() []MemRLPolicyRule {
	modes := []MemRLMode{MemRLModeOff, MemRLModeObserve, MemRLModeEnforce}
	buckets := []MemRLAttemptBucket{
		MemRLAttemptBucketInitial,
		MemRLAttemptBucketMiddle,
		MemRLAttemptBucketFinal,
		MemRLAttemptBucketOverflow,
	}
	rules := make([]MemRLPolicyRule, 0, len(modes)*len(defaultMemRLFailureClasses)*len(buckets)+len(modes))

	for _, mode := range modes {
		for _, failureClass := range defaultMemRLFailureClasses {
			for _, bucket := range buckets {
				ruleID := fmt.Sprintf("%s.%s.%s", mode, failureClass, bucket)
				rules = append(rules, MemRLPolicyRule{
					RuleID:        ruleID,
					Mode:          mode,
					FailureClass:  failureClass,
					AttemptBucket: bucket,
					Action:        defaultActionForRule(mode, failureClass, bucket),
					Priority:      100,
				})
			}
		}
		// Per-mode wildcard fallback keeps policy closed under new bucket values.
		rules = append(rules, MemRLPolicyRule{
			RuleID:        fmt.Sprintf("%s.fallback", mode),
			Mode:          mode,
			FailureClass:  MemRLFailureClassAny,
			AttemptBucket: MemRLAttemptBucketAny,
			Action:        MemRLActionEscalate,
			Priority:      0,
		})
	}

	return rules
}

func defaultActionForRule(mode MemRLMode, failureClass MemRLFailureClass, bucket MemRLAttemptBucket) MemRLAction {
	if bucket == MemRLAttemptBucketFinal || bucket == MemRLAttemptBucketOverflow {
		return MemRLActionEscalate
	}
	if mode == MemRLModeEnforce && failureClass == MemRLFailureClassCrankBlocked {
		return MemRLActionEscalate
	}
	return MemRLActionRetry
}

func defaultMemRLRollbackMatrix() []MemRLRollbackTrigger {
	return []MemRLRollbackTrigger{
		{
			TriggerID:           "enforce_escalation_rate_high",
			Metric:              "escalation_rate",
			MetricSourceCommand: "ao rpi status --output json",
			LookbackWindow:      "24h",
			MinSampleSize:       10,
			Threshold:           "escalation_rate > 0.35",
			OperatorAction:      "set MEMRL_MODE=observe and rerun from validation",
			VerificationCommand: "MEMRL_MODE=observe ao rpi status --output json",
		},
		{
			TriggerID:           "unknown_failure_class_ratio_high",
			Metric:              "unknown_failure_class_ratio",
			MetricSourceCommand: "rg -n \"unknown_failure_class\" .agents/rpi/phase-*-result.json",
			LookbackWindow:      "48h",
			MinSampleSize:       5,
			Threshold:           "unknown_failure_class_ratio > 0.10",
			OperatorAction:      "set MEMRL_MODE=off and open corrective issue for failure-class mapping",
			VerificationCommand: "MEMRL_MODE=off ao rpi status --output json",
		},
		{
			TriggerID:           "missing_policy_metadata_detected",
			Metric:              "missing_metadata_count",
			MetricSourceCommand: "rg -n \"missing_metadata\" .agents/rpi/phased-orchestration.log",
			LookbackWindow:      "24h",
			MinSampleSize:       1,
			Threshold:           "missing_metadata_count >= 1",
			OperatorAction:      "set MEMRL_MODE=off and repair instrumentation before re-enabling",
			VerificationCommand: "rg -n \"MEMRL_MODE=off\" .agents/rpi/phased-orchestration.log",
		},
	}
}

// EvaluateDefaultMemRLPolicy evaluates against the canonical v1 policy package.
func EvaluateDefaultMemRLPolicy(input MemRLPolicyInput) MemRLPolicyDecision {
	return EvaluateMemRLPolicy(DefaultMemRLPolicyContract(), input)
}

// EvaluateMemRLPolicy deterministically resolves one policy decision.
func EvaluateMemRLPolicy(contract MemRLPolicyContract, input MemRLPolicyInput) MemRLPolicyDecision {
	mode := input.Mode
	if !isValidMemRLMode(mode) {
		mode = contract.DefaultMode
	}

	bucket := input.AttemptBucket
	if bucket == "" {
		bucket = BucketMemRLAttempt(input.Attempt, input.MaxAttempts)
	}

	metadataPresent := input.MetadataPresent
	if !metadataPresent && input.FailureClass != "" && bucket != "" {
		metadataPresent = true
	}

	baseDecision := MemRLPolicyDecision{
		Mode:            mode,
		FailureClass:    input.FailureClass,
		AttemptBucket:   bucket,
		MetadataPresent: metadataPresent,
	}

	if !metadataPresent || input.FailureClass == "" || bucket == "" {
		baseDecision.Action = contract.MissingMetadataAction
		baseDecision.RuleID = "default.missing_metadata"
		baseDecision.Reason = "missing_metadata"
		return baseDecision
	}

	if !IsKnownMemRLFailureClass(input.FailureClass) {
		baseDecision.Action = contract.UnknownFailureClassAction
		baseDecision.RuleID = "default.unknown_failure_class"
		baseDecision.Reason = "unknown_failure_class"
		return baseDecision
	}

	var candidates []MemRLPolicyRule
	for _, rule := range contract.Rules {
		if rule.Mode != mode {
			continue
		}
		if rule.FailureClass != MemRLFailureClassAny && rule.FailureClass != input.FailureClass {
			continue
		}
		if rule.AttemptBucket != MemRLAttemptBucketAny && rule.AttemptBucket != bucket {
			continue
		}
		candidates = append(candidates, rule)
	}

	if len(candidates) == 0 {
		baseDecision.Action = contract.UnknownFailureClassAction
		baseDecision.RuleID = "default.no_matching_rule"
		baseDecision.Reason = "no_matching_rule"
		return baseDecision
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		si := ruleSpecificity(candidates[i])
		sj := ruleSpecificity(candidates[j])
		if si != sj {
			return si > sj
		}
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].RuleID < candidates[j].RuleID
	})

	chosen := candidates[0]
	baseDecision.Action = chosen.Action
	baseDecision.RuleID = chosen.RuleID
	baseDecision.Reason = "rule_match_specificity_priority_rule_id"
	return baseDecision
}

func ruleSpecificity(rule MemRLPolicyRule) int {
	score := 0
	if rule.FailureClass != MemRLFailureClassAny {
		score++
	}
	if rule.AttemptBucket != MemRLAttemptBucketAny {
		score++
	}
	return score
}

// ValidateMemRLPolicyContract ensures policy and rollback matrix are complete.
func ValidateMemRLPolicyContract(contract MemRLPolicyContract) error {
	if contract.SchemaVersion < 1 {
		return fmt.Errorf("schema_version must be >= 1")
	}
	if !isValidMemRLMode(contract.DefaultMode) {
		return fmt.Errorf("invalid default_mode: %q", contract.DefaultMode)
	}
	if !isValidMemRLAction(contract.UnknownFailureClassAction) {
		return fmt.Errorf("invalid unknown_failure_class_action: %q", contract.UnknownFailureClassAction)
	}
	if !isValidMemRLAction(contract.MissingMetadataAction) {
		return fmt.Errorf("invalid missing_metadata_action: %q", contract.MissingMetadataAction)
	}
	if len(contract.TieBreakRules) == 0 {
		return fmt.Errorf("tie_break_rules must not be empty")
	}
	if len(contract.Rules) == 0 {
		return fmt.Errorf("rules must not be empty")
	}
	for _, rule := range contract.Rules {
		if err := validatePolicyRule(rule); err != nil {
			return err
		}
	}
	if len(contract.RollbackMatrix) == 0 {
		return fmt.Errorf("rollback_matrix must not be empty")
	}
	for _, trigger := range contract.RollbackMatrix {
		if err := validateRollbackTrigger(trigger); err != nil {
			return err
		}
	}
	return nil
}

// validatePolicyRule validates a single policy rule.
func validatePolicyRule(rule MemRLPolicyRule) error {
	if rule.RuleID == "" {
		return fmt.Errorf("rule_id must not be empty")
	}
	if !isValidMemRLMode(rule.Mode) {
		return fmt.Errorf("rule %s has invalid mode %q", rule.RuleID, rule.Mode)
	}
	if !isValidMemRLAction(rule.Action) {
		return fmt.Errorf("rule %s has invalid action %q", rule.RuleID, rule.Action)
	}
	if !isValidAttemptBucket(rule.AttemptBucket) {
		return fmt.Errorf("rule %s has invalid attempt_bucket %q", rule.RuleID, rule.AttemptBucket)
	}
	if rule.FailureClass != MemRLFailureClassAny && !IsKnownMemRLFailureClass(rule.FailureClass) {
		return fmt.Errorf("rule %s has unknown failure_class %q", rule.RuleID, rule.FailureClass)
	}
	return nil
}

// validateRollbackTrigger validates a single rollback trigger.
func validateRollbackTrigger(trigger MemRLRollbackTrigger) error {
	switch {
	case trigger.TriggerID == "":
		return fmt.Errorf("rollback trigger id must not be empty")
	case trigger.Metric == "":
		return fmt.Errorf("rollback trigger %s missing metric", trigger.TriggerID)
	case trigger.MetricSourceCommand == "":
		return fmt.Errorf("rollback trigger %s missing metric_source_command", trigger.TriggerID)
	case trigger.LookbackWindow == "":
		return fmt.Errorf("rollback trigger %s missing lookback_window", trigger.TriggerID)
	case trigger.MinSampleSize <= 0:
		return fmt.Errorf("rollback trigger %s min_sample_size must be > 0", trigger.TriggerID)
	case trigger.Threshold == "":
		return fmt.Errorf("rollback trigger %s missing threshold", trigger.TriggerID)
	case trigger.OperatorAction == "":
		return fmt.Errorf("rollback trigger %s missing operator_action", trigger.TriggerID)
	case trigger.VerificationCommand == "":
		return fmt.Errorf("rollback trigger %s missing verification_command", trigger.TriggerID)
	}
	return nil
}
