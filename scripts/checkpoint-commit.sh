#!/usr/bin/env bash
# Usage: scripts/checkpoint-commit.sh <skill> <phase> "<message>"
# Commits .agents/<skill>/ artifacts. No-op if nothing changed.
# Used by orchestration skills (rpi, crank, evolve) for compaction resilience.
set -euo pipefail

SKILL="${1:?Usage: checkpoint-commit.sh <skill> <phase> <message>}"
PHASE="${2:?Usage: checkpoint-commit.sh <skill> <phase> <message>}"
MSG="${3:?Usage: checkpoint-commit.sh <skill> <phase> <message>}"

# Guard: workers must NOT commit (lead-only-commit rule)
if [ "${CRANK_WORKER:-}" = "true" ]; then
  echo "checkpoint-commit: skipped (worker mode)"
  exit 0
fi

git add ".agents/${SKILL}/" 2>/dev/null || true
if git diff --cached --quiet 2>/dev/null; then
  echo "checkpoint-commit: nothing to commit for ${SKILL}/${PHASE}"
  exit 0
fi

git commit -m "${SKILL}: ${PHASE} — ${MSG}"
echo "checkpoint-commit: committed ${SKILL}/${PHASE}"
