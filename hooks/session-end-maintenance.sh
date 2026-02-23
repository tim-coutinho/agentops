#!/usr/bin/env bash
# SessionEnd: forge learnings + maturity maintenance (serialized via lock).

# Kill switch
[ "${AGENTOPS_HOOKS_DISABLED:-}" = "1" ] && exit 0

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
ROOT="$(cd "$ROOT" 2>/dev/null && pwd -P 2>/dev/null || printf '%s' "$ROOT")"
AO_DIR="$ROOT/.agents/ao"
LOCK_FILE="$AO_DIR/session-end-heavy.lock"

mkdir -p "$AO_DIR" 2>/dev/null || true

AO_TIMEOUT_BIN="timeout"
command -v "$AO_TIMEOUT_BIN" >/dev/null 2>&1 || AO_TIMEOUT_BIN="gtimeout"

run_ao_quick() {
    local seconds="$1"; shift
    if command -v "$AO_TIMEOUT_BIN" >/dev/null 2>&1; then
        "$AO_TIMEOUT_BIN" "$seconds" ao "$@" 2>/dev/null
        return $?
    fi
    ao "$@" 2>/dev/null
}

run_maintenance() {
    command -v ao >/dev/null 2>&1 || return 0

    run_ao_quick 6 forge transcript --last-session --queue --quiet || true
    run_ao_quick 4 maturity --scan || true

    if [ "${AGENTOPS_EVICTION_DISABLED:-0}" != "1" ]; then
        run_ao_quick 4 maturity --expire --archive || true
        run_ao_quick 4 maturity --evict --archive || true
    fi
}

# Serialize across processes
if command -v flock >/dev/null 2>&1; then
    exec 9>"$LOCK_FILE" || exit 0
    flock -n 9 || exit 0
    run_maintenance
    exit 0
fi

# Fallback lock
LOCK_DIR="${LOCK_FILE}.d"
if mkdir "$LOCK_DIR" 2>/dev/null; then
    trap 'rmdir "$LOCK_DIR" 2>/dev/null || true' EXIT
    run_maintenance
fi

exit 0
