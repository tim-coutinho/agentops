# Quality Mode

## When to Use

Use `--quality` when:
- Post-mortem findings are accumulating faster than they're consumed
- All GOALS pass but `next-work.jsonl` has unconsumed high-severity items
- You want to resolve context-hot findings from a just-completed epic
- Running immediately after `/post-mortem` to action its findings

Do NOT use `--quality` when:
- GOALS have critical failures (build broken, tests failing)
- No `next-work.jsonl` exists or is empty
- You want standard fitness-driven improvement

## Quality Score

Simple severity-weighted score:

```
score = 100 - (high_count * 10) - (medium_count * 3)
```

Where counts are unconsumed findings remaining in next-work.jsonl.

| Score | Meaning |
|-------|---------|
| 90-100 | Excellent — few or no findings remaining |
| 70-89 | Good — medium-severity items remain |
| 50-69 | Attention needed — high-severity items remain |
| <50 | Quality debt — many high-severity findings |

## Priority Cascade (Quality Mode)

1. High-severity unconsumed findings → /rpi
2. Medium-severity unconsumed findings → /rpi
3. Failing GOALS.yaml goals → /rpi (standard behavior)
4. Open beads → /implement
5. Nothing → stagnation (3 idle cycles)

## Marking Findings Consumed

When evolve picks a finding from next-work.jsonl, mark it consumed:
- Set `consumed: true`
- Set `consumed_by: "evolve-quality:cycle-N"`
- Set `consumed_at: "<timestamp>"`

If the /rpi cycle fails (regression), un-mark the finding (set consumed back to false).

## Artifacts

| File | Purpose |
|------|---------|
| `cycle-history.jsonl` | Same as standard mode + `quality_score` field |
| `fitness-latest.json` | Same as standard mode (goals measurement) |
| `quality-trajectory.md` | Quality score over time (written at teardown) |

## Interaction with Standard Mode

Quality mode and standard mode share:
- The same cycle-history.jsonl
- The same fitness measurement (goals are still checked)
- The same stagnation detection (3 idle cycles)
- The same circuit breaker (60 minutes)

They differ in work selection priority: quality mode picks findings first, standard picks goals first.
