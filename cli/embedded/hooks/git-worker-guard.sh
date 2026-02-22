#!/bin/bash
# git-worker-guard.sh - PreToolUse hook: block git commit/push/add-all for swarm workers
# Workers write files only; lead commits. Prevents merge conflicts in parallel swarms.

# Kill switch
[ "${AGENTOPS_HOOKS_DISABLED:-}" = "1" ] && exit 0

# Read all of stdin (hook pipes JSON)
INPUT=$(cat)

# Extract tool_input.command from JSON
if command -v jq >/dev/null 2>&1; then
    CMD=$(echo "$INPUT" | jq -r '.tool_input.command // ""' 2>/dev/null)
else
    CMD=$(echo "$INPUT" | grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"command"[[:space:]]*:[[:space:]]*"//;s/"$//')
fi

# No command → exit silently
if [ -z "$CMD" ] || [ "$CMD" = "null" ]; then
    exit 0
fi

# Hot path: if command doesn't contain "git" at all, skip (<50ms)
case "$CMD" in
    *git*) ;;
    *) exit 0 ;;
esac

# Check if this is a dangerous git command
BLOCKED=0

# git commit or git push → always check identity
if echo "$CMD" | grep -qE 'git\s+(commit|push)'; then
    BLOCKED=1
fi

# git add -A / git add . / git add --all → block
if echo "$CMD" | grep -qE 'git\s+add\s+(-A|\.(\s|$|&&)|--all)'; then
    BLOCKED=1
fi

# Not a blocked command → allow
if [ "$BLOCKED" -eq 0 ]; then
    exit 0
fi

# Source hook-helpers from plugin install dir, not repo root (security: prevents malicious repo sourcing)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/hook-helpers.sh
. "$SCRIPT_DIR/../lib/hook-helpers.sh"

ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
ROOT="$(cd "$ROOT" 2>/dev/null && pwd -P 2>/dev/null || printf '%s' "$ROOT")"

# Check agent identity via CLAUDE_AGENT_NAME
if [ -n "$CLAUDE_AGENT_NAME" ]; then
    case "$CLAUDE_AGENT_NAME" in
        worker-*)
            write_failure "worker_guard" "git commit" 2 "worker $CLAUDE_AGENT_NAME attempted git commit/push"
            echo "Workers must NOT commit. Write files and report via SendMessage to the team lead." >&2
            exit 2
            ;;
        *)
            exit 0
            ;;
    esac
fi

# Fallback: check for swarm-role marker file
if [ -f "$ROOT/.agents/swarm-role" ]; then
    ROLE=$(cat "$ROOT/.agents/swarm-role")
    case "$ROLE" in
        worker*)
            write_failure "worker_guard" "git commit" 2 "worker role '$ROLE' attempted git commit/push"
            echo "Workers must NOT commit. Write files and report via SendMessage to the team lead." >&2
            exit 2
            ;;
    esac
fi

# No worker identity detected → allow
exit 0
