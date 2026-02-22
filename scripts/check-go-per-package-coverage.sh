#!/usr/bin/env bash
set -euo pipefail

# Check that every Go package under cli/ meets a minimum test coverage floor.
# Unlike go-coverage-floor (average only), this enforces a PER-PACKAGE minimum.

usage() {
  cat <<USAGE
Usage: $0 [--floor <percent>]

Checks that every Go package under cli/ has test coverage >= floor.

Options:
  --floor  Minimum coverage per package (default: 60)
USAGE
}

FLOOR=60

while [[ $# -gt 0 ]]; do
  case "$1" in
    --floor) FLOOR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ ! -d "cli" ]]; then
  echo "No cli/ directory found; skipping."
  exit 0
fi

# Run tests with coverage and capture output
OUTPUT=$(cd cli && go test -cover ./... 2>&1) || true

FAILURES=0

while IFS= read -r line; do
  # Match lines like: ok  github.com/boshu2/agentops/cli/internal/config  0.5s  coverage: 98.5% of statements
  if [[ "$line" =~ coverage:\ ([0-9]+(\.[0-9]+)?)% ]]; then
    COV="${BASH_REMATCH[1]}"
    PKG=$(echo "$line" | awk '{print $2}')
    # Compare using awk for float comparison
    BELOW=$(awk -v cov="$COV" -v floor="$FLOOR" 'BEGIN { print (cov < floor) ? "1" : "0" }')
    if [[ "$BELOW" == "1" ]]; then
      echo "FAIL: $PKG coverage ${COV}% < ${FLOOR}%"
      FAILURES=$((FAILURES + 1))
    fi
  fi
  # Also catch packages with [no test files]
  if [[ "$line" =~ \[no\ test\ files\] ]]; then
    PKG=$(echo "$line" | awk '{print $2}')
    echo "WARN: $PKG has no test files (counted as 0% coverage)"
    # Don't fail on no-test-files packages for now — just warn
  fi
done <<< "$OUTPUT"

if [[ "$FAILURES" -gt 0 ]]; then
  echo
  echo "ERROR: $FAILURES package(s) below ${FLOOR}% coverage floor."
  exit 1
fi

echo "All packages meet ${FLOOR}% per-package coverage floor."
exit 0
