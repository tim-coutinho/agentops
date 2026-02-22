# Trace Report Template

Write trace reports to: `.agents/research/YYYY-MM-DD-trace-<concept-slug>.md`

## Full Template

```markdown
# Trace: <Concept>

**Date:** YYYY-MM-DD
**Query:** <original concept>
**Sources searched:** CASS, Handoffs, Git, Research

## Summary

<2-3 sentence overview of how the concept evolved>

## Timeline

| Date | Source | Event | Evidence |
|------|--------|-------|----------|
| ... | ... | ... | ... |

## Key Decisions

### Decision 1: <title>
- **Date:** YYYY-MM-DD
- **Source:** <CASS session / Handoff / Git commit>
- **What:** <what was decided>
- **Why:** <reasoning if known>
- **Evidence:** <link/path>

### Decision 2: <title>
...

## Evolution Summary

<How the concept changed over time, key inflection points>

## Current State

<Where the concept stands now based on most recent evidence>

## Related Concepts

- <related concept 1> - see `/trace <concept1>`
- <related concept 2> - see `/trace <concept2>`

## Sources

### CASS Sessions
| Date | Session Path | Score |
|------|--------------|-------|
| ... | ... | ... |

### Handoff Documents
| Date | File | Context |
|------|------|---------|
| ... | ... | ... |

### Git Commits
| Date | SHA | Message |
|------|-----|---------|
| ... | ... | ... |

### Research/Learnings
| Date | File |
|------|------|
| ... | ... |
```

## Timeline Construction

Merge results from all sources into the Timeline table.

**Deduplication rules:**
- Same content within 24 hours = single event (note multiple sources)
- Same session ID = single event
- Preserve ALL sources as evidence

**Sorting:**
- Chronological order (oldest first)
- Show evolution of the concept over time
