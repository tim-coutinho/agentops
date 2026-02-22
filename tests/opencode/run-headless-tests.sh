#!/usr/bin/env bash
# run-headless-tests.sh — Test AgentOps skills in OpenCode headless mode
#
# Usage:
#   ./run-headless-tests.sh [--tier N] [--skill NAME] [--timeout SECS] [--attempts N] [--help]
#
# Runs OpenCode headless (opencode run) against AgentOps skills using the
# configured model (default: devstral/devstral-2). Captures output, exit code,
# duration, and generates per-skill logs.
#
# Brownian Ratchet: Each skill runs N attempts. Pass if ANY attempt succeeds.
# Local inference is free — more attempts = better capability assessment.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUTPUT_DIR="$REPO_ROOT/.agents/opencode-tests"
DATE=$(date +%Y-%m-%d)
MODEL="${OPENCODE_TEST_MODEL:-devstral/devstral-2}"
TIMEOUT=180
MAX_ATTEMPTS=3
TIER=""
SKILL_FILTER=""
COMPARE_MODE=false
COMPARE_MODELS=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Source assertion framework if available
ASSERTIONS_LOADED=false
if [[ -f "$SCRIPT_DIR/assertions.sh" ]]; then
    # shellcheck source=assertions.sh
    source "$SCRIPT_DIR/assertions.sh"
    ASSERTIONS_LOADED=true
fi
if [[ -f "$SCRIPT_DIR/skill-assertions.sh" ]]; then
    # shellcheck source=skill-assertions.sh
    source "$SCRIPT_DIR/skill-assertions.sh"
fi

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Options:
  --tier N          Run only tier N (1, 2, or 3)
  --skill NAME      Run only a specific skill
  --timeout SECS    Timeout per test attempt (default: 180)
  --attempts N      Attempts per skill (default: 3, brownian ratchet)
  --model MODEL     Model to use (default: devstral/devstral-2)
  --compare         Run against all configured models, produce comparison
  --models LIST     Comma-separated model list for comparison
  --help            Show this help

Brownian Ratchet:
  Each skill runs N attempts. Skill PASSES if ANY attempt succeeds.
  Local inference is free — more attempts = better capability assessment.

  Pass rate (e.g., 2/3) indicates reliability:
    3/3  = reliable    (skill works consistently)
    1/3  = flaky       (skill works but needs prompt tuning)
    0/3  = broken      (skill needs fixes or model can't handle it)

Environment:
  NODE_TLS_REJECT_UNAUTHORIZED  Set to 0 for self-signed certs (auto-set)
  OPENCODE_TEST_MODEL           Override default model
  OH_MY_OPENCODE_DISABLED       Set to 1 to disable oh-my-opencode hooks

Output:
  .agents/opencode-tests/<date>-<skill>.log            Best attempt output
  .agents/opencode-tests/<date>-<skill>-attempt-N.log   Per-attempt output
  .agents/opencode-tests/<date>-summary.txt             Test summary
  .agents/opencode-tests/<date>-model-comparison.txt    Model comparison (--compare)
EOF
    exit 0
}

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --tier) TIER="$2"; shift 2 ;;
        --skill) SKILL_FILTER="$2"; shift 2 ;;
        --timeout) TIMEOUT="$2"; shift 2 ;;
        --attempts) MAX_ATTEMPTS="$2"; shift 2 ;;
        --model) MODEL="$2"; shift 2 ;;
        --compare) COMPARE_MODE=true; shift ;;
        --models) COMPARE_MODELS="$2"; shift 2 ;;
        --help) usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

mkdir -p "$OUTPUT_DIR"

# TLS bypass for self-signed RunAI certs
export NODE_TLS_REJECT_UNAUTHORIZED=0

# Strip ANSI escape codes from output
strip_ansi() {
    sed 's/\x1b\[[0-9;]*m//g' | sed 's/\x1b\[0m//g' | sed 's/\r//g'
}

# Load prompt from external file or use inline default
load_prompt() {
    local skill="$1"
    local default_prompt="$2"
    local prompt_file="$SCRIPT_DIR/prompts/${skill}.txt"

    if [[ -f "$prompt_file" ]]; then
        cat "$prompt_file"
    else
        echo "$default_prompt"
    fi
}

