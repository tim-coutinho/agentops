#!/usr/bin/env bash
# Test: Distributed mode checkpoint recovery
# Verifies:
# - CHECKPOINT message format and content
# - Replacement demigod context handoff
# - Partial work preservation
# - Orchestrator (crank) handling of CHECKPOINT
#
# References:
# - skills/crank/SKILL.md: Distributed Mode section, Step 8 (Handle Checkpoints)
# - skills/implement/SKILL.md: Distributed Mode Context Checkpoint section
# - skills/swarm/SKILL.md: Distributed Mode Architecture
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/test-helpers.sh"

# Override MAX_TURNS for complex checkpoint prompts
export MAX_TURNS=5

echo "=== Test: Distributed Mode Checkpoint Recovery ==="
echo ""
echo "This test validates checkpoint message handling for context exhaustion"
echo "in distributed mode (tmux + Agent Mail) orchestration."
echo ""

# Track test results
PASSED=0
FAILED=0

run_test() {
    local test_name="$1"
    local test_func="$2"

    echo "Running: $test_name"
    if $test_func; then
        PASSED=$((PASSED + 1))
        echo -e "  ${GREEN}[PASS]${NC} $test_name"
    else
        FAILED=$((FAILED + 1))
        echo -e "  ${RED}[FAIL]${NC} $test_name"
    fi
    echo ""
}

# Helper to run claude with retry on empty response
run_claude_retry() {
    local prompt="$1"
    local timeout="${2:-90}"
    local output
    local retries=2

    for ((i=0; i<retries; i++)); do
        output=$(run_claude "$prompt" "$timeout" 2>&1) || true
        if [[ -n "$output" ]] && [[ "$output" != *"Reached max turns"* ]]; then
            echo "$output"
            return 0
        fi
        sleep 2
    done
    echo "$output"
}

# =============================================================================
# Test 1: CHECKPOINT Message Format Knowledge
# Verify the skill documents describe checkpoint message format
# =============================================================================
test_checkpoint_message_format() {
    local prompt='In the agentops skills, what fields are included in a CHECKPOINT message sent by a demigod when context is exhausted? List the key fields.'

    output=$(run_claude_retry "$prompt" 90)

    # Verify key CHECKPOINT message fields are mentioned
    local found=0

    # Check for bead/issue reference
    if echo "$output" | grep -qiE "(bead|issue)"; then
        found=$((found + 1))
    fi

    # Check for reason/context high mention
    if echo "$output" | grep -qiE "(reason|context|high|exhaust)"; then
        found=$((found + 1))
    fi

    # Check for commit/progress mention
    if echo "$output" | grep -qiE "(commit|progress|partial|work)"; then
        found=$((found + 1))
    fi

    # Check for next steps/successor mention
    if echo "$output" | grep -qiE "(next|step|successor|remain|guidance)"; then
        found=$((found + 1))
    fi

    if [[ $found -ge 3 ]]; then
        return 0
    else
        echo "  Expected at least 3 of: bead, reason/context, commit/progress, next steps"
        echo "  Found $found matches"
        return 1
    fi
}

# =============================================================================
# Test 2: Context Exhaustion Trigger
# Verify the skill describes when CHECKPOINT should be sent
# =============================================================================
test_context_exhaustion_trigger() {
    local prompt='When should a demigod in distributed mode send a CHECKPOINT message? What context threshold triggers this?'

    output=$(run_claude_retry "$prompt" 90)

    # Verify context threshold is mentioned (80% or high context)
    if ! echo "$output" | grep -qiE "(80|context.*high|high.*context|exhaust|threshold|before.*exit)"; then
        echo "  Expected mention of context threshold (80%) or exhaustion"
        return 1
    fi

    return 0
}

# =============================================================================
# Test 3: Replacement Demigod Spawning
# Verify crank spawns replacement demigod with checkpoint context
# =============================================================================
test_replacement_demigod_spawn() {
    local prompt='In agentops distributed mode, when crank receives a CHECKPOINT message, how does it spawn a replacement demigod? What context does the replacement receive?'

    output=$(run_claude_retry "$prompt" 90)

    # Verify replacement spawn mechanism mentioned
    local found=0

    # Check for spawn/replacement mention
    if echo "$output" | grep -qiE "(spawn|replacement|fresh|new|successor)"; then
        found=$((found + 1))
    fi

    # Check for context passing
    if echo "$output" | grep -qiE "(commit|checkpoint|context|partial|resume|continue)"; then
        found=$((found + 1))
    fi

    # Check for issue/bead passing
    if echo "$output" | grep -qiE "(issue|bead|task|work)"; then
        found=$((found + 1))
    fi

    if [[ $found -ge 2 ]]; then
        return 0
    else
        echo "  Expected spawn mechanism and context passing"
        echo "  Found $found matches"
        return 1
    fi
}

