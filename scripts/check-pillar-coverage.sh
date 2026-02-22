#!/usr/bin/env bash
# check-pillar-coverage.sh — Verify GOALS.yaml has goals for all 4 pillars.
set -uo pipefail

GOALS_FILE="${1:-GOALS.yaml}"
pillars=$(yq '.goals[].pillar' "$GOALS_FILE" | sort -u)

for p in knowledge-compounding validated-acceleration goal-driven-automation zero-friction-workflow; do
  if ! echo "$pillars" | grep -q "$p"; then
    echo "MISSING pillar: $p" >&2
    exit 1
  fi
done
