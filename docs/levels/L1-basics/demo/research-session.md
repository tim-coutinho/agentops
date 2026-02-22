# Demo: /research Session

A realistic transcript of exploring how authentication works in a codebase.

---

## Before

- Fresh Claude Code session
- Unfamiliar with this codebase
- No `.agents/` directory exists

---

## Session

```
You: /research "how authentication works"

Claude: I'll explore the authentication system in this codebase.

[Glob] **/*.{ts,py} containing "auth"
Found: src/auth/middleware.ts, src/auth/session.ts, src/auth/jwt.ts, tests/auth.test.ts

[Grep] "login|authenticate|verify" in src/auth/
Found 12 matches across 3 files

[Read] src/auth/middleware.ts (45 lines)
[Read] src/auth/jwt.ts (62 lines)

**Findings:**

The auth system uses JWT tokens with Redis session storage:

1. **Entry point**: `middleware.ts` - validates token on every request
2. **Token handling**: `jwt.ts` - creates/verifies tokens, 15-min expiry
3. **Session storage**: Redis-backed, tokens refreshed on activity

Key flow:
- Request hits middleware -> extracts Bearer token
- Token verified via `jwt.verify()` -> user ID extracted
- Session checked in Redis -> request proceeds or 401

The refresh logic (jwt.ts:38-42) extends sessions on each valid request,
so active users stay logged in indefinitely.
```

---

## After

- You now understand the auth flow
- Key files identified for future reference
- No files were created - this was conversation only

---

## What You Learned

1. Claude uses glob/grep to find relevant files quickly
2. Research output lives in the conversation, not on disk
3. Good research identifies the "entry point" and traces the flow
4. At L1, you carry this knowledge in your head (or copy it somewhere)