# =============================================================================
# Test 4: Partial Work Preservation
# Verify partial commits are not lost during checkpoint
# =============================================================================
test_partial_work_preservation() {
    local prompt='In distributed mode checkpoint recovery, how is partial work preserved? What happens to commits made before context exhaustion?'

    output=$(run_claude_retry "$prompt" 90)

    # Verify partial work preservation mechanism
    if ! echo "$output" | grep -qiE "(commit|preserve|saved|git|partial|not.*lost|continue|resume)"; then
        echo "  Expected mention of commit/work preservation"
        return 1
    fi

    return 0
}

# =============================================================================
# Test 5: Orchestrator Message Handling
# Verify crank handles CHECKPOINT in inbox polling
# =============================================================================
test_orchestrator_checkpoint_handling() {
    local prompt='In crank skill distributed mode, what action does the orchestrator take when it receives a CHECKPOINT message from a demigod? List the steps.'

    output=$(run_claude_retry "$prompt" 90)

    # Verify orchestrator actions mentioned
    local found=0

    # Check for parsing/reading checkpoint
    if echo "$output" | grep -qiE "(parse|read|receive|fetch|inbox)"; then
        found=$((found + 1))
    fi

    # Check for spawning replacement
    if echo "$output" | grep -qiE "(spawn|replacement|new|fresh|demigod|agent)"; then
        found=$((found + 1))
    fi

    # Check for context/state handling
    if echo "$output" | grep -qiE "(commit|checkpoint|context|state|progress|resume)"; then
        found=$((found + 1))
    fi

    if [[ $found -ge 2 ]]; then
        return 0
    else
        echo "  Expected inbox handling and replacement spawning"
        echo "  Found $found matches"
        return 1
    fi
}

# =============================================================================
# Test 6: CHECKPOINT vs FAILED Distinction
# Verify checkpoint is distinct from failure
# =============================================================================
test_checkpoint_vs_failed() {
    local prompt='What is the difference between a CHECKPOINT message and a FAILED message in agentops distributed mode? When would each be sent?'

    output=$(run_claude_retry "$prompt" 90)

    # Verify distinction is clear
    local found=0

    # Check for checkpoint context/continuation aspect
    if echo "$output" | grep -qiE "(context|continu|resume|progress|partial|successor)"; then
        found=$((found + 1))
    fi

    # Check for failed/error aspect
    if echo "$output" | grep -qiE "(fail|error|cannot|unable|block|impossible)"; then
        found=$((found + 1))
    fi

    # Check for distinction being made
    if echo "$output" | grep -qiE "(different|distinct|versus|while|whereas|but|however)"; then
        found=$((found + 1))
    fi

    if [[ $found -ge 2 ]]; then
        return 0
    else
        echo "  Expected clear distinction between CHECKPOINT and FAILED"
        echo "  Found $found matches"
        return 1
    fi
}

# =============================================================================
# Test 7: Next Steps for Successor
# Verify checkpoint includes guidance for replacement
# =============================================================================
test_successor_guidance() {
    local prompt='What guidance should a CHECKPOINT message include for the successor demigod? What does "Next Steps for Successor" contain?'

    output=$(run_claude_retry "$prompt" 90)

    # Verify guidance content mentioned
    if ! echo "$output" | grep -qiE "(next|step|remain|continue|guidance|instruction|successor|todo|what.*left|incomplete)"; then
        echo "  Expected mention of successor guidance/next steps"
        return 1
    fi

    return 0
}

