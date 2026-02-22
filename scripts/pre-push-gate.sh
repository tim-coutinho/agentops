#!/usr/bin/env bash
# pre-push-gate.sh — lightweight validation before push
#
# Runs the minimum checks to prevent broken code from landing on main.
# Designed to be fast (~10-20s cached) while catching the failures that
# ci-local-release.sh would catch later.
#
# Checks:
#   1. Go build + vet (if cli/ changed)
#   2. Go race tests on changed packages (via validate-go-fast.sh)
#   3. Embedded hooks sync (cli/embedded/ matches hooks/)
#   4. Skill count sync
#
# Usage:
#   scripts/pre-push-gate.sh          # Run directly
#   (also called from .githooks/pre-push)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

errors=0
pass() { echo -e "${GREEN}  ok${NC}  $1"; }
fail() { echo -e "${RED}FAIL${NC}  $1"; errors=$((errors + 1)); }

echo "pre-push gate: validating before push..."

# --- 1. Go build + vet ---
if command -v go >/dev/null 2>&1 && [[ -f cli/go.mod ]]; then
    # Check if any Go files changed vs upstream
    go_changed=$(git diff --name-only '@{upstream}...HEAD' -- 'cli/*.go' 'cli/**/*.go' 'cli/go.mod' 'cli/go.sum' 2>/dev/null || true)
    if [[ -n "$go_changed" ]]; then
        if (cd cli && go build -o /dev/null ./cmd/ao 2>&1); then
            pass "go build"
        else
            fail "go build"
        fi
        if (cd cli && go vet ./... 2>&1); then
            pass "go vet"
        else
            fail "go vet"
        fi
    else
        pass "go build (no Go changes)"
    fi
fi

# --- 2. Go race tests on changed packages (120s timeout, warn-only) ---
if [[ -x scripts/validate-go-fast.sh ]]; then
    race_output="$(mktemp)"
    race_rc=0
    if command -v timeout >/dev/null 2>&1; then
        timeout 120 scripts/validate-go-fast.sh > "$race_output" 2>&1 || race_rc=$?
    elif command -v gtimeout >/dev/null 2>&1; then
        # macOS: coreutils installed via Homebrew provides gtimeout.
        gtimeout 120 scripts/validate-go-fast.sh > "$race_output" 2>&1 || race_rc=$?
    else
        scripts/validate-go-fast.sh > "$race_output" 2>&1 || race_rc=$?
    fi

    if [[ "$race_rc" -eq 0 ]]; then
        pass "go test -race (changed scope)"
    elif [[ "$race_rc" -eq 124 ]]; then
        # Exit code 124 = timeout reached. Warn but don't block.
        echo "  WARNING: race tests timed out after 120s — not blocking push"
        pass "go test -race (timed out, non-blocking)"
    else
        fail "go test -race (changed scope)"
        # Show failure details to help debugging.
        tail -20 "$race_output" 2>/dev/null | sed 's/^/    /'
    fi
    rm -f "$race_output"
fi

# --- 3. Embedded hooks sync ---
stale=0
for src in hooks/session-start.sh hooks/hooks.json; do
    embedded="cli/embedded/$src"
    if [[ -f "$src" ]] && [[ -f "$embedded" ]]; then
        if ! diff -q "$src" "$embedded" >/dev/null 2>&1; then
            stale=1
            break
        fi
    fi
done
if [[ "$stale" -eq 1 ]]; then
    fail "embedded hooks stale (run: cd cli && make sync-hooks)"
else
    pass "embedded hooks in sync"
fi

# --- 4. Skill count sync ---
if [[ -x scripts/sync-skill-counts.sh ]]; then
    if scripts/sync-skill-counts.sh --check >/dev/null 2>&1; then
        pass "skill counts in sync"
    else
        fail "skill counts out of sync (run: scripts/sync-skill-counts.sh)"
    fi
fi

# --- Summary ---
echo ""
if [[ $errors -gt 0 ]]; then
    echo -e "${RED}pre-push gate: BLOCKED ($errors failures)${NC}"
    exit 1
else
    echo -e "${GREEN}pre-push gate: passed${NC}"
    exit 0
fi
