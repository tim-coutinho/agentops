# Agent Mail Protocol Reference

This document defines the message protocol used for distributed mode coordination between crank, swarm, implement, and inbox skills via Agent Mail.

## Message Types

| Type | Direction | Purpose |
|------|-----------|---------|
| `BEAD_ACCEPTED` | Demigod -> Orchestrator | Confirms demigod has accepted work |
| `PROGRESS` | Demigod -> Orchestrator | Periodic status updates |
| `HELP_REQUEST` | Demigod -> Chiron | Request guidance when stuck |
| `HELP_RESPONSE` | Chiron -> Demigod | Response to help request |
| `OFFERING_READY` | Demigod -> Orchestrator | Work complete, ready for review |
| `DONE` | Demigod -> Orchestrator | Alternate completion signal |
| `FAILED` | Demigod -> Orchestrator | Implementation failed |
| `CHECKPOINT` | Demigod -> Orchestrator | Context exhaustion with partial progress |
| `SPAWN_REQUEST` | Orchestrator -> Spawner | Request to spawn new worker |
| `SPAWN_ACK` | Spawner -> Orchestrator | Acknowledge spawn request |

## Message Formats

### BEAD_ACCEPTED

Sent when a demigod accepts a bead to work on.

```markdown
Subject: BEAD_ACCEPTED
# or
Subject: [<bead-id>] BEAD_ACCEPTED

Body:
Accepted bead: <bead-id>
Title: <issue title>
Starting implementation at: <timestamp>
```

**Fields:**
- `Accepted bead:` - The bead/issue identifier
- `Title:` - Human-readable issue title
- `Starting implementation at:` - ISO8601 timestamp

---

### PROGRESS

Periodic status updates during implementation.

```markdown
Subject: PROGRESS
# or
Subject: [<bead-id>] PROGRESS

Body:
Bead: <bead-id>
Step: <current step description>
Status: <what's happening>
Context usage: <N>%
Files touched: <comma-separated list>
```

**Fields:**
- `Bead:` - The bead/issue identifier
- `Step:` - Current step (e.g., "Step 4 - implementing auth")
- `Status:` - Brief status description
- `Context usage:` - Approximate context window percentage (0-100)
- `Files touched:` - Files modified so far

**Recommended frequency:** Every 5 minutes or after each major step.

---

### HELP_REQUEST

Sent when a demigod is stuck and needs guidance.

```markdown
Subject: HELP_REQUEST

Body:
Bead: <bead-id>
Issue Type: STUCK | SPEC_UNCLEAR | BLOCKED | TECHNICAL

## Problem
<description of the issue>

## What I Tried
<approaches attempted>

## Files Touched
- path/to/file1
- path/to/file2

## Question
<specific question needing an answer>
```

**Issue Types:**
- `STUCK` - Cannot proceed, unclear how
- `SPEC_UNCLEAR` - Specification is ambiguous
- `BLOCKED` - External dependency blocking progress
- `TECHNICAL` - Technical problem needing expert help

**AckRequired:** `true` - Sender expects a response.

---

### HELP_RESPONSE

Response to a HELP_REQUEST.

```markdown
Subject: HELP_RESPONSE

Body:
<guidance text>
```

Should be sent in the same thread as the HELP_REQUEST.

---

### OFFERING_READY

Work complete and ready for review.

```markdown
Subject: OFFERING_READY

Body:
Bead: <bead-id>
Status: DONE

## Changes
- Commit: <commit-sha>
- Files: <comma-separated list>

## Self-Validation
- Tests: PASS | FAIL
- Lint: PASS | FAIL
- Build: PASS | FAIL

## Summary
<brief description of what was implemented>
```

**Fields:**
- `Commit:` - Git commit SHA for the changes
- `Files:` - List of changed files
- `Tests/Lint/Build:` - Validation status (PASS or FAIL)
- `Summary:` - Human-readable summary

**AckRequired:** `true` - Orchestrator should acknowledge.

---

### DONE

Alternate completion message (simpler than OFFERING_READY).

```markdown
Subject: DONE
# or
Subject: [<bead-id>] DONE

Body:
Bead: <bead-id>
Status: DONE

## Changes
- Commit: <commit-sha>
- Files: <list>

## Self-Validation
- Tests: PASS | FAIL
- Lint: PASS | FAIL
- Build: PASS | FAIL

## Summary
<description>
```

Same format as OFFERING_READY.

---

### FAILED

Implementation failed with reason.

