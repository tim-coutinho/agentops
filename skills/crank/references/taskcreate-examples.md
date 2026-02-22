# TaskCreate Examples

> Copy-paste-ready TaskCreate patterns for each crank mode.

---

## SPEC WAVE TaskCreate

Use when `--test-first` is set and issue is spec-eligible (feature/bugfix/refactor).

```
TaskCreate(
  subject="SPEC: <issue-title>",
  description="Generate contract for beads issue <issue-id>.

Details from beads:
<paste issue details from bd show>

You are a spec writer. Generate a contract for this issue.

FIRST: Explore the codebase to understand existing patterns, types, and interfaces
relevant to this issue. Use Glob and Read to examine the code.

THEN: Read the contract template at skills/crank/references/contract-template.md.

Generate a contract following the template. Include:
- At least 3 invariants
- At least 3 test cases mapped to invariants
- Concrete types and interfaces from the actual codebase

If inputs are missing or the issue is underspecified, write BLOCKED with reason.

Output: .agents/specs/contract-<issue-id>.md

```validation
files_exist:
  - .agents/specs/contract-<issue-id>.md
content_check:
  - file: .agents/specs/contract-<issue-id>.md
    patterns:
      - "## Invariants"
      - "## Test Cases"
```

Mark task complete when contract is written and validation passes.",
  activeForm="Writing spec for <issue-id>"
)
```

---

## TEST WAVE TaskCreate

Use when `--test-first` is set, SPEC WAVE is complete, and issue is spec-eligible.

```
TaskCreate(
  subject="TEST: <issue-title>",
  description="Generate FAILING tests for beads issue <issue-id>.

Details from beads:
<paste issue details from bd show>

You are a test writer. Generate FAILING tests from the contract.

Read ONLY the contract at .agents/specs/contract-<issue-id>.md.
You may read codebase structure (imports, types, interfaces) but NOT existing
implementation details.

Generate tests that:
- Cover ALL test cases from the contract's Test Cases table
- Cover ALL invariants (at least one test per invariant)
- All tests MUST FAIL when run (RED state)
- Follow existing test patterns in the codebase

Do NOT read or reference existing implementation code.
Do NOT write implementation code.

Output: test files in the appropriate location for the project's test framework.

```validation
files_exist:
  - <test-file-path-1>
  - <test-file-path-2>
red_gate:
  command: "<test-command>"
  expected: "FAIL"
  reason: "All new tests must fail (RED state) before implementation"
```

Mark task complete when tests are written and ALL tests FAIL.",
  activeForm="Writing tests for <issue-id>"
)
```

---

## GREEN Mode TaskCreate

Use when `--test-first` is set, SPEC and TEST waves are complete, and issue is spec-eligible.

```
TaskCreate(
  subject="<issue-id>: <issue-title>",
  description="Implement beads issue <issue-id> (GREEN mode).

Details from beads:
<paste issue details from bd show>

**GREEN Mode:** Failing tests exist. Make them pass. Do NOT modify test files.

Failing tests are at:
- <test-file-path-1>
- <test-file-path-2>

Contract is at: .agents/specs/contract-<issue-id>.md

Follow GREEN Mode rules from /implement SKILL.md:
1. Read failing tests and contract FIRST
2. Write minimal implementation to pass tests
3. Do NOT modify test files
4. Do NOT add tests (already written)
5. Validate by running test suite

Execute using /implement <issue-id>. Mark complete when all tests pass.",
  activeForm="Implementing <issue-id> (GREEN)"
)
```

---

## Standard IMPL TaskCreate

Use for non-spec-eligible issues (docs/chore/ci) or when `--test-first` is NOT set.

```
TaskCreate(
  subject="<issue-id>: <issue-title>",
  description="Implement beads issue <issue-id>.

Details from beads:
<paste issue details from bd show>

Execute using /implement <issue-id>. Mark complete when done.

```validation
<optional validation metadata specific to this issue>
build:
  command: "<build-command>"
  expected: "success"
tests:
  command: "<test-command>"
  expected: "pass"
files_exist:
  - <expected-output-file-1>
  - <expected-output-file-2>
```
",
  activeForm="Implementing <issue-id>"
)
```

---

## Notes

- **Subject patterns:**
  - SPEC WAVE: `SPEC: <issue-title>` (no issue ID)
  - TEST WAVE: `TEST: <issue-title>` (no issue ID)
  - GREEN/IMPL: `<issue-id>: <issue-title>` (with issue ID)

- **Validation blocks:**
  - Fenced with triple backticks and `validation` language tag
  - Always include for SPEC and TEST waves
  - Optional but recommended for GREEN/IMPL waves
  - Consumed by lead during wave validation

- **activeForm:**
  - Shows in TaskList UI while worker is active
  - Keep concise (3-5 words)
  - Include issue ID for easy tracking

- **Worker context:**
  - SPEC: codebase read access, contract template
  - TEST: contract only, codebase structure (not implementations)
  - GREEN: failing tests (immutable), contract, issue description
  - IMPL: full codebase access, issue description

- **Category-based skipping:**
  - docs/chore/ci issues bypass SPEC and TEST waves
  - Use standard IMPL TaskCreate for these even if `--test-first` is set
