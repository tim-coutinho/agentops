#!/usr/bin/env bash
# Smoke test for pre-mortem-gate.sh hook
# Validates: script exists, syntax, kill switches, and basic flow

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source shared colors and helpers
source "${SCRIPT_DIR}/../lib/colors.sh"

errors=0
fail() { echo -e "${RED}  ✗${NC} $1"; ((errors++)) || true; }

HOOK="$REPO_ROOT/hooks/pre-mortem-gate.sh"

# =============================================================================
# Structural checks
# =============================================================================

log "Testing pre-mortem-gate.sh structure..."

# Test 1: Script exists
if [[ -f "$HOOK" ]]; then
    pass "Script exists"
else
    fail "Script not found at hooks/pre-mortem-gate.sh"
    echo -e "${RED}FAILED${NC} - Cannot continue without script"
    exit 1
fi

# Test 2: Script is executable or has valid bash syntax
if bash -n "$HOOK" 2>/dev/null; then
    pass "Valid bash syntax"
else
    fail "Bash syntax error in pre-mortem-gate.sh"
fi

# Test 3: Has kill switch (AGENTOPS_HOOKS_DISABLED)
if grep -q 'AGENTOPS_HOOKS_DISABLED' "$HOOK"; then
    pass "Kill switch: AGENTOPS_HOOKS_DISABLED"
else
    fail "Missing kill switch: AGENTOPS_HOOKS_DISABLED"
fi

# Test 4: Has specific gate bypass (AGENTOPS_SKIP_PRE_MORTEM_GATE)
if grep -q 'AGENTOPS_SKIP_PRE_MORTEM_GATE' "$HOOK"; then
    pass "Gate bypass: AGENTOPS_SKIP_PRE_MORTEM_GATE"
else
    fail "Missing gate bypass: AGENTOPS_SKIP_PRE_MORTEM_GATE"
fi

# Test 5: Worker exemption
if grep -q 'AGENTOPS_WORKER' "$HOOK"; then
    pass "Worker exemption: AGENTOPS_WORKER"
else
    fail "Missing worker exemption"
fi

# Test 6: Checks for jq availability (graceful degradation)
if grep -q 'command -v jq' "$HOOK"; then
    pass "jq availability check (graceful degradation)"
else
    fail "Missing jq availability check"
fi

# Test 7: Only gates Skill tool calls for crank
if grep -q 'TOOL_NAME.*Skill' "$HOOK" || grep -q '"Skill"' "$HOOK"; then
    pass "Gates Skill tool calls only"
else
    fail "Missing Skill tool name check"
fi

# Test 8: Checks for --skip-pre-mortem bypass
if grep -q 'skip-pre-mortem' "$HOOK"; then
    pass "--skip-pre-mortem bypass in args"
else
    fail "Missing --skip-pre-mortem bypass check"
fi

# =============================================================================
# Behavioral checks
# =============================================================================

log "Testing pre-mortem-gate.sh behavior..."

# Test 9: Kill switch disables gate (exit 0)
EXIT_CODE=0
echo '{"tool_name":"Skill","tool_input":{"skill":"crank","args":"ag-test"}}' | \
    AGENTOPS_HOOKS_DISABLED=1 bash "$HOOK" >/dev/null 2>&1 || EXIT_CODE=$?
if [[ $EXIT_CODE -eq 0 ]]; then
    pass "AGENTOPS_HOOKS_DISABLED=1 exits 0"
else
    fail "AGENTOPS_HOOKS_DISABLED=1 exits $EXIT_CODE (expected 0)"
fi

# Test 10: Worker exemption (exit 0)
EXIT_CODE=0
echo '{"tool_name":"Skill","tool_input":{"skill":"crank","args":"ag-test"}}' | \
    AGENTOPS_WORKER=1 bash "$HOOK" >/dev/null 2>&1 || EXIT_CODE=$?
if [[ $EXIT_CODE -eq 0 ]]; then
    pass "AGENTOPS_WORKER=1 exits 0"
else
    fail "AGENTOPS_WORKER=1 exits $EXIT_CODE (expected 0)"
fi

# Test 11: Non-Skill tool calls pass through (exit 0)
EXIT_CODE=0
echo '{"tool_name":"Bash","tool_input":{"command":"ls"}}' | \
    bash "$HOOK" >/dev/null 2>&1 || EXIT_CODE=$?
if [[ $EXIT_CODE -eq 0 ]]; then
    pass "Non-Skill tool passes through"
else
    fail "Non-Skill tool exits $EXIT_CODE (expected 0)"
fi

# Test 12: Non-crank skill calls pass through (exit 0)
EXIT_CODE=0
echo '{"tool_name":"Skill","tool_input":{"skill":"vibe","args":"recent"}}' | \
    bash "$HOOK" >/dev/null 2>&1 || EXIT_CODE=$?
if [[ $EXIT_CODE -eq 0 ]]; then
    pass "Non-crank skill passes through"
else
    fail "Non-crank skill exits $EXIT_CODE (expected 0)"
fi

# =============================================================================
# Summary
# =============================================================================
echo ""
echo -e "${BLUE}═══════════════════════════════════════════${NC}"

if [[ $errors -gt 0 ]]; then
    echo -e "${RED}FAILED${NC} - $errors errors"
    exit 1
else
    echo -e "${GREEN}PASSED${NC} - All pre-mortem-gate tests passed"
    exit 0
fi
