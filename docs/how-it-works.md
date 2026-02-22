# How It Works

> Agent output quality is determined by context input quality. Every pattern below — fresh context per worker, ratcheted progress, least-privilege loading — exists to ensure the right information is in the right window at the right time.

Parallel agents produce noisy output; councils filter it; ratchets lock progress so it can never regress.

## The Brownian Ratchet

*A mechanism borrowed from molecular physics: random motion is captured by one-way gates, converting chaos into forward progress.*

Chaos in, locked progress out.

```
  ╭─ agent-1 ─→ ✓ ─╮
  ├─ agent-2 ─→ ✗ ─┤   3 attempts, 1 fails
  ├─ agent-3 ─→ ✓ ─┤   council catches it
  ╰─ council ──→ PASS   ratchet locks the result
                  ↓
          can't go backward
```

Spawn parallel agents (chaos), validate with multi-model council (filter), merge to main (ratchet). Failed agents are cheap — fresh context means no contamination.

See also: [Brownian Ratchet (deep dive)](brownian-ratchet.md)

## Ralph Wiggum Pattern — Fresh Context Every Wave

*Named after Ralph Wiggum's "I'm helping!" -- each worker starts fresh with no memory of previous workers, ensuring complete isolation between waves.*

```
  Wave 1:  spawn 3 workers → write files → lead validates → lead commits
  Wave 2:  spawn 2 workers → ...same pattern, zero accumulated context
```

Every wave gets a fresh worker set. Every worker gets clean context. No bleed-through between waves. The lead is the only one who commits.

Supports both Codex sub-agents (`spawn_agent`) and Claude agent teams (`TeamCreate`).

Operational contract reference: `skills/shared/references/ralph-loop-contract.md` (reverse-engineered from `ghuntley/how-to-ralph-wiggum` and mapped to AgentOps primitives).

## Two-Tier Execution Model

Skills follow a strict rule: **the orchestrator never forks; the workers it spawns always fork.**

| Tier | Skills | Behavior |
|------|--------|----------|
| **NO-FORK** (orchestrators) | evolve, rpi, crank, vibe, post-mortem, pre-mortem | Stay in main session — operator sees progress and can intervene |
| **FORK** (worker spawners) | council, codex-team | Fork into subagents — results merge back via filesystem |

This was learned through production experience: orchestrators that forked (`context: fork` in SKILL.md) became invisible — the operator couldn't see cycle-by-cycle evolve progress, phase transitions in rpi, or wave reports from crank. The fix: remove `context: fork` from orchestrators, keep it only on worker spawners.

`/swarm` is a special case — it's an orchestrator (no fork) that spawns runtime workers via `TeamCreate`/`spawn_agent`. The workers are runtime sub-agents, not SKILL.md skills.

Full classification: [`SKILL-TIERS.md`](../skills/SKILL-TIERS.md)

## Agent Backends — Runtime-Native Orchestration

Skills auto-select the best available backend:

1. Runtime-native backend first:
   Claude sessions → Claude native teams (`TeamCreate` + `SendMessage`)
   Codex sessions → Codex sub-agents (`spawn_agent`)
2. Secondary/mixed backend only when explicitly requested
3. Background task fallback (`Task(run_in_background=true)`)

```
  Council:                               Swarm:
  ╭─ judge-1 ──╮                  ╭─ worker-1 ──╮
  ├─ judge-2 ──┼→ consolidate     ├─ worker-2 ──┼→ validate + commit
  ╰─ judge-3 ──╯                  ╰─ worker-3 ──╯
```

**Claude teams setup** (optional):
```json
// ~/.claude/settings.json
{
  "teammateMode": "tmux",
  "env": { "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1" }
}
```

## Hooks — The Workflow Enforces Itself

12 hooks. All have a kill switch: `AGENTOPS_HOOKS_DISABLED=1`.

| Hook | Trigger | What it does |
|------|---------|-------------|
| Push gate | `git push` | Blocks push if `/vibe` hasn't passed |
| Pre-mortem gate | `/crank` invocation | Blocks `/crank` if `/pre-mortem` hasn't passed |
| Worker guard | `git commit` | Blocks workers from committing (lead-only) |
| Dangerous git guard | `force-push`, `reset --hard` | Blocks destructive git commands |
| Standards injector | Write/Edit | Auto-injects language-specific coding rules |
| Ratchet nudge | Any prompt | "Run /vibe before pushing" |
| Task validation | Task completed | Validates metadata before accepting |
| Session start | Session start | Knowledge injection, stale state cleanup |
| Ratchet advance | After Bash | Locks progress gates |
| Stop team guard | Session stop | Prevents premature stop with active teams |
| Precompact snapshot | Before compaction | Saves state before context compaction |
| Pending cleaner | Session start | Cleans stale pending state |

All hooks use `lib/hook-helpers.sh` for structured error recovery — failures include suggested next actions and auto-handoff context.

## Compaction Resilience — Long Runs That Don't Lose State

LLM context compaction can destroy loop state mid-run. Any skill that runs for hours (especially `/evolve`) must store state on disk, not in LLM memory.

The pattern:
1. **Write state to disk after every step** — `cycle-history.jsonl`, fitness snapshots, heartbeat
2. **Recover from disk on every resume** — read last cycle number from JSONL, not from conversation context
3. **Verify writes succeeded** — read back the entry, compare, stop if mismatch

Hard gates in `/evolve`:
- Pre-cycle: fitness snapshot must exist and be valid JSON before the regression gate runs
- Post-cycle: cycle-history.jsonl write is verified; failure = stop
- Loop entry: continuity check confirms cycle N was logged before starting N+1

This was validated in production: 116 evolve cycles ran ~7 hours overnight. The first run revealed that without disk-based recovery, context compaction silently broke tracking after cycle 1 — the agent continued producing valuable work but without formal regression gating. The hardening above prevents this class of failure.

## Context Windowing — Bounded Execution for Large Codebases

For repos over ~1500 files, `/rpi` uses deterministic shards to keep each worker's context window bounded. Run `scripts/rpi/context-window-contract.sh` before `/rpi` to enable sharding. This prevents context overflow and keeps worker quality consistent regardless of codebase size.

## See Also

- [Architecture](ARCHITECTURE.md) — System design and component overview
- [Brownian Ratchet](brownian-ratchet.md) — AI-native development philosophy
- [The Science](the-science.md) — Research behind knowledge decay and compounding
- [Glossary](GLOSSARY.md) — Definitions of key terms and metaphors
