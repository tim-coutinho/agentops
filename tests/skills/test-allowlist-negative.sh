#!/usr/bin/env bash
# test-allowlist-negative.sh — Negative fixture test for validate-skill allowlist parsing
#
# Proves that validate-skill.sh catches drift between documented flags
# and canonical reference files. Creates temp fixtures with intentional
# mismatches and asserts validation fails.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VALIDATE="$SCRIPT_DIR/validate-skill.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

PASS=0
FAIL=0

assert_fails() {
    local desc="$1"
    shift
    if "$@" >/dev/null 2>&1; then
        echo -e "  ${RED}✗${NC} $desc (expected failure, got success)"
        FAIL=$((FAIL + 1))
    else
        echo -e "  ${GREEN}✓${NC} $desc"
        PASS=$((PASS + 1))
    fi
}

assert_passes() {
    local desc="$1"
    shift
    if "$@" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} $desc"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}✗${NC} $desc (expected success, got failure)"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== Allowlist Negative Fixture Tests ==="

# --- Test 1: Baseline passes (sanity check) ---
echo ""
echo "Test 1: Baseline council validation passes"
assert_passes "council allowlist matches references" \
    "$VALIDATE" "$REPO_ROOT/skills/council"

# --- Test 2: Drift in technique allowlist ---
echo ""
echo "Test 2: Detect technique allowlist drift"

TMPDIR_2=$(mktemp -d)
trap "rm -rf $TMPDIR_2" EXIT

# Copy council skill to temp
/bin/cp -R "$REPO_ROOT/skills/council" "$TMPDIR_2/council"

# Inject a fake technique into the reference doc (simulates drift)
echo '| `fake-technique` | Intentionally invalid technique for testing |' \
    >> "$TMPDIR_2/council/references/brainstorm-techniques.md"

assert_fails "catches drifted technique in reference doc" \
    "$VALIDATE" "$TMPDIR_2/council"

# --- Test 3: Drift in profile allowlist ---
echo ""
echo "Test 3: Detect profile allowlist drift"

TMPDIR_3=$(mktemp -d)
# Re-register cleanup
trap "rm -rf $TMPDIR_2 $TMPDIR_3" EXIT

/bin/cp -R "$REPO_ROOT/skills/council" "$TMPDIR_3/council"

# Inject a fake profile into the reference doc
echo '| `ultra-fast` | haiku | 1 | 30 | Intentionally invalid profile |' \
    >> "$TMPDIR_3/council/references/model-profiles.md"

assert_fails "catches drifted profile in reference doc" \
    "$VALIDATE" "$TMPDIR_3/council"

# --- Test 4: Missing reference file ---
echo ""
echo "Test 4: Missing reference file detected"

TMPDIR_4=$(mktemp -d)
trap "rm -rf $TMPDIR_2 $TMPDIR_3 $TMPDIR_4" EXIT

/bin/cp -R "$REPO_ROOT/skills/council" "$TMPDIR_4/council"
rm -f "$TMPDIR_4/council/references/brainstorm-techniques.md"

assert_fails "catches missing brainstorm-techniques.md" \
    "$VALIDATE" "$TMPDIR_4/council"

# --- Summary ---
echo ""
TOTAL=$((PASS + FAIL))
echo "=== Results: $PASS/$TOTAL passed ==="

if [ "$FAIL" -gt 0 ]; then
    echo -e "${RED}$FAIL test(s) failed${NC}"
    exit 1
fi
echo -e "${GREEN}All tests passed${NC}"
