# Python Standards (Tier 1)

## Required
- `ruff check` passes (or `flake8`)
- `ruff format` (or `black`) for formatting
- Type hints on public functions
- Docstrings on public classes/functions

## Error Handling
- Never bare `except:` - always specify exception type
- Use `raise ... from e` to preserve stack traces
- Log before raising in library code

## Common Issues
| Pattern | Problem | Fix |
|---------|---------|-----|
| `except Exception:` | Too broad | Catch specific exceptions |
| `# type: ignore` | Hiding problems | Fix the type error |
| `eval()` / `exec()` | Security risk | Use safer alternatives |
| Mutable default args | Shared state bugs | Use `None` + conditional |

## Security
- Never use `eval()`, `exec()`, or `__import__()` with untrusted input
- Use `secrets` module for tokens, not `random`
- Validate and sanitize all external input (user data, file paths, URLs)
- Use parameterized queries for SQL — never string formatting

## Testing
- pytest preferred
- `conftest.py` for shared fixtures
- Mock external services, not internal code
