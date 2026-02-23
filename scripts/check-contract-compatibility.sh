#!/usr/bin/env bash
set -euo pipefail

# check-contract-compatibility.sh
# Dynamic contract-compatibility gate.
#
# Validates:
#   1. All contract files referenced in docs/INDEX.md exist on disk
#   2. All contract .md files' embedded schema/file references resolve
#   3. All *.schema.json files are valid JSON
#   4. All contracts on disk are catalogued in docs/INDEX.md (orphan check)

ROOT="${1:-.}"
CONTRACTS_DIR="$ROOT/docs/contracts"
INDEX="$ROOT/docs/INDEX.md"
BRIDGE="$ROOT/docs/ol-bridge-contracts.md"

failures=0
warnings=0

fail() { echo "FAIL: $1"; failures=$((failures + 1)); }
warn() { echo "WARN: $1"; warnings=$((warnings + 1)); }
pass() { echo "  OK: $1"; }

echo "=== Contract compatibility gate ==="
echo ""

# ── Check 1: docs/contracts/ directory exists ──

if [[ ! -d "$CONTRACTS_DIR" ]]; then
  fail "docs/contracts/ directory not found"
  echo ""
  echo "Contract compatibility check failed ($failures failure(s))."
  exit 1
fi

# ── Check 2: INDEX.md references resolve ──

echo "--- INDEX.md link resolution ---"
if [[ -f "$INDEX" ]]; then
  # Extract markdown links pointing into contracts/ (outside code blocks)
  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue
    if [[ -f "$ROOT/docs/$ref" ]]; then
      pass "$ref"
    else
      fail "INDEX.md references $ref but file not found"
    fi
  done < <(awk '/^```/{skip=!skip; next} !skip{print}' "$INDEX" \
    | grep -oE '\]\(contracts/[A-Za-z0-9_./-]+\)' \
    | sed 's/\](//; s/)//' | sort -u)
else
  fail "docs/INDEX.md not found"
fi
echo ""

# ── Check 3: Bridge doc references resolve ──

echo "--- Bridge doc reference resolution ---"
if [[ -f "$BRIDGE" ]]; then
  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue
    if [[ -f "$ROOT/$ref" ]]; then
      pass "$ref"
    else
      fail "ol-bridge-contracts.md references $ref but file not found"
    fi
  done < <(awk '/^```/{skip=!skip; next} !skip{print}' "$BRIDGE" \
    | grep -oE 'docs/contracts/[A-Za-z0-9_./-]+' | sort -u)
else
  warn "docs/ol-bridge-contracts.md not found (optional)"
fi
echo ""

# ── Check 4: Contract .md files' embedded references resolve ──

echo "--- Contract .md cross-references ---"
for md in "$CONTRACTS_DIR"/*.md; do
  [[ -f "$md" ]] || continue
  basename="$(basename "$md")"
  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue
    # Try resolving relative to contracts dir, then relative to docs/, then repo root
    if [[ -f "$CONTRACTS_DIR/$ref" ]] || [[ -f "$ROOT/docs/$ref" ]] || [[ -f "$ROOT/$ref" ]]; then
      pass "$basename -> $ref"
    else
      fail "$basename references $ref but file not found"
    fi
  done < <(awk '/^```/{skip=!skip; next} !skip{print}' "$md" \
    | grep -oE '[A-Za-z0-9_.-]+\.schema\.json' | sort -u)
done
echo ""

# ── Check 5: All *.schema.json files are valid JSON ──

echo "--- Schema JSON validation ---"
for schema in "$CONTRACTS_DIR"/*.schema.json; do
  [[ -f "$schema" ]] || continue
  basename="$(basename "$schema")"
  if jq empty "$schema" 2>/dev/null; then
    pass "$basename is valid JSON"
  else
    fail "$basename is not valid JSON"
  fi
done
# Also check schemas/ dir at repo root
if [[ -d "$ROOT/schemas" ]]; then
  for schema in "$ROOT/schemas"/*.schema.json; do
    [[ -f "$schema" ]] || continue
    basename="$(basename "$schema")"
    if jq empty "$schema" 2>/dev/null; then
      pass "schemas/$basename is valid JSON"
    else
      fail "schemas/$basename is not valid JSON"
    fi
  done
fi
echo ""

# ── Check 6: All *.json example files are valid JSON ──

echo "--- Example JSON validation ---"
for example in "$CONTRACTS_DIR"/*.example.json; do
  [[ -f "$example" ]] || continue
  basename="$(basename "$example")"
  if jq empty "$example" 2>/dev/null; then
    pass "$basename is valid JSON"
  else
    fail "$basename is not valid JSON"
  fi
done
echo ""

# ── Check 7: Orphan detection — files on disk not in INDEX.md ──

echo "--- Orphan detection ---"
if [[ -f "$INDEX" ]]; then
  for contract in "$CONTRACTS_DIR"/*; do
    [[ -f "$contract" ]] || continue
    basename="$(basename "$contract")"
    if grep -q "$basename" "$INDEX" 2>/dev/null; then
      pass "$basename catalogued in INDEX.md"
    else
      warn "$basename exists on disk but not in INDEX.md"
    fi
  done
fi
echo ""

# ── Summary ──

echo "=== Summary ==="
echo "Failures: $failures"
echo "Warnings: $warnings"

if [[ "$failures" -gt 0 ]]; then
  echo ""
  echo "Contract compatibility check failed."
  exit 1
fi

echo ""
echo "Contract compatibility check passed."
