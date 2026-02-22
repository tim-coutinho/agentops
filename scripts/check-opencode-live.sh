#!/usr/bin/env bash
# check-opencode-live.sh — GOALS.yaml gate for OpenCode live tier 1 tests
#
# Runs Tier 1 headless tests with brownian ratchet (3 attempts per skill).
# Passes if ≥4/6 skills succeed (allows 2 failures for model variability).
#
# Gracefully skips if:
#   - opencode not in PATH (structural check only)
#   - No model configured or reachable

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HARNESS="$REPO_ROOT/tests/opencode/run-headless-tests.sh"

# Gate: opencode must be installed
if ! command -v opencode &>/dev/null; then
    echo "SKIP: opencode not in PATH"
    exit 0  # Graceful skip — don't fail the goal just because opencode isn't installed
fi

# Gate: harness must exist
if [[ ! -x "$HARNESS" ]]; then
    echo "FAIL: harness not found or not executable: $HARNESS"
    exit 1
fi

# Run tier 1 with 3 attempts
echo "Running OpenCode Tier 1 headless tests (3 attempts per skill)..."
"$HARNESS" --tier 1 --attempts 3 2>&1

# Parse summary for pass count
SUMMARY_FILE="$REPO_ROOT/.agents/opencode-tests/$(date +%Y-%m-%d)-summary.txt"
if [[ ! -f "$SUMMARY_FILE" ]]; then
    echo "FAIL: summary file not found: $SUMMARY_FILE"
    exit 1
fi

# Count skills that passed (PASS or FLAKY = at least 1 attempt succeeded)
PASSED=$(grep -cE '(PASS|FLAKY)' "$SUMMARY_FILE" || echo "0")
TOTAL=$(wc -l < "$SUMMARY_FILE" | tr -d ' ')

echo ""
echo "Gate result: $PASSED/$TOTAL skills passed"

# Threshold: ≥4 of 6 tier 1 skills must pass
if [[ $PASSED -ge 4 ]]; then
    echo "PASS: $PASSED/$TOTAL meets threshold (≥4)"
    exit 0
else
    echo "FAIL: $PASSED/$TOTAL below threshold (need ≥4)"
    exit 1
fi
