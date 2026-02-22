# Failure Categories Taxonomy

## Failure Tracking

**Track failures by TYPE - not all failures are equal:**

| Failure Type | Counts Toward Limit? | Action |
|--------------|----------------------|--------|
| `root_cause_not_found` | YES | Re-investigate from Phase 1 |
| `fix_failed_tests` | YES | New hypothesis in Phase 3 |
| `design_rejected` | YES | Rethink approach |
| `execution_timeout` | NO (reset counter) | Retry same approach |
| `external_dependency` | NO (escalate) | Report blocker |

## The 3-Failure Rule

- Count only `root_cause_not_found`, `fix_failed_tests`, `design_rejected`
- After 3 such failures: **STOP and question architecture**
- Output: "3+ fix attempts failed. Escalating to architecture review."
- Do NOT count timeouts or external blockers toward limit

## Track in Issue Notes

```bash
bd update <issue-id> --append-notes "FAILURE: <type> at $(date -Iseconds) - <reason>" 2>/dev/null
```

## Checking Failure Count

```bash
# Count failures (excluding timeouts and external blockers)
failures=$(bd show <issue-id> --json 2>/dev/null | jq '[.notes[]? | select(startswith("FAILURE:")) | select(contains("root_cause") or contains("fix_failed") or contains("design_rejected"))] | length')

if [[ "$failures" -ge 3 ]]; then
    echo "3+ fix attempts failed. Escalating to architecture review."
    bd update <issue-id> --append-notes "ESCALATION: Architecture review needed after 3 failures" 2>/dev/null
    exit 1
fi
```
