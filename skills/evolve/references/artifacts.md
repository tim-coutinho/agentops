# /evolve Artifacts

| File | Purpose |
|------|---------|
| `GOALS.yaml` | Fitness goals (repo root) |
| `.agents/evolve/fitness-0-baseline.json` | Cycle-0 baseline snapshot (comparison anchor) |
| `.agents/evolve/cycle-0-report.md` | Baseline report (failing goals by weight) |
| `.agents/evolve/last-sweep-date` | Timestamp of last comprehensive sweep (cycle-0 refresh gate) |
| `.agents/evolve/fitness-{N}.json` | Pre-cycle fitness snapshot (continuous values) |
| `.agents/evolve/fitness-{N}-post.json` | Post-cycle fitness snapshot (for regression comparison) |
| `.agents/evolve/cycle-history.jsonl` | Cycle outcomes log (includes commit SHAs) |
| `.agents/evolve/session-summary.md` | Session wrap-up |
| `.agents/evolve/session-fitness-delta.md` | Session fitness trajectory (baseline to final delta) |
| `.agents/evolve/STOP` | Local kill switch |
| `.agents/evolve/KILLED.json` | Kill acknowledgment |
| `~/.config/evolve/KILL` | External kill switch |
