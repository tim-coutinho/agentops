package types

import "fmt"

// Sentinel errors for MemRL policy contract validation. Using sentinels
// allows callers to match with errors.Is for reliable error handling.
var (
	// ErrSchemaVersionInvalid is returned when schema_version is less than 1.
	ErrSchemaVersionInvalid = fmt.Errorf("schema_version must be >= 1")

	// ErrTieBreakRulesEmpty is returned when tie_break_rules is empty.
	ErrTieBreakRulesEmpty = fmt.Errorf("tie_break_rules must not be empty")

	// ErrRulesEmpty is returned when rules is empty.
	ErrRulesEmpty = fmt.Errorf("rules must not be empty")

	// ErrRollbackMatrixEmpty is returned when rollback_matrix is empty.
	ErrRollbackMatrixEmpty = fmt.Errorf("rollback_matrix must not be empty")

	// ErrRuleIDEmpty is returned when a rule_id is empty.
	ErrRuleIDEmpty = fmt.Errorf("rule_id must not be empty")

	// ErrTriggerIDEmpty is returned when a rollback trigger id is empty.
	ErrTriggerIDEmpty = fmt.Errorf("rollback trigger id must not be empty")
)
