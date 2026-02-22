#!/usr/bin/env bash
# tasks-sync.sh - Beads Light: sync tasks to/from .tasks.json
# Usage: tasks-sync.sh list|ready|show|claim|complete|add|import|lock-health
# Exit codes: 0=success, 1=failure. lock-health: 0=healthy, 1=unhealthy.

set -euo pipefail

TASKS_FILE="${TASKS_FILE:-.tasks.json}"
TASKS_LOCK="${TASKS_LOCK:-.tasks.lock}"
LOCK_TIMEOUT="${TASKS_LOCK_TIMEOUT:-10}"

LOCK_BACKEND=""
LOCK_FD=-1
LOCK_HELD=0

detect_lock_backend() {
    if command -v flock >/dev/null 2>&1; then
        echo "flock"
    else
        echo "pidfile"
    fi
}

is_pid_running() {
    local pid="$1"
    [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" >/dev/null 2>&1
}

acquire_lock_with_flock() {
    exec {LOCK_FD}>"$TASKS_LOCK"
    if ! flock -w "$LOCK_TIMEOUT" "$LOCK_FD"; then
        exec {LOCK_FD}>&- 2>/dev/null || true
        LOCK_FD=-1
        echo "Error: lock timeout (${LOCK_TIMEOUT}s) using flock on $TASKS_LOCK" >&2
        return 1
    fi
    return 0
}

acquire_lock_with_pidfile() {
    local waited=0
    while true; do
        if (set -o noclobber; printf "%s\n" "$$" > "$TASKS_LOCK") 2>/dev/null; then
            return 0
        fi

        local owner_pid=""
        owner_pid="$(cat "$TASKS_LOCK" 2>/dev/null || true)"

        # Stale or malformed lock file, clear and retry.
        if [[ -z "$owner_pid" ]] || ! is_pid_running "$owner_pid"; then
            rm -f "$TASKS_LOCK"
            continue
        fi

        if (( waited >= LOCK_TIMEOUT )); then
            echo "Error: lock timeout (${LOCK_TIMEOUT}s) waiting for $TASKS_LOCK (owner pid: $owner_pid)" >&2
            return 1
        fi

        sleep 1
        ((waited++))
    done
}

acquire_lock() {
    if (( LOCK_HELD == 1 )); then
        return 0
    fi

    LOCK_BACKEND="$(detect_lock_backend)"
    if [[ "$LOCK_BACKEND" == "flock" ]]; then
        acquire_lock_with_flock
    else
        acquire_lock_with_pidfile
    fi
    LOCK_HELD=1
}

release_lock() {
    if (( LOCK_HELD == 0 )); then
        return 0
    fi

    if [[ "$LOCK_BACKEND" == "flock" ]]; then
        if [[ "$LOCK_FD" =~ ^[0-9]+$ ]] && (( LOCK_FD >= 0 )); then
            flock -u "$LOCK_FD" 2>/dev/null || true
            exec {LOCK_FD}>&-
        fi
        LOCK_FD=-1
    else
        local owner_pid=""
        owner_pid="$(cat "$TASKS_LOCK" 2>/dev/null || true)"
        if [[ -z "$owner_pid" ]] || [[ "$owner_pid" == "$$" ]]; then
            rm -f "$TASKS_LOCK"
        fi
    fi

    LOCK_HELD=0
}

trap release_lock EXIT

# Initialize empty tasks file if needed
init_tasks() {
    if [[ ! -f "$TASKS_FILE" ]]; then
        echo "[]" > "$TASKS_FILE"
    fi
}

# List tasks (for demigods to read)
cmd_list() {
    init_tasks
    jq -r '.[] | "\(.id)\t\(.status)\t\(.owner // "none")\t\(.subject)"' "$TASKS_FILE"
}

# Get ready tasks (pending, no blockers, no owner)
cmd_ready() {
    init_tasks
    jq -r '.[] | select(.status == "pending" and (.owner == null or .owner == "")) | .id' "$TASKS_FILE"
}

# Show a specific task
cmd_show() {
    local task_id="$1"
    init_tasks
    jq ".[] | select(.id == \"$task_id\")" "$TASKS_FILE"
}

# Claim a task (set status=in_progress, owner=demigod-N)
cmd_claim() {
    local task_id="$1"
    local owner="${2:-$$}"

    acquire_lock
    init_tasks

    local tmp
    tmp="$(mktemp)"
    jq "map(if .id == \"$task_id\" then .status = \"in_progress\" | .owner = \"$owner\" else . end)" "$TASKS_FILE" > "$tmp"
    mv "$tmp" "$TASKS_FILE"
    release_lock

    echo "Claimed: $task_id by $owner"
}

# Complete a task
cmd_complete() {
    local task_id="$1"

    acquire_lock
    init_tasks

    local tmp
    tmp="$(mktemp)"
    jq "map(if .id == \"$task_id\" then .status = \"completed\" else . end)" "$TASKS_FILE" > "$tmp"
    mv "$tmp" "$TASKS_FILE"
    release_lock

    echo "Completed: $task_id"
}

# Add a task (for manual creation)
cmd_add() {
    local subject="$1"
    local description="${2:-}"

    acquire_lock
    init_tasks

    local max_id
    max_id="$(jq -r '.[].id' "$TASKS_FILE" | sort -n | tail -1)"
    local new_id
    new_id="$((${max_id:-0} + 1))"

    local tmp
    tmp="$(mktemp)"
    jq ". + [{\"id\": \"$new_id\", \"subject\": \"$subject\", \"description\": \"$description\", \"status\": \"pending\", \"owner\": null, \"blockedBy\": []}]" "$TASKS_FILE" > "$tmp"
    mv "$tmp" "$TASKS_FILE"
    release_lock

    echo "Created task #$new_id: $subject"
}

# Import tasks from stdin
cmd_import_json() {
    acquire_lock
    cat > "$TASKS_FILE"
    release_lock
    echo "Imported tasks to $TASKS_FILE"
}

# Verify lock health for preflight checks
cmd_lock_health() {
    local health_timeout="${TASKS_HEALTH_TIMEOUT:-2}"
    local original_timeout="$LOCK_TIMEOUT"
    LOCK_TIMEOUT="$health_timeout"

    if acquire_lock; then
        local backend="$LOCK_BACKEND"
        release_lock
        LOCK_TIMEOUT="$original_timeout"
        echo "status=healthy backend=${backend} lock_file=${TASKS_LOCK} timeout=${health_timeout}s"
        return 0
    fi

    local backend="${LOCK_BACKEND:-unknown}"
    LOCK_TIMEOUT="$original_timeout"
    echo "status=unhealthy backend=${backend} lock_file=${TASKS_LOCK} timeout=${health_timeout}s" >&2
    return 1
}

# Main
case "${1:-list}" in
    list)        cmd_list ;;
    ready)       cmd_ready ;;
    show)        cmd_show "$2" ;;
    claim)       cmd_claim "$2" "${3:-}" ;;
    complete)    cmd_complete "$2" ;;
    add)         cmd_add "$2" "${3:-}" ;;
    import)      cmd_import_json ;;
    lock-health) cmd_lock_health ;;
    *)
        echo "Usage: $0 list|ready|show|claim|complete|add|import|lock-health"
        exit 1
        ;;
esac
