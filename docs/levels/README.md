# AgentOps Levels — Progressive Learning Path

Inspired by Kelsey Hightower's approach: learn by doing, one level at a time.

## Tiers vs Levels

AgentOps has two progression axes:

- **Tiers (0-3)** = what tools you install. See the [Install section](../../README.md#install).
- **Levels (L1-L5)** = what concepts you learn. That's this document.

Tiers tell you **what to install**. Levels tell you **what to learn**. You can be at Tier 0 (skills only, no CLI) and still work through all five learning levels. The CLI and beads enhance the experience but aren't required to learn the concepts.

| Tier | Required for Levels | What It Adds |
|------|-------------------|-------------|
| Tier 0 (skills only) | L1-L2 | `/research`, `/pre-mortem`, `/vibe` work immediately |
| Tier 1 (+ `ao` CLI) | L2-L3 | Knowledge persists across sessions, auto-inject/extract |
| Tier 2 (+ beads) | L3-L5 | Issue tracking, epic orchestration |
| Tier 3 (+ codex) | Any | Cross-vendor consensus (`/council --mixed`) |

## Philosophy

Each level builds on the previous. Master L1 before attempting L2. The levels are designed to be:

- **Progressive** — Each level adds ONE new concept
- **Practical** — Real demos, not just theory
- **Incremental** — Break changes into small, verifiable steps

## The Five Levels

| Level | Name | Core Concept | New Capability |
|-------|------|--------------|----------------|
| L1 | Basics | Single-session work | `/research`, `/implement` (single issue) |
| L2 | Persistence | `.agents/` output | State survives sessions |
| L3 | State Management | Issue tracking | `/plan`, beads integration |
| L4 | Parallelization | Wave execution | `/implement-wave` |
| L5 | Orchestration | Full autonomy | `/crank`, gastown multi-agent |

## Progression Path

```
L1 (Gateway)
    ↓
L2 (Add persistence)
    ↓
L3 (Add tracking)
    ↓
L4 (Add parallelism)
    ↓
L5 (Full autonomy)
```

## Level Details

### L1 — Basics
Single-session work. No state persistence. Use `/research` to explore, `/implement` to build. Changes exist only in git.

### L2 — Persistence
Add `.agents/` directory. Research documents, patterns, and learnings persist across sessions. The AI has memory.

### L3 — State Management
Add issue tracking with beads. `/plan` creates structured work items. Track progress across sessions.

### L4 — Parallelization
Execute independent work in parallel. `/implement-wave` runs multiple issues concurrently. Speed through unblocked work.

### L5 — Orchestration
Full autonomous operation. `/crank` handles epic-to-completion via the ODMCR reconciliation loop. Integrates with gastown for multi-agent parallelization.

## Getting Started

Start with [L1-basics/](./L1-basics/). Read its README, run the demos, then progress to L2.

## Directory Contents

Each level directory contains:
- `README.md` — What you'll learn, prerequisites, available commands
- `demo/` — Real session transcripts showing the level in action
