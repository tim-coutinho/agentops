#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

errors=0

# Count actual goals in GOALS.yaml
actual_count=$(grep -c "^  - id:" "$REPO_ROOT/GOALS.yaml")

# Extract README claim (line like "44 measurable goals") — optional
readme_claim=$(grep -oE '[0-9]+ measurable goals' "$REPO_ROOT/README.md" | head -1 | grep -oE '^[0-9]+' || echo "")

echo "=== Goal Count Validation ==="
echo "  GOALS.yaml actual: $actual_count"

if [[ -n "$readme_claim" ]]; then
    echo "  README.md claim:   $readme_claim"
    if [[ "$actual_count" -ne "$readme_claim" ]]; then
        echo "FAIL: GOALS.yaml has $actual_count goals but README.md claims $readme_claim"
        errors=$((errors + 1))
    fi
else
    echo "  README.md claim:   (none — no hardcoded count, OK)"
fi
echo ""

# Also check GOALS.yaml count comment if present
yaml_claim=$(grep -oE '^# [0-9]+ goals:' "$REPO_ROOT/GOALS.yaml" | head -1 | grep -oE '[0-9]+' || echo "")
if [[ -n "$yaml_claim" ]]; then
    echo "  GOALS.yaml comment: $yaml_claim"
    if [[ "$actual_count" -ne "$yaml_claim" ]]; then
        echo "FAIL: GOALS.yaml comment says $yaml_claim but actual count is $actual_count"
        errors=$((errors + 1))
    fi
fi

if [[ "$errors" -gt 0 ]]; then
    echo ""
    echo "FAIL: $errors mismatch(es) found"
    exit 1
else
    echo ""
    echo "PASS: Goal counts consistent (actual=$actual_count)"
    exit 0
fi
