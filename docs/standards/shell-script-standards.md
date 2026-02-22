# Shell Script Standards

<!-- Canonical source: gitops/docs/standards/shell-script-standards.md -->
<!-- Last synced: 2026-01-19 -->

> **Purpose:** Standardized shell scripting conventions for this repository.

## Scope

This document covers: bash compatibility, shellcheck integration, error handling, input validation, logging, and security patterns.

**Related:**
- [Python Style Guide](./python-style-guide.md) - Python coding conventions
- [Tag Vocabulary](./tag-vocabulary.md) - Documentation standards

---

## Quick Reference

| Standard | Value | Validation |
|----------|-------|------------|
| **Shell** | Bash 4.0+ | `bash --version` |
| **Shebang** | `#!/usr/bin/env bash` | First line of script |
| **Flags** | `set -eEuo pipefail` | Line 2 or 3 |
| **Linter** | shellcheck | `.shellcheckrc` at repo root |

---

## Required Patterns

### Shebang and Flags

Every shell script MUST start with:

```bash
#!/usr/bin/env bash
set -eEuo pipefail
```

**Why:**
- `#!/usr/bin/env bash` - Finds bash in PATH (works on macOS and Linux)
- `set -e` - Exit on error
- `set -E` - ERR trap is inherited by shell functions
- `set -u` - Exit on undefined variable
- `set -o pipefail` - Fail if any command in a pipe fails

### Variable Quoting

```bash
# GOOD - Quoted variables, safe defaults
namespace="${NAMESPACE:-default}"
kubectl get pods -n "${namespace}"

# BAD - Unquoted variables (word splitting, globbing risks)
kubectl get pods -n $namespace
```

---

## Shellcheck Integration

All scripts must pass shellcheck validation:

```bash
shellcheck scripts/*.sh
```

### Repository Configuration

Create `.shellcheckrc` at repo root:

```ini
# .shellcheckrc
# Can't follow non-constant source
disable=SC1090
# Not following sourced files
disable=SC1091
# Consider invoking separately (pipefail handles this)
disable=SC2312
```

### Common Shellcheck Fixes

| Issue | Fix |
|-------|-----|
| SC2086 (word splitting) | Quote variables: `"$var"` |
| SC2164 (cd can fail) | `cd /path || exit 1` |
| SC2046 (word splitting in $()) | Quote: `"$(command)"` |
| SC2181 (checking $?) | Use `if command; then` directly |

### Disable Rules Sparingly

Only disable when truly necessary:

```bash
# shellcheck disable=SC2086
# Reason: Word splitting is intentional for flag array
$tool_cmd $flags_array "$input_file"
```

---

## Error Handling

### ERR Trap for Debug Context

Add an ERR trap to provide context on failure:

```bash
#!/usr/bin/env bash
set -eEuo pipefail

on_error() {
    local exit_code=$?
    echo "ERROR: Script failed on line $LINENO with exit code $exit_code" >&2
    exit "$exit_code"
}
trap on_error ERR
```

### Exit Code Documentation

Document exit codes in script headers:

```bash
#!/usr/bin/env bash
# Usage: ./scripts/my_script.sh <config>
#
# Exit Codes:
#   0 - Success
#   1 - Argument error
#   2 - Missing dependency
#   3 - Configuration error
#   4 - Validation failed
#   5 - User cancelled

set -eEuo pipefail
```

### Cleanup Pattern

```bash
#!/usr/bin/env bash
set -eEuo pipefail

# Create temp directory
TMPDIR=$(mktemp -d)

cleanup() {
    rm -rf "$TMPDIR"
}
trap cleanup EXIT

# Your code here - cleanup runs on exit or error
```

### Checking Command Success

```bash
# GOOD - Direct conditional
if kubectl get namespace "$ns" &>/dev/null; then
    echo "Namespace exists"
else
    echo "Creating namespace"
    kubectl create namespace "$ns"
fi

# BAD - Capturing exit code (unnecessary)
kubectl get namespace "$ns"
result=$?
if [[ $result -eq 0 ]]; then
    ...
fi
```

---

## Logging Functions

Use consistent logging functions across scripts:

```bash
# Logging functions
log()  { echo "[$(date '+%H:%M:%S')] $*"; }
warn() { echo "[$(date '+%H:%M:%S')] WARNING: $*" >&2; }
err()  { echo "[$(date '+%H:%M:%S')] ERROR: $*" >&2; }
die()  { err "$*"; exit 1; }

# Usage
log "Processing namespace: ${NAMESPACE}"
warn "Timeout exceeded, retrying..."
die "Required tool 'kubectl' not found"
```

