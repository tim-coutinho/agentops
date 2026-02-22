#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0; FAIL=0
check() { if bash -c "$2"; then echo "PASS: $1"; PASS=$((PASS + 1)); else echo "FAIL: $1"; FAIL=$((FAIL + 1)); fi; }

check "SKILL.md exists" "[ -f '$SKILL_DIR/SKILL.md' ]"
check "SKILL.md has YAML frontmatter" "head -1 '$SKILL_DIR/SKILL.md' | grep -q '^---$'"
check "name is inbox" "grep -q '^name: inbox' '$SKILL_DIR/SKILL.md'"
check "mentions Agent Mail" "grep -q 'Agent Mail' '$SKILL_DIR/SKILL.md'"
check "mentions HELP_REQUEST" "grep -q 'HELP_REQUEST' '$SKILL_DIR/SKILL.md'"
check "mentions MCP tools" "grep -q 'mcp__' '$SKILL_DIR/SKILL.md'"
check "mentions watch mode" "grep -qi 'watch' '$SKILL_DIR/SKILL.md'"

echo ""; echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && exit 0 || exit 1
