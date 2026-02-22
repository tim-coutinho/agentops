#!/usr/bin/env bash
# Run a single explicit skill request test
# Usage: ./run-test.sh <skill-name>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../claude-code/test-helpers.sh"

SKILL_NAME="${1:-}"

if [[ -z "$SKILL_NAME" ]]; then
    echo "Usage: $0 <skill-name>"
    echo "Example: $0 research"
    exit 1
fi

PROMPT_FILE="$SCRIPT_DIR/prompts/${SKILL_NAME}.txt"

if [[ ! -f "$PROMPT_FILE" ]]; then
    echo "Prompt file not found: $PROMPT_FILE"
    exit 1
fi

PROMPT=$(cat "$PROMPT_FILE")

echo "Testing explicit skill request: $SKILL_NAME"
echo "Prompt: $PROMPT"
echo ""

# Run Claude with stream-json output
LOG_FILE=$(run_claude_json "$PROMPT" 120) || true

# Verify the skill was triggered
assert_skill_triggered "$LOG_FILE" "$SKILL_NAME" "Skill triggered"

# Verify no premature tool calls
assert_no_premature_tools "$LOG_FILE" "No premature tools"

echo ""
echo "Log file: $LOG_FILE"
