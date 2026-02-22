# AO-Olympus Ownership Matrix

> Decision record for splitting responsibilities between AgentOps (`ao`) and Olympus (`ol`) while building durable autonomy.

**Version:** 3
**Date:** 2026-02-12

## Decision Summary

Keep AgentOps as the cognitive workflow and operator UX layer. Build durable runtime, execution safety, and scale substrate in Olympus.

- `ao` owns skill semantics, operator-facing workflows, and learning extraction UX.
- Olympus owns execution authority, context compilation, checkpoints, validation authority, and runtime safety boundaries.
- Bridge contracts are minimized to 3 surfaces only.

This preserves deterministic-first validation, safety gate placement, and clean separation of concerns.

## Scope

This matrix defines:
- authoritative vs conforming ownership for AO/Olympus interactions,
- bridge contract boundaries,
- migration and rollback rules.

This matrix does not define:
- internal Olympus implementation details beyond public spec contracts,
- product/tenant/commercial policy.

## Precedence and Conflict Resolution

### Normative Precedence

When AO and Olympus documents conflict, apply this order:

1. Olympus current-state normative specs in `docs/specs/index.md` precedence order.
2. Olympus security and validation invariants (`security.md`, `validation-authority.md`) are non-negotiable.
3. Olympus runtime contracts (`autonomy-runtime.md`) govern activation, rollback, and determinism boundaries.
4. AO skill semantics apply only within the above bounds.
5. Bridge contracts cannot weaken any higher-precedence safety/validation rule.

### Ownership Rule (No Ambiguous Cells)

Each capability row must declare:
- one **Authoritative System** (defines behavior/contract), and
- one **Conforming System** (adapts/consumes within that behavior).

"Bridge" is not a team and is never accountable for implementation.

## Status Semantics

This matrix separates "is the capability real" from "is the schema-enveloped bridge enforced":

- **Standalone Status** describes what exists in the authoritative system today, without requiring schema-enveloped bridge enforcement.
- **Bridge Envelope Status** describes whether the 3 bridge envelopes are implemented and enforced in code (schema-versioned parsing/validation + conformance tests), not whether any ad-hoc transport exists.

This avoids claiming bridge-dependent capabilities are "enforceable" when the envelopes are still spec-level.

## Implementation Status Legend

### Standalone Status

- `IMPLEMENTED`: available and enforceable in the authoritative system.
- `SPEC_ONLY`: defined in specs but not fully implemented end-to-end.
- `FUTURE`: out of cycle-1 scope; do not use as migration prerequisite unless explicitly called out.

### Bridge Envelope Status

- `ENVELOPE_IMPLEMENTED`: schema-versioned envelopes are produced/consumed and validated with conformance tests.
- `SPEC_ONLY`: envelope contract exists only in docs; current integration may be via raw prompts/files/exit codes.
- `N/A`: no bridge envelope is required.

## Bridge Contract Surfaces (Collapsed to 3)

All AO↔Olympus interop is limited to **three contract surfaces**:

- `INVOCATION_ENVELOPE`
- `STATE_CHECKPOINT_HANDOFF`
- `OBSERVABILITY_EVENTS`

**Important:** As of this matrix version, the envelopes are **SPEC_ONLY** until implemented in code with conformance tests.

## Olympus Role Mapping Disclaimer

The "Olympus Role Mapping" column references mt-olympus role topology (`mt-olympus.md`). These roles are **target-state directional** and may not correspond 1:1 to current-state runtime entities. Do not assume these roles exist as current runtime processes; treat them as conceptual labels mapped to current CLI/code paths until mt-olympus promotion.

## Ownership Matrix

