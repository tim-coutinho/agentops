#!/usr/bin/env bash
# Smoke test for skill invocation via Claude CLI
# Tests that skills can be triggered and produce expected outputs

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../claude-code" && pwd)/test-helpers.sh"

# Guard: skip if Claude CLI not available
if ! command -v claude &> /dev/null; then
    echo "SKIP: claude CLI not found in PATH"
    exit 0
fi

passed=0
failed=0
skipped=0

echo "═══════════════════════════════════════════"
echo "Skill Invocation Smoke Tests"
echo "═══════════════════════════════════════════"
echo ""

# Create isolated test environment
TEST_PROJECT=$(create_test_project)
trap 'cleanup_test_project "$TEST_PROJECT"' EXIT

cd "$TEST_PROJECT"

# Set MAX_TURNS for all tests
export MAX_TURNS=3

# Test 1: /status skill
echo "Test 1: /status skill"
LOG_FILE=$(run_claude_json "/status" 120) || true

test_passed=true
if ! assert_skill_triggered "$LOG_FILE" "status" "Status skill triggered"; then
    test_passed=false
fi

if $test_passed; then
    ((passed++)) || true
else
    ((failed++)) || true
fi
echo ""

# Test 2: /knowledge skill
echo "Test 2: /knowledge skill"
LOG_FILE=$(run_claude_json "/knowledge" 120) || true

test_passed=true
if ! assert_skill_triggered "$LOG_FILE" "knowledge" "Knowledge skill triggered"; then
    test_passed=false
fi

if $test_passed; then
    ((passed++)) || true
else
    ((failed++)) || true
fi
echo ""

# Test 3: /research skill
echo "Test 3: /research skill"
LOG_FILE=$(run_claude_json "/research what are the main components of this project?" 120) || true

test_passed=true
if ! assert_skill_triggered "$LOG_FILE" "research" "Research skill triggered"; then
    test_passed=false
fi

# Verify research creates artifacts in .agents/research/
if [[ -d ".agents/research" ]]; then
    research_files=$(find .agents/research -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$research_files" -gt 0 ]]; then
        echo -e "  ${GREEN}[PASS]${NC} Research artifacts created ($research_files files)"
    else
        echo -e "  ${YELLOW}[WARN]${NC} No research artifacts found (may be expected if skill failed)"
    fi
else
    echo -e "  ${YELLOW}[WARN]${NC} .agents/research directory not created"
fi

if $test_passed; then
    ((passed++)) || true
else
    ((failed++)) || true
fi
echo ""

print_summary "$passed" "$failed" "$skipped"
