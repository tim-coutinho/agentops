#!/usr/bin/env bash
# pre-commit-gate.sh — cyclomatic complexity gate for cli/internal/
#
# Fails if any production function in cli/internal/ has CC >= 8.
# Run directly or via .git/hooks/pre-commit.
#
# Soft dependency on gocyclo: warns but does not block if not installed.
# Install: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
#
# Trigger: only runs when Go files in cli/internal/ are staged.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

INTERNAL_DIR="cli/internal"
CC_THRESHOLD=7  # fail if CC > 7 (i.e., CC >= 8)

# Only run when Go files in cli/internal/ are staged.
staged_internal=$(git diff --cached --name-only 2>/dev/null \
    | grep "^${INTERNAL_DIR}/.*\.go$" \
    | grep -v '_test\.go$' \
    || true)

if [[ -z "$staged_internal" ]]; then
    exit 0
fi

# Soft dependency: warn but do not block if gocyclo is not installed.
if ! command -v gocyclo >/dev/null 2>&1; then
    echo "WARNING: gocyclo not installed; skipping complexity gate."
    echo "  Install: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest"
    exit 0
fi

# Scan ALL production Go files in cli/internal/ (not just staged files).
# This prevents staged changes from introducing regressions elsewhere.
mapfile -t target_files < <(
    find "$INTERNAL_DIR" -name '*.go' ! -name '*_test.go' -type f 2>/dev/null \
    | sort \
    || true
)

if [[ ${#target_files[@]} -eq 0 ]]; then
    exit 0
fi

echo "pre-commit: checking cyclomatic complexity in cli/internal/ (threshold: CC >= 8)..."

violations=$(gocyclo -over "$CC_THRESHOLD" "${target_files[@]}" 2>/dev/null || true)

if [[ -n "$violations" ]]; then
    echo ""
    echo "BLOCKED: functions with CC >= 8 found in cli/internal/:"
    echo "$violations"
    echo ""
    echo "Refactor the above functions before committing."
    echo "  Tip: Run 'gocyclo -over $CC_THRESHOLD $INTERNAL_DIR' to check locally."
    exit 1
fi

echo "  ok  complexity gate (no CC >= 8 in cli/internal/)"
exit 0
