#!/bin/bash
# Validate pr-implement skill
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

echo "=== PR Implement Skill Validation ==="
echo ""

check_pattern "SKILL.md has isolation pre-check" "$SKILL_DIR/SKILL.md" "[Ii]solation[[:space:]]+[Pp]re-[Cc]heck"
check_pattern "SKILL.md has isolation post-check" "$SKILL_DIR/SKILL.md" "[Ii]solation[[:space:]]+[Pp]ost-[Cc]heck"
check_pattern "SKILL.md has implementation phase" "$SKILL_DIR/SKILL.md" "Phase[[:space:]]+[0-9]+:?[[:space:]]+[Ii]mplementation"
check_pattern "SKILL.md has Examples section" "$SKILL_DIR/SKILL.md" "^## Examples"
check_pattern "SKILL.md has Troubleshooting section" "$SKILL_DIR/SKILL.md" "^## Troubleshooting"

echo ""
echo "=== Results ==="
echo "Checks: $CHECKS"
echo "Errors: $ERRORS"

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "FAIL: pr-implement skill validation failed"
    exit 1
else
    echo ""
    echo "PASS: pr-implement skill validation passed"
    exit 0
fi
