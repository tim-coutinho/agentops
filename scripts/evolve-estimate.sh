#!/usr/bin/env bash
set -euo pipefail

# evolve-estimate.sh
# Estimates how many evolve cycles remain to clear violations for a given goal.
# Reads cycle-history.jsonl for average violations-per-cycle and runs the
# relevant check script to count current violations.

HISTORY_FILE=".agents/evolve/cycle-history.jsonl"

usage() {
  cat <<USAGE
Usage: $0 <goal-name> [--history <path>]

Estimates remaining evolve cycles to clear all violations for a given goal.

Counts current violations by running the relevant check script, then reads
cycle history to compute average violations fixed per cycle.

Arguments:
  <goal-name>   Goal ID (e.g., go-absolute-complexity-ceiling)

Options:
  --history  Path to cycle-history.jsonl (default: .agents/evolve/cycle-history.jsonl)
  -h, --help Show this help

Supported goals:
  go-absolute-complexity-ceiling  Cyclomatic complexity > 10 in cli/
  go-per-package-coverage-floor   Packages below 60% coverage in cli/

Examples:
  $0 go-absolute-complexity-ceiling
  $0 go-per-package-coverage-floor --history .agents/evolve/cycle-history.jsonl
USAGE
}

# Parse arguments
GOAL=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --history) HISTORY_FILE="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    -*)        echo "Unknown flag: $1" >&2; usage; exit 2 ;;
    *)
      if [[ -z "$GOAL" ]]; then
        GOAL="$1"; shift
      else
        echo "Unexpected argument: $1" >&2; usage; exit 2
      fi
      ;;
  esac
done

if [[ -z "$GOAL" ]]; then
  echo "ERROR: Goal name is required." >&2
  usage
  exit 2
fi

# Count current violations based on goal type
count_violations() {
  local goal="$1"
  case "$goal" in
    go-absolute-complexity-ceiling)
      if ! command -v gocyclo >/dev/null 2>&1; then
        echo "gocyclo not found; install with: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest" >&2
        exit 2
      fi
      if [[ ! -d "cli/" ]]; then
        echo "0"
        return
      fi
      local go_files
      mapfile -t go_files < <(find cli/ -name '*.go' ! -name '*_test.go' -type f 2>/dev/null || true)
      if [[ ${#go_files[@]} -eq 0 ]]; then
        echo "0"
        return
      fi
      local violations
      violations=$(gocyclo -over 10 "${go_files[@]}" 2>/dev/null || true)
      if [[ -z "$violations" ]]; then
        echo "0"
      else
        echo "$violations" | wc -l | tr -d ' '
      fi
      ;;
    go-per-package-coverage-floor)
      if [[ ! -d "cli" ]]; then
        echo "0"
        return
      fi
      local output
      output=$(cd cli && go test -cover ./... 2>&1) || true
      local failures=0
      while IFS= read -r line; do
        if [[ "$line" =~ coverage:\ ([0-9]+(\.[0-9]+)?)% ]]; then
          local cov="${BASH_REMATCH[1]}"
          local below
          below=$(awk -v cov="$cov" -v floor="60" 'BEGIN { print (cov < floor) ? "1" : "0" }')
          if [[ "$below" == "1" ]]; then
            failures=$((failures + 1))
          fi
        fi
      done <<< "$output"
      echo "$failures"
      ;;
    *)
      echo "ERROR: Unsupported goal '$goal'." >&2
      echo "Supported goals: go-absolute-complexity-ceiling, go-per-package-coverage-floor" >&2
      exit 2
      ;;
  esac
}

# Compute average violations fixed per cycle from history
compute_avg_rate() {
  local goal="$1"
  local history_file="$2"

  if [[ ! -f "$history_file" ]]; then
    echo ""
    return
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo ""
    return
  fi

  # Extract cycles matching this goal that have violations_fixed
  local rate
  rate=$(jq -r --arg g "$goal" '
    select(.goal_id == $g and .violations_fixed != null) | .violations_fixed
  ' "$history_file" 2>/dev/null | awk '{sum += $1; count++} END {
    if (count > 0) printf "%.1f", sum/count; else print ""
  }')

  # If no violations_fixed data, we cannot compute a rate
  if [[ -z "$rate" ]]; then
    echo ""
    return
  fi

  echo "$rate"
}

# Main
VIOLATIONS=$(count_violations "$GOAL")

AVG_RATE=$(compute_avg_rate "$GOAL" "$HISTORY_FILE")

if [[ "$VIOLATIONS" == "0" ]]; then
  echo "Goal $GOAL: 0 violations -- already clear!"
  exit 0
fi

if [[ -n "$AVG_RATE" ]] && awk -v r="$AVG_RATE" 'BEGIN { exit (r > 0) ? 0 : 1 }'; then
  CYCLES=$(awk -v v="$VIOLATIONS" -v r="$AVG_RATE" 'BEGIN {
    c = v / r
    # Round up
    if (c != int(c)) c = int(c) + 1
    printf "%d", c
  }')
  echo "Goal $GOAL: $VIOLATIONS violations, ~$CYCLES cycles to clear at current rate ($AVG_RATE violations/cycle)"
else
  echo "Goal $GOAL: $VIOLATIONS violations (no cycle history available for rate estimation)"
fi

exit 0
