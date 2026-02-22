#!/usr/bin/env bash
# Release smoke test - verify all skills are loadable
# Usage: ./tests/release-smoke-test.sh [--full]
#
# Default: Fast verification (~30s) - checks components are registered
# --full:  Slow verification (~10min) - invokes each skill individually

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$SCRIPT_DIR/claude-code/test-helpers.sh"

# Logging (redefine to avoid conflict with macOS log command)
log() { echo -e "${BLUE}[TEST]${NC} $1"; }
pass() { echo -e "${GREEN}  ✓${NC} $1"; }
fail() { echo -e "${RED}  ✗${NC} $1"; }

# Expected counts — computed dynamically from skill directories
EXPECTED_SKILLS=$(find "$REPO_ROOT/skills" -maxdepth 2 -name SKILL.md -type f | wc -l | tr -d ' ')

# Parse args
FULL_TEST=false
[[ "${1:-}" == "--full" ]] && FULL_TEST=true
[[ "${1:-}" == "--help" ]] && { echo "Usage: $0 [--full]"; echo "  --full  Run slow individual tests (~10min)"; exit 0; }

echo ""
echo "═══════════════════════════════════════════"
echo "     AgentOps Release Smoke Test"
echo "═══════════════════════════════════════════"
echo ""

if $FULL_TEST; then
    # =========================================================================
    # FULL TEST: Individual invocation of each skill
    # =========================================================================
    log "Running FULL test (individual invocations)..."

    # Build skills array dynamically from skill directories
    SKILLS=()
    for skill_dir in "$REPO_ROOT"/skills/*/; do
        [[ -f "${skill_dir}SKILL.md" ]] && SKILLS+=("$(basename "$skill_dir")")
    done

    passed=0
    failed=0

    for skill in "${SKILLS[@]}"; do
        if timeout 45 claude -p "Invoke agentops:$skill skill" \
            --plugin-dir "$REPO_ROOT" --dangerously-skip-permissions --max-turns 3 >/dev/null 2>&1; then
            pass "$skill"
            ((passed++))
        else
            fail "$skill"
            ((failed++))
        fi
    done

    print_summary "$passed" "$failed" 0
    exit $((failed > 0))
fi

# =============================================================================
# FAST TEST: Single prompt to verify all components are registered
# =============================================================================
log "Running FAST test (registration check)..."
echo ""

# Create a prompt that asks Claude to list available agentops skills
PROMPT='List all available agentops skills. Format your response as:

SKILLS: [comma-separated list]
COUNTS: skills=N

Only list agentops: prefixed items. Be thorough - check the Skill tool for skills.'

log "Querying Claude for registered components..."

output=$(timeout 120 claude -p "$PROMPT" \
    --plugin-dir "$REPO_ROOT" \
    --dangerously-skip-permissions \
    --max-turns 5 2>&1) || {
    fail "Claude query failed"
    exit 1
}

# Parse the output
echo "$output"
echo ""

# Extract counts from output
skill_count=$(echo "$output" | grep -oE 'skills?[=:] ?[0-9]+' | grep -oE '[0-9]+' | head -1 || echo "0")

# Fallback: count comma-separated items if explicit count not found
if [[ -z "$skill_count" ]] || [[ "$skill_count" == "0" ]]; then
    skills_line=$(echo "$output" | grep -i "^SKILLS:" | head -1)
    if [[ -n "$skills_line" ]]; then
        skill_count=$(echo "$skills_line" | tr ',' '\n' | wc -l | tr -d ' ')
    fi
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo "Release Smoke Test Results"
echo -e "${BLUE}───────────────────────────────────────────${NC}"

passed=0
failed=0

# Check skills
if [[ "$skill_count" -ge "$EXPECTED_SKILLS" ]]; then
    pass "Skills: $skill_count found (expected $EXPECTED_SKILLS)"
    ((passed++)) || true
else
    fail "Skills: $skill_count found (expected $EXPECTED_SKILLS)"
    ((failed++)) || true
fi

echo -e "${BLUE}───────────────────────────────────────────${NC}"
echo -e "  Total:  ${GREEN}$passed passed${NC}, ${RED}$failed failed${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"

if [[ $failed -gt 0 ]]; then
    echo ""
    echo -e "${RED}RELEASE BLOCKED${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}RELEASE READY: All components registered${NC}"
exit 0
