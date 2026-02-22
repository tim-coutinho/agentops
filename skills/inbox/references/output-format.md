# Output Format

## Display Template

Format output as:

```markdown
# Agent Mail Inbox

**Agent:** <agent-name>
**Project:** <project-key>
**Checked:** <timestamp>

## Pending Messages (<count>)

| From | Subject | Thread | Age |
|------|---------|--------|-----|
| ... | ... | ... | ... |

## HELP_REQUESTs Needing Response (<count>)

| From | Issue | Problem | Waiting |
|------|-------|---------|---------|
| demigod-gt-123 | gt-123 | STUCK - Can't find auth module | 15m |
| ... | ... | ... | ... |

## Recent Completions (<count>)

| Agent | Issue | Status | Completed |
|-------|-------|--------|-----------|
| demigod-gt-124 | gt-124 | DONE | 2m ago |
| ... | ... | ... | ... |

## Actions Needed

- [ ] Respond to HELP_REQUEST from demigod-gt-123 (waiting 15m)
- [ ] Acknowledge completion of gt-124
```

## Integration Points

| System | Integration |
|--------|-------------|
| **Beads** | Thread IDs often match beads issue IDs (e.g., gt-123) |
| **Demigod** | Demigods send progress, help requests, completions |
| **Mayor** | Mayor monitors inbox for coordination decisions |
| **Chiron** | Chiron watches for HELP_REQUESTs to answer |

## Example Session

```bash
# Quick check
/inbox

# Output:
# Agent Mail Inbox
# Agent: boden
# Project: ~/gt/olympus
#
# Pending Messages (2)
# - PROGRESS from demigod-ol-527 (5m ago)
# - HELP_REQUEST from demigod-ol-528 (2m ago)
#
# HELP_REQUESTs Needing Response (1)
# - demigod-ol-528: STUCK - Can't find skill template format
#
# Actions Needed:
# - [ ] Respond to HELP_REQUEST from demigod-ol-528

# Start monitoring
/inbox --watch
```
