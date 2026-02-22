---
description: Extract session learnings to .agents/learnings/
---

# /retro

Captures what you learned during a session. Saves insights to `.agents/learnings/` so they survive context clears.

## Usage

```
/retro
/retro "specific topic"
```

## When to Use

- End of a productive session
- After solving a tricky problem
- When you discover something worth remembering

## Steps

1. Claude reviews the session
2. Identifies key learnings and patterns
3. Writes to `.agents/learnings/YYYY-MM-DD-topic.md`
4. Summary shown in conversation

## Output

```
.agents/learnings/2025-01-15-auth-debugging.md
```

## Example

```
You: /retro

Claude: Reviewing this session...

**Session learnings saved to:** .agents/learnings/2025-01-15-session.md

Key insights:
- JWT tokens must be refreshed before 401, not after
- The middleware order in Express matters for auth
- Redis connection pooling prevents timeouts
```

## What Gets Captured

- Problems solved and how
- Patterns discovered
- Mistakes to avoid
- Useful commands or techniques

## Next

- Start fresh session, load learnings with "read .agents/learnings/..."
- `/research` for your next exploration
