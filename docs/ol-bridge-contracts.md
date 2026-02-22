# OL-AO Bridge Contracts

> Interchange formats, exit codes, and version negotiation for the Olympus (ol) ↔ AgentOps (ao) CLI bridge.

## 0. Canonical Bridge Contract Surfaces (Normative)

AO↔Olympus interop is limited to **three contract surfaces**:

1. `INVOCATION_ENVELOPE`
2. `STATE_CHECKPOINT_HANDOFF`
3. `OBSERVABILITY_EVENTS`

This document may define multiple payload profiles, but those profiles MUST map to one of the three surfaces above. Do not add new independent bridge surfaces without updating this section.

**Enforcement status:** The schema-enveloped bridge in this section is a target contract. Until Olympus emits/parses these envelopes and conformance tests exist, envelope enforcement MUST be treated as `SPEC_ONLY`.

### 0.1 `INVOCATION_ENVELOPE`

Used when AO declares intent for Olympus execution (or vice versa), including skill dispatch and context/input metadata.

```json
{
  "schema_version": 1,
  "contract": "INVOCATION_ENVELOPE",
  "source_system": "ao",
  "target_system": "ol",
  "operation": "skill.invoke",
  "payload": {},
  "meta": {
    "timestamp": "2026-02-12T00:00:00Z",
    "correlation_id": "uuid"
  }
}
```

### 0.2 `STATE_CHECKPOINT_HANDOFF`

Used for runtime phase/checkpoint/result handoff and deterministic state transitions.

```json
{
  "schema_version": 1,
  "contract": "STATE_CHECKPOINT_HANDOFF",
  "source_system": "ol",
  "target_system": "ao",
  "operation": "checkpoint.update",
  "payload": {
    "quest_id": "ol-572",
    "phase": "VALIDATE",
    "status": "PASS"
  },
  "meta": {
    "timestamp": "2026-02-12T00:00:00Z",
    "correlation_id": "uuid"
  }
}
```

### 0.3 `OBSERVABILITY_EVENTS`

Used for lifecycle/audit/rollback evidence events.

```json
{
  "schema_version": 1,
  "contract": "OBSERVABILITY_EVENTS",
  "source_system": "ol",
  "target_system": "ao",
  "operation": "event.emit",
  "payload": {
    "event_type": "runtime.kill_switch.enforced",
    "severity": "warn",
    "message": "Quest canceled and checkpoint preserved"
  },
  "meta": {
    "timestamp": "2026-02-12T00:00:00Z",
    "correlation_id": "uuid"
  }
}
```

### 0.4 Profile Mapping

| This Section | Surface |
|---|---|
| Learning interchange format | `INVOCATION_ENVELOPE` |
| Exit code contract | `STATE_CHECKPOINT_HANDOFF` |
| Validation result format | `STATE_CHECKPOINT_HANDOFF` |
| Storage path conventions | `INVOCATION_ENVELOPE` |
| Lifecycle/audit/kill switch evidence | `OBSERVABILITY_EVENTS` |

### 0.5 Schema v1 Baseline (Required vs Optional)

For `schema_version: 1`, envelopes MUST include:
- `schema_version` (int, required)
- `contract` (string, required; one of `INVOCATION_ENVELOPE`, `STATE_CHECKPOINT_HANDOFF`, `OBSERVABILITY_EVENTS`)
- `source_system` (string, required; `ao` or `ol`)
- `target_system` (string, required; `ao` or `ol`)
- `operation` (string, required)
- `payload` (object, required; may be empty `{}`)
- `meta.timestamp` (RFC3339 string, required)

Envelopes SHOULD include:
- `meta.correlation_id` (string, recommended; stable across retries)

Validation / default rules (normative):
- Writers MUST include `schema_version`.
- If `schema_version` is absent, treat it as `1` (legacy reader behavior; writers MUST NOT omit it).
- If `schema_version` is less than `1`, the receiver MUST reject the payload.
- If `schema_version` is greater than supported, the receiver MUST reject the payload.
- If required fields are missing, or `contract` is unknown, the receiver MUST reject the payload.

