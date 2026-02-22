#!/usr/bin/env bash
set -euo pipefail
SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0; FAIL=0

check() { if bash -c "$2"; then echo "PASS: $1"; PASS=$((PASS + 1)); else echo "FAIL: $1"; FAIL=$((FAIL + 1)); fi; }

check "SKILL.md exists" "[ -f '$SKILL_DIR/SKILL.md' ]"
check "SKILL.md has YAML frontmatter" "head -1 '$SKILL_DIR/SKILL.md' | grep -q '^---$'"
check "SKILL.md has name: retro" "grep -q '^name: retro' '$SKILL_DIR/SKILL.md'"
check "references/ directory exists" "[ -d '$SKILL_DIR/references' ]"
check "references/ has at least 1 file" "[ \$(ls '$SKILL_DIR/references/' | wc -l) -ge 1 ]"
check "SKILL.md mentions .agents/learnings/ output" "grep -q '\.agents/learnings/' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions .agents/retros/ output" "grep -q '\.agents/retros/' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions knowledge flywheel" "grep -qi 'flywheel' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions ao forge" "grep -q 'ao forge' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions learning categories" "grep -qi 'category\|debugging\|architecture\|process' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions confidence levels" "grep -qi 'confidence.*high\|high.*medium.*low' '$SKILL_DIR/SKILL.md'"
check "SKILL.md mentions vibe results integration" "grep -qi 'vibe.results\|vibe-results' '$SKILL_DIR/SKILL.md'"

echo ""; echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && exit 0 || exit 1
