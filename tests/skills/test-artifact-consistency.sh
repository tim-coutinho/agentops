#!/usr/bin/env bash
# test-artifact-consistency.sh
# Validates artifact-consistency script behavior:
# - fenced code exclusion
# - placeholder filtering
# - verbose output shape
# - allowlist filtering

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CHECK_SCRIPT="$REPO_ROOT/skills/flywheel/scripts/artifact-consistency.sh"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

PASS=0
FAIL=0

assert_eq() {
  local desc="$1"
  local actual="$2"
  local expected="$3"
  if [[ "$actual" == "$expected" ]]; then
    echo -e "  ${GREEN}✓${NC} $desc"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $desc (expected '$expected', got '$actual')"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local desc="$1"
  local haystack="$2"
  local needle="$3"
  if grep -q --fixed-strings "$needle" <<< "$haystack"; then
    echo -e "  ${GREEN}✓${NC} $desc"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $desc (missing '$needle')"
    FAIL=$((FAIL + 1))
  fi
}

assert_not_contains() {
  local desc="$1"
  local haystack="$2"
  local needle="$3"
  if grep -q --fixed-strings "$needle" <<< "$haystack"; then
    echo -e "  ${RED}✗${NC} $desc (unexpected '$needle')"
    FAIL=$((FAIL + 1))
  else
    echo -e "  ${GREEN}✓${NC} $desc"
    PASS=$((PASS + 1))
  fi
}

metric() {
  local output="$1"
  local key="$2"
  awk -F= -v k="$key" '$1 == k { print $2; exit }' <<< "$output"
}

run_in_fixture() {
  local fixture="$1"
  shift
  (
    cd "$fixture"
    "$CHECK_SCRIPT" "$@"
  )
}

echo "=== Artifact Consistency Tests ==="

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

# Test 1: fenced-code + placeholder filtering
mkdir -p "$TMP_DIR/t1/.agents/research"
touch "$TMP_DIR/t1/.agents/exists.md"
cat > "$TMP_DIR/t1/.agents/research/check.md" <<'EOF'
Literal valid ref: .agents/exists.md
Template ref to ignore: .agents/research/YYYY-template.md

```bash
# fenced code ref should be ignored:
cat .agents/never-created.md
```
EOF

OUT_1="$(run_in_fixture "$TMP_DIR/t1" --no-allowlist)"
assert_eq "fenced/placeholder total refs" "$(metric "$OUT_1" "TOTAL_REFS")" "1"
assert_eq "fenced/placeholder broken refs" "$(metric "$OUT_1" "BROKEN_REFS")" "0"
assert_eq "fenced/placeholder status" "$(metric "$OUT_1" "STATUS")" "Healthy"

# Test 2: verbose output format for broken refs
mkdir -p "$TMP_DIR/t2/.agents/research"
cat > "$TMP_DIR/t2/.agents/research/check.md" <<'EOF'
Broken ref: .agents/missing.md
EOF

OUT_2="$(run_in_fixture "$TMP_DIR/t2" --no-allowlist --verbose)"
assert_eq "verbose broken refs count" "$(metric "$OUT_2" "BROKEN_REFS")" "1"
assert_contains "verbose line includes source -> target" "$OUT_2" "BROKEN_REF=.agents/research/check.md -> .agents/missing.md"

# Test 3: allowlist filtering
mkdir -p "$TMP_DIR/t3/.agents/research"
cat > "$TMP_DIR/t3/.agents/research/check.md" <<'EOF'
Broken ref but allowlisted: .agents/missing.md
EOF
cat > "$TMP_DIR/t3/allowlist.txt" <<'EOF'
* -> .agents/missing.md
EOF

OUT_3="$(run_in_fixture "$TMP_DIR/t3" --allowlist "$TMP_DIR/t3/allowlist.txt" --verbose)"
assert_eq "allowlist suppresses broken refs" "$(metric "$OUT_3" "BROKEN_REFS")" "0"
assert_not_contains "allowlist suppresses verbose BROKEN_REF lines" "$OUT_3" "BROKEN_REF="

echo ""
TOTAL=$((PASS + FAIL))
echo "=== Results: $PASS/$TOTAL passed ==="

if [[ "$FAIL" -gt 0 ]]; then
  echo -e "${RED}$FAIL test(s) failed${NC}"
  exit 1
fi

echo -e "${GREEN}All tests passed${NC}"
exit 0