```markdown
Subject: FAILED

Body:
Bead: <bead-id>
Status: FAILED

## Failure
Type: TESTS_FAIL | BUILD_FAIL | SPEC_IMPOSSIBLE | CONTEXT_HIGH | ERROR
Reason: <description>
Internal Attempts: <count>

## Partial Progress
- Commit: <partial-commit-sha>
- Files: <list>

## Recommendation
<what needs to happen to unblock>
```

**Failure Types:**
- `TESTS_FAIL` - Tests do not pass
- `BUILD_FAIL` - Build/compilation fails
- `SPEC_IMPOSSIBLE` - Specification cannot be implemented
- `CONTEXT_HIGH` - Context window exhausted
- `ERROR` - Unexpected error

---

### CHECKPOINT

Context exhaustion with partial progress for successor.

```markdown
Subject: CHECKPOINT

Body:
Bead: <bead-id>
Reason: CONTEXT_HIGH | MANUAL | TIMEOUT

## Progress
- Commit: <partial-commit-sha>
- Description: <what's done, what remains>
- Context usage: <N>%

## Next Steps for Successor
<guidance for replacement demigod>
```

**Checkpoint Reasons:**
- `CONTEXT_HIGH` - Context window above 80%
- `MANUAL` - Manual checkpoint requested
- `TIMEOUT` - Time limit reached

---

### SPAWN_REQUEST

Request to spawn a new worker.

```markdown
Subject: SPAWN_REQUEST

Body:
Issue: <bead-id>
Resume: true | false
Checkpoint: <commit-sha>
Orchestrator: <orchestrator-id>
```

**Fields:**
- `Issue:` - Bead/issue to work on
- `Resume:` - Whether resuming from checkpoint
- `Checkpoint:` - Commit SHA to resume from (if Resume=true)
- `Orchestrator:` - Requesting orchestrator

---

### SPAWN_ACK

Acknowledge spawn request.

```markdown
Subject: SPAWN_ACK

Body:
Issue: <bead-id>
Status: spawned | failed
Session: <session-name>
```

---

## Subject Line Conventions

Subjects may include bead ID in brackets for easy filtering:

```
[ol-527.1] PROGRESS
[ol-527.1] OFFERING_READY
[ol-527.2] HELP_REQUEST
```

Parser extracts bead ID from:
1. Body field: `Bead: <id>` or `Accepted bead: <id>`
2. Subject brackets: `[<id>] TYPE`
3. Subject prefix: `<id>: TYPE`

---

## Threading

All messages for a bead should use the same `thread_id` (typically the bead ID):

```json
{
  "thread_id": "ol-527.1",
  "subject": "PROGRESS",
  ...
}
```

This enables:
- Grouping related messages
- Thread summarization
- Conversation reconstruction

---

## Acknowledgement

Messages with `ack_required: true` expect acknowledgement:

| Message Type | Ack Required |
|--------------|--------------|
| `BEAD_ACCEPTED` | No |
| `PROGRESS` | No |
| `HELP_REQUEST` | Yes |
| `OFFERING_READY` | Yes |
| `DONE` | No |
| `FAILED` | No |
| `CHECKPOINT` | No |
| `SPAWN_REQUEST` | Yes |

---

## Parser Usage (Go)

```go
import "github.com/boshu2/agentops/cli/internal/agentmail"

// Parse single message
parser := agentmail.NewParser()
msg, err := parser.Parse(rawMessage)
if err != nil {
    // handle error
}

// Check message type
switch msg.Type {
case agentmail.MessageTypeOfferingReady:
    fmt.Println("Complete:", msg.Parsed.CommitSHA)
case agentmail.MessageTypeFailed:
    fmt.Println("Failed:", msg.Parsed.Reason)
case agentmail.MessageTypeHelpRequest:
    fmt.Println("Help needed:", msg.Parsed.Question)
}

// Filter helpers
completions := agentmail.FilterCompletions(messages)
successes := agentmail.FilterSuccesses(messages)
pending := agentmail.FilterPending(messages)
beadMsgs := agentmail.FilterByBeadID(messages, "ol-527.1")
```

---

## Integration Points

| Skill | Uses Parser For |
|-------|-----------------|
| `/crank` | Monitoring wave completion, routing help requests |
| `/swarm` | Tracking demigod status, wave completion |
| `/implement` | Sending status messages (not parsing) |
| `/inbox` | Displaying messages, filtering, summarizing |

---

## Error Handling

The parser is lenient by default:
- Missing fields return empty strings
- Invalid types return `MessageTypeUnknown`
- Malformed bodies are partially parsed

For strict mode:
```go
parser := agentmail.NewParser()
parser.Strict = true
```

In strict mode, malformed messages return errors.