On rejection, the receiver MUST emit an `OBSERVABILITY_EVENTS` error and fail closed (no checkpoint mutation).

### 0.6 MemRL Policy Package (AO Export)

AgentOps now exports a deterministic MemRL policy package that Olympus can consume as static contract artifacts:

- `docs/contracts/memrl-policy.schema.json`
- `docs/contracts/memrl-policy.profile.example.json`
- `docs/contracts/memrl-policy-integration.md`

The policy package defines:

- canonical `memrl_mode` contract: `off|observe|enforce`
- deterministic truth table: `memrl_mode × failure_class × attempt_bucket -> action`
- action set restricted to `retry|escalate`
- explicit unknown/missing/tie-break/default behavior
- rollback matrix with mechanical triggers and verification commands

Status: AO-side export is implemented. Olympus-side envelope/runtime enforcement remains `SPEC_ONLY` until Olympus adds schema-enforced consumption in code with conformance tests.

## 1. Learning Interchange Format

### OL → AO (harvest → forge)

OL `harvestCandidate` maps to AO `Candidate` as follows:

| OL Field (`harvestCandidate`) | AO Field (`Candidate`) | Transform |
|-------------------------------|------------------------|-----------|
| `id` | `id` | Direct copy |
| `type` ("LEARNING", "ANTI_PATTERN") | `type` | Map: `LEARNING` → `learning`, `ANTI_PATTERN` → `failure` |
| `title` | `content` | Direct copy |
| `summary` | `context` | Direct copy |
| `category` | `metadata["category"]` | Store in metadata |
| `quest_id` | `metadata["quest_id"]` | Store in metadata |
| `source_artifacts` | `source.transcript_path` | Join with `;` |
| `tags` | `metadata["tags"]` | Store in metadata |
| `created_at` | `extracted_at` | Parse RFC3339 |
| _(missing)_ | `utility` | Default: `0.5` |
| _(missing)_ | `is_current` | Default: `true` |
| _(missing)_ | `maturity` | Default: `provisional` |
| _(missing)_ | `tier` | Default: `bronze` |
| _(missing)_ | `source.session_id` | Set to `ol-harvest-<quest_id>` |

### File Format

OL harvest outputs markdown with YAML frontmatter to `.agents/learnings/`. AO `inject` discovers files at this same path.

**Transport note:** The file system under `.agents/` is the canonical transport for this profile, but the contract surface is still `INVOCATION_ENVELOPE` (this is not an extra bridge surface).

**Required frontmatter fields for AO compatibility:**

```yaml
---
id: "learn-2026-02-09-abc12345"
type: learning           # learning | failure | decision | solution | reference
source: "ol-harvest"     # provenance marker
quest_id: "ol-572"       # OL quest traceability
created_at: "2026-02-09T12:00:00Z"
tags: ["go", "testing"]
---
```

**AO reads:** `id`, `type`, `created_at` from frontmatter. Title from first `# ` heading. Summary from body text.

### AO → OL (anti-patterns → constraints)

AO learnings with `type: failure` or `maturity: anti-pattern` can be converted to OL constraints:

| AO Field (`Candidate`) | OL Field (`Constraint`) | Transform |
|-------------------------|-------------------------|-----------|
| `content` | `pattern` | First sentence or title |
| `context` | `detection` | Full description |
| _(needs manual)_ | `test_template` | Derive from content or leave empty |
| `id` | `source` | `ao-learning-<id>` |
| `confidence` | `confidence` | Direct copy (default 0.5 if missing) |
| _(missing)_ | `status` | Default: `proposed` |

### File Location

AO writes anti-patterns/failures to `.agents/learnings/`. OL reads constraints from `.ol/constraints/quarantine.json`. The bridge must translate between formats.

<!-- FUTURE: ao export-constraints not yet implemented -->
> **Not yet implemented:** `ao export-constraints --format=ol` — would write to `.ol/constraints/quarantine.json`. **Owner:** AO. **Migration note:** treat as Phase 2 prerequisite for AO→OL constraint propagation.