# =============================================================================
# Test 8: Checkpoint Commit Reference
# Verify checkpoint includes commit SHA for partial work
# =============================================================================
test_checkpoint_commit_reference() {
    local prompt='Does a CHECKPOINT message include a git commit SHA? Why is this important for the replacement demigod?'

    output=$(run_claude_retry "$prompt" 90)

    # Verify commit reference explained
    local found=0

    # Check for commit mention
    if echo "$output" | grep -qiE "(commit|sha|hash|git)"; then
        found=$((found + 1))
    fi

    # Check for why it matters
    if echo "$output" | grep -qiE "(partial|progress|work|change|state|resume|continue|reference|pick.*up)"; then
        found=$((found + 1))
    fi

    if [[ $found -ge 2 ]]; then
        return 0
    else
        echo "  Expected commit SHA and its importance"
        echo "  Found $found matches"
        return 1
    fi
}

# =============================================================================
# Test 9: Agent Mail Integration
# Verify checkpoint uses Agent Mail MCP tools
# =============================================================================
test_agent_mail_integration() {
    local prompt='What MCP tool does a demigod use to send a CHECKPOINT message in distributed mode? What parameters does it require?'

    output=$(run_claude_retry "$prompt" 90)

    # Verify Agent Mail tool mentioned
    if ! echo "$output" | grep -qiE "(send.*message|agent.*mail|mcp|message|project.*key|sender|subject)"; then
        echo "  Expected Agent Mail tool reference"
        return 1
    fi

    return 0
}

# =============================================================================
# Test 10: End-to-End Checkpoint Flow
# Verify understanding of full checkpoint recovery flow
# =============================================================================
test_checkpoint_flow_e2e() {
    local prompt='Describe the end-to-end flow when a demigod hits context exhaustion: from detecting high context, to sending CHECKPOINT, to crank spawning replacement, to the replacement continuing work.'

    output=$(run_claude_retry "$prompt" 120)

    # Verify flow components mentioned
    local found=0

    # Detection
    if echo "$output" | grep -qiE "(detect|context|high|80|threshold|monitor)"; then
        found=$((found + 1))
    fi

    # Send checkpoint
    if echo "$output" | grep -qiE "(send|checkpoint|message|agent.*mail)"; then
        found=$((found + 1))
    fi

    # Crank receives
    if echo "$output" | grep -qiE "(crank|orchestrat|receive|inbox|poll)"; then
        found=$((found + 1))
    fi

    # Spawn replacement
    if echo "$output" | grep -qiE "(spawn|replacement|new|fresh|successor)"; then
        found=$((found + 1))
    fi

    # Continue work
    if echo "$output" | grep -qiE "(continue|resume|pick.*up|complete|finish)"; then
        found=$((found + 1))
    fi

    if [[ $found -ge 4 ]]; then
        return 0
    else
        echo "  Expected full checkpoint flow (detection -> send -> receive -> spawn -> continue)"
        echo "  Found $found/5 components"
        return 1
    fi
}

# =============================================================================
# Run All Tests
# =============================================================================

echo "Test 1: CHECKPOINT Message Format"
run_test "test_checkpoint_message_format" test_checkpoint_message_format

echo "Test 2: Context Exhaustion Trigger"
run_test "test_context_exhaustion_trigger" test_context_exhaustion_trigger

echo "Test 3: Replacement Demigod Spawning"
run_test "test_replacement_demigod_spawn" test_replacement_demigod_spawn

echo "Test 4: Partial Work Preservation"
run_test "test_partial_work_preservation" test_partial_work_preservation

echo "Test 5: Orchestrator Checkpoint Handling"
run_test "test_orchestrator_checkpoint_handling" test_orchestrator_checkpoint_handling

echo "Test 6: CHECKPOINT vs FAILED Distinction"
run_test "test_checkpoint_vs_failed" test_checkpoint_vs_failed

echo "Test 7: Next Steps for Successor"
run_test "test_successor_guidance" test_successor_guidance

echo "Test 8: Checkpoint Commit Reference"
run_test "test_checkpoint_commit_reference" test_checkpoint_commit_reference

echo "Test 9: Agent Mail Integration"
run_test "test_agent_mail_integration" test_agent_mail_integration

echo "Test 10: End-to-End Checkpoint Flow"
run_test "test_checkpoint_flow_e2e" test_checkpoint_flow_e2e

# =============================================================================
# Summary
# =============================================================================
echo "========================================"
echo " Checkpoint Recovery Test Summary"
echo "========================================"
echo ""
echo "  Passed:  $PASSED"
echo "  Failed:  $FAILED"
echo ""

if [[ $FAILED -gt 0 ]]; then
    echo "STATUS: FAILED"
    exit 1
else
    echo "=== All checkpoint recovery tests passed ==="
    exit 0
fi
