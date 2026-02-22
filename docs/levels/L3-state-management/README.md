# L3 — State Management

Add issue tracking with beads for structured work.

## What You'll Learn

- Using `/plan` to decompose work into issues
- Beads commands for issue lifecycle
- Tracking dependencies between tasks
- Session close protocol

## Prerequisites

- Completed L2-persistence
- Comfortable with `.agents/` directory
- Beads CLI installed (`pip install beads`)

## Available Commands

| Command | Purpose |
|---------|---------|
| `/plan <goal>` | Decompose goal into beads issues |
| `/research <topic>` | Same as L2 |
| `/implement [id]` | Execute specific issue, then close it |
| `/retro [topic]` | Same as L2 |

## Beads Commands

```bash
bd ready                    # Show unblocked issues
bd list --status open       # All open issues
bd show <id>                # View issue details
bd update <id> --status in_progress
bd close <id> --reason "Done"
bd sync                     # Sync at session end
```

## Key Concepts

- **Issues**: Atomic units of work
- **Dependencies**: Issues can block each other
- **Session close**: `bd sync` before push

## What's NOT at This Level

- No parallel execution
- No `/crank` (autonomous execution)

## Next Level

Once comfortable with issue tracking, progress to [L4-parallelization](../L4-parallelization/) to execute waves.
