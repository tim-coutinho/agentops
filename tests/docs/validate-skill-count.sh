#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

errors=0

# Helper: extract a number from a pattern in a file
# Usage: extract_number "regex-with-capture-group" file
# Returns the first captured group (number) or NOT_FOUND
extract_number() {
  local pattern="$1" file="$2"
  local result
  result=$(sed -n "${pattern}p" "$file" | head -1)
  if [[ -z "$result" ]]; then
    echo "NOT_FOUND"
  else
    echo "$result"
  fi
}

# --- Actual counts from disk ---

actual_total=$(find "$REPO_ROOT/skills" -mindepth 1 -maxdepth 1 -type d -not -name '.*' | wc -l | tr -d ' ')

# Count skills listed in SKILL-TIERS.md user-facing table (between "### User-Facing" and "### Internal")
actual_user_facing=$(sed -n '/^### User-Facing/,/^### Internal/p' "$REPO_ROOT/skills/SKILL-TIERS.md" \
  | grep -c '^| \*\*')

# Count skills listed in SKILL-TIERS.md internal table (after "### Internal Skills")
actual_internal=$(sed -n '/^### Internal Skills/,/^---$/p' "$REPO_ROOT/skills/SKILL-TIERS.md" \
  | grep -c '^| ')
# Subtract header row
actual_internal=$((actual_internal - 1))

echo "=== Actual counts from disk ==="
echo "  Skill directories: $actual_total"
echo "  SKILL-TIERS.md user-facing table rows: $actual_user_facing"
echo "  SKILL-TIERS.md internal table rows: $actual_internal"
echo "  Table total: $((actual_user_facing + actual_internal))"
echo ""

# --- Consistency: table rows vs directory count ---

table_total=$((actual_user_facing + actual_internal))
if [[ "$table_total" -ne "$actual_total" ]]; then
  echo "MISMATCH: SKILL-TIERS.md tables list $table_total skills, actual directories: $actual_total"
  errors=$((errors + 1))
fi

# --- Extract claimed counts from SKILL-TIERS.md headers ---

# "### User-Facing Skills (21)" -> 21
tiers_user_claim=$(extract_number 's/.*### User-Facing Skills (\([0-9]*\)).*/\1/' "$REPO_ROOT/skills/SKILL-TIERS.md")
# "### Internal Skills (10)" -> 10
tiers_internal_claim=$(extract_number 's/.*### Internal Skills (\([0-9]*\)).*/\1/' "$REPO_ROOT/skills/SKILL-TIERS.md")

echo "=== SKILL-TIERS.md header claims ==="
echo "  User-facing claim: $tiers_user_claim"
echo "  Internal claim: $tiers_internal_claim"
echo ""

if [[ "$tiers_user_claim" != "NOT_FOUND" && "$tiers_user_claim" -ne "$actual_user_facing" ]]; then
  echo "MISMATCH: SKILL-TIERS.md header says $tiers_user_claim user-facing, table has $actual_user_facing"
  errors=$((errors + 1))
fi

if [[ "$tiers_internal_claim" != "NOT_FOUND" && "$tiers_internal_claim" -ne "$actual_internal" ]]; then
  echo "MISMATCH: SKILL-TIERS.md header says $tiers_internal_claim internal, table has $actual_internal"
  errors=$((errors + 1))
fi

# --- Extract counts from CLAUDE.md ---

# "All 32 skills (22 user-facing, 10 internal)"
claude_total=$(extract_number 's/.*All \([0-9]*\) skills.*/\1/' "$REPO_ROOT/CLAUDE.md")
claude_user=$(extract_number 's/.*(\([0-9][0-9]*\) user-facing.*/\1/' "$REPO_ROOT/CLAUDE.md")
claude_internal=$(extract_number 's/.* \([0-9][0-9]*\) internal).*/\1/' "$REPO_ROOT/CLAUDE.md")

echo "=== CLAUDE.md claims ==="
echo "  Total: $claude_total"
echo "  User-facing: $claude_user"
echo "  Internal: $claude_internal"
echo ""

if [[ "$claude_total" != "NOT_FOUND" && "$claude_total" -ne "$actual_total" ]]; then
  echo "MISMATCH: CLAUDE.md says $claude_total total, actual is $actual_total"
  errors=$((errors + 1))
fi

if [[ "$claude_user" != "NOT_FOUND" && "$claude_user" -ne "$actual_user_facing" ]]; then
  echo "MISMATCH: CLAUDE.md says $claude_user user-facing, SKILL-TIERS.md table has $actual_user_facing"
  errors=$((errors + 1))
fi

if [[ "$claude_internal" != "NOT_FOUND" && "$claude_internal" -ne "$actual_internal" ]]; then
  echo "MISMATCH: CLAUDE.md says $claude_internal internal, SKILL-TIERS.md table has $actual_internal"
  errors=$((errors + 1))
fi

# --- Extract counts from README.md ---

# Badge: "skills-32-"
readme_badge_total=$(extract_number 's/.*skills-\([0-9]*\)-.*/\1/' "$REPO_ROOT/README.md")

# Text: "32 skills across" or "All 32 skills"
readme_text_total=$(extract_number 's/.*[^0-9]\([0-9][0-9]*\) skills.*/\1/' "$REPO_ROOT/README.md")

