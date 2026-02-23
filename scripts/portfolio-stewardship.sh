#!/usr/bin/env bash
set -euo pipefail

# portfolio-stewardship.sh
# Reports open epic backlog and flags stale epics for pruning.
#
# Usage:
#   scripts/portfolio-stewardship.sh

if ! command -v bd >/dev/null 2>&1; then
  echo "bd CLI not available — skipping portfolio stewardship."
  exit 0
fi

echo "== Portfolio Stewardship Report =="
echo "Generated: $(date -Iseconds)"
echo

# Capture bd ready once to avoid multiple calls
ready_output=$(bd ready 2>/dev/null || true)

open_epics=$(echo "$ready_output" | grep -c '\[epic\]' || true)
open_tasks=$(echo "$ready_output" | grep -c '.' || true)

echo "Open epics: $open_epics"
echo "Open tasks: $open_tasks"
echo

echo "Top 20 open epics:"
echo "$ready_output" | grep '\[epic\]' | head -20 || echo "No open epics found."
echo
