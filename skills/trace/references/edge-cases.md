# Edge Cases

How to handle common failure modes during trace execution.

## No CASS Results

```
IF cass search returns 0 results:
  - Log: "No session transcripts mention '<concept>'"
  - Continue with other sources
  - Note in report: "Concept not found in session history"
```

## No Handoff Documents

```
IF .agents/handoff/ doesn't exist OR no matches:
  - Log: "No handoff documents mention '<concept>'"
  - Continue with other sources
  - Note in report: "Concept not documented in handoffs"
```

## Ambiguous Concept (Too Many Results)

```
IF CASS returns >20 results:
  - Show top 10 by score
  - Ask user: "Many sessions mention this. Want to narrow by date range or workspace?"
  - Suggest related but more specific concepts
```

## All Sources Empty

```
IF all 4 searches return nothing:
  - Report: "No provenance found for '<concept>'"
  - Suggest: "Try related terms: <suggestions>"
  - Ask: "Is this concept documented somewhere else?"
```