# Run a single attempt of a skill test
# Returns 0 if attempt passed, 1 if failed
run_attempt() {
    local skill="$1"
    local prompt="$2"
    local attempt="$3"
    local logfile="$4"

    local exit_code=0
    timeout "$TIMEOUT" opencode run \
        -m "$MODEL" \
        --title "test-${skill}-attempt-${attempt}" \
        --dir "$REPO_ROOT" \
        "$prompt" \
        > "$logfile.raw" 2>&1 || exit_code=$?

    # Strip ANSI codes
    strip_ansi < "$logfile.raw" > "$logfile"
    rm -f "$logfile.raw"

    local output_size
    output_size=$(wc -c < "$logfile" | tr -d ' ')

    # Basic pass: exit 0 + >50 bytes
    if [[ $exit_code -eq 0 && $output_size -gt 50 ]]; then
        # Run content assertions if available
        if [[ "$ASSERTIONS_LOADED" == "true" ]] && type -t "assert_skill_${skill}" &>/dev/null; then
            if "assert_skill_${skill}" "$logfile" "$REPO_ROOT" 2>/dev/null; then
                return 0
            else
                return 1  # Assertions failed despite output
            fi
        fi
        return 0
    fi
    return 1
}

# Run a skill test with brownian ratchet retry loop
run_test() {
    local skill="$1"
    local default_prompt="$2"
    local tier="$3"

    # Load prompt (external file or inline default)
    local prompt
    prompt=$(load_prompt "$skill" "$default_prompt")

    printf "${BLUE}[T${tier}]${NC} Testing %-20s " "$skill"

    local overall_start
    overall_start=$(date +%s)

    local attempts_passed=0
    local best_bytes=0
    local best_attempt=0
    local best_logfile=""
    local total_duration=0
    local attempt_results=""

    for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
        local attempt_logfile="$OUTPUT_DIR/${DATE}-${skill}-attempt-${attempt}.log"
        local attempt_start
        attempt_start=$(date +%s)

        if run_attempt "$skill" "$prompt" "$attempt" "$attempt_logfile"; then
            attempts_passed=$((attempts_passed + 1))
            local bytes
            bytes=$(wc -c < "$attempt_logfile" | tr -d ' ')
            if [[ $bytes -gt $best_bytes ]]; then
                best_bytes=$bytes
                best_attempt=$attempt
                best_logfile="$attempt_logfile"
            fi
            attempt_results="${attempt_results}+"
        else
            attempt_results="${attempt_results}-"
        fi

        local attempt_end
        attempt_end=$(date +%s)
        total_duration=$((total_duration + attempt_end - attempt_start))

        # Short pause between attempts to avoid hammering
        if [[ $attempt -lt $MAX_ATTEMPTS ]]; then
            sleep 2
        fi
    done

    # Copy best attempt as the canonical log
    local canonical_logfile="$OUTPUT_DIR/${DATE}-${skill}.log"
    if [[ -n "$best_logfile" ]]; then
        cp "$best_logfile" "$canonical_logfile"
    elif [[ -f "$OUTPUT_DIR/${DATE}-${skill}-attempt-1.log" ]]; then
        cp "$OUTPUT_DIR/${DATE}-${skill}-attempt-1.log" "$canonical_logfile"
    fi

    # Determine overall status (ratchet: pass if ANY attempt succeeded)
    local status="FAIL"
    local color="$RED"
    local rate_color="$RED"
    if [[ $attempts_passed -eq $MAX_ATTEMPTS ]]; then
        status="PASS"
        color="$GREEN"
        rate_color="$GREEN"
    elif [[ $attempts_passed -gt 0 ]]; then
        status="FLAKY"
        color="$YELLOW"
        rate_color="$YELLOW"
    fi

    # Display result with pass rate
    printf "${color}%-8s${NC} [${rate_color}%d/%d${NC}] (%s) %ds\n" \
        "$status" "$attempts_passed" "$MAX_ATTEMPTS" "$attempt_results" "$total_duration"

    # Append to summary
    printf "%-20s %-8s tier=%d rate=%d/%d pattern=%s bytes=%d duration=%ds model=%s\n" \
        "$skill" "$status" "$tier" "$attempts_passed" "$MAX_ATTEMPTS" \
        "$attempt_results" "$best_bytes" "$total_duration" "$MODEL" \
        >> "$OUTPUT_DIR/${DATE}-summary.txt"

    # Return overall status for counting
    if [[ $attempts_passed -gt 0 ]]; then
        return 0
    fi
    return 1
}

