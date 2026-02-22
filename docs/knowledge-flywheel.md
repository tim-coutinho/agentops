# The Knowledge Flywheel

> **"Every session makes the next one smarter."**

## The Problem

AI assistants forget everything between sessions. Your team solves the same problems over and over.

## The Solution

Olympus mines your Claude transcripts and extracts reusable knowledge:
- Decisions and their rationale
- Solutions that worked
- Patterns worth repeating
- Mistakes to avoid

## The Flywheel

```
Mine transcripts → Extract knowledge → Index for recall → Apply to new work → Learn more
                                                                    ↓
                                                         Compounds over time
```

## Architecture

```
┌───────────────────────────────────────────────────────────┐
│                      THE FLYWHEEL                          │
│                                                            │
│  .agents/patterns/  ──▶  Next /research reads              │
│  .agents/retros/    ──▶  Smart Connections indexes         │
│  .agents/learnings/ ──▶  Knowledge compounds               │
│                                                            │
└───────────────────────────────────────────────────────────┘
                              │
                              ▼
                         NEXT CHAIN
```

## Knowledge Stores

| Store | Content | Updated By |
|-------|---------|------------|
| `.agents/learnings/` | Lessons learned | `/forge`, `/post-mortem` |
| `.agents/patterns/` | Reusable patterns | `/forge`, `/retro` |
| `.agents/retros/` | Retrospectives | `/retro`, `/post-mortem` |
| `.agents/ao/` | Session index, provenance | `ao forge` |

## The Compounding Effect

| Timeline | Claude Knows |
|----------|--------------|
| Day 1 | Nothing - fresh start |
| Week 1 | Your coding patterns |
| Month 1 | Your codebase |
| Month 3 | Your organization |

## See Also

- [Brownian Ratchet Philosophy](brownian-ratchet.md)
- `/forge` - Extract knowledge from transcripts
- `/inject` - Recall knowledge at session start
