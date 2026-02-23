#!/usr/bin/env bash
# Usage: scripts/log-telemetry.sh <skill> <event> [key=value...]
# Appends structured entry to .agents/ao/skill-telemetry.jsonl
set -euo pipefail

SKILL="${1:?Usage: log-telemetry.sh <skill> <event> [key=value...]}"
EVENT="${2:?Usage: log-telemetry.sh <skill> <event> [key=value...]}"
shift 2

TIMESTAMP=$(date -Iseconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S%z)
SCHEMA_VERSION=1

json_quote() {
  jq -Rn --arg s "$1" '$s'
}

# Build JSON object safely (all string values are JSON-escaped).
JSON="{\"schema_version\":${SCHEMA_VERSION},\"skill\":$(json_quote "${SKILL}"),\"event\":$(json_quote "${EVENT}"),\"timestamp\":$(json_quote "${TIMESTAMP}")"

# Append key=value pairs
for kv in "$@"; do
  key="${kv%%=*}"
  val="${kv#*=}"
  # Try to detect numeric values
  if [[ "$val" =~ ^-?[0-9]+$ ]]; then
    JSON="${JSON},$(json_quote "${key}"):${val}"
  elif [[ "$val" =~ ^-?[0-9]+\.[0-9]+$ ]]; then
    JSON="${JSON},$(json_quote "${key}"):${val}"
  else
    JSON="${JSON},$(json_quote "${key}"):$(json_quote "${val}")"
  fi
done

JSON="${JSON}}"

mkdir -p .agents/ao
echo "$JSON" >> .agents/ao/skill-telemetry.jsonl
