#!/bin/bash
# Validate openai-docs skill
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

echo "=== OpenAI Docs Skill Validation ==="
echo ""

check_pattern "SKILL.md has OpenAI docs MCP search tool" "$SKILL_DIR/SKILL.md" "mcp__openaiDeveloperDocs__search_openai_docs"
check_pattern "SKILL.md has OpenAI docs MCP fetch tool" "$SKILL_DIR/SKILL.md" "mcp__openaiDeveloperDocs__fetch_openai_doc"
check_pattern "SKILL.md has Examples section" "$SKILL_DIR/SKILL.md" "^## Examples"
check_pattern "SKILL.md has Troubleshooting section" "$SKILL_DIR/SKILL.md" "^## Troubleshooting"
check_pattern "SKILL.md restricts fallback to official domains" "$SKILL_DIR/SKILL.md" "developers\.openai\.com|platform\.openai\.com"

echo ""
echo "=== Results ==="
echo "Checks: $CHECKS"
echo "Errors: $ERRORS"

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "FAIL: openai-docs skill validation failed"
    exit 1
else
    echo ""
    echo "PASS: openai-docs skill validation passed"
    exit 0
fi
