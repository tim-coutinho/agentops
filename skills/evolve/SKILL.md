---
name: evolve
description: Goal-driven fitness-scored improvement loop. Measures goals, picks worst gap, runs /rpi, compounds via knowledge flywheel. Also pulls from open beads when goals all pass.
user-invocable: true
metadata:
  tier: execution
  dependencies:
    - rpi         # required - executes each improvement cycle
    - post-mortem # required - auto-runs at teardown to harvest learnings
  triggers:
    - evolve
    - improve everything
    - autonomous improvement
    - run until done
---

# /evolve — Goal-Driven Compounding Loop

> Measure what's wrong. Fix the worst thing. Measure again. Compound.

Thin fitness-scored loop over `/rpi`. Three work sources in priority order:
1. **Failing GOALS.yaml goals** (highest-weight first)
2. **Open beads issues** (`bd ready` — when all goals pass)
3. **Harvested next-work.jsonl** (from prior /rpi post-mortems)

**Dormancy is success.** When all sources are empty, stop. Don't manufacture work.

```bash
/evolve                      # Run until kill switch or stagnation
/evolve --max-cycles=5       # Cap at 5 cycles
/evolve --dry-run            # Show what would be worked on, don't execute
/evolve --beads-only         # Skip goals measurement, work beads backlog only
```

## Execution Steps

**YOU MUST EXECUTE THIS WORKFLOW. Do not just describe it.**

### Step 0: Setup

```bash
mkdir -p .agents/evolve
ao inject 2>/dev/null || true
```

Recover cycle number from disk (survives context compaction):
```bash
if [ -f .agents/evolve/cycle-history.jsonl ]; then
  CYCLE=$(( $(tail -1 .agents/evolve/cycle-history.jsonl | jq -r '.cycle // 0') + 1 ))
else
  CYCLE=1
fi
SESSION_START_SHA=$(git rev-parse HEAD)
IDLE_STREAK=0
```

Parse flags: `--max-cycles=N` (default unlimited), `--dry-run`, `--beads-only`, `--skip-baseline`.

### Step 0.5: Baseline (first run only)

Skip if `--skip-baseline` or `--beads-only` or baseline already exists.

```bash
if [ ! -f .agents/evolve/fitness-0-baseline.json ]; then
  ao goals measure --json --timeout 60 > .agents/evolve/fitness-0-baseline.json
fi
```

### Step 1: Kill Switch Check

Run at the TOP of every cycle:

```bash
[ -f ~/.config/evolve/KILL ] && echo "KILL: $(cat ~/.config/evolve/KILL)" && exit 0
[ -f .agents/evolve/STOP ] && echo "STOP: $(cat .agents/evolve/STOP 2>/dev/null)" && exit 0
```

### Step 2: Measure Fitness

Skip if `--beads-only`.

```bash
ao goals measure --json --timeout 60 > .agents/evolve/fitness-${CYCLE}-pre.json
```

This writes a fitness snapshot to `.agents/evolve/`. If `ao goals measure` is unavailable, read `GOALS.yaml` and run each goal's `check` command manually. Mark timeouts as `"result": "skip"`.

### Step 3: Select Work

**Priority 1 — Failing goals** (skip if `--beads-only`):
```bash
FAILING=$(jq -r '.goals[] | select(.result=="fail") | .goal_id' .agents/evolve/fitness-${CYCLE}-pre.json | head -1)
```
Pick the highest-weight failing goal. Skip goals that regressed 3 consecutive cycles.

**Priority 2 — Open beads** (when all goals pass or `--beads-only`):
```bash
READY_ISSUE=$(bd ready -n 1 2>/dev/null | head -1 | awk '{print $2}')
```
Pick the highest-priority unblocked issue. Use `bd show $READY_ISSUE` for details.

**Priority 3 — Harvested work** from `.agents/rpi/next-work.jsonl` (unconsumed entries).

**Nothing found?** Increment IDLE_STREAK. Stop after 3 consecutive idle cycles (stagnation = success).

If `--dry-run`: report what would be worked on and go to Teardown.

### Step 4: Execute

For a **failing goal**:
```
Invoke /rpi "Improve {goal_id}: {description}" --auto --max-cycles=1
```

