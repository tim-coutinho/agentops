# /evolve Artifacts

## Committed to Git

| File | Purpose |
|------|---------|
| `GOALS.yaml` | Fitness goals (repo root) |
| `.agents/evolve/fitness-0-baseline.json` | Cycle-0 baseline snapshot (comparison anchor) |
| `.agents/evolve/cycle-history.jsonl` | Cycle outcomes log (includes commit SHAs) |

## Local Only (gitignored)

| File | Purpose |
|------|---------|
| `.agents/evolve/fitness-latest.json` | Pre-cycle fitness snapshot (rolling, overwritten each cycle) |
| `.agents/evolve/fitness-latest-post.json` | Post-cycle fitness snapshot (for regression comparison) |
| `.agents/evolve/session-summary.md` | Session wrap-up |
| `.agents/evolve/session-fitness-delta.md` | Session fitness trajectory (baseline to final delta) |
| `.agents/evolve/STOP` | Local kill switch |
| `~/.config/evolve/KILL` | External kill switch |

## Removed (legacy)

These files are no longer generated. Older repos may have them in git history:

| File | Replacement |
|------|-------------|
| `.agents/evolve/fitness-{N}-pre.json` | `fitness-latest.json` (rolling) |
| `.agents/evolve/fitness-{N}-post.json` | `fitness-latest-post.json` (rolling) |
| `.agents/evolve/cycle-0-report.md` | Inlined into session-summary.md |
| `.agents/evolve/last-sweep-date` | No longer needed (baseline gate uses fitness-0-baseline.json) |
| `.agents/evolve/KILLED.json` | Kill switch acknowledgment removed (STOP file is sufficient) |
