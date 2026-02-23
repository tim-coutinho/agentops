---
name: inbox
description: 'Agent mail inbox. Check pending messages, HELP_REQUESTs, and inter-agent notifications. Triggers: "inbox", "check messages", "check mail", "any messages".'
skill_api_version: 1
user-invocable: true
metadata:
  tier: session
  dependencies: []
---

# /inbox — Agent Mail Inbox

> **Purpose:** Check the agent mail inbox for pending messages, HELP_REQUESTs, and inter-agent notifications.

**YOU MUST EXECUTE THIS WORKFLOW. Do not just describe it.**

**CLI dependencies:** gt — required. This skill wraps `gt mail inbox`.

---

## Quick Start

```bash
/inbox              # Check pending messages
```

---

## Execution Steps

### Step 1: Check Inbox

```bash
if command -v gt &>/dev/null; then
  gt mail inbox
else
  echo "gt CLI not installed. Install with: brew install gt"
  exit 1
fi
```

### Step 2: Display Results

Show the inbox contents to the user. If there are pending messages, display them with sender, timestamp, and content. If the inbox is empty, confirm no pending messages.

### Step 3: Suggest Action

If messages contain HELP_REQUESTs, suggest responding. If inbox is empty, confirm and suggest returning to current work.

---

## Examples

### Checking Inbox With Messages

**User says:** `/inbox`

**What happens:**
1. Agent runs `gt mail inbox` to retrieve pending messages
2. Agent displays messages with sender, timestamp, and content
3. Agent suggests responding to any HELP_REQUESTs

**Result:** User sees pending messages and can act on them.

### Checking Empty Inbox

**User says:** `/inbox`

**What happens:**
1. Agent runs `gt mail inbox` and finds no messages
2. Agent confirms inbox is empty

**Result:** User confirms no pending messages and continues current work.

---

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| "gt CLI not installed" | gt not in PATH | Install with `brew install gt` |
| No messages shown but expected | Messages already read or expired | Check `gt mail` for full mail history |
| Permission error | gt not configured for this workspace | Run `gt init` to configure the workspace |
