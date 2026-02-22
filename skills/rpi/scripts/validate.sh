#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0; FAIL=0

check() { if bash -c "$2"; then echo "PASS: $1"; PASS=$((PASS + 1)); else echo "FAIL: $1"; FAIL=$((FAIL + 1)); fi; }

check "SKILL.md exists" "[ -f '$SKILL_DIR/SKILL.md' ]"
check "SKILL.md has YAML frontmatter" "head -1 '$SKILL_DIR/SKILL.md' | grep -q '^---$'"
check "SKILL.md has name: rpi" "grep -q '^name: rpi' '$SKILL_DIR/SKILL.md'"
check "references/ directory exists" "[ -d '$SKILL_DIR/references' ]"
check "references/ has at least 3 files" "[ \$(ls '$SKILL_DIR/references/' | wc -l) -ge 3 ]"
check "SKILL.md mentions research phase" "grep -qi 'research' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions plan phase" "grep -qi '/plan' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions pre-mortem phase" "grep -qi 'pre-mortem' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions crank phase" "grep -qi '/crank' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions vibe phase" "grep -qi '/vibe' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions post-mortem phase" "grep -qi 'post-mortem' '$SKILL_DIR/SKILL.md'"

echo ""; echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && exit 0 || exit 1
