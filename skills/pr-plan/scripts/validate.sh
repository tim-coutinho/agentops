#!/bin/bash
# Validate pr-plan skill
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

ERRORS=0
CHECKS=0

check_pattern() {
    local desc="$1"
    local file="$2"
    local pattern="$3"

    CHECKS=$((CHECKS + 1))
    if grep -qiE "$pattern" "$file" 2>/dev/null; then
        echo "✓ $desc"
    else
        echo "✗ $desc (pattern '$pattern' not found in $file)"
        ERRORS=$((ERRORS + 1))
    fi
}

echo "=== PR Plan Skill Validation ==="
echo ""

check_pattern "SKILL.md has scope definition" "$SKILL_DIR/SKILL.md" "[Ss]cope"
check_pattern "SKILL.md has acceptance criteria" "$SKILL_DIR/SKILL.md" "[Aa]cceptance[[:space:]]+[Cc]riteria"
check_pattern "SKILL.md has risk assessment" "$SKILL_DIR/SKILL.md" "[Rr]isk[[:space:]]+[Aa]ssessment"
check_pattern "SKILL.md has Examples section" "$SKILL_DIR/SKILL.md" "^## Examples"
check_pattern "SKILL.md has Troubleshooting section" "$SKILL_DIR/SKILL.md" "^## Troubleshooting"

echo ""
echo "=== Results ==="
echo "Checks: $CHECKS"
echo "Errors: $ERRORS"

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "FAIL: pr-plan skill validation failed"
    exit 1
else
    echo ""
    echo "PASS: pr-plan skill validation passed"
    exit 0
fi
