---
description: Make changes, validate, commit
---

# /implement

Execute a code change with validation. Use after you know what to change.

---

## Usage

```
/implement
```

---

## Steps

1. **State the change** - Tell Claude what to modify and why
2. **Claude makes changes** - Files are edited or created
3. **Validate** - Run tests or linting
4. **Commit** - Changes saved to git

---

## Output

Modified files, validation results, git commit.

---

## Example

```
You: /implement - Add logging to the auth module

Claude: Adding logging to auth.py...
[reads auth.py, edits auth.py]
$ pytest tests/test_auth.py
3 passed
$ git commit -m "feat(auth): add logging for login attempts"
```

---

## Next

`/research <topic>` to explore, or `git show HEAD` to review.