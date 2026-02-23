# Teardown Procedure

**Auto-run /post-mortem on the full evolution session:**

```
/post-mortem "evolve session: $CYCLE cycles, goals improved: X, harvested: Y"
```

This captures learnings from the ENTIRE evolution run (all cycles, all /rpi invocations) in one council review. The post-mortem harvests follow-up items into `next-work.jsonl`, feeding the next `/evolve` session.

**Compute session fitness trajectory:**

```bash
# Check if both baseline and final snapshot exist
if [ -f .agents/evolve/fitness-0-baseline.json ] && [ -f .agents/evolve/fitness-latest.json ]; then
  baseline = load(".agents/evolve/fitness-0-baseline.json")
  final = load(".agents/evolve/fitness-latest.json")

  # Compute delta — goals that flipped between baseline and final
  improved_count = 0
  regressed_count = 0
  unchanged_count = 0
  delta_rows = []

  for final_goal in final.goals:
    baseline_goal = baseline.goals.find(g => g.id == final_goal.id)
    baseline_result = baseline_goal ? baseline_goal.result : "unknown"
    final_result = final_goal.result

    if baseline_result == "fail" and final_result == "pass":
      delta = "improved"
      improved_count += 1
    elif baseline_result == "pass" and final_result == "fail":
      delta = "regressed"
      regressed_count += 1
    else:
      delta = "unchanged"
      unchanged_count += 1

    delta_rows.append({goal_id: final_goal.id, baseline_result, final_result, delta})

  # Write session-fitness-delta.md with trajectory table
  cat > .agents/evolve/session-fitness-delta.md << EOF
  # Session Fitness Trajectory

  | goal_id | baseline_result | final_result | delta |
  |---------|-----------------|--------------|-------|
  $(for row in delta_rows: "| ${row.goal_id} | ${row.baseline_result} | ${row.final_result} | ${row.delta} |")

  **Summary:** ${improved_count} improved, ${regressed_count} regressed, ${unchanged_count} unchanged
  EOF

  # Include delta summary in user-facing teardown report
  log "Fitness trajectory: ${improved_count} improved, ${regressed_count} regressed, ${unchanged_count} unchanged"
fi
```

**Then write session summary:**

```bash
cat > .agents/evolve/session-summary.md << EOF
# /evolve Session Summary

**Date:** $(date -Iseconds)
**Cycles:** $CYCLE of $MAX_CYCLES
**Goals measured:** $(wc -l < GOALS.yaml goals)

## Cycle History
$(cat .agents/evolve/cycle-history.jsonl)

## Final Fitness
$(cat .agents/evolve/fitness-latest.json)

## Post-Mortem
<path to post-mortem report from above>

## Next Steps
- Run \`/evolve\` again to continue improving
- Run \`/evolve --dry-run\` to check current fitness without executing
- Create \`~/.config/evolve/KILL\` to prevent future runs
- Create \`.agents/evolve/STOP\` for a one-time local stop
EOF
```

Report to user:
```
## /evolve Complete

Cycles: N of M
Goals improved: X
Goals regressed: Y (reverted)
Goals unchanged: Z
Post-mortem: <verdict> (see <report-path>)

Run `/evolve` again to continue improving.
```
