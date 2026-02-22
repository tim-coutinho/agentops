---
description: Explore a codebase to understand how it works
---

# /research

Investigates code structure, patterns, and behavior. Use before making changes to unfamiliar code.

## Usage

```
/research "your topic"
```

## Steps

1. Claude searches for relevant files using glob/grep
2. Claude reads key files and traces connections
3. Claude summarizes findings in the conversation

## Output

Findings appear directly in the conversation. At L1, nothing persists to disk.

## Example

```
You: /research "how does authentication work"

Claude: I'll explore the authentication system.
[Searches for auth-related files]
[Reads src/auth/middleware.ts, src/auth/session.ts]

**Findings:**
- JWT-based auth in middleware.ts
- Sessions stored in Redis (session.ts:42)
- Token refresh happens on each request
```

## Next

Run `/implement` to make changes based on your research.
