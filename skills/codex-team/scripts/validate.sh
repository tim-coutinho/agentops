#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0; FAIL=0

check() { if bash -c "$2"; then echo "PASS: $1"; PASS=$((PASS + 1)); else echo "FAIL: $1"; FAIL=$((FAIL + 1)); fi; }

check "SKILL.md exists" "[ -f '$SKILL_DIR/SKILL.md' ]"
check "SKILL.md has YAML frontmatter" "head -1 '$SKILL_DIR/SKILL.md' | grep -q '^---$'"
check "SKILL.md has name: codex-team" "grep -q '^name: codex-team' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions codex" "grep -qi 'codex' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions parallel" "grep -qi 'parallel' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions merge" "grep -qi 'merge' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions multi-wave" "grep -qi 'multi-wave' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions spawn" "grep -qi 'spawn' '$SKILL_DIR/SKILL.md'"
check "SKILL.md documents output directory" "grep -q '\.agents/codex-team/' '$SKILL_DIR/SKILL.md'"

echo ""; echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && exit 0 || exit 1
