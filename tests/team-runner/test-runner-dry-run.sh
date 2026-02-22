#!/usr/bin/env bash
# Test team-runner.sh dry-run mode
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RUNNER="${REPO_ROOT}/lib/scripts/team-runner.sh"
FIXTURES="${SCRIPT_DIR}/fixtures"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

PASS=0
FAIL=0

assert_eq() {
    local desc="$1" expected="$2" actual="$3"
    if [[ "$expected" == "$actual" ]]; then
        echo "  PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $desc (expected=$expected, actual=$actual)"
        FAIL=$((FAIL + 1))
    fi
}

assert_contains() {
    local desc="$1" pattern="$2" text="$3"
    if echo "$text" | grep -q "$pattern"; then
        echo "  PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $desc (pattern '$pattern' not found)"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== Test: team-runner.sh dry-run ==="

# Test 1: Syntax check
echo "Test 1: Syntax check"
bash -n "$RUNNER"
assert_eq "syntax valid" "0" "$?"

# Test 2: Dry run with sample spec
echo "Test 2: Dry run execution"
OUTPUT=$(TEAM_RUNNER_DRY_RUN=1 bash "$RUNNER" "$FIXTURES/sample-team-spec.json" 2>&1)
assert_eq "exit code 0" "0" "$?"
assert_contains "shows codex exec" "codex exec" "$OUTPUT"
assert_contains "shows model" "gpt-5.3-codex" "$OUTPUT"
assert_contains "shows agent name" "test-agent-1" "$OUTPUT"
assert_contains "all agents passed" "All agents passed" "$OUTPUT"
assert_contains "report generated" "team-report.md" "$OUTPUT"

# Test 3: Dry run produces report
echo "Test 3: Report generation"
assert_eq "report exists" "true" "$(test -f ".agents/teams/test-run-001/team-report.md" && echo true || echo false)"

# Test 4: Sandbox level mapping
echo "Test 4: Sandbox level mapping"
assert_contains "full-auto for workspace-write" "full-auto" "$OUTPUT"
assert_contains "read-only for read-only agent" "read-only" "$OUTPUT"

echo ""
echo "Results: $PASS passed, $FAIL failed"
[[ $FAIL -eq 0 ]]
