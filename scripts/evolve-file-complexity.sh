#!/usr/bin/env bash
set -euo pipefail

# evolve-file-complexity.sh
# Reports top files by count of high-complexity functions.
# Informational only — does not fail the build.
#
# Used by /evolve to surface hotspot files (e.g., rpi_phased.go) early.

usage() {
  cat <<USAGE
Usage: $0 [--dir <path>] [--threshold <n>] [--top <n>]

Reports top files by count of high-complexity functions.
Informational only — does not fail.

Options:
  --dir        Directory to scan (default: cli/)
  --threshold  Complexity threshold to flag (default: 7)
  --top        Number of top files to report (default: 5)
USAGE
}

DIR="cli/"
THRESHOLD=7
TOP=5

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)       DIR="$2"; shift 2 ;;
    --threshold) THRESHOLD="$2"; shift 2 ;;
    --top)       TOP="$2"; shift 2 ;;
    -h|--help)   usage; exit 0 ;;
    *)           echo "Unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

if ! command -v gocyclo >/dev/null 2>&1; then
  echo "gocyclo not found; install with: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest"
  echo "Skipping per-file complexity analysis."
  exit 0
fi

if [[ ! -d "$DIR" ]]; then
  echo "Directory $DIR does not exist; skipping."
  exit 0
fi

mapfile -t GO_FILES < <(find "$DIR" -name '*.go' ! -name '*_test.go' -type f 2>/dev/null | sort || true)

if [[ ${#GO_FILES[@]} -eq 0 ]]; then
  echo "No non-test Go files found in $DIR."
  exit 0
fi

echo "=== Per-File Complexity Hotspots (threshold: CC >= ${THRESHOLD}, dir: ${DIR}) ==="

RESULTS=$(gocyclo -over "$THRESHOLD" "${GO_FILES[@]}" 2>/dev/null || true)

if [[ -z "$RESULTS" ]]; then
  echo "No functions exceed complexity threshold ${THRESHOLD} in ${DIR}."
  exit 0
fi

# gocyclo output: <complexity> <package> <function> <file:line:col>
# Count high-complexity functions per file, sort by count descending, show top N
echo "$RESULTS" | awk '{
  path = $4
  sub(/:[0-9]+:[0-9]+$/, "", path)
  count[path]++
  if ($1 + 0 > max[path] + 0) max[path] = $1
}
END {
  for (f in count) {
    printf "%d\t%d\t%s\n", count[f], max[f], f
  }
}' | sort -rn | head -"$TOP" | while IFS=$'\t' read -r count max_cc file; do
  echo "  ${file}: ${count} function(s) with CC >= ${THRESHOLD} (max CC=${max_cc})"
done

echo ""
echo "Top ${TOP} complexity hotspot files reported (informational only)."
exit 0
