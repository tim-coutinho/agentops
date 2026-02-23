#!/usr/bin/env bash
set -euo pipefail

# portfolio-stewardship.sh
# Reports open epic backlog and flags stale epics for pruning.
#
# Usage:
#   scripts/portfolio-stewardship.sh

if ! command -v bd >/dev/null 2>&1; then
  echo "bd CLI is required for portfolio stewardship."
  exit 1
fi

echo "== Portfolio Stewardship Report =="
echo "Generated: $(date -Iseconds)"
echo

open_epics=$(bd count --type epic --status open 2>/dev/null || echo 0)
open_tasks=$(bd count --type task --status open 2>/dev/null || echo 0)

echo "Open epics: $open_epics"
echo "Open tasks: $open_tasks"
echo

echo "Top 20 open epics:"
bd list --type epic --status open 2>/dev/null | head -20
echo

echo "Stale epic candidates (>30 days since update):"
bd stale --days 30 --type epic 2>/dev/null || echo "No stale epic candidates found."
