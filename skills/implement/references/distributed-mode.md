# Distributed Mode: Agent Mail Coordination

> Extracted from implement SKILL.md. Covers Agent Mail integration for distributed/multi-agent execution.

> **When:** Agent Mail MCP tools are available (distributed mode or multi-agent coordination)

Distributed mode enhances /implement with real-time coordination via MCP Agent Mail. This enables:
- Progress reporting to orchestrators (Mayor/Delphi)
- Help requests instead of blocking on user input
- Receiving guidance mid-execution
- Coordination with parallel workers (file reservations)

## Mode Detection

Detect the operational mode at skill start:

```bash
# Check for Agent Mail availability
AGENT_MAIL_AVAILABLE=false

# Method 1: Check if running inside a distributed worker context
if [ -n "$OLYMPUS_DEMIGOD_ID" ]; then
    AGENT_MAIL_AVAILABLE=true
fi

# Method 2: Check for explicit flag
# /implement <issue-id> --mode=distributed --thread-id <id>
# /implement <issue-id> --agent-mail --thread-id <id>  # Back-compat alias
```

**Mode Semantics:**
| Mode | Condition | Behavior |
|------|-----------|----------|
| `local` | No Agent Mail | Current behavior (all steps above) |
| `distributed` | Agent Mail available | Enhanced coordination (steps below) |

## Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `--mode=local` | Force local mode | Default |
| `--mode=distributed` | Enable Agent Mail coordination | - |
| `--agent-mail` | Enable Agent Mail coordination (back-compat alias) | `false` |
| `--thread-id` | Thread ID for message grouping (usually bead-id) | `<issue-id>` |
| `--demigod-id` | Agent identity for message sender | `demigod-<issue-id>` |
| `--orchestrator` | Who to send status messages to | `<orchestrator-agent-name>` |

## Distributed Mode Execution Steps

When `--mode=distributed` (or `--agent-mail`) is enabled OR `$OLYMPUS_DEMIGOD_ID` is set:

### Step 0: Initialize Agent Mail Session

```bash
# If not already registered (Demigod handles this), register self
# This is typically done by a worker wrapper (e.g., tmux session), but implement can work standalone
if [ -z "$OLYMPUS_DEMIGOD_ID" ]; then
    # Standalone distributed mode - need to register
    DEMIGOD_ID="${DEMIGOD_ID:-demigod-$(date +%s)}"
else
    DEMIGOD_ID="$OLYMPUS_DEMIGOD_ID"
fi
```

**MCP Tool Call (if standalone):**
```
Tool: mcp__mcp-agent-mail__register_agent
Parameters:
  project_key: "<project-key>"
  program: "implement-skill"
  model: "<model>"
  task_description: "Implementing <issue-id>"
```

### Step 1: Send ACCEPTED Message

After claiming the issue (Step 2 in base flow), notify the orchestrator:

**MCP Tool Call:**
```
Tool: mcp__mcp-agent-mail__send_message
Parameters:
  project_key: "<project-key>"
  sender_name: "<demigod-id>"
  to: "<orchestrator>"
  subject: "BEAD_ACCEPTED"
  body_md: |
    Accepted bead: <issue-id>
    Title: <issue-title>
    Starting implementation at: <timestamp>
  thread_id: "<issue-id>"
  ack_required: false
```

### Step 2: Check Inbox Before Major Steps

Before Steps 3, 4, 5 in base flow, check for incoming messages:

**MCP Tool Call:**
```
Tool: mcp__mcp-agent-mail__fetch_inbox
Parameters:
  project_key: "olympus"
  agent_name: "<demigod-id>"
```

**Handle incoming messages:**
| Message Type | Action |
|--------------|--------|
| `GUIDANCE` | Incorporate guidance into approach |
| `NUDGE` | Respond with progress update |
| `TERMINATE` | Exit gracefully (checkpoint work) |

### Step 3: Send PROGRESS Messages Periodically

After each major step, send progress update:

**Timing:**
- After context gathering (Step 3)
- During implementation (Step 4) - every ~5 minutes of work
- After verification (Step 5)

**MCP Tool Call:**
```
Tool: mcp__mcp-agent-mail__send_message
Parameters:
  project_key: "olympus"
  sender_name: "<demigod-id>"
  to: "<orchestrator>"
  subject: "PROGRESS"
  body_md: |
    Bead: <issue-id>
    Step: <current-step>
    Status: <what's happening>
    Context usage: <approximate %>
    Files touched: <list>
  thread_id: "<issue-id>"
  ack_required: false
```

### Step 4: Send HELP_REQUEST When Stuck

**Replace user prompts with Agent Mail help requests:**

Instead of:
```
"I'm stuck on X. Should I proceed with Y or Z?"
```