# Define test cases
# Format: "skill|prompt|tier"
declare -a TESTS=()

# Tier 1: Should work (read-only / tool-independent)
TIER1_TESTS=(
    "status|Load the status skill and run it. Show current project status.|1"
    "knowledge|Load the knowledge skill and search for 'council patterns'. Show results.|1"
    "complexity|Load the complexity skill and analyze the file skills/council/SKILL.md for complexity.|1"
    "doc|Load the doc skill and check documentation coverage for skills/research/ directory.|1"
    "handoff|Load the handoff skill and create a handoff summary for this test session.|1"
    "retro|Load the retro skill and extract learnings from the most recent work in .agents/learnings/.|1"
)

# Tier 2: Degraded mode (fallback possible)
TIER2_TESTS=(
    "research|Load the research skill and research the testing infrastructure in this repo. Use --auto mode. Do inline exploration only, do not spawn agents.|2"
    "plan|Load the plan skill and plan adding a README badge for test coverage. Use --auto mode.|2"
    "pre-mortem|Load the pre-mortem skill with --quick and validate the most recent plan in .agents/plans/.|2"
    "implement|Load the implement skill. Check what beads issues are ready to work on using bd ready.|2"
    "vibe|Load the vibe skill with --quick and review the file skills/status/SKILL.md for quality.|2"
    "bug-hunt|Load the bug-hunt skill and investigate any test failures in the tests/ directory.|2"
    "learn|Load the learn skill and save this insight: OpenCode headless mode requires NODE_TLS_REJECT_UNAUTHORIZED=0 for self-signed certs on RunAI clusters.|2"
    "trace|Load the trace skill and trace the decision history for the council architecture in this project.|2"
)

# Tier 3: Expected failure (hard blockers)
TIER3_TESTS=(
    "council|Load the council skill and validate skills/status/SKILL.md using multi-model consensus.|3"
    "crank|Load the crank skill. Show what epic would need to be cranked.|3"
    "swarm|Load the swarm skill and describe what it would do to spawn 2 workers.|3"
    "rpi|Load the rpi skill. Show what the full RPI lifecycle would look like for this project.|3"
    "codex-team|Load the codex-team skill and describe what it would do to spawn 2 codex agents.|3"
)

# Filter tests by tier
for test in "${TIER1_TESTS[@]}"; do
    if [[ -z "$TIER" || "$TIER" == "1" ]]; then
        TESTS+=("$test")
    fi
done
for test in "${TIER2_TESTS[@]}"; do
    if [[ -z "$TIER" || "$TIER" == "2" ]]; then
        TESTS+=("$test")
    fi
done
for test in "${TIER3_TESTS[@]}"; do
    if [[ -z "$TIER" || "$TIER" == "3" ]]; then
        TESTS+=("$test")
    fi
done