### Colored Output (Optional)

```bash
# Color codes (use sparingly)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'  # No Color

log_success() { echo -e "${GREEN}[OK]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $*" >&2; }
log_error()   { echo -e "${RED}[ERR]${NC} $*" >&2; }
```

---

## Script Organization

### Template Structure

```bash
#!/usr/bin/env bash
# ===================================================================
# Script: <name>
# Purpose: <one-line description>
# Usage: ./<script> [args]
#
# Exit Codes:
#   0 - Success
#   1 - Argument error
#   2 - Missing dependency
# ===================================================================

set -eEuo pipefail

# -------------------------------------------------------------------
# Configuration
# -------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-default}"
DEFAULT_TIMEOUT=300

# -------------------------------------------------------------------
# Functions
# -------------------------------------------------------------------

log()  { echo "[$(date '+%H:%M:%S')] $*"; }
err()  { echo "[$(date '+%H:%M:%S')] ERROR: $*" >&2; }
die()  { err "$*"; exit 1; }

on_error() {
    local exit_code=$?
    err "Script failed on line $LINENO with exit code $exit_code"
    exit "$exit_code"
}
trap on_error ERR

cleanup() {
    rm -rf "$TMPDIR" 2>/dev/null || true
}

validate_args() {
    if [[ $# -lt 1 ]]; then
        echo "Usage: $0 <required-arg>" >&2
        exit 1
    fi
}

check_dependencies() {
    local missing=()
    for cmd in kubectl jq; do
        if ! command -v "$cmd" &>/dev/null; then
            missing+=("$cmd")
        fi
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
        die "Missing dependencies: ${missing[*]}"
    fi
}

# -------------------------------------------------------------------
# Main
# -------------------------------------------------------------------

main() {
    validate_args "$@"
    check_dependencies

    log "Starting with namespace: $NAMESPACE"

    # Main logic here
}

# Setup
TMPDIR=$(mktemp -d)
trap cleanup EXIT

# Run
main "$@"
```

---

## Security

### Secret Handling

**Never pass secrets as CLI arguments** - they're visible in `ps aux`:

```bash
# BAD - Secrets visible in process list
kubectl create secret generic my-secret --from-literal=token="$TOKEN"

# GOOD - Pass via stdin
echo "$TOKEN" | kubectl create secret generic my-secret --from-literal=token=-

# GOOD - Use file-based approach
echo "$SECRET" > "$TMPDIR/secret"
chmod 600 "$TMPDIR/secret"
kubectl create secret generic my-secret --from-file=token="$TMPDIR/secret"
```

### Input Validation

#### Kubernetes Resource Names (RFC 1123)

```bash
validate_namespace() {
    local ns="$1"
    if [[ ! "$ns" =~ ^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$ ]] && \
       [[ ! "$ns" =~ ^[a-z0-9]$ ]]; then
        die "Invalid namespace format: $ns (must be RFC 1123 label)"
    fi
}
```

#### Path Traversal Prevention

```bash
validate_path() {
    local path="$1"
    case "$path" in
        *..*)
            die "Path traversal detected: $path"
            ;;
    esac
}
```

### Sed Injection Prevention

User input in sed replacement strings can have special meaning:

```bash
# BAD - Injection possible with special characters
NAME="test&id"  # & has special meaning in sed
sed "s/{{NAME}}/$NAME/g" template.txt  # Inserts command result!

# GOOD - Escape special characters
escape_sed_replacement() {
    printf '%s' "$1" | sed -e 's/[&/\]/\\&/g'
}

escaped_name=$(escape_sed_replacement "$NAME")
sed "s/{{NAME}}/$escaped_name/g" template.txt
```

### JSON Construction

Use `jq` for safe JSON construction:

```bash
# BAD - String interpolation (injection risk)
json="{\"name\": \"$NAME\", \"value\": \"$VALUE\"}"

# GOOD - Use jq for proper escaping
json=$(jq -n --arg name "$NAME" --arg value "$VALUE" \
    '{name: $name, value: $value}')
```

---

## Common Patterns

### Polling with Timeout

