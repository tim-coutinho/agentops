# Distributed Mode: tmux + Agent Mail (Experimental)

> **Status: Experimental.** Local mode (native teams) is the recommended execution method. Distributed mode requires Agent Mail and tmux and has not been battle-tested. Use for long-running epics where process isolation and persistence are critical.

> **When:** MCP Agent Mail is available AND you want true process isolation, persistent workers, and robust coordination.

Distributed mode spawns real tmux sessions instead of Task tool background agents. Each demigod runs in its own Claude process with full lifecycle management.

## Why Distributed Mode?

| Local (Task tool) | Distributed (tmux + Agent Mail) |
|---------------------|----------------------------|
| Background agents in Mayor's process | Separate tmux sessions |
| Coupled to Mayor lifecycle | Persistent if Mayor crashes |
| No inter-agent coordination | Agent Mail messaging |
| No file conflict prevention | File reservations |
| Simple, fast to spawn | More setup, more robust |
| Good for small jobs | Better for large/long jobs |

## Mode Detection

At skill start, detect which mode to use:

```bash
# Method 1: Explicit flag
# /swarm --mode=distributed <tasks>

# Method 2: Auto-detect Agent Mail availability
MODE="local"

# Check for Agent Mail MCP tools (look for register_agent tool)
if mcp-tools 2>/dev/null | grep -q "mcp-agent-mail"; then
    AGENT_MAIL_AVAILABLE=true
fi

# Check for Agent Mail HTTP endpoint
if curl -s http://localhost:8765/health >/dev/null 2>&1; then
    AGENT_MAIL_HTTP=true
fi

# Distributed requires: Agent Mail available + explicit flag
if [ "$MODE_FLAG" = "distributed" ] && [ "$AGENT_MAIL_AVAILABLE" = "true" -o "$AGENT_MAIL_HTTP" = "true" ]; then
    MODE="distributed"
fi
```

**Decision matrix:**

| `--mode` | Agent Mail | Result |
|----------|------------|--------|
| Not set | Not available | Local |
| Not set | Available | Local (explicit opt-in required) |
| `--mode=local` | Any | Local |
| `--mode=distributed` | Not available | **Error: Agent Mail required** |
| `--mode=distributed` | Available | Distributed |

## Distributed Mode Invocation

```
/swarm --mode=distributed [--max-workers=N]
/swarm --mode=distributed --bead-ids ol-527.1,ol-527.2,ol-527.3
```

**Parameters:**

| Parameter | Description | Default |
|-----------|-------------|---------|
| `--mode=distributed` | Enable tmux + Agent Mail mode | - |
| `--max-workers=N` | Max concurrent demigods | 5 |
| `--bead-ids` | Specific beads to work (comma-separated) | Auto from `bd ready` |
| `--wait` | Wait for all demigods to complete | false |
| `--timeout` | Max time to wait (if --wait) | 30m |

## Distributed Mode Architecture

```
Mayor Session (this session)
    |
    +-> Mode Detection: `--mode=distributed` + Agent Mail available -> Distributed
    |
    +-> Identify wave: bd ready -> [ol-527.1, ol-527.2, ol-527.3]
    |
    +-> Spawn: tmux new-session for each worker
    |       demigod-ol-527-1 -> runs `/implement ol-527.1 --mode=distributed`
    |       demigod-ol-527-2 -> runs `/implement ol-527.2 --mode=distributed`
    |       demigod-ol-527-3 -> runs `/implement ol-527.3 --mode=distributed`
    |
    +-> Coordinate: Agent Mail messages
    |       Each demigod sends ACCEPTED, PROGRESS, DONE/FAILED
    |       Mayor monitors via fetch_inbox
    |       File reservations prevent conflicts
    |
    +-> Validate: On DONE, optionally run /vibe --remote
    |
    +-> Repeat: New wave when workers complete
```

## Distributed Mode Execution Steps

Given `/swarm --mode=distributed`:

### Step 1: Pre-flight Checks

Run equivalent checks manually before starting:

```bash
# Check tmux is available
which tmux >/dev/null 2>&1 || {
    echo "Error: tmux required for distributed mode. Install: brew install tmux"
    exit 1
}

# Check claude CLI is available
which claude >/dev/null 2>&1 || {
    echo "Error: claude CLI required for distributed mode"
    exit 1
}

# Check Agent Mail is available
AGENT_MAIL_OK=false
if curl -s http://localhost:8765/health >/dev/null 2>&1; then
    AGENT_MAIL_OK=true
fi
# OR check MCP tools

if [ "$AGENT_MAIL_OK" != "true" ]; then
    echo "Error: Agent Mail required for distributed mode"
    echo "Start your Agent Mail MCP server (implementation-specific). See docs/agent-mail.md."
    exit 1
fi

# Check claim-lock health for shared task claims
scripts/tasks-sync.sh lock-health >/dev/null 2>&1 || {
    echo "Error: claim-lock health check failed (scripts/tasks-sync.sh lock-health)"
    exit 1
}
```

### Step 2: Register Mayor with Agent Mail

Register the Mayor session to receive messages from demigods.

```
MCP Tool: register_agent
Parameters:
  project_key: <absolute path to project>
  program: "claude-code"
  model: "opus"
  task_description: "Mayor orchestrating swarm for wave"
```

**Store the returned agent name as `MAYOR_NAME`.**

### Step 3: Identify Wave (Ready Beads)

Same as local mode, get the beads to work:

```bash
# Get ready beads
READY_BEADS=$(bd ready --json 2>/dev/null | jq -r '.[].id' | head -$MAX_WORKERS)

# Or use explicit bead list
if [ -n "$BEAD_IDS" ]; then
    READY_BEADS=$(echo "$BEAD_IDS" | tr ',' '\n')
fi

# Count
WAVE_SIZE=$(echo "$READY_BEADS" | wc -l | tr -d ' ')
```

If no ready beads, exit with message.

### Step 4: Spawn Demigods via tmux

For each ready bead, spawn a demigod session:

```bash
for BEAD_ID in $READY_BEADS; do
    # Generate session name
    SESSION_NAME="demigod-$(echo $BEAD_ID | tr '.' '-')"

    # Check for existing session
    if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        echo "Session $SESSION_NAME already exists, skipping"
        continue
    fi

    # Spawn demigod in new tmux session
    tmux new-session -d -s "$SESSION_NAME" "claude -p '/implement $BEAD_ID --mode=distributed --thread-id $BEAD_ID'"

    # Verify session started
    if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        echo "Spawned: $SESSION_NAME for $BEAD_ID"
        SPAWNED_COUNT=$((SPAWNED_COUNT + 1))
    else
        echo "Failed to spawn: $SESSION_NAME"
    fi

    # Rate limit to avoid API cascade
    sleep 2
done
```

**Key points:**
- Use `-d` for detached sessions (run in background)
- Session names derived from bead IDs for easy correlation
- Rate limit spawns (2 seconds) to avoid API rate limits
- Verify each session started before continuing

### Step 5: Monitor via Agent Mail

Poll Agent Mail inbox for demigod messages:

```
MCP Tool: fetch_inbox
Parameters:
  project_key: <project path>
  agent_name: <MAYOR_NAME>
  limit: 50
  include_bodies: true
```

**Process messages by type:**

| Subject Pattern | Action |
|-----------------|--------|
| `[<bead-id>] ACCEPTED` | Log demigod started work |
| `[<bead-id>] PROGRESS` | Log progress, check health |
| `[<bead-id>] HELP_REQUEST` | Alert for manual intervention or route to Chiron |
| `[<bead-id>] DONE` | Mark complete, check if wave finished |
| `[<bead-id>] FAILED` | Log failure, decide on retry or escalate |

### Step 6: Track Completion

Maintain completion state:

```bash
# Track per-bead status
declare -A BEAD_STATUS
for BEAD_ID in $READY_BEADS; do
    BEAD_STATUS[$BEAD_ID]="spawned"
done

# Update on DONE/FAILED messages
# BEAD_STATUS[$BEAD_ID]="done" or "failed"

# Check if wave complete
DONE_COUNT=0
FAILED_COUNT=0
for BEAD_ID in "${!BEAD_STATUS[@]}"; do
    case "${BEAD_STATUS[$BEAD_ID]}" in
        done) DONE_COUNT=$((DONE_COUNT + 1)) ;;
        failed) FAILED_COUNT=$((FAILED_COUNT + 1)) ;;
    esac
done

if [ $((DONE_COUNT + FAILED_COUNT)) -eq $WAVE_SIZE ]; then
    WAVE_COMPLETE=true
fi
```

### Step 7: Report Results

When wave completes (or on `--wait` timeout):

```markdown
## Swarm Distributed Mode Results

**Wave completed:** <timestamp>
**Beads in wave:** <WAVE_SIZE>
**Successful:** <DONE_COUNT>
**Failed:** <FAILED_COUNT>

### Completed Beads
| Bead ID | Demigod | Commit | Summary |
|---------|---------|--------|---------|
| ol-527.1 | GreenCastle | abc123 | Added auth middleware |
| ol-527.2 | BlueMountain | def456 | Fixed rate limiting |

### Failed Beads
| Bead ID | Demigod | Reason | Recommendation |
|---------|---------|--------|----------------|
| ol-527.3 | RedValley | Tests failed | Re-run with spec clarification |

### Active Sessions
| Session | Bead | Status | Runtime |
|---------|------|--------|---------|
| demigod-ol-527-1 | ol-527.1 | done | 15m |
| demigod-ol-527-2 | ol-527.2 | done | 12m |
| demigod-ol-527-3 | ol-527.3 | failed | 18m |
```

### Step 8: Cleanup Completed Sessions

Optionally clean up tmux sessions for completed beads:

```bash
# Clean up done sessions (keep failed for debugging)
for BEAD_ID in "${!BEAD_STATUS[@]}"; do
    if [ "${BEAD_STATUS[$BEAD_ID]}" = "done" ]; then
        SESSION_NAME="demigod-$(echo $BEAD_ID | tr '.' '-')"
        tmux kill-session -t "$SESSION_NAME" 2>/dev/null
    fi
done
```

**Or keep all sessions for review:** Use `--keep-sessions` flag to preserve all tmux sessions for post-mortem analysis.

## Distributed Mode Helpers

Use these helpers with distributed mode swarm:

| Helper | Purpose |
|--------|---------|
| `tmux list-sessions` | List running worker sessions |
| `tmux attach -t <session>` | Attach to a worker session for debugging |
| `/inbox` | Check Agent Mail for pending messages |
| `/vibe --remote <session>` | Validate a worker's work before accepting |

## Distributed Mode File Reservations

File reservations prevent conflicts when multiple demigods edit files.

**How it works:**
1. Each demigod claims files before editing (via Agent Mail `file_reservation_paths`)
2. If another demigod tries to claim the same file, it sees a conflict warning
3. Demigods release reservations when done

**Mayor can view reservations:**

```
MCP Tool: get_file_reservations (if available)
Parameters:
  project_key: <project path>
```

**On conflict:**
- Demigod sends PROGRESS message noting the conflict
- Mayor decides: wait, reassign, or allow parallel work
- Advisory reservations don't block, just warn

## Distributed Mode Error Handling

### Demigod Session Crashes

If a tmux session dies unexpectedly:

```bash
# Check session health
tmux has-session -t "$SESSION_NAME" 2>/dev/null || {
    echo "Session $SESSION_NAME died"

    # Check bead status
    STATUS=$(bd show $BEAD_ID --json 2>/dev/null | jq -r '.status')

    if [ "$STATUS" = "in_progress" ]; then
        # Unclaim bead for retry
        bd update $BEAD_ID --status open --assignee "" 2>/dev/null
        echo "Bead $BEAD_ID released for retry"
    fi
}
```

### Agent Mail Server Crashes