| Capability | Authoritative System | Conforming System | Olympus Role Mapping (target) | Current Integration (Today) | Bridge Surface | Standalone Status | Bridge Envelope Status | Notes |
|---|---|---|---|---|---|---|---|---|
| Skill semantics (`/research`, `/plan`, `/crank`, `/vibe`, `/council`) | AO | Olympus | Apollo (consumer), Zeus (boundary) | Prompt injection + AO artifacts | `INVOCATION_ENVELOPE` | `IMPLEMENTED` | `SPEC_ONLY` | AO defines workflow semantics; Olympus executes within safety boundaries |
| Operator UX (CLI flags, hooks UX, reports) | AO | Olympus | Apollo (consumer) | AO CLI + hooks | - | `IMPLEMENTED` | `N/A` | AO remains operator shell |
| Quest orchestration + execution mode mutex | Olympus | AO | Zeus, Apollo | Olympus checkpoint CAS | `STATE_CHECKPOINT_HANDOFF` | `IMPLEMENTED` | `SPEC_ONLY` | Must respect existing execution-mode CAS semantics |
| Actor lifecycle scheduling (activation/passivation/fairness) | Olympus | AO | Apollo | CLI-driven waves (cycle 1) | `STATE_CHECKPOINT_HANDOFF` | `SPEC_ONLY` | `SPEC_ONLY` | Out of full fleet orchestration scope in cycle 1 |
| Durable checkpoints and run-ledger evidence | Olympus | AO | Zeus | Checkpoint + quest events | `STATE_CHECKPOINT_HANDOFF` | `IMPLEMENTED` | `SPEC_ONLY` | Runtime authority remains Olympus |
| Identity seed + **cryptographic** signed lineage | Olympus | AO | Zeus | N/A | `OBSERVABILITY_EVENTS` | `FUTURE` | `SPEC_ONLY` | Canonical specs cover attribution fields; cryptographic signing is FUTURE until specified |
| Context compiler (assembly/hash/diff) | Olympus | AO | Apollo, Delphi | `ol context build` + deterministic hashing | `INVOCATION_ENVELOPE` | `IMPLEMENTED` | `SPEC_ONLY` | AO provides inputs; Olympus owns assembly policy |
| Context inputs (learnings, skill outputs, hints) | AO | Olympus | Moirai (producer), Apollo (consumer) | `.agents/*` artifacts | `INVOCATION_ENVELOPE` | `IMPLEMENTED` | `SPEC_ONLY` | Input quality is AO concern |
| Deterministic validation authority (`ol validate stage1`) | Olympus | AO | Athena | CLI invocation + JSON output | `STATE_CHECKPOINT_HANDOFF` | `IMPLEMENTED` | `SPEC_ONLY` | Deterministic-first pipeline authority |
| Qualitative council semantics and reporting | AO | Olympus | Athena (consumer) | AO `/council` reports | `INVOCATION_ENVELOPE` | `IMPLEMENTED` | `SPEC_ONLY` | AO owns council prompt/report semantics |
| Stigmergic coordination medium (beads + git + `.agents/`) | Olympus | AO | Apollo, Demigod, Moirai | Beads + git + filesystem artifacts | `STATE_CHECKPOINT_HANDOFF` | `IMPLEMENTED` | `SPEC_ONLY` | Coordination is artifact-first, not direct messaging coupling |
| Knowledge extraction UX (`/retro`, `/post-mortem`) | AO | Olympus | Moirai | AO skills + `.agents/learnings` | `OBSERVABILITY_EVENTS` | `IMPLEMENTED` | `SPEC_ONLY` | AO produces learnings artifacts |
| Knowledge storage format (frozen interchange spec) | AO | Olympus | Moirai (consumer) | YAML frontmatter + markdown | `INVOCATION_ENVELOPE` | `IMPLEMENTED` | `SPEC_ONLY` | Governed by `docs/ol-bridge-contracts.md` |
| Knowledge compilation to tests/constraints | Olympus | AO | Athena, Chiron | `internal/constraints/*` | `STATE_CHECKPOINT_HANDOFF` | `IMPLEMENTED` | `SPEC_ONLY` | Olympus constraint engine authority |
| Knowledge injection into context bundles | Olympus | AO | Apollo, Delphi | Basic injection exists | `INVOCATION_ENVELOPE` | `IMPLEMENTED` | `SPEC_ONLY` | Deterministic provenance-preserving injection remains `SPEC_ONLY` |
| Tiered memory policy (hot/warm/cold/policy) | AO | Olympus | Zeus (policy boundary), Delphi | N/A | `INVOCATION_ENVELOPE` | `FUTURE` | `SPEC_ONLY` | No canonical Olympus memory-tier policy spec yet |
| Tiered memory implementation (stores/indexes/query engine) | Olympus | AO | Apollo, Delphi | N/A | `STATE_CHECKPOINT_HANDOFF` | `FUTURE` | `SPEC_ONLY` | Explicitly out of cycle-1 scope |
| Constraint export (`ao export-constraints`) | AO | Olympus | Moirai (producer), Athena (consumer) | N/A | `INVOCATION_ENVELOPE` | `FUTURE` | `SPEC_ONLY` | **Phase 2 prerequisite** (bead `ag-q7n` follow-up) |
| Runtime budget/time/lease enforcement | Olympus | AO | Zeus, Apollo | Partial (cycle 1 bounds) | `STATE_CHECKPOINT_HANDOFF` | `SPEC_ONLY` | `SPEC_ONLY` | AO may request intent; Olympus enforces hard limits |
| Runtime SLOs, scaling, failure recovery | Olympus | AO | Apollo, Zeus | N/A | `OBSERVABILITY_EVENTS` | `FUTURE` | `SPEC_ONLY` | Do not treat as implemented capability |
| Kill switch enforcement | Olympus | AO | Zeus, Apollo | `ol quest cancel` | `OBSERVABILITY_EVENTS` | `SPEC_ONLY` | `SPEC_ONLY` | AO advisory, Olympus authoritative enforcement |

