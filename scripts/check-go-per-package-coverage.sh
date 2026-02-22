#!/usr/bin/env bash
set -euo pipefail

# Check that every Go package under cli/ meets a minimum test coverage floor.
# Unlike go-coverage-floor (average only), this enforces a PER-PACKAGE minimum.

usage() {
  cat <<USAGE
Usage: $0 [--floor <percent>] [--track-delta]

Checks that every Go package under cli/ has test coverage >= floor.

Options:
  --floor        Minimum coverage per package (default: 60)
  --track-delta  Save current measurement and compare against previous snapshot.
                 Snapshots stored in .agents/evolve/coverage-snapshots/.
                 Reports per-package delta (e.g., "cmd/ao: 51.8% -> 53.2% (+1.4%)")
USAGE
}

FLOOR=60
TRACK_DELTA=false
SNAPSHOT_DIR=".agents/evolve/coverage-snapshots"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --floor)       FLOOR="$2"; shift 2 ;;
    --track-delta) TRACK_DELTA=true; shift ;;
    -h|--help)     usage; exit 0 ;;
    *)             echo "Unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ ! -d "cli" ]]; then
  echo "No cli/ directory found; skipping."
  exit 0
fi

# Run tests with coverage and capture output
OUTPUT=$(cd cli && go test -cover ./... 2>&1) || true

FAILURES=0

# Temporary file to collect current coverage data (for --track-delta)
CURRENT_DATA=$(mktemp)
trap 'rm -f "$CURRENT_DATA"' EXIT

while IFS= read -r line; do
  # Match lines like: ok  github.com/boshu2/agentops/cli/internal/config  0.5s  coverage: 98.5% of statements
  if [[ "$line" =~ coverage:\ ([0-9]+(\.[0-9]+)?)% ]]; then
    COV="${BASH_REMATCH[1]}"
    PKG=$(echo "$line" | awk -F'\t' '{print $2}' | sed 's/^ *//;s/ *$//')
    echo "$PKG $COV" >> "$CURRENT_DATA"
    # Compare using awk for float comparison
    BELOW=$(awk -v cov="$COV" -v floor="$FLOOR" 'BEGIN { print (cov < floor) ? "1" : "0" }')
    if [[ "$BELOW" == "1" ]]; then
      echo "FAIL: $PKG coverage ${COV}% < ${FLOOR}%"
      FAILURES=$((FAILURES + 1))
    fi
  fi
  # Also catch packages with [no test files]
  if [[ "$line" =~ \[no\ test\ files\] ]]; then
    PKG=$(echo "$line" | awk -F'\t' '{print $2}' | sed 's/^ *//;s/ *$//')
    echo "WARN: $PKG has no test files (counted as 0% coverage)"
    # Don't fail on no-test-files packages for now — just warn
  fi
done <<< "$OUTPUT"

if [[ "$FAILURES" -gt 0 ]]; then
  echo
  echo "ERROR: $FAILURES package(s) below ${FLOOR}% coverage floor."
  # Still do track-delta even on failure (informational)
fi

# --track-delta: save snapshot and compare against previous
if [[ "$TRACK_DELTA" == "true" ]]; then
  mkdir -p "$SNAPSHOT_DIR"

  # Find the most recent previous snapshot
  PREV_SNAPSHOT=""
  if [[ -d "$SNAPSHOT_DIR" ]]; then
    PREV_SNAPSHOT=$(find "$SNAPSHOT_DIR" -name '*.tsv' -type f 2>/dev/null | sort | tail -1 || true)
  fi

  # Save current snapshot
  TIMESTAMP=$(date -u +%Y%m%d-%H%M%S)
  GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
  CURRENT_SNAPSHOT="$SNAPSHOT_DIR/${TIMESTAMP}-${GIT_SHA}.tsv"
  # Header
  echo "# Coverage snapshot: $TIMESTAMP (git: $GIT_SHA)" > "$CURRENT_SNAPSHOT"
  sort "$CURRENT_DATA" >> "$CURRENT_SNAPSHOT"

  echo ""
  echo "=== Coverage Delta ==="

  if [[ -n "$PREV_SNAPSHOT" && -f "$PREV_SNAPSHOT" ]]; then
    PREV_LABEL=$(head -1 "$PREV_SNAPSHOT" | sed 's/^# Coverage snapshot: //')
    echo "Comparing against: $PREV_LABEL"
    echo ""

    # Build associative arrays and compare
    # Read previous snapshot into temp file (skip comment lines)
    PREV_DATA=$(mktemp)
    grep -v '^#' "$PREV_SNAPSHOT" > "$PREV_DATA" 2>/dev/null || true

    # For each current package, look up previous value and compute delta
    while read -r pkg cov; do
      prev_cov=$(awk -v p="$pkg" '$1 == p { print $2 }' "$PREV_DATA" 2>/dev/null || true)
      if [[ -n "$prev_cov" ]]; then
        delta=$(awk -v c="$cov" -v p="$prev_cov" 'BEGIN { printf "%+.1f", c - p }')
        # Strip the full module path down to a short name
        short_pkg=$(echo "$pkg" | sed 's|.*/cli/||; s|.*/||')
        echo "  $short_pkg: ${prev_cov}% -> ${cov}% (${delta}%)"
      else
        short_pkg=$(echo "$pkg" | sed 's|.*/cli/||; s|.*/||')
        echo "  $short_pkg: ${cov}% (new package)"
      fi
    done < <(sort "$CURRENT_DATA")

    # Check for removed packages
    while read -r pkg prev_cov; do
      if ! grep -q "^$pkg " "$CURRENT_DATA" 2>/dev/null; then
        short_pkg=$(echo "$pkg" | sed 's|.*/cli/||; s|.*/||')
        echo "  $short_pkg: ${prev_cov}% -> (removed)"
      fi
    done < "$PREV_DATA"

    rm -f "$PREV_DATA"
  else
    echo "No previous snapshot found; baseline saved."
  fi

  echo "Snapshot saved: $CURRENT_SNAPSHOT"
fi

if [[ "$FAILURES" -gt 0 ]]; then
  exit 1
fi

echo "All packages meet ${FLOOR}% per-package coverage floor."
exit 0