If Agent Mail becomes unavailable mid-swarm:

1. Demigods continue working (graceful degradation)
2. Messages queue locally (if supported)
3. Mayor loses visibility but work continues
4. On restart, poll beads status directly:

```bash
for BEAD_ID in $READY_BEADS; do
    STATUS=$(bd show $BEAD_ID --json 2>/dev/null | jq -r '.status')
    echo "$BEAD_ID: $STATUS"
done
```

### Timeout Handling

If `--wait` times out:

```markdown
## Swarm Timeout

Wave did not complete within <timeout>.

### Still Running
| Session | Bead | Runtime | Action |
|---------|------|---------|--------|
| demigod-ol-527-3 | ol-527.3 | 32m | Consider `tmux attach -t demigod-ol-527-3` |

### Options
1. Continue waiting: `/swarm --mode=distributed --wait --timeout 60m`
2. Attach to slow workers: `tmux attach -t demigod-ol-527-3`
3. Kill and retry: `tmux kill-session -t demigod-ol-527-3`
```

## Distributed vs Local Mode Summary

| Behavior | Local | Distributed |
|----------|---------|---------|
| Spawn mechanism | `TeamCreate` + `Task(team_name=...)` | `tmux new-session -d` |
| Worker entry point | Inline prompt (teammate) | `/implement <bead-id> --mode=distributed` |
| Process isolation | Team per wave (fresh context) | Separate processes |
| Persistence | Tied to Mayor | Survives Mayor crash |
| Coordination | `SendMessage` (native teams) | Agent Mail messages |
| File conflicts | Workers report to lead | File reservations |
| Retry mechanism | `SendMessage` to idle worker | Re-spawn |
| Debugging | Limited | `tmux attach -t <session>` |
| Resource overhead | Low | Medium (N tmux sessions) |
| Setup requirements | None (native teams built-in) | tmux + Agent Mail |

## When to Use Distributed Mode

| Scenario | Recommendation |
|----------|---------------|
| Quick parallel tasks (<5 min each) | Local |
| Long-running work (>10 min each) | Distributed |
| Need to debug stuck workers | Distributed |
| Multi-file changes across workers | Distributed (file reservations) |
| Mayor might disconnect | Distributed (persistence) |
| Complex coordination needed | Distributed |
| Simple, isolated tasks | Local |

## Example: Full Distributed Mode Swarm

```bash
# 1. Start Agent Mail (if not running)
# Start your Agent Mail MCP server (implementation-specific)
# See docs/agent-mail.md

# 2. In Claude session, run distributed mode swarm
/swarm --mode=distributed --max-workers=3 --wait

# Output:
# Pre-flight: tmux OK, Agent Mail OK
# Registered as Mayor: GoldenPeak
# Wave 1: 3 ready beads
# Spawning: demigod-ol-527-1 for ol-527.1
# Spawning: demigod-ol-527-2 for ol-527.2
# Spawning: demigod-ol-527-3 for ol-527.3
# Monitoring...
# [15:32] ACCEPTED from GreenCastle (ol-527.1)
# [15:32] ACCEPTED from BlueMountain (ol-527.2)
# [15:33] ACCEPTED from RedValley (ol-527.3)
# [15:40] PROGRESS from GreenCastle: Step 4 - implementing auth
# [15:45] DONE from GreenCastle (ol-527.1) - commit abc123
# [15:48] DONE from BlueMountain (ol-527.2) - commit def456
# [15:55] FAILED from RedValley (ol-527.3) - tests failed
#
# Wave complete: 2 done, 1 failed
# Sessions cleaned up (except failed)
# Use `tmux attach -t demigod-ol-527-3` to debug the failed worker
```

## Fallback Behavior

If distributed mode requested but requirements not met:

```
Error: Distributed mode requires tmux and Agent Mail.

Missing:
- [ ] tmux: Install with `brew install tmux`
- [x] Agent Mail: Running at localhost:8765

Falling back to local mode? [y/N]
```

If user confirms, degrade to local mode execution. Otherwise, exit with error.
