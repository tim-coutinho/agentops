#!/usr/bin/env bash
# sync-skill-counts.sh — Single-source-of-truth skill count updater.
# Reads actual counts from disk + SKILL-TIERS.md, patches all doc files.
# Run after adding/removing a skill to keep all references in sync.
#
# Usage: scripts/sync-skill-counts.sh [--check]
#   --check   Dry-run: report mismatches without modifying files (exit 1 if any)
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK_ONLY=false
[[ "${1:-}" == "--check" ]] && CHECK_ONLY=true

# --- Derive truth from disk ---

TOTAL=$(find "$REPO_ROOT/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')

USER_FACING=$(sed -n '/^### User-Facing/,/^### Internal/p' "$REPO_ROOT/skills/SKILL-TIERS.md" \
  | grep -c '^| \*\*')

INTERNAL=$(sed -n '/^### Internal Skills/,/^---$/p' "$REPO_ROOT/skills/SKILL-TIERS.md" \
  | grep -c '^| ' || echo 0)
INTERNAL=$((INTERNAL - 1))  # subtract header row

echo "Skill counts from disk:"
echo "  Total:       $TOTAL"
echo "  User-facing: $USER_FACING"
echo "  Internal:    $INTERNAL"
echo ""

if [[ $((USER_FACING + INTERNAL)) -ne "$TOTAL" ]]; then
  echo "ERROR: SKILL-TIERS.md tables ($((USER_FACING + INTERNAL))) != directories ($TOTAL)"
  echo "Fix SKILL-TIERS.md first — add/remove the skill row, then re-run."
  exit 1
fi

# --- Define all patch targets ---
changes=0

patch_file() {
  local file="$1" pattern="$2" desc="$3"
  if [[ ! -f "$file" ]]; then
    echo "SKIP: $file (not found)"
    return
  fi
  if $CHECK_ONLY; then
    # Apply the sed pattern to a copy and compare — if output differs, there's drift
    local patched
    patched=$(sed "$pattern" "$file")
    if [[ "$patched" == "$(cat "$file")" ]]; then
      echo "OK:   $desc"
    else
      echo "DRIFT: $desc"
      changes=$((changes + 1))
    fi
    return
  fi
  local before after
  before=$(md5 -q "$file" 2>/dev/null || md5sum "$file" | cut -d' ' -f1)
  sed -i '' "$pattern" "$file" 2>/dev/null || sed -i "$pattern" "$file"
  after=$(md5 -q "$file" 2>/dev/null || md5sum "$file" | cut -d' ' -f1)
  if [[ "$before" != "$after" ]]; then
    echo "UPDATED: $desc"
    changes=$((changes + 1))
  else
    echo "OK:      $desc"
  fi
}

# SKILL-TIERS.md header counts
patch_file "$REPO_ROOT/skills/SKILL-TIERS.md" \
  "s/### User-Facing Skills ([0-9]*)/### User-Facing Skills ($USER_FACING)/" \
  "SKILL-TIERS.md user-facing header"

patch_file "$REPO_ROOT/skills/SKILL-TIERS.md" \
  "s/### Internal Skills ([0-9]*)/### Internal Skills ($INTERNAL)/" \
  "SKILL-TIERS.md internal header"

# CLAUDE.md: "All N skills (M user-facing, K internal)"
patch_file "$REPO_ROOT/CLAUDE.md" \
  "s/All [0-9]* skills ([0-9]* user-facing, [0-9]* internal)/All $TOTAL skills ($USER_FACING user-facing, $INTERNAL internal)/" \
  "CLAUDE.md skill counts"

# README.md badge: "skills-N-"
patch_file "$REPO_ROOT/README.md" \
  "s/skills-[0-9]*-/skills-${TOTAL}-/" \
  "README.md badge"

# README.md text: "N skills total: M user-facing across three tiers, plus K internal"
patch_file "$REPO_ROOT/README.md" \
  "s/[0-9]* skills total: [0-9]* user-facing across three tiers, plus [0-9]* internal/${TOTAL} skills total: ${USER_FACING} user-facing across three tiers, plus ${INTERNAL} internal/" \
  "README.md text count"

# README.md: "All N skills work without it"
patch_file "$REPO_ROOT/README.md" \
  "s/All [0-9]* skills work without it/All ${TOTAL} skills work without it/" \
  "README.md ao CLI sentence"

# docs/SKILLS.md header: "all N AgentOps skills (M user-facing + K internal)"
patch_file "$REPO_ROOT/docs/SKILLS.md" \
  "s/all [0-9]* AgentOps skills ([0-9]* user-facing + [0-9]* internal)/all ${TOTAL} AgentOps skills (${USER_FACING} user-facing + ${INTERNAL} internal)/" \
  "docs/SKILLS.md header"

# docs/SKILLS.md behavioral: "All N skills have"
patch_file "$REPO_ROOT/docs/SKILLS.md" \
  "s/All [0-9]* skills have/All ${TOTAL} skills have/" \
  "docs/SKILLS.md behavioral line"

# docs/ARCHITECTURE.md: "N skills (M user-facing, K internal)"
patch_file "$REPO_ROOT/docs/ARCHITECTURE.md" \
  "s/[0-9]* skills ([0-9]* user-facing, [0-9]* internal)/${TOTAL} skills (${USER_FACING} user-facing, ${INTERNAL} internal)/" \
  "docs/ARCHITECTURE.md"

# marketplace.json: two description fields with skill counts
patch_file "$REPO_ROOT/.claude-plugin/marketplace.json" \
  "s/[0-9]* skills with Knowledge Flywheel/${TOTAL} skills with Knowledge Flywheel/" \
  "marketplace.json metadata description"

patch_file "$REPO_ROOT/.claude-plugin/marketplace.json" \
  "s/[0-9]* skills ([0-9]* user-facing, [0-9]* internal)/${TOTAL} skills (${USER_FACING} user-facing, ${INTERNAL} internal)/" \
  "marketplace.json plugin description"

# README.md summary: "N skills: M user-facing, K internal"
patch_file "$REPO_ROOT/README.md" \
  "s/[0-9]* skills: [0-9]* user-facing, [0-9]* internal/${TOTAL} skills: ${USER_FACING} user-facing, ${INTERNAL} internal/" \
  "README.md skills summary"

# README.md reference: "all N skills:"
patch_file "$REPO_ROOT/README.md" \
  "s/all [0-9]* skills:/all ${TOTAL} skills:/" \
  "README.md skills reference"

# PRODUCT.md: "The N skills,"
patch_file "$REPO_ROOT/PRODUCT.md" \
  "s/The [0-9]* skills,/The ${TOTAL} skills,/" \
  "PRODUCT.md skill count"

# using-agentops/SKILL.md: "Available Skills (M user-facing)"
patch_file "$REPO_ROOT/skills/using-agentops/SKILL.md" \
  "s/Available Skills ([0-9]* user-facing)/Available Skills (${USER_FACING} user-facing)/" \
  "using-agentops/SKILL.md user-facing count"

echo ""

# --- Verify with existing validator ---
if ! $CHECK_ONLY; then
  echo "=== Verifying with validate-skill-count.sh ==="
  if bash "$REPO_ROOT/tests/docs/validate-skill-count.sh" > /dev/null 2>&1; then
    echo "PASS: All counts verified consistent"
  else
    echo "FAIL: Counts still inconsistent after patching — check output above"
    exit 1
  fi
fi

echo ""
if [[ "$changes" -gt 0 ]]; then
  if $CHECK_ONLY; then
    echo "DRIFT: $changes file(s) have stale skill counts. Run: scripts/sync-skill-counts.sh"
    exit 1
  else
    echo "DONE: $changes file(s) updated. Counts synced to $TOTAL total ($USER_FACING user-facing, $INTERNAL internal)."
  fi
else
  echo "DONE: All counts already in sync."
fi