For a **beads issue**:
```
Invoke /implement {issue_id}
```
Or for an epic with children: `Invoke /crank {epic_id}`.

For **harvested work**:
```
Invoke /rpi "{item_title}" --auto --max-cycles=1
```
Then mark the item consumed in next-work.jsonl.

### Step 5: Regression Gate

After execution, verify nothing broke:

```bash
# Fast gate: build + vet + test
cd cli && go build ./cmd/ao/ && go vet ./... && go test ./... -count=1 -timeout 120s
```

If not `--beads-only`, also re-measure to produce a fitness-${CYCLE}-post snapshot:
```bash
ao goals measure --json --timeout 60 --goal $GOAL_ID > .agents/evolve/fitness-${CYCLE}-post.json
```

**If regression detected** (previously-passing goal now fails):
```bash
git revert HEAD --no-edit  # single commit
# or for multiple commits:
git revert --no-commit ${CYCLE_START_SHA}..HEAD && git commit -m "revert: evolve cycle ${CYCLE} regression"
```
Set outcome to "regressed".

### Step 6: Log Cycle + Commit

**HARD GATE: Every cycle MUST be logged and committed. This is what survives compaction.**

```bash
# Append to cycle history
echo "{\"cycle\":${CYCLE},\"target\":\"${TARGET}\",\"result\":\"${OUTCOME}\",\"sha\":\"$(git rev-parse --short HEAD)\",\"timestamp\":\"$(date -Iseconds)\"}" >> .agents/evolve/cycle-history.jsonl

# Verify write
LAST=$(tail -1 .agents/evolve/cycle-history.jsonl | jq -r '.cycle')
[ "$LAST" != "$CYCLE" ] && echo "FATAL: cycle log write failed" && exit 1

# Telemetry
bash scripts/log-telemetry.sh evolve cycle-complete cycle=${CYCLE} goal=${TARGET} outcome=${OUTCOME} 2>/dev/null || true

# Commit checkpoint (survives compaction)
git add .agents/evolve/ && git commit -m "evolve: cycle ${CYCLE} -- ${TARGET} ${OUTCOME}" || true
```

### Step 7: Loop or Stop

```bash
CYCLE=$((CYCLE + 1))
# Stop if max-cycles reached
# Otherwise: go to Step 1
```

Push periodically (every 3-5 cycles or at teardown):
```bash
git push
```

### Teardown

1. Run `/post-mortem "evolve session: ${CYCLE} cycles"` to harvest learnings.
2. Push all commits: `git push`.
3. Report summary:

```
## /evolve Complete
Cycles: N | Improved: X | Regressed: Y (reverted) | Unchanged: Z
```

## Examples

**Basic:** `/evolve --max-cycles=5` — measures goals, fixes highest-weight failure, gates, repeats for 5 cycles.

**Beads only:** `/evolve --beads-only` — skips goals measurement, works through `bd ready` backlog.

**Dry run:** `/evolve --dry-run` — shows what would be worked on without executing.

See `references/examples.md` for detailed walkthroughs.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Loop exits immediately | Remove `~/.config/evolve/KILL` or `.agents/evolve/STOP` |
| Stagnation after 3 idle cycles | All work sources empty — this is success |
| `ao goals measure` hangs | Use `--timeout 30` flag or `--beads-only` to skip |
| Regression gate reverts | Review reverted changes, narrow scope, re-run |

See `references/cycle-history.md` for advanced troubleshooting.

## References

- `references/cycle-history.md` — JSONL format, recovery protocol, kill switch
- `references/compounding.md` — Knowledge flywheel and work harvesting
- `references/goals-schema.md` — GOALS.yaml format and continuous metrics
- `references/parallel-execution.md` — Parallel /swarm architecture
- `references/teardown.md` — Trajectory computation and session summary
- `references/examples.md` — Detailed usage examples
- `references/artifacts.md` — Generated files registry

## See Also

- `skills/rpi/SKILL.md` — Full lifecycle orchestrator (called per cycle)
- `skills/crank/SKILL.md` — Epic execution (called for beads epics)
- `GOALS.yaml` — Fitness goals for this repo