## Knowledge Ownership Split (Explicit)

To avoid ambiguous "shared" ownership, knowledge is split into four concerns:

1. **Extraction UX**: AO authoritative (`/retro`, `/post-mortem`).
2. **Storage format**: AO authoritative frozen interchange spec (`docs/ol-bridge-contracts.md`).
3. **Compilation-to-tests/constraints**: Olympus authoritative (`internal/constraints/*`).
4. **Injection-into-context**: Olympus authoritative (context build/runtime assembly).

## Schema Versioning Mechanism

Bridge contracts use **monotonic integer** versioning.

Rules:
1. Every bridge payload includes `schema_version: <int>`.
2. If `schema_version` is absent, treat as `1`.
3. Version increments by `+1` for any contract change.
4. Changes must be additive when possible.
5. Readers must default-safe when new fields are absent.
6. Unknown fields must not fail parsing by default.
7. If `schema_version` is greater than supported, the receiver must emit an `OBSERVABILITY_EVENTS` error and fail closed (no checkpoint mutation).
8. Breaking changes require:
   - dual-read compatibility window, and
   - explicit migration gate before old-version removal.

## Migration Plan (Feature-Flagged Activation)

### Phase 1: Define Contracts and Gates

- Define (not freeze) the 3 bridge schemas.
- Add conformance tests for:
  - invocation envelope,
  - checkpoint handoff,
  - observability events.
- Add compatibility tests for schema-version forward/backward safe behavior.
- Keep runtime paths disabled by default.

### Phase 2: Controlled Activation (No Shadow Mode)

- Use feature-flagged, environment-scoped activation per `autonomy-runtime.md`.
- Do not run continuous dual-runtime shadow comparison for council outputs.
- Activate on selected quests/workloads with explicit rollback trigger documented beforehand.
- **Prerequisite:** `ao export-constraints` implemented and integrated.

### Phase 3: Cutover by Capability Class

- Promote capabilities from `SPEC_ONLY` to `IMPLEMENTED` only after conformance + operational gates pass.
- Keep AO local fallback available for non-runtime-critical paths.
- Do not promote `FUTURE` rows as cutover blockers unless explicitly ratified.

## Rollback Criteria (Concrete)

Rollback is required when any trigger is met:

1. **Metric trigger:**
   - deterministic validation pass rate regresses below agreed baseline,
   - checkpoint mutation/replay errors exceed threshold,
   - runtime safety violations breach guardrail threshold.
2. **State preserved:**
   - quest checkpoints,
   - run-ledger entries,
   - quest-event audit trail.
3. **Post-rollback mode:**
   - return to pre-activation runtime behavior (feature flags off).
4. **Max data loss window:**
   - zero committed checkpoint loss,
   - at most one in-flight activity window not yet checkpointed.

Pre-activation requirement: numeric thresholds and alerting rules must be recorded in the runbook before turning flags on.

## Kill Switch Composition Model

Kill-switch composition is explicit:

1. **AO kill switch is advisory (early warning).**
   - AO hooks may request stop/escalation.
2. **Olympus kill switch is authoritative (hard boundary).**
   - execute `ol quest cancel`,
   - open/escalate circuit breaker,
   - terminate active runtime processes for the cancelled quest,
   - preserve checkpoint and audit evidence.

AO cannot override Olympus hard-stop enforcement.

## Acceptance Criteria

The split is successful when:

1. AO skill semantics remain stable while Olympus runtime authority is preserved.
2. Bridge conformance tests pass for all 3 contract surfaces.
3. Standalone `IMPLEMENTED` rows are not misread as "bridge envelope implemented".
4. Deterministic-first, council-second validation ordering remains intact.
5. Kill-switch execution preserves checkpoint/audit state while halting quest execution.

## Related Docs

- `docs/ol-bridge-contracts.md`
- `docs/ARCHITECTURE.md`
- `<olympus-root>/docs/specs/index.md`
- `<olympus-root>/docs/specs/autonomy-runtime.md`
- `<olympus-root>/docs/specs/mt-olympus.md`
- `<olympus-root>/.agents/council/2026-02-12-council-ao-ownership-report.md`
