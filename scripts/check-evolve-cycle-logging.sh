#!/usr/bin/env bash
set -euo pipefail

# Validate evolve cycle-history.jsonl integrity.
# Checks: file exists (if evolve has run), entries have required fields,
# cycle numbers are monotonically increasing with no gaps.

HISTORY=".agents/evolve/cycle-history.jsonl"

# If no evolve directory exists, skip gracefully (evolve hasn't run yet)
if [[ ! -d ".agents/evolve" ]]; then
  echo "No .agents/evolve/ directory — evolve has not run yet. Skipping."
  exit 0
fi

# If evolve directory exists but no history file, check for fitness snapshots
# which would indicate cycles ran without logging (the exact bug we're catching)
if [[ ! -f "$HISTORY" ]]; then
  SNAPSHOT_COUNT=$(find .agents/evolve -name 'fitness-*-post.json' 2>/dev/null | wc -l | tr -d ' ')
  if [[ "$SNAPSHOT_COUNT" -gt 0 ]]; then
    echo "ERROR: Found $SNAPSHOT_COUNT post-cycle fitness snapshots but no cycle-history.jsonl."
    echo "This indicates evolve cycles ran without logging — the tracking bug this goal prevents."
    exit 1
  fi
  echo "No cycle-history.jsonl and no post-cycle snapshots. Evolve has not completed any cycles."
  exit 0
fi

# Validate each entry has required fields
REQUIRED_FIELDS='cycle goal_id result timestamp'
LINE_NUM=0
ERRORS=0
PREV_CYCLE=-1

while IFS= read -r line; do
  LINE_NUM=$((LINE_NUM + 1))

  # Skip empty lines
  [[ -z "$line" ]] && continue

  # Validate JSON
  if ! echo "$line" | jq empty 2>/dev/null; then
    echo "ERROR: Line $LINE_NUM is not valid JSON"
    ERRORS=$((ERRORS + 1))
    continue
  fi

  # Check required fields (goal_id or goal_ids)
  for field in cycle result timestamp; do
    VALUE=$(echo "$line" | jq -r ".$field // empty")
    if [[ -z "$VALUE" ]]; then
      echo "ERROR: Line $LINE_NUM missing required field: $field"
      ERRORS=$((ERRORS + 1))
    fi
  done

  # Check that either goal_id or goal_ids exists
  GOAL_ID=$(echo "$line" | jq -r '.goal_id // empty')
  GOAL_IDS=$(echo "$line" | jq -r '.goal_ids // empty')
  if [[ -z "$GOAL_ID" && -z "$GOAL_IDS" ]]; then
    echo "ERROR: Line $LINE_NUM missing both goal_id and goal_ids"
    ERRORS=$((ERRORS + 1))
  fi

  # Check cycle number monotonicity
  CYCLE=$(echo "$line" | jq -r '.cycle // -1')
  if [[ "$PREV_CYCLE" -ge 0 ]]; then
    EXPECTED=$((PREV_CYCLE + 1))
    if [[ "$CYCLE" -ne "$EXPECTED" ]]; then
      echo "ERROR: Cycle gap at line $LINE_NUM: expected cycle $EXPECTED, got $CYCLE"
      ERRORS=$((ERRORS + 1))
    fi
  fi
  PREV_CYCLE="$CYCLE"

done < "$HISTORY"

if [[ "$LINE_NUM" -eq 0 ]]; then
  echo "WARN: cycle-history.jsonl exists but is empty."
  exit 0
fi

if [[ "$ERRORS" -gt 0 ]]; then
  echo
  echo "ERROR: $ERRORS integrity issues found in cycle-history.jsonl ($LINE_NUM entries checked)."
  exit 1
fi

echo "cycle-history.jsonl OK: $LINE_NUM entries, cycles 1-$PREV_CYCLE, no gaps, all fields present."
exit 0