# Model comparison mode
run_comparison() {
    local models=()

    if [[ -n "$COMPARE_MODELS" ]]; then
        IFS=',' read -ra models <<< "$COMPARE_MODELS"
    else
        # Auto-detect from opencode config
        local config_file="${HOME}/.config/opencode/opencode.json"
        if [[ -f "$config_file" ]] && command -v jq &>/dev/null; then
            while IFS= read -r provider; do
                while IFS= read -r model_id; do
                    models+=("${provider}/${model_id}")
                done < <(jq -r ".provider.\"${provider}\".models | keys[]" "$config_file" 2>/dev/null)
            done < <(jq -r '.provider | keys[]' "$config_file" 2>/dev/null)
        fi
    fi

    if [[ ${#models[@]} -lt 2 ]]; then
        echo "Error: Need at least 2 models for comparison. Found: ${models[*]:-none}"
        echo "Use --models 'model1,model2' or configure models in ~/.config/opencode/opencode.json"
        exit 1
    fi

    local comparison_file="$OUTPUT_DIR/${DATE}-model-comparison.txt"
    printf "%-20s" "Skill" > "$comparison_file"
    for m in "${models[@]}"; do
        printf " | %-20s" "$m" >> "$comparison_file"
    done
    printf "\n" >> "$comparison_file"
    printf "%s\n" "$(printf '%.0s-' {1..80})" >> "$comparison_file"

    echo ""
    echo "================================================================"
    echo " Model Comparison"
    echo " Models: ${models[*]}"
    echo " Tests:  ${#TESTS[@]}"
    echo " Attempts per test: $MAX_ATTEMPTS"
    echo "================================================================"
    echo ""

    for test_spec in "${TESTS[@]}"; do
        IFS='|' read -r skill prompt tier <<< "$test_spec"
        if [[ -n "$SKILL_FILTER" && "$skill" != "$SKILL_FILTER" ]]; then
            continue
        fi

        printf "%-20s" "$skill" >> "$comparison_file"

        for m in "${models[@]}"; do
            MODEL="$m"
            # Suppress individual test output in comparison mode
            local attempts_passed=0
            for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
                local attempt_logfile="$OUTPUT_DIR/${DATE}-${skill}-${m//\//-}-attempt-${attempt}.log"
                if run_attempt "$skill" "$(load_prompt "$skill" "$prompt")" "$attempt" "$attempt_logfile" 2>/dev/null; then
                    attempts_passed=$((attempts_passed + 1))
                fi
                [[ $attempt -lt $MAX_ATTEMPTS ]] && sleep 2
            done
            printf " | %d/%d %-14s" "$attempts_passed" "$MAX_ATTEMPTS" \
                "$(if [[ $attempts_passed -eq $MAX_ATTEMPTS ]]; then echo '(reliable)'; elif [[ $attempts_passed -gt 0 ]]; then echo '(flaky)'; else echo '(broken)'; fi)" \
                >> "$comparison_file"
            printf "${BLUE}[T${tier}]${NC} %-20s %-25s %d/%d\n" "$skill" "$m" "$attempts_passed" "$MAX_ATTEMPTS"
        done
        printf "\n" >> "$comparison_file"
    done

    echo ""
    echo "Comparison: $comparison_file"
}

# Handle comparison mode
if [[ "$COMPARE_MODE" == "true" ]]; then
    run_comparison
    exit 0
fi

# Header
echo ""
echo "================================================================"
echo " OpenCode Headless Skills Test (Brownian Ratchet)"
echo " Model:    $MODEL"
echo " Date:     $DATE"
echo " Repo:     $REPO_ROOT"
echo " Tests:    ${#TESTS[@]}"
echo " Attempts: ${MAX_ATTEMPTS} per skill"
echo " Timeout:  ${TIMEOUT}s per attempt"
echo "================================================================"
echo ""

# Clear summary
> "$OUTPUT_DIR/${DATE}-summary.txt"

# Run tests
pass=0
flaky=0
fail=0
total=0

for test_spec in "${TESTS[@]}"; do
    IFS='|' read -r skill prompt tier <<< "$test_spec"

    # Filter by skill name if specified
    if [[ -n "$SKILL_FILTER" && "$skill" != "$SKILL_FILTER" ]]; then
        continue
    fi

    total=$((total + 1))

    # run_test returns 0 if any attempt passed, 1 if all failed
    if run_test "$skill" "$prompt" "$tier"; then
        last_status=$(tail -1 "$OUTPUT_DIR/${DATE}-summary.txt" | awk '{print $2}')
        case "$last_status" in
            PASS) pass=$((pass + 1)) ;;
            FLAKY) flaky=$((flaky + 1)) ;;
            *) pass=$((pass + 1)) ;;
        esac
    else
        fail=$((fail + 1))
    fi
done

# Footer
echo ""
echo "================================================================"
echo " Results: ${pass} PASS / ${flaky} FLAKY / ${fail} FAIL  (${total} total)"
echo " Summary: $OUTPUT_DIR/${DATE}-summary.txt"
echo " Logs:    $OUTPUT_DIR/${DATE}-*.log"
echo ""
echo " Ratchet: PASS = all attempts succeeded"
echo "          FLAKY = some attempts succeeded (needs prompt tuning)"
echo "          FAIL = no attempts succeeded (broken or model can't handle)"
echo "================================================================"