Do this:
```
Tool: mcp__mcp-agent-mail__send_message
Parameters:
  project_key: "olympus"
  sender_name: "<demigod-id>"
  to: "chiron@olympus"  # Help goes to Chiron (coach)
  subject: "HELP_REQUEST"
  body_md: |
    Bead: <issue-id>
    Issue Type: STUCK | SPEC_UNCLEAR | BLOCKED | TECHNICAL

    ## Problem
    <describe the issue>

    ## What I Tried
    <approaches attempted>

    ## Files Touched
    - path/to/file.py
    - path/to/other.py

    ## Question
    <specific question needing answer>
  thread_id: "<issue-id>"
  ack_required: true
```

**Then wait for HELP_RESPONSE:**
```
# Poll inbox for response (max 2 minutes per help-response timeout)
for i in {1..24}; do
    # Check inbox
    # If HELP_RESPONSE found, continue with guidance
    # If timeout exceeded, proceed with best judgment or fail gracefully
    sleep 5
done
```

### Step 5: Report Completion via Agent Mail

After Step 7 (closing the issue), send completion message:

**On Success:**
```
Tool: mcp__mcp-agent-mail__send_message
Parameters:
  project_key: "olympus"
  sender_name: "<demigod-id>"
  to: "<orchestrator>"
  subject: "OFFERING_READY"
  body_md: |
    Bead: <issue-id>
    Status: DONE

    ## Changes
    - Commit: <commit-sha>
    - Files: <changed-files>

    ## Self-Validation
    - Tests: PASS/FAIL
    - Lint: PASS/FAIL
    - Build: PASS/FAIL

    ## Summary
    <brief description of what was implemented>
  thread_id: "<issue-id>"
  ack_required: true
```

**On Failure:**
```
Tool: mcp__mcp-agent-mail__send_message
Parameters:
  project_key: "olympus"
  sender_name: "<demigod-id>"
  to: "<orchestrator>"
  subject: "FAILED"
  body_md: |
    Bead: <issue-id>
    Status: FAILED

    ## Failure
    Type: TESTS_FAIL | BUILD_FAIL | SPEC_IMPOSSIBLE | ERROR
    Reason: <description>
    Internal Attempts: <count>

    ## Partial Progress
    - Commit: <partial-commit-sha if any>
    - Files: <files-modified>

    ## Recommendation
    <what needs to happen to unblock>
  thread_id: "<issue-id>"
  ack_required: false
```

## File Reservations

When running in distributed mode with parallel workers, claim files before editing:

**Before Step 4 (implementation):**
```
Tool: mcp__mcp-agent-mail__file_reservation_paths
Parameters:
  project_key: "olympus"
  agent_name: "<demigod-id>"
  paths:
    - "path/to/file1.py"
    - "path/to/file2.py"
  exclusive: false  # Advisory, not blocking
```

**After Step 7 (completion):**
```
Tool: mcp__mcp-agent-mail__release_file_reservations
Parameters:
  project_key: "olympus"
  agent_name: "<demigod-id>"
```

## Context Checkpoint

If context usage is high (>80%), send checkpoint before exiting:

```
Tool: mcp__mcp-agent-mail__send_message
Parameters:
  project_key: "olympus"
  sender_name: "<demigod-id>"
  to: "<orchestrator>"
  subject: "CHECKPOINT"
  body_md: |
    Bead: <issue-id>
    Reason: CONTEXT_HIGH

    ## Progress
    - Commit: <partial-commit-sha>
    - Description: <what's done, what remains>
    - Context usage: <pct>%

    ## Next Steps for Successor
    <guidance for replacement demigod>
  thread_id: "<issue-id>"
  ack_required: false
```

## Key Rules

- **Always check inbox** - Messages may contain critical guidance
- **Send progress regularly** - Orchestrator needs visibility
- **Use HELP_REQUEST** - Don't block on user, ask Chiron
- **Reserve files** - Prevents conflicts with parallel workers
- **Checkpoint on context high** - Preserve progress for successor
- **Acknowledge important messages** - Confirms receipt to sender

## Distributed vs Local Mode Summary

| Behavior | Local | Distributed |
|----------|---------|---------|
| Progress reporting | None | PROGRESS messages |
| When stuck | Ask user / proceed | HELP_REQUEST to Chiron |
| Completion | bd update only | OFFERING_READY + bd update |
| Failure | Report to user | FAILED message + report |
| File conflicts | Race conditions | Advisory reservations |
| Guidance | None | Check inbox for GUIDANCE |

## Integration with Distributed Workers

When `/implement` is run inside a distributed worker (e.g., a tmux session):
1. Environment variables may already be set (`$OLYMPUS_DEMIGOD_ID`, etc.)
2. Agent registration already done
3. File reservations already claimed
4. Just focus on implementation + progress + completion messages

When /implement is called standalone with `--agent-mail`:
1. Must register self
2. Must claim own files
3. Full distributed mode behavior
