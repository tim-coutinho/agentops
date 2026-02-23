#!/usr/bin/env bash
# AgentOps Session Start Hook (minimal flywheel)
# Creates .agents/ directories, runs extract+inject, injects skill context.

# Kill switches
[ "${AGENTOPS_HOOKS_DISABLED:-}" = "1" ] && exit 0
[ "${AGENTOPS_SESSION_START_DISABLED:-}" = "1" ] && exit 0

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
PLUGIN_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
ROOT="$(cd "$ROOT" 2>/dev/null && pwd -P 2>/dev/null || printf '%s' "$ROOT")"
AO_DIR="$ROOT/.agents/ao"

HOOK_ERROR_LOG="$AO_DIR/hook-errors.log"

log_hook_fail() {
    mkdir -p "$AO_DIR" 2>/dev/null || return 0
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) HOOK_FAIL: $1" >> "$HOOK_ERROR_LOG" 2>/dev/null || true
}

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

cd "$ROOT" 2>/dev/null || true

# Ensure .agents/ directories exist
for dir in .agents/research .agents/products .agents/retros .agents/learnings \
           .agents/patterns .agents/council .agents/knowledge/pending \
           .agents/plans .agents/rpi .agents/ao; do
    mkdir -p "$ROOT/$dir" 2>/dev/null
done

# Auto-gitignore .agents/
if [ "${AGENTOPS_GITIGNORE_AUTO:-1}" != "0" ] && [ -d "$ROOT/.git" ]; then
    GITIGNORE="$ROOT/.gitignore"
    if [ -f "$GITIGNORE" ]; then
        grep -q '\.agents/' "$GITIGNORE" 2>/dev/null || \
            printf '\n# AgentOps session artifacts\n.agents/\n' >> "$GITIGNORE" 2>/dev/null
    else
        printf '# AgentOps session artifacts\n.agents/\n' > "$GITIGNORE" 2>/dev/null
    fi
fi
if [ ! -f "$ROOT/.agents/.gitignore" ]; then
    cat > "$ROOT/.agents/.gitignore" 2>/dev/null <<'EOF'
*
!.gitignore
!README.md
EOF
fi

# Flywheel: extract pending queue + inject prior knowledge
if command -v ao &>/dev/null; then
    run_ao_quick 5 extract || log_hook_fail "ao extract"
    run_ao_quick 5 inject --apply-decay --format markdown --max-tokens 1000 || log_hook_fail "ao inject"
fi

# Inject using-agentops skill context
SKILL_FILE="${PLUGIN_ROOT}/skills/using-agentops/SKILL.md"
if [ -f "$SKILL_FILE" ]; then
    using_agentops_content=$(cat "$SKILL_FILE")
else
    using_agentops_content="(AgentOps skill content unavailable)"
fi

full_content="<EXTREMELY_IMPORTANT>
You have AgentOps superpowers.

**Below is the full content of your 'agentops:using-agentops' skill - your introduction to using AgentOps skills. For all other skills, use the 'Skill' tool:**

${using_agentops_content}
</EXTREMELY_IMPORTANT>"

if command -v jq &>/dev/null; then
    additional_context=$(printf '%s' "$full_content" | jq -Rs '.')
    cat <<HOOKEOF
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": ${additional_context}
  }
}
HOOKEOF
else
    # Minimal fallback — escape newlines and quotes
    escaped=$(printf '%s' "$full_content" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')
    cat <<HOOKEOF
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "${escaped}"
  }
}
HOOKEOF
fi

exit 0
