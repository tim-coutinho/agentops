# MemRL Policy Integration (AO Export for Olympus)

This document defines how Olympus should consume the AgentOps MemRL policy package without requiring Olympus code changes.

## Artifacts
- Schema: `docs/contracts/memrl-policy.schema.json`
- Example profile: `docs/contracts/memrl-policy.profile.example.json`
- Runtime source of truth: `cli/internal/types/memrl_policy.go`

## Contract
Policy decision is deterministic and maps:

`memrl_mode × failure_class × attempt_bucket -> action`

- `memrl_mode`: `off | observe | enforce`
- `failure_class`:
  - `pre_mortem_fail`
  - `crank_blocked`
  - `crank_partial`
  - `vibe_fail`
  - `phase_timeout`
  - `phase_stall`
  - `phase_exit_error`
- `attempt_bucket`: `initial | middle | final | overflow`
- `action`: `retry | escalate`

## Boundary and Default Behavior
- Unknown `failure_class`: action = `escalate` (fail-closed)
- Missing metadata: action = `escalate` (fail-closed)
- Tie-break rules:
  1. Exact `failure_class` + exact `attempt_bucket` before wildcard matches
  2. Higher `priority` before lower `priority`
  3. Lexical `rule_id` order as final deterministic tie-break
- No randomness and no wall-clock dependence in evaluator.

## Runtime Mode Semantics in AgentOps
- `off`: preserve legacy retry behavior (strict parity path)
- `observe`: evaluate and log policy decisions, but keep legacy selected action
- `enforce`: evaluate and enforce policy decision (`retry|escalate`)

## Olympus Hook Consumption
Olympus can consume the exported policy package as a static AO artifact:
1. Load `memrl-policy.profile.example.json` (or generated profile) and validate against `memrl-policy.schema.json`.
2. Map Olympus retry context to:
   - `memrl_mode`
   - `failure_class`
   - `attempt_bucket`
3. Execute the returned action:
   - `retry`: continue retry loop
   - `escalate`: stop retry and escalate

This is an AO export contract and does not change Olympus runtime ownership boundaries.

## Rollback Matrix (Mechanical Triggers)
Rollback triggers are part of the profile payload (`rollback_matrix`) and include:
- metric source command
- lookback window
- minimum sample size
- threshold
- operator action
- verification command

Use these triggers to switch mode safely (`enforce -> observe -> off`) while preserving auditability.

## Migration Notes
1. Start with `MEMRL_MODE=off` in production to guarantee strict parity.
2. Move to `MEMRL_MODE=observe` and collect policy/audit logs.
3. Promote to `MEMRL_MODE=enforce` after replay + parity + conformance tests pass.
4. If any rollback trigger fires, follow matrix operator action and verification command immediately.
