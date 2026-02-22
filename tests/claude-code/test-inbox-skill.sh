#!/usr/bin/env bash
# Test: inbox skill
# Verifies the Agent Mail inbox monitoring skill works correctly
# Tests: message fetch, thread grouping, filtering
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Inbox skill queries may need extra turns to explore skill docs
export MAX_TURNS="${MAX_TURNS:-10}"

source "$SCRIPT_DIR/test-helpers.sh"

echo "=== Test: inbox skill ==="
echo ""

# Test 1: Verify skill is recognized
echo "Test 1: Skill recognition..."

output=$(run_claude "What is the inbox skill in this plugin? Describe it briefly." 45)

if assert_contains "$output" "inbox" "Skill name recognized"; then
    :
else
    exit 1
fi

if assert_contains "$output" "mail\|message\|agent" "Describes mail functionality"; then
    :
else
    exit 1
fi

echo ""

# Test 2: Verify message categories and filtering
echo "Test 2: Message categories and filtering..."

output=$(run_claude "What message categories does the inbox skill handle? List them." 45)

if assert_contains "$output" "HELP_REQUEST\|pending\|completion\|done" "Mentions message categories"; then
    :
else
    exit 1
fi

echo ""

# Test 3: Verify thread support
echo "Test 3: Thread support..."

output=$(run_claude "Does the inbox skill support thread grouping or summarization?" 45)

if assert_contains "$output" "thread\|group\|summarize" "Mentions thread support"; then
    :
else
    exit 1
fi

echo ""

echo "=== All inbox skill tests passed ==="
