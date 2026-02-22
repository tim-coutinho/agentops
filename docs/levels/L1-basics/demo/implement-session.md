# Demo: /implement Session

A realistic transcript of adding a feature with validation and commit.

---

## Before

- You've already researched the auth module
- Know where to make changes
- Tests exist and pass

---

## Session

```
You: /implement - Add logging when users fail authentication

Claude: I'll add logging for failed auth attempts.

[Read] src/auth/middleware.ts

I see the auth check at line 23. I'll add logging before the 401 response.

[Edit] src/auth/middleware.ts
- Added: import { logger } from '../utils/logger'
- Added at line 28: logger.warn('Auth failed', { ip: req.ip, path: req.path })

[Bash] pytest tests/test_auth.py
===== 4 passed in 0.3s =====

[Bash] npm run lint
No issues found

Changes validated. Ready to commit?

You: Yes

[Bash] git add src/auth/middleware.ts
[Bash] git commit -m "feat(auth): log failed authentication attempts"
[main a1b2c3d] feat(auth): log failed authentication attempts
 1 file changed, 3 insertions(+)

Done. Failed auth attempts now logged with IP and path.
```

---

## After

- `src/auth/middleware.ts` modified with logging
- Tests pass, lint clean
- Commit created on current branch

---

## What You Learned

1. Always validate before committing (tests, lint)
2. Claude shows you what changed before committing
3. Small, focused changes are easier to validate
4. The commit message follows conventional format
5. At L1, you're doing the orchestration - Claude just executes