## 2. Exit Code Contract

### OL Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Normal completion |
| 1 | Error | Invalid input, test failure, merge conflict |
| 2 | Escalation | Human/agent intervention needed (iteration limit, hard blocker) |
| 42 | Complete | Zeus step: all phases done |

### AO Exit Codes

AO skills run inside Claude Code sessions and don't use exit codes directly. Instead, they signal via:
- File artifacts (`.agents/council/*` with verdicts)
- Promise tags (`<promise>DONE</promise>`, `<promise>BLOCKED</promise>`)

### Bridge Mapping

When AO invokes `ol` commands:

| ol exit code | AO interpretation |
|--------------|-------------------|
| 0 | Proceed to next phase |
| 1 | Log error, retry or escalate |
| 2 | Present to user or invoke fallback skill |
| 42 | Epic/quest complete |

Note: `ol validate stage1 -o json` may exit `0` even when `passed: false` (it returns after encoding JSON). Consumers MUST gate on the JSON `passed` field for that command until Olympus changes this behavior.

When OL dispatches to Claude Code (which runs AO skills):

Olympus treats **artifacts** as the authoritative completion signal. Exit codes are secondary and may be ambiguous (e.g., user cancel vs skill error).

**Normative definition: Expected artifacts (DD8)**
- "Expected artifacts" are phase-specific glob patterns relative to repo root.
- Checks MUST be scoped by `dispatch_start_time` (phase entry time): for freshness-checked patterns, a matching file MUST have `mtime > dispatch_start_time`.
- Note: `dispatch_start_time` is stable across retries for a phase. This intentionally allows artifacts produced by a prior attempt in the same phase to satisfy the check (crash recovery).
- Optional checks do not gate success.

Current phase patterns (authoritative in Olympus code; may drift):
- `RESEARCH`: `.agents/research/*` (required)
- `PLAN`: `.agents/plans/*` (required), `.beads/issues.jsonl` (optional)
- `RETRO`: `.agents/retros/*` (required)
- `FLYWHEEL`: `.agents/learnings/*` (optional)

**Normative interpretation (DD7.3 + DD8)**
- If **required artifacts exist** (artifact check OK), treat the dispatch as **SUCCESS**, even if the subprocess exited non-zero (crash recovery).
- If **required artifacts are missing**, treat the dispatch as **FAILURE/RETRY**, even if the subprocess exited 0.

| Subprocess exit | Artifact check (`artifact_check_ok`) | Olympus interpretation |
|---|---|---|
| non-zero | true | **SUCCESS** (treat as phase success; clear dispatch state; no retry) |
| 0 | false | **RETRY/FAIL** (retry up to budget; if still missing, fail closed, typically exit `1`) |
| non-zero | false | **RETRY/ESCALATE** (retry up to budget; if a deterministic breaker triggers, exit `2`; otherwise fail closed, exit `1`) |

## 3. Versioning and Capability Negotiation

### 3.1 Schema Versioning (Normative)

Bridge payload versioning uses a **monotonic integer** field: `schema_version`.

Rules:
1. Every bridge payload MUST include `schema_version`.
2. Increment by `+1` for any contract change.
3. Prefer additive changes.
4. Readers MUST be default-safe when new fields are absent.
5. Unknown fields MUST NOT fail parsing by default.
6. Breaking changes require a dual-read compatibility window and explicit migration gate.

### 3.2 Capability Query

Before bridge commands call across CLIs, verify the other CLI exists and supports the expected interface:

```bash
# AO checking OL
ol_version=$(ol --version 2>/dev/null) || { echo "ol CLI not found"; exit 1; }

# OL checking AO
ao_version=$(ao version 2>/dev/null) || { echo "ao CLI not found"; exit 1; }
```

### 3.3 Feature Detection

Rather than version parsing, use feature detection:

```bash
# Check if ol harvest supports --format flag
ol harvest --help 2>&1 | grep -q "\-\-format" && OL_HAS_FORMAT=true

# Check if ao inject supports --ol-constraints flag
ao inject --help 2>&1 | grep -q "ol-constraints" && AO_HAS_OL=true

# Check if ol validate stage1 exists
ol validate stage1 --help 2>/dev/null && OL_HAS_STAGE1=true
```

### 3.4 Graceful Degradation

| Missing capability | Fallback |
|-------------------|----------|
| `ol` not on PATH | Skip OL integration, pure AO mode |
| `ao` not on PATH | Skip AO integration, pure OL mode |
| `ol harvest --format=ao` not supported | Manual file copy from `.agents/learnings/` |
| `ao inject --ol-constraints` not supported | Skip constraint injection |
| `.ol/` directory missing | Not an Olympus project, skip all OL features |

### 3.5 MemRL Policy Migration

Roll out MemRL policy integration in deterministic stages:

1. `MEMRL_MODE=off` (strict parity with legacy retry behavior)
2. `MEMRL_MODE=observe` (audit decisions, no enforcement)
3. `MEMRL_MODE=enforce` (enforce policy action)

Rollback is mechanical via `rollback_matrix` triggers in the policy profile. If any trigger fires, demote mode (`enforce -> observe -> off`) and verify using the trigger's verification command.

## 4. Validation Result Format

### OL Stage1Result → AO Vibe Input

#### Current State (Implemented): Raw `Stage1Result` JSON

Today, `ol validate stage1 -o json` emits **raw `Stage1Result` JSON** (no envelope).

Important: In JSON mode, consumers MUST gate on `passed` (not process exit code). `passed: false` may still exit `0` after emitting JSON.

Example (raw, abbreviated):

```json
{
  "quest_id": "ol-572",
  "bead_id": "ol-572.3",
  "worktree": "/path/to/worktree",
  "passed": true,
  "steps": [
    {"name": "go test", "exit_code": 0, "passed": true, "duration": "3.4s"}
  ],
  "summary": "all steps passed"
}
```

#### Future (SPEC_ONLY): Enveloped `Stage1Result` via `STATE_CHECKPOINT_HANDOFF`

Once Olympus implements schema-enveloped emit/parse + conformance tests, `Stage1Result` MAY be carried inside a `STATE_CHECKPOINT_HANDOFF` envelope.

```json
{
  "schema_version": 1,
  "contract": "STATE_CHECKPOINT_HANDOFF",
  "source_system": "ol",
  "target_system": "ao",
  "operation": "validate.stage1.result",
  "payload": {
    "quest_id": "ol-572",
    "bead_id": "ol-572.3",
    "worktree": "/path/to/worktree",
    "passed": true,
    "steps": [
      {"name": "go build", "passed": true, "duration": "1.2s"},
      {"name": "go vet", "passed": true, "duration": "0.8s"},
      {"name": "go test", "passed": true, "duration": "3.4s"}
    ],
    "summary": "all steps passed"
  },
  "meta": {
    "timestamp": "2026-02-12T00:00:00Z",
    "correlation_id": "uuid"
  }
}
```

**AO vibe integration:** Include OL stage1 results in the vibe report as a "deterministic validation" section. If `passed: false`, auto-FAIL the vibe without running council.

## 5. Storage Path Conventions

| Artifact | OL Location | AO Location | Shared? |
|----------|-------------|-------------|---------|
| Learnings | `.agents/learnings/` | `.agents/learnings/` | **Yes** — both read/write |
| Patterns | `.agents/patterns/` | `.agents/patterns/` | **Yes** — both read/write |
| Sessions | `.ol/quests/*/` | `.agents/ao/sessions/` | No |
| Constraints | `.ol/constraints/` | N/A | OL only |
| Council reports | N/A | `.agents/council/` | AO only |
| Validation runs | `.ol/runs/` | N/A | OL only |
| Ratchet chain | N/A | `.agents/ao/chain.jsonl` | AO only |

**Key insight:** `.agents/learnings/` and `.agents/patterns/` are already shared. No unification needed for the primary knowledge flow. The bridge work is about format compatibility within these shared directories.
