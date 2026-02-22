#!/usr/bin/env bash
set -euo pipefail

# Check that no Go function in the given directory exceeds a complexity threshold.
# Unlike check-go-complexity.sh (delta-only), this checks ALL functions — not just changed ones.

usage() {
  cat <<USAGE
Usage: $0 [--dir <path>] [--threshold <n>]

Checks that no non-test Go function exceeds the given cyclomatic complexity threshold.

Options:
  --dir        Directory to scan (default: cli/)
  --threshold  Max allowed complexity (default: 10)
USAGE
}

DIR="cli/"
THRESHOLD=10

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)       DIR="$2"; shift 2 ;;
    --threshold) THRESHOLD="$2"; shift 2 ;;
    -h|--help)   usage; exit 0 ;;
    *)           echo "Unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if ! command -v gocyclo >/dev/null 2>&1; then
  echo "gocyclo not found; install with: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest" >&2
  exit 2
fi

if [[ ! -d "$DIR" ]]; then
  echo "Directory $DIR does not exist; skipping." >&2
  exit 0
fi

# Find all non-test Go files
mapfile -t GO_FILES < <(find "$DIR" -name '*.go' ! -name '*_test.go' -type f 2>/dev/null || true)

if [[ ${#GO_FILES[@]} -eq 0 ]]; then
  echo "No non-test Go files found in $DIR."
  exit 0
fi

# Run gocyclo and capture functions over threshold
VIOLATIONS=$(gocyclo -over "$THRESHOLD" "${GO_FILES[@]}" 2>/dev/null || true)

if [[ -z "$VIOLATIONS" ]]; then
  echo "All functions in $DIR are below complexity $THRESHOLD."
  exit 0
fi

echo "ERROR: Functions exceeding complexity $THRESHOLD in $DIR:"
echo "$VIOLATIONS"
exit 1