```bash
wait_for_condition() {
    local timeout=${1:-300}
    local interval=${2:-10}
    local condition_cmd="${3}"

    local elapsed=0
    while ! eval "$condition_cmd" &>/dev/null; do
        if [[ $elapsed -ge $timeout ]]; then
            err "Timeout waiting for condition after ${timeout}s"
            return 1
        fi
        log "Waiting... (${elapsed}s/${timeout}s)"
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done
    return 0
}

# Usage
wait_for_condition 300 10 "kubectl get pod my-pod -o jsonpath='{.status.phase}' | grep -q Running"
```

### Parallel Execution with Background Jobs

```bash
run_parallel() {
    local pids=()
    local failures=()

    for item in "$@"; do
        process_item "$item" &
        pids+=($!)
    done

    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            failures+=("$pid")
        fi
    done

    if [[ ${#failures[@]} -gt 0 ]]; then
        err "Failed jobs: ${#failures[@]}"
        return 1
    fi
}
```

### Kubernetes Resource Checks

```bash
# Check if resource exists
resource_exists() {
    local kind="$1"
    local name="$2"
    local ns="${3:-}"

    local ns_flag=""
    [[ -n "$ns" ]] && ns_flag="-n $ns"

    # shellcheck disable=SC2086
    kubectl get "$kind" "$name" $ns_flag &>/dev/null
}

# Usage
if resource_exists deployment my-app my-namespace; then
    log "Deployment exists"
fi
```

---

## Testing

### Manual Testing

```bash
# Test with shellcheck
shellcheck ./script.sh

# Test with bash
bash ./script.sh --help

# Test with set -x for debugging
bash -x ./script.sh
```

### BATS Framework (Optional)

For complex scripts, use BATS for automated testing:

```bash
# test/test_script.bats
#!/usr/bin/env bats

@test "script requires argument" {
    run ./script.sh
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "script validates namespace format" {
    run ./script.sh --namespace "INVALID_NS"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid namespace" ]]
}
```

---

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| `unbound variable` | Using unset variable | Add default: `${VAR:-default}` |
| `command not found` | Missing dependency | Add to `check_dependencies()` |
| `syntax error near unexpected token` | Missing quote or semicolon | Check quoting, line endings |
| Script works locally, fails in CI | Different bash version | Check `bash --version`, use portable syntax |
| `permission denied` | Script not executable | `chmod +x script.sh` |
| `bad substitution` | Using bash-only syntax in sh | Use `#!/usr/bin/env bash` |
| Word splitting issues | Unquoted variable | Quote: `"${var}"` |
| `No such file or directory` | Wrong path, spaces in path | Quote paths, check existence |

---

## Anti-Patterns

<!-- vibe:ignore - table shows anti-patterns with example bad code -->
| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| Parsing ls Output | `for f in $(ls)` | Breaks on spaces, special chars | `for f in *` or `find` |
| Cat Abuse | `cat file \| grep` | Useless use of cat | `grep pattern file` |
| Backticks | `` `command` `` | Hard to nest, escape | `$(command)` |
| No Error Handling | Missing `set -e` | Silent failures | Add `set -eEuo pipefail` |
| Secrets in Arguments | `--password="secret"` | Visible in `ps aux` | Use stdin or temp file |
| Hardcoded Paths | `/home/user/file` | Breaks on other machines | Use variables, `$HOME` |
| No Shellcheck | Skipping linting | Bugs, security issues | Run `shellcheck` always |
| eval Abuse | `eval "$user_input"` | Command injection | Avoid eval, validate input |

---

## AI Agent Guidelines

When AI agents write shell scripts for this repo:

| Guideline | Rationale |
|-----------|-----------|
| ALWAYS add `set -eEuo pipefail` | Fail-fast on errors |
| ALWAYS run `shellcheck` before committing | Catches bugs and security issues |
| ALWAYS quote variables | Prevents word splitting, globbing |
| ALWAYS document exit codes in header | Users know what errors mean |
| NEVER use `eval` with user input | Command injection risk |
| NEVER pass secrets as CLI arguments | Visible in process list |
| PREFER `[[ ]]` over `[ ]` | Safer, more features |
| PREFER `$(command)` over backticks | Nestable, readable |
| PREFER functions over inline code | Testable, reusable |

---

## Summary

**Key Takeaways:**
1. Bash 4.0+ with `set -eEuo pipefail`
2. All scripts must pass shellcheck
3. Quote all variables: `"${var}"`
4. Use logging functions: `log`, `warn`, `err`, `die`
5. Add ERR trap for debug context
6. Never pass secrets as CLI arguments
7. Use `jq` for JSON construction
8. Validate all user input
9. Document exit codes in script headers
10. Check Common Errors table for troubleshooting
11. Avoid named Anti-Patterns
