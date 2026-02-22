#!/bin/bash
# Validate pr-retro skill
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

echo "=== PR Retro Skill Validation ==="
echo ""

check_pattern "SKILL.md has outcome analysis" "$SKILL_DIR/SKILL.md" "[Oo]utcome[[:space:]]+[Aa]nalysis"
check_pattern "SKILL.md has lessons learned" "$SKILL_DIR/SKILL.md" "[Ll]essons[[:space:]]+[Ll]earned"
check_pattern "SKILL.md has process updates" "$SKILL_DIR/SKILL.md" "[Uu]pdates[[:space:]]+to[[:space:]]+[Pp]rocess"
check_pattern "SKILL.md has Examples section" "$SKILL_DIR/SKILL.md" "^## Examples"
check_pattern "SKILL.md has Troubleshooting section" "$SKILL_DIR/SKILL.md" "^## Troubleshooting"

echo ""
echo "=== Results ==="
echo "Checks: $CHECKS"
echo "Errors: $ERRORS"

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "FAIL: pr-retro skill validation failed"
    exit 1
else
    echo ""
    echo "PASS: pr-retro skill validation passed"
    exit 0
fi
