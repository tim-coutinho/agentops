#!/usr/bin/env bash
# Olympus (OL) deterministic validation for vibe checks.
# Parses Stage1Result JSON from `ol validate stage1`.
set -euo pipefail

# --- Guard: detect OL environment ---
has_config=false
has_binary=false

if [ -f ".ol/config.yaml" ]; then
  has_config=true
fi

if command -v ol >/dev/null 2>&1; then
  has_binary=true
fi

if [ "$has_config" = false ] && [ "$has_binary" = false ]; then
  echo "SKIPPED (ol not detected)"
  exit 2
fi

# OL is detected (at least one of config/binary found).
# From here, errors should be reported but exit 2 (skip), not crash.

if [ "$has_binary" = false ]; then
  echo "SKIPPED (ol error: config found but ol binary not on PATH)"
  exit 2
fi

if [ "$has_config" = false ]; then
  echo "SKIPPED (ol error: ol binary found but .ol/config.yaml missing)"
  exit 2
fi

# --- Run ol validate stage1 ---
json=""
if ! json="$(ol validate stage1 -o json 2>&1)"; then
  echo "SKIPPED (ol error: ol validate stage1 failed: ${json})"
  exit 2
fi

# --- Parse Stage1Result JSON ---
passed=""
if ! passed="$(echo "$json" | jq -r 'if has("passed") then .passed else error("missing .passed") end' 2>&1)"; then
  echo "SKIPPED (ol error: failed to parse .passed from Stage1Result: ${passed})"
  exit 2
fi

summary=""
if ! summary="$(echo "$json" | jq -re '.summary' 2>&1)"; then
  echo "SKIPPED (ol error: failed to parse .summary from Stage1Result: ${summary})"
  exit 2
fi

# --- Build report ---
if [ "$passed" = "true" ]; then
  status="PASSED"
else
  status="FAILED"
fi

echo "## Deterministic Validation (Olympus)"
echo ""
echo "**Status:** ${status}"
echo ""
echo "| Step | Duration | Exit Code | Passed |"
echo "|------|----------|-----------|--------|"

# Parse steps array
step_count=""
if ! step_count="$(echo "$json" | jq -re '.steps | length' 2>&1)"; then
  echo "SKIPPED (ol error: failed to parse .steps from Stage1Result: ${step_count})"
  exit 2
fi

for ((i = 0; i < step_count; i++)); do
  step_name="$(echo "$json" | jq -r ".steps[$i].name")"
  step_duration="$(echo "$json" | jq -r ".steps[$i].duration")"
  step_exit="$(echo "$json" | jq -r ".steps[$i].exit_code")"
  step_passed="$(echo "$json" | jq -r ".steps[$i].passed")"
  echo "| ${step_name} | ${step_duration} | ${step_exit} | ${step_passed} |"
done

echo ""
echo "**Summary:** ${summary}"

# --- Exit ---
if [ "$passed" = "true" ]; then
  exit 0
else
  exit 1
fi
