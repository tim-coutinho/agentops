# /evolve Examples

## Infinite Autonomous Improvement

**User says:** `/evolve`

**What happens:**
1. Agent checks kill switch files (none found, continues)
2. Agent measures fitness against GOALS.yaml (3 of 5 goals passing)
3. Agent selects worst-failing goal by weight (test-pass-rate)
4. Agent invokes `/rpi "Improve test-pass-rate"` with full lifecycle
5. Agent re-measures fitness post-cycle (test-pass-rate now passing, all others unchanged)
6. Agent logs cycle to history, increments cycle counter
7. Agent loops to next cycle, selects next failing goal
8. After all goals pass, agent checks harvested work from post-mortem — finds 3 items
9. Agent works through harvested items, each generating more via post-mortem
10. After 3 consecutive idle cycles (no failing goals, no harvested work), agent runs `/post-mortem` and writes session summary
11. To stop earlier: create `~/.config/evolve/KILL` or `.agents/evolve/STOP`

**Result:** Runs forever — fixing goals, consuming harvested work, re-measuring. Only stops on kill switch or stagnation (3 idle cycles).

## Dry-Run Mode

**User says:** `/evolve --dry-run`

**What happens:**
1. Agent measures fitness (3 of 5 goals passing)
2. Agent identifies worst-failing goal (doc-coverage, weight 5)
3. Agent reports what would be worked on: "Dry run: would work on 'doc-coverage' (weight: 5)"
4. Agent shows harvested work queue (2 items from prior RPI cycles)
5. Agent stops without executing

**Result:** Fitness report and next-action preview without code changes.

## Regression with Revert

**User says:** `/evolve --max-cycles=3`

**What happens:**
1. Agent improves goal A in cycle 1 (commit abc123)
2. Agent measures fitness post-cycle: goal A passes, but goal B now fails (regression)
3. Agent reverts commit abc123 with annotated message
4. Agent logs regression to history, moves to next goal
5. Agent completes 3 cycles (cap reached), runs post-mortem

**Result:** Fitness regressions are auto-reverted, preventing compounding failures.

## Parallel Goal Improvement

**User says:** `/evolve --parallel`

**What happens:**
1. Agent checks kill switch (none found)
2. Agent measures fitness against GOALS.yaml (4 of 7 goals failing)
3. Agent selects top 3 independent failing goals by weight, filtered for independence via `select_parallel_goals`
4. Agent creates TaskList tasks for each goal, sets up artifact isolation
5. Agent invokes `/swarm --worktrees` — spawns 3 fresh-context workers in isolated worktrees
6. Each worker runs a full `/rpi` cycle independently (research → plan → crank → vibe → post-mortem)
7. `/swarm` completes — all 3 workers done, lead merges worktrees
8. Agent re-measures ALL goals (single regression gate for entire wave)
9. No regression detected — logs cycle with `goal_ids` array and `parallel: true`
10. Next cycle: 1 remaining failing goal → sequential (only 1 goal, no parallelism needed)
11. After all goals pass, checks harvested work, eventually stagnation → teardown

**Result:** 3 goals fixed in ~1 cycle instead of 3 sequential cycles. ~3x speedup for independent goals. Each worker's /post-mortem feeds the knowledge flywheel independently.

## Parallel with Regression Revert

**User says:** `/evolve --parallel --max-cycles=2`

**What happens:**
1. Cycle 1: 3 parallel goals attempted via `/swarm --worktrees`
2. Post-wave regression gate detects goal C started failing after goals A and B were improved
3. Agent reverts entire parallel wave (all merged worktree commits) using `cycle_start_sha`
4. Logs cycle with `result: "regressed"` and all 3 `goal_ids`
5. Cycle 2: Agent retries — `select_parallel_goals` still selects same 3 (different check scripts)
6. This time no regression — all 3 improvements are clean
7. Max cycles reached (2), runs teardown with `/post-mortem`

**Result:** Parallel regression detected and reverted cleanly. Entire wave rolled back as a unit. Retry in next cycle succeeds.
