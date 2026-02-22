#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0; FAIL=0

check() { if bash -c "$2"; then echo "PASS: $1"; PASS=$((PASS + 1)); else echo "FAIL: $1"; FAIL=$((FAIL + 1)); fi; }

check "SKILL.md exists" "[ -f '$SKILL_DIR/SKILL.md' ]"
check "SKILL.md has YAML frontmatter" "head -1 '$SKILL_DIR/SKILL.md' | grep -q '^---$'"
check "SKILL.md has name: evolve" "grep -q '^name: evolve' '$SKILL_DIR/SKILL.md'"
check "references/ directory exists" "[ -d '$SKILL_DIR/references' ]"
check "references/ has at least 1 file" "[ \$(ls '$SKILL_DIR/references/' | wc -l) -ge 1 ]"
check "SKILL.md mentions kill switch" "grep -qi 'kill switch' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions fitness" "grep -qi 'fitness' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions GOALS.yaml" "grep -q 'GOALS.yaml' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions cycle" "grep -qi 'cycle' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions /rpi" "grep -q '/rpi' '$SKILL_DIR/SKILL.md'"
# Behavioral contracts from retro learnings (2026-02-12)
check "SKILL.md has KILL file path" "grep -q 'KILL' '$SKILL_DIR/SKILL.md'"
check "SKILL.md documents regression detection" "grep -qi 'regression' '$SKILL_DIR/SKILL.md'"
check "SKILL.md documents snapshot enforcement" "grep -qi 'snapshot' '$SKILL_DIR/SKILL.md'"
check "SKILL.md documents session_start_sha" "grep -qi 'session.start.sha\|cycle_start_sha' '$SKILL_DIR/SKILL.md'"
check "SKILL.md documents continuous values" "grep -qi 'continuous\|value.*threshold' '$SKILL_DIR/SKILL.md'"
check "SKILL.md documents full regression gate" "grep -qi 'full.*regression\|all goals' '$SKILL_DIR/SKILL.md'"
check "SKILL.md documents post-cycle snapshot" "grep -q 'fitness-.*-post' '$SKILL_DIR/SKILL.md'"

echo ""; echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && exit 0 || exit 1
