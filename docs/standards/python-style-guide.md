# Python Style Guide

<!-- Canonical source: gitops/docs/standards/python-style-guide.md -->
<!-- Last synced: 2026-01-19 -->

> **Purpose**: Unified Python coding standards for this repository.

## Scope

This document covers: code formatting, complexity management, refactoring patterns, naming, type hints, error handling, logging, and testing.

**Related:**
- [Shell Script Standards](./shell-script-standards.md) - Bash scripting conventions
- [Tag Vocabulary](./tag-vocabulary.md) - Documentation standards

---

## Table of Contents

1. [Python Version](#python-version)
2. [Project Structure](#project-structure)
3. [Package Management](#package-management)
4. [Code Formatting](#code-formatting)
5. [Code Complexity](#code-complexity)
6. [Naming Conventions](#naming-conventions)
7. [Type Hints](#type-hints)
8. [Docstrings](#docstrings)
9. [Error Handling](#error-handling)
10. [Logging](#logging)
11. [Testing](#testing)
12. [CLI Script Template](#cli-script-template)

---

## Python Version

**Required:** Python 3.12+

All scripts MUST use Python 3.12 or later. This ensures:
- Modern syntax support (match statements, improved type hints)
- Performance improvements
- Consistency with container images

**pyproject.toml configuration:**
```toml
[project]
requires-python = ">=3.12"
```

---

## Project Structure

Standard Python structure:

```text
scripts/
├── my_script.py             # Standalone CLI tools
├── lib/                     # Shared libraries (if needed)
│   ├── __init__.py
│   └── helpers.py
└── tests/                   # Test suite
    ├── __init__.py
    ├── conftest.py          # Pytest fixtures
    └── test_my_script.py
```

**Key Principles:**
- CLI scripts are standalone files in `scripts/`
- Shared code goes in `scripts/lib/`
- Tests mirror script structure

---

## Package Management

**Quick Reference:**

| Tool | Use For | Install |
|------|---------|---------|
| **uv** | Project dependencies | `brew install uv` |
| **pipx** | Global CLI tools | `brew install pipx` |
| **brew** | System tools | (macOS default) |
| ~~pip~~ | Avoid | Use uv instead |

### When to Use What

```
Need to...                          -> Use
---------------------------------------------
Install project dependencies        -> uv sync
Add a new library to project        -> uv add requests
Install a CLI tool globally         -> pipx install ruff
Install a system tool               -> brew install shellcheck
Quick one-off script                -> uv run script.py
```

### Project Dependencies (uv)

Use `uv` for all project-level Python dependencies. It's 10-100x faster than pip and creates deterministic builds via lock files.

```bash
# Initialize a new project
uv init my-project
cd my-project

# Add dependencies
uv add requests pyyaml        # Runtime deps
uv add --dev pytest ruff      # Dev deps

# Install from existing pyproject.toml
uv sync                       # Creates/updates uv.lock

# Run a script with project deps
uv run python my_script.py
```

**Always commit `uv.lock`** - this ensures reproducible builds across machines.

### Global CLI Tools (pipx)

Use `pipx` for Python CLI tools you want available everywhere, isolated from projects.

```bash
# Install CLI tools globally
pipx install ruff             # Linter/formatter
pipx install radon            # Complexity analysis
pipx install xenon            # Complexity enforcement
pipx install pre-commit       # Git hooks

# Upgrade all
pipx upgrade-all

# Run without installing
pipx run cowsay "hello"
```

**Why not pip install globally?** Global pip installs pollute your system Python and cause version conflicts. pipx creates isolated venvs per tool.

### What NOT to Do

```bash
# Don't use pip globally
pip install requests          # Pollutes system Python
sudo pip install anything     # Even worse

# Don't mix package managers in a project
pip install requests          # Now you have pip AND uv deps
uv add pyyaml                 # Conflicts likely

# Don't commit venv/
git add .venv/                # Use .gitignore

# Do this instead
uv add requests               # Add to project
uv sync                       # Install everything
```

---

## Code Formatting

**Tool:** [ruff](https://github.com/astral-sh/ruff) (linter + formatter)

**Configuration in `pyproject.toml`:**
```toml
[tool.ruff]
line-length = 100
target-version = "py312"

[tool.ruff.lint]
select = [
    "E",   # pycodestyle errors
    "W",   # pycodestyle warnings
    "F",   # pyflakes
    "I",   # isort
    "N",   # pep8-naming
    "UP",  # pyupgrade
    "B",   # flake8-bugbear
    "C4",  # flake8-comprehensions
    "SIM", # flake8-simplify
]
ignore = [
    "E501",  # line-too-long (handled by formatter)
]

[tool.ruff.format]
quote-style = "double"
indent-style = "space"
```

**Usage:**
```bash
# Check linting
ruff check scripts/

# Auto-fix issues
ruff check --fix scripts/

# Format code
ruff format scripts/
```

---

## Code Complexity

**Required:** Maximum cyclomatic complexity of 10 (Grade B) per function

**Why it matters:**
- Cyclomatic complexity = number of independent paths through code
- CC > 10 means exponentially more test cases needed for coverage
- High complexity correlates with defect density
- Humans (and LLMs) struggle to reason about deeply nested logic

### Complexity Grades

| Grade | CC Range | Meaning | Action |
|-------|----------|---------|--------|
| **A** | 1-5 | Simple, low risk | Ideal |
| **B** | 6-10 | Moderate, acceptable | Acceptable |
| **C** | 11-20 | Complex, hard to test | Refactor when touching |
| **D** | 21-30 | Very complex, high risk | Must refactor |
| **E** | 31-40 | Extremely complex | Urgent refactor |
| **F** | 41+ | Unmaintainable | Critical - block merges |

### Checking Complexity Locally

**Tools:** `radon` (analysis) + `xenon` (enforcement)

```bash
# Install
pipx install radon xenon

# Check specific file
radon cc scripts/my_script.py -s -a

# Fail if any function exceeds Grade B (CC > 10)
xenon scripts/ --max-absolute B

# Show only Grade C or worse
radon cc scripts/ -s -n C
```

### Reducing Complexity

#### Pattern 1: Dispatch Pattern (Handler Registry)

**When to use:** Functions with if/elif chains that dispatch based on mode or type.

```python
# Bad - if/elif chain (CC=18+)
def main():
    if args.patch:
        # 90 lines of patch logic
    elif args.read:
        # 20 lines of read logic
    else:
        # 100 lines of write logic

# Good - Dispatch pattern (CC=6)
def _handle_patch_mode(args, client):
    """Handle --patch mode."""
    # Focused patch logic

def _handle_read_mode(args, client):
    """Handle --read mode."""
    # Focused read logic

def main():
    args = parse_args()
    client = build_client()

    if args.patch:
        _handle_patch_mode(args, client)
    elif args.read:
        _handle_read_mode(args, client)
    else:
        _handle_write_mode(args, client)
```

#### Pattern 2: Early Returns (Guard Clauses)

```python
# Bad - Deep nesting (CC=8)
def validate_document(doc):
    if doc:
        if doc.content:
            if len(doc.content) > 0:
                if doc.tenant:
                    return True
    return False

# Good - Guard clauses (CC=4)
def validate_document(doc):
    if not doc:
        return False
    if not doc.content:
        return False
    if len(doc.content) == 0:
        return False
    if not doc.tenant:
        return False
    return True
```

#### Pattern 3: Lookup Tables

```python
# Bad - Each 'or' adds +1 CC
def normalize_field(key, value):
    if key == "tls.crt" or key == "tls.key" or key == "ca":
        return normalize_cert_field(value)
    elif key == "config.json":
        return normalize_pull_secret_json(value)
    else:
        return value

# Good - O(1) lookup
NORMALIZERS = {
    "tls.crt": normalize_cert_field,
    "tls.key": normalize_cert_field,
    "ca": normalize_cert_field,
    "config.json": normalize_pull_secret_json,
}

def normalize_field(key, value):
    normalizer = NORMALIZERS.get(key)
    return normalizer(value) if normalizer else value
```

### Helper Naming Convention

| Prefix | Meaning | Example |
|--------|---------|---------|
| `_handle_` | Mode/dispatch handler | `_handle_patch_mode()` |
| `_process_` | Processing helper | `_process_secret()` |
| `_validate_` | Validation helper | `_validate_cert()` |
| `_setup_` | Initialization helper | `_setup_mount_point()` |
| `_normalize_` | Data normalization | `_normalize_cert_field()` |
| `_build_` | Construction | `_build_audit_metadata()` |

---

## Naming Conventions

Follow PEP 8 naming conventions:

| Element | Convention | Example |
|---------|------------|---------|
| **Modules** | `snake_case.py` | `my_script.py` |
| **Classes** | `PascalCase` | `MyClient` |
| **Functions** | `snake_case()` | `get_secret()` |
| **Variables** | `snake_case` | `mount_point` |
| **Constants** | `UPPER_SNAKE_CASE` | `MAX_RETRIES` |
| **Private** | `_leading_underscore` | `_internal_helper()` |

---

## Type Hints

**Required:** Type hints for all public functions

**Preferred Style:**
- Use modern syntax (`list[str]` not `List[str]`)
- Use `|` for unions (`str | None` not `Optional[str]`)
- Add return type annotations

```python
from __future__ import annotations

# Good - Modern syntax
def process_secrets(
    secrets: dict[str, dict],
    only: set[str] | None = None,
) -> list[str]:
    """Process secrets and return names processed."""
    results: list[str] = []
    for name, payload in secrets.items():
        if only and name not in only:
            continue
        results.append(name)
    return results

# Bad - Old syntax
from typing import Dict, List, Optional

def process_secrets(
    secrets: Dict[str, Dict],
    only: Optional[List[str]] = None,
) -> List[str]:
    pass
```

---

## Docstrings

**Required:** All public functions MUST have docstrings

**Style:** Google-style docstrings

```python
def verify_secret_after_write(
    client: hvac.Client,
    mount_point: str,
    name: str,
    expected_payload: dict[str, Any],
) -> bool:
    """Verify secret was written correctly.

    Args:
        client: Vault client
        mount_point: KV v2 mount point
        name: Secret name
        expected_payload: Expected secret data

    Returns:
        True if verification passed, False if any check failed

    Raises:
        hvac.exceptions.InvalidPath: If secret path is invalid
    """
    pass
```

---

## Error Handling

**Principles:**
1. **Use specific exception types** - Never catch bare `Exception` (unless re-raising)
2. **Fail fast** - Don't swallow errors silently
3. **Provide context** - Include relevant data in error messages
4. **Log warnings** - If catching exceptions, log what happened

### Good Patterns

```python
# Good - Specific exception, logged
try:
    cert_info = validate_certificate(payload["tls.crt"])
except subprocess.CalledProcessError as exc:
    logging.warning(f"Certificate validation failed: {exc}")

# Good - Specific types for format detection
try:
    decoded = base64.b64decode(data)
except (UnicodeDecodeError, base64.binascii.Error, ValueError) as exc:
    logging.debug(f"Not base64, assuming PEM format: {exc}")
    decoded = data

# Good - Re-raise with context
try:
    result = subprocess.run(cmd, check=True, capture_output=True)
except subprocess.CalledProcessError as exc:
    raise RuntimeError(f"Command failed: {cmd}") from exc
```

### Bad Patterns

```python
# Bad - Bare exception, swallowed
try:
    validate_something()
except Exception:
    pass  # Silent failure!

# Bad - Catching Exception without re-raising
try:
    process_data()
except Exception as e:
    logging.error(f"Error: {e}")
    return None  # Hides the problem
```

---

## Logging

**Library:** Python standard `logging` module

```python
import logging

logging.basicConfig(
    format="%(asctime)s %(levelname)s %(message)s",
    level=logging.INFO,
)

# Use module logger
log = logging.getLogger(__name__)

# Or use root logger for simple scripts
logging.info("Processing secret: %s", secret_name)
```

**Log Levels:**
- `DEBUG` - Detailed diagnostic (development only)
- `INFO` - Key events, progress
- `WARNING` - Recoverable issues
- `ERROR` - Operation failed

**Best Practices:**
```python
# Good - Context included
logging.info(f"Prepared {secret_name}: {preview}")
logging.warning(f"Security policy check failed for {key}: {exc}")

# Bad - No context
logging.info("Processing...")
logging.error(str(e))
```

---

## Testing

**Framework:** pytest

**Structure:**
```text
scripts/
├── my_script.py
└── tests/
    ├── conftest.py           # Shared fixtures
    ├── test_my_script.py     # Unit tests
    └── e2e/                   # End-to-end tests
        ├── conftest.py       # Testcontainers fixtures
        └── test_integration.py
```

**Running tests:**
```bash
# Run all tests
pytest scripts/tests/

# Run with coverage
pytest scripts/tests/ --cov=scripts --cov-report=term-missing

# Run only E2E tests
pytest scripts/tests/e2e/ -m e2e
```

**Naming:**
- Test files: `test_*.py`
- Test functions: `test_*`

### Testcontainers for E2E Tests

**What it is:** [Testcontainers](https://testcontainers.com/) is a library that spins up real Docker containers during test execution, then tears them down automatically.

**Why use it instead of mocks:**

| Approach | Problem |
|----------|---------|
| Mocks | Don't catch real DB behavior (SQL syntax, constraints) |
| SQLite substitute | Different SQL dialect, missing extensions |
| Shared test DB | Tests interfere with each other, flaky results |
| **Testcontainers** | Real DB, isolated per run, reproducible |

**When to use testcontainers:**
- Testing database queries
- Testing against message queues (Redis, RabbitMQ)
- Any E2E test that needs real infrastructure

**Installation:**
```bash
pip install testcontainers[postgres]  # PostgreSQL
pip install testcontainers[redis]     # Redis
```

**Example - PostgreSQL:**
```python
# tests/e2e/conftest.py
import pytest
from testcontainers.postgres import PostgresContainer

@pytest.fixture(scope="session")
def postgres_container():
    """Spin up PostgreSQL for E2E tests."""
    with PostgresContainer("postgres:16") as postgres:
        yield postgres
    # Container automatically cleaned up

@pytest.fixture
def db_connection(postgres_container):
    """Get connection to test database."""
    import psycopg
    conn_str = postgres_container.get_connection_url()
    with psycopg.connect(conn_str) as conn:
        yield conn
```

**pytest marker configuration (pyproject.toml):**
```toml
[tool.pytest.ini_options]
markers = [
    "e2e: marks tests as end-to-end (require Docker for testcontainers)",
]
```

---

## CLI Script Template

Standard template for standalone CLI scripts:

```python
#!/usr/bin/env python3
"""One-line description of what this script does.

Usage:
    python3 script_name.py --config config.yaml --apply

Exit Codes:
    0 - Success
    1 - Argument/configuration error
    2 - Runtime error
"""

from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path
from typing import Any

logging.basicConfig(
    format="%(asctime)s %(levelname)s %(message)s",
    level=logging.INFO,
)


def die(message: str) -> None:
    """Print error message and exit with code 1."""
    logging.error(message)
    sys.exit(1)


def parse_args() -> argparse.Namespace:
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", default="config.yaml", help="Path to config file")
    parser.add_argument("--apply", action="store_true", help="Apply changes")
    parser.add_argument("--dry-run", action="store_true", help="Show what would be done")
    return parser.parse_args()


def main() -> int:
    """Main entry point."""
    args = parse_args()

    if not args.apply:
        args.dry_run = True
        logging.info("Dry-run mode (use --apply to make changes)")

    # Main logic here
    try:
        # ... implementation
        pass
    except Exception as exc:
        logging.error(f"Failed: {exc}")
        return 2

    return 0


if __name__ == "__main__":
    sys.exit(main())
```

---

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| `ruff check` fails | Code style violations | Run `ruff check --fix` |
| `xenon` reports Grade C+ | CC > 10 in function | Apply refactoring patterns above |
| `TypeError: NoneType` | Missing null check | Add guard clause or `if x is not None` |
| Import cycle error | Circular imports | Move imports inside function or restructure |
| `ModuleNotFoundError` | Missing dependency | Run `uv sync` or `uv add <package>` |
| Type hint error | Wrong type annotation | Use modern syntax: `list[str]` not `List[str]` |
| f-string in logging | Using `f"..."` with logging | Use `%s` formatting: `logging.info("x: %s", x)` |

---

## Anti-Patterns

| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| God Function | Single function >100 lines | Untestable, high CC | Dispatch pattern, extract helpers |
| Bare Except | `except Exception: pass` | Hides bugs silently | Specific exceptions, log warnings |
| Print Debugging | `print("debug:", x)` | No levels, no timestamps | Use `logging.debug()` |
| Stringly Typed | Using strings for enums | No validation, typos | Use `enum.Enum` or Literal types |
| Deep Nesting | 4+ indent levels | Hard to follow, high CC | Early returns, extract functions |
| Global State | Module-level mutable state | Hidden dependencies, race conditions | Pass state explicitly |
| Any Typed | `def foo(x: Any)` | Defeats type checking | Use generics or specific types |
| Magic Strings | `if mode == "patch"` | Typo-prone, scattered | Define constants: `MODE_PATCH = "patch"` |

---

## AI Agent Guidelines

When AI agents write Python for this repo:

| Guideline | Rationale |
|-----------|-----------|
| ALWAYS run `ruff check` before committing | Catches style issues immediately |
| ALWAYS check complexity with `radon cc -s` | Prevents CC creep |
| NEVER add `# type: ignore` without comment | Explain why type check fails |
| NEVER use `subprocess.shell=True` | Security risk, use list args |
| PREFER `pathlib.Path` over `os.path` | Modern, cross-platform |
| PREFER dataclasses over dicts | Type safety, IDE support |

---

## Summary

**Key Takeaways:**
1. Python 3.12+ required
2. Use `ruff` for formatting and linting
3. **Cyclomatic complexity <= 10** per function (use `radon`/`xenon`)
4. Type hints required for public functions
5. Google-style docstrings for public APIs
6. Specific exception types, never bare `except Exception:`
7. Use `logging`, never `print()`
8. Follow CLI script template for consistency
9. Check Common Errors table for quick troubleshooting
10. Avoid named Anti-Patterns
