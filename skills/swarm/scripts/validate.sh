#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0; FAIL=0

check() { if bash -c "$2"; then echo "PASS: $1"; PASS=$((PASS + 1)); else echo "FAIL: $1"; FAIL=$((FAIL + 1)); fi; }

check "SKILL.md exists" "[ -f '$SKILL_DIR/SKILL.md' ]"
check "SKILL.md has YAML frontmatter" "head -1 '$SKILL_DIR/SKILL.md' | grep -q '^---$'"
check "SKILL.md has name: swarm" "grep -q '^name: swarm' '$SKILL_DIR/SKILL.md'"
check "Local mode documented" "grep -q 'Local' '$SKILL_DIR/SKILL.md'"
check "Distributed mode documented" "grep -q 'Distributed' '$SKILL_DIR/SKILL.md'"
check "Backend references documented" "grep -q 'backend-claude-teams' '$SKILL_DIR/SKILL.md'"
check "Shared backend docs exist" "[ -f '$SKILL_DIR/../shared/references/backend-claude-teams.md' ]"
check "Cleanup lifecycle documented" "grep -qE 'TeamDelete|close_agent|cleanup' '$SKILL_DIR/SKILL.md'"

echo ""; echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && exit 0 || exit 1
