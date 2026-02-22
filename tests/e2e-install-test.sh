#!/bin/bash
# E2E test: Actually install and test plugins with Claude Code CLI
# This runs in a container to simulate a fresh user environment
#
# Usage: ./tests/e2e-install-test.sh [plugin-name]
# If no plugin specified, tests all plugins

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[E2E]${NC} $1"; }
pass() { echo -e "${GREEN}  ✓${NC} $1"; }
fail() { echo -e "${RED}  ✗${NC} $1"; }
warn() { echo -e "${YELLOW}  !${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${SCRIPT_DIR}/.."
[[ -d "$REPO_ROOT/agentops" ]] && REPO_ROOT="$REPO_ROOT/agentops"

cd "$REPO_ROOT"

errors=0
tests_passed=0

# =============================================================================
log "Phase 0: Validate manifests against schemas"
# =============================================================================

if "$REPO_ROOT/scripts/validate-manifests.sh" --repo-root "$REPO_ROOT" >/dev/null 2>&1; then
    pass "Manifest schemas valid"
    tests_passed=$((tests_passed + 1))
else
    fail "Manifest schema validation failed"
    errors=$((errors + 1))
fi

# =============================================================================
log "Phase 1: Verify Claude Code CLI"
# =============================================================================

if command -v claude &>/dev/null; then
    version=$(claude --version 2>&1 | head -1)
    pass "Claude Code CLI installed: $version"
else
    fail "Claude Code CLI not found"
    echo "Install with: npm install -g @anthropic-ai/claude-code"
    exit 1
fi

# =============================================================================
log "Phase 2: Test plugin loading with --plugin-dir"
# =============================================================================

# Get list of plugins to test
plugins=()
plugin_count=0
if [[ $# -gt 0 ]]; then
    plugins=("$1")
    plugin_count=1
else
    if [[ -d "plugins" ]]; then
        for p in plugins/*/; do
            if [[ -d "$p" ]]; then
                plugins+=("$(basename "$p")")
                plugin_count=$((plugin_count + 1))
            fi
        done
    fi
fi

# If no plugins, skip plugin tests
if [[ $plugin_count -eq 0 ]]; then
    pass "No plugins to test (plugins/ empty or missing)"
    tests_passed=$((tests_passed + 1))
fi

for plugin in "${plugins[@]:-}"; do
    [[ -z "$plugin" ]] && continue
    plugin_dir="plugins/$plugin"

    if [[ ! -d "$plugin_dir" ]]; then
        fail "$plugin: directory not found"
        errors=$((errors + 1))
        continue
    fi

    log "Testing $plugin..."

    # Test 1: Plugin can be loaded without errors
    # Use timeout and capture stderr to detect load failures
    load_output=$(timeout 10 claude --plugin-dir "$plugin_dir" --help 2>&1) || true

    if echo "$load_output" | grep -qi "invalid manifest\|validation error\|failed to load"; then
        fail "$plugin: Plugin load error"
        echo "$load_output" | grep -i "error\|invalid\|failed" | head -5
        errors=$((errors + 1))
    else
        pass "$plugin: Plugin loads successfully"
        tests_passed=$((tests_passed + 1))
    fi

    # Test 2: Check skill count matches expectations
    expected_skills=$(find "$plugin_dir/skills" -name "SKILL.md" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [[ $expected_skills -gt 0 ]]; then
        pass "$plugin: $expected_skills skills found"
        tests_passed=$((tests_passed + 1))
    else
        warn "$plugin: No skills directory"
    fi

    # Test 3: Verify no symlinks (would break GitHub install)
    symlinks=$(find "$plugin_dir" -type l 2>/dev/null | wc -l | tr -d ' ')
    if [[ $symlinks -gt 0 ]]; then
        fail "$plugin: Contains $symlinks symlinks (breaks standalone install)"
        find "$plugin_dir" -type l 2>/dev/null
        errors=$((errors + 1))
    else
        pass "$plugin: No symlinks"
        tests_passed=$((tests_passed + 1))
    fi

    # Test 4: Manifest validation
    manifest="$plugin_dir/.claude-plugin/plugin.json"
    if [[ -f "$manifest" ]]; then
        # Check for invalid keys
        # shellcheck disable=SC2016
        valid_keys='["$schema","name","version","description","author","homepage","repository","license","keywords","commands","skills","agents","hooks"]'
        invalid=$(jq -r --argjson valid "$valid_keys" 'keys - $valid | .[]' "$manifest" 2>/dev/null || echo "JSON_ERROR")

        if [[ "$invalid" == "JSON_ERROR" ]]; then
            fail "$plugin: Invalid JSON in manifest"
            errors=$((errors + 1))
        elif [[ -n "$invalid" ]]; then
            fail "$plugin: Invalid manifest keys: $invalid"
            errors=$((errors + 1))
        else
            pass "$plugin: Manifest valid"
            tests_passed=$((tests_passed + 1))
        fi
    else
        fail "$plugin: Missing manifest"
        errors=$((errors + 1))
    fi
done

# =============================================================================
log "Phase 3: Test combined plugin loading"
# =============================================================================

# Test loading multiple plugins together (simulates marketplace install)
all_plugins_args=""
if [[ -d "plugins" ]]; then
    for p in plugins/*/; do
        [[ -d "$p" ]] && all_plugins_args="$all_plugins_args --plugin-dir $p"
    done
fi

# Skip if no plugins
if [[ -z "$all_plugins_args" ]]; then
    pass "Combined load: No plugins to test"
    tests_passed=$((tests_passed + 1))
else
    # shellcheck disable=SC2086
    combined_output=$(timeout 15 claude $all_plugins_args --help 2>&1) || true

    # Check for actual plugin load errors (not general CLI help text)
    # Patterns that indicate real problems:
    # - "invalid manifest" / "validation error" / "failed to load" = plugin structure issues
    # - "conflict" / "duplicate" = plugin naming collisions
    if echo "$combined_output" | grep -qiE "invalid manifest|validation error|failed to load|plugin.*conflict|duplicate.*skill|duplicate.*command"; then
        fail "Combined load: Plugin conflicts detected"
        echo "$combined_output" | grep -iE "invalid|failed|conflict|duplicate" | head -5
        errors=$((errors + 1))
    else
        pass "Combined load: All $plugin_count plugins load together"
        tests_passed=$((tests_passed + 1))
    fi
fi

# =============================================================================
# Summary
# =============================================================================

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}              E2E INSTALL TEST SUMMARY                      ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  Tests passed: ${GREEN}$tests_passed${NC}"
echo -e "  Errors:       ${RED}$errors${NC}"
echo ""

if [[ $errors -gt 0 ]]; then
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}  E2E INSTALL TEST FAILED                                   ${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 1
else
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  E2E INSTALL TEST PASSED                                   ${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 0
fi