# "auto-loaded, 10 total"
readme_internal=$(extract_number 's/.*auto-loaded, \([0-9]*\) total.*/\1/' "$REPO_ROOT/README.md")

echo "=== README.md claims ==="
echo "  Badge total: $readme_badge_total"
echo "  Text total: $readme_text_total"
echo "  Internal: $readme_internal"
echo ""

if [[ "$readme_badge_total" != "NOT_FOUND" && "$readme_badge_total" -ne "$actual_total" ]]; then
  echo "MISMATCH: README.md badge says $readme_badge_total total, actual is $actual_total"
  errors=$((errors + 1))
fi

if [[ "$readme_text_total" != "NOT_FOUND" && "$readme_text_total" -ne "$actual_total" ]]; then
  echo "MISMATCH: README.md text says $readme_text_total total, actual is $actual_total"
  errors=$((errors + 1))
fi

if [[ "$readme_internal" != "NOT_FOUND" && "$readme_internal" -ne "$actual_internal" ]]; then
  echo "MISMATCH: README.md says $readme_internal internal, SKILL-TIERS.md table has $actual_internal"
  errors=$((errors + 1))
fi

# --- Extract counts from README.md summary line ---

# "37 skills: 27 user-facing, 10 internal"
readme_summary_total=$(extract_number 's/^\([0-9][0-9]*\) skills: [0-9]* user-facing.*/\1/' "$REPO_ROOT/README.md")
readme_summary_user=$(extract_number 's/.*[0-9]* skills: \([0-9][0-9]*\) user-facing.*/\1/' "$REPO_ROOT/README.md")

echo "=== README.md summary claims ==="
echo "  Summary total: $readme_summary_total"
echo "  Summary user-facing: $readme_summary_user"
echo ""

if [[ "$readme_summary_total" != "NOT_FOUND" && "$readme_summary_total" -ne "$actual_total" ]]; then
  echo "MISMATCH: README.md summary says $readme_summary_total total, actual is $actual_total"
  errors=$((errors + 1))
fi

if [[ "$readme_summary_user" != "NOT_FOUND" && "$readme_summary_user" -ne "$actual_user_facing" ]]; then
  echo "MISMATCH: README.md summary says $readme_summary_user user-facing, actual is $actual_user_facing"
  errors=$((errors + 1))
fi

# --- Extract counts from PRODUCT.md ---

product_total=$(extract_number 's/.*The \([0-9][0-9]*\) skills,.*/\1/' "$REPO_ROOT/PRODUCT.md")

echo "=== PRODUCT.md claims ==="
echo "  Total: $product_total"
echo ""

if [[ "$product_total" != "NOT_FOUND" && "$product_total" -ne "$actual_total" ]]; then
  echo "MISMATCH: PRODUCT.md says $product_total total, actual is $actual_total"
  errors=$((errors + 1))
fi

# --- Extract counts from using-agentops/SKILL.md ---

agentops_user=$(extract_number 's/.*Available Skills (\([0-9][0-9]*\) user-facing).*/\1/' "$REPO_ROOT/skills/using-agentops/SKILL.md")

echo "=== using-agentops/SKILL.md claims ==="
echo "  User-facing: $agentops_user"
echo ""

if [[ "$agentops_user" != "NOT_FOUND" && "$agentops_user" -ne "$actual_user_facing" ]]; then
  echo "MISMATCH: using-agentops/SKILL.md says $agentops_user user-facing, actual is $actual_user_facing"
  errors=$((errors + 1))
fi

# --- Cross-file consistency ---

echo "=== Cross-file consistency ==="

# Compare all total claims against each other
totals=()
[[ "$claude_total" != "NOT_FOUND" ]] && totals+=("CLAUDE.md:$claude_total")
[[ "$readme_badge_total" != "NOT_FOUND" ]] && totals+=("README-badge:$readme_badge_total")
[[ "$readme_text_total" != "NOT_FOUND" ]] && totals+=("README-text:$readme_text_total")
[[ "$readme_summary_total" != "NOT_FOUND" ]] && totals+=("README-summary:$readme_summary_total")
[[ "$product_total" != "NOT_FOUND" ]] && totals+=("PRODUCT:$product_total")
if [[ "$tiers_user_claim" != "NOT_FOUND" && "$tiers_internal_claim" != "NOT_FOUND" ]]; then
  totals+=("SKILL-TIERS-headers:$((tiers_user_claim + tiers_internal_claim))")
fi

if [[ ${#totals[@]} -gt 1 ]]; then
  first_val="${totals[0]#*:}"
  for entry in "${totals[@]:1}"; do
    val="${entry#*:}"
    src="${entry%%:*}"
    if [[ "$val" -ne "$first_val" ]]; then
      echo "MISMATCH: Cross-file total disagreement: ${totals[0]} vs $src:$val"
      errors=$((errors + 1))
    fi
  done
fi

echo ""

# --- Summary ---

if [[ "$errors" -gt 0 ]]; then
  echo "FAIL: $errors mismatch(es) found"
  exit 1
else
  echo "PASS: All skill counts consistent (total=$actual_total, user-facing=$actual_user_facing, internal=$actual_internal)"
  exit 0
fi
