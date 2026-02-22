# Phase Data Contracts

How each consolidated phase passes data to the next. Artifacts are filesystem-based; no in-memory coupling between phases.

| Transition | Output | Extraction | Input to Next |
|------------|--------|------------|---------------|
| → Discovery | Research doc, plan doc, pre-mortem report, epic ID | Latest files in `.agents/research/`, `.agents/plans/`, `.agents/council/`; epic from `bd list --type epic --status open` | `epic_id`, `pre_mortem` verdict, and discovery summary are persisted in phased state |
| Discovery → Implementation | Epic execution context + discovery summary | `phased-state.json` + `.agents/rpi/phase-1-summary.md` | `/crank <epic-id>` with prior-phase context |
| Implementation → Validation | Completed/partial crank status + implementation summary | `bd children <epic-id>` + `.agents/rpi/phase-2-summary.md` | `/vibe` + `/post-mortem` with implementation context |
| Validation → Next Cycle (optional) | Vibe/post-mortem verdicts + harvested follow-up work | Latest council reports + `.agents/rpi/next-work.jsonl` | Stop, loop (`--loop`), or suggest next `/rpi` (`--spawn-next`) |
