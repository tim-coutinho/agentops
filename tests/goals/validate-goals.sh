#!/usr/bin/env bash
# Validate GOALS.yaml schema and fitness function integrity
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
GOALS_FILE="$REPO_ROOT/GOALS.yaml"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

errors=0

pass() { echo -e "${GREEN}  ✓${NC} $1"; }
fail() { echo -e "${RED}  ✗${NC} $1"; errors=$((errors + 1)); }

echo "Validating GOALS.yaml..."

# 1. File exists
if [[ ! -f "$GOALS_FILE" ]]; then
    fail "GOALS.yaml not found at $GOALS_FILE"
    exit 1
fi
pass "GOALS.yaml exists"

# 2. Valid YAML (use python since yq may not be installed)
if python3 -c "import yaml; yaml.safe_load(open('$GOALS_FILE'))" 2>/dev/null; then
    pass "Valid YAML syntax"
else
    fail "Invalid YAML syntax"
    exit 1
fi

# 3. Has version field
if grep -q '^version:' "$GOALS_FILE"; then
    pass "Has version field"
else
    fail "Missing version field"
fi

# 4. Has mission field
if grep -q '^mission:' "$GOALS_FILE"; then
    pass "Has mission field"
else
    fail "Missing mission field"
fi

# 5. Each goal has required fields (id, description, check, weight)
goal_count=0
missing_fields=0

# Extract goal IDs
while IFS= read -r id; do
    goal_count=$((goal_count + 1))
done < <(grep '^\s*- id:' "$GOALS_FILE" | sed 's/.*id:\s*//' | tr -d '"' | tr -d "'")

if [[ $goal_count -gt 0 ]]; then
    pass "Found $goal_count goals"
else
    fail "No goals found"
fi

# Check each goal block has description, check, weight
desc_count=$(grep -c '^\s*description:' "$GOALS_FILE" || true)
check_count=$(grep -c '^\s*check:' "$GOALS_FILE" || true)
weight_count=$(grep -c '^\s*weight:' "$GOALS_FILE" || true)

if [[ $desc_count -ge $goal_count ]]; then
    pass "All goals have description field"
else
    fail "Some goals missing description ($desc_count of $goal_count)"
fi

if [[ $check_count -ge $goal_count ]]; then
    pass "All goals have check field"
else
    fail "Some goals missing check ($check_count of $goal_count)"
fi

if [[ $weight_count -ge $goal_count ]]; then
    pass "All goals have weight field"
else
    fail "Some goals missing weight ($weight_count of $goal_count)"
fi

# 6. No duplicate IDs
dup_count=$(grep '^\s*- id:' "$GOALS_FILE" | sed 's/.*id:\s*//' | tr -d '"' | tr -d "'" | sort | uniq -d | wc -l | tr -d ' ')
if [[ $dup_count -eq 0 ]]; then
    pass "No duplicate goal IDs"
else
    fail "Found $dup_count duplicate goal IDs"
fi

# 7. Weights are numeric and in range 1-10
bad_weights=0
while IFS= read -r w; do
    w=$(echo "$w" | tr -d ' ')
    if ! [[ "$w" =~ ^[0-9]+$ ]] || [[ "$w" -lt 1 ]] || [[ "$w" -gt 10 ]]; then
        bad_weights=$((bad_weights + 1))
    fi
done < <(grep '^\s*weight:' "$GOALS_FILE" | sed 's/.*weight:\s*//' | tr -d '"')

if [[ $bad_weights -eq 0 ]]; then
    pass "All weights in range 1-10"
else
    fail "$bad_weights weights out of range"
fi

echo ""
if [[ $errors -eq 0 ]]; then
    echo -e "${GREEN}GOALS.yaml validation passed${NC}"
    exit 0
else
    echo -e "${RED}GOALS.yaml validation failed ($errors errors)${NC}"
    exit 1
fi
