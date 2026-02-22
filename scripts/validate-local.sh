#!/bin/bash
# Local plugin validation - run before pushing
# Usage: ./scripts/validate-local.sh
#
# Validates:
# 1. Manifest schema conformance (versioned schemas)
# 2. No symlinks (breaks GitHub install)
# 3. SKILL.md frontmatter for all skills
# 4. Hook preflight checks
# 5. Lightweight Go race tests for changed packages only
# 6. Actually loads with claude --plugin-dir
#
# Install as pre-push hook:
#   ln -sf ../../scripts/validate-local.sh .git/hooks/pre-push

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; errors=$((errors + 1)); }
warn() { echo -e "${YELLOW}!${NC} $1"; }

errors=0
cd "$REPO_ROOT"

echo ""
echo "🔍 Running pre-push plugin validation..."
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  AgentOps Plugin Validation"
echo "═══════════════════════════════════════════════════════"
echo ""

echo "── Manifests ──"
if "$REPO_ROOT/scripts/validate-manifests.sh" --repo-root "$REPO_ROOT" >/dev/null 2>&1; then
    pass "Manifests match versioned schemas"
else
    fail "Manifest schema validation failed"
fi
echo ""

# 3. Check for symlinks
echo "── Symlinks ──"
symlink_list=$(find . -type l ! -path "./.git/*" 2>/dev/null || true)
if [[ -n "$symlink_list" ]]; then
    symlinks=$(echo "$symlink_list" | wc -l | tr -d ' ')
    fail "Contains $symlinks symlinks (breaks standalone install):"
    echo "$symlink_list" | sed 's/^/    /'
else
    pass "No symlinks"
fi
echo ""

# 4. Check skills
echo "── Skills ──"
if [[ -d "skills" ]]; then
    skill_count=0
    skill_errors=0
    for skill_dir in skills/*/; do
        [[ ! -d "$skill_dir" ]] && continue
        skill_name=$(basename "$skill_dir")
        skill_file="$skill_dir/SKILL.md"

        if [[ ! -f "$skill_file" ]]; then
            fail "Skill $skill_name: missing SKILL.md"
            skill_errors=$((skill_errors + 1))
            continue
        fi

        # Check frontmatter
        if ! head -1 "$skill_file" | grep -q "^---$"; then
            fail "Skill $skill_name: no YAML frontmatter"
            skill_errors=$((skill_errors + 1))
            continue
        fi

        if ! grep -q "^name:" "$skill_file"; then
            fail "Skill $skill_name: missing 'name' in frontmatter"
            skill_errors=$((skill_errors + 1))
            continue
        fi

        skill_count=$((skill_count + 1))
    done

    if [[ $skill_errors -eq 0 ]] && [[ $skill_count -gt 0 ]]; then
        pass "$skill_count skills valid"
    fi
else
    warn "No skills/ directory found"
fi
echo ""

# 5. Check agents
echo "── Agents ──"
if [[ -d "agents" ]]; then
    agent_count=$(find agents -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')
    if [[ $agent_count -gt 0 ]]; then
        pass "$agent_count agents found"
    else
        warn "No agent files found"
    fi
else
    warn "No agents/ directory found"
fi
echo ""

# 6. Hook preflight checklist
echo "── Hook Preflight ──"
if [[ -x "scripts/validate-hook-preflight.sh" ]]; then
    if hook_preflight_output=$(./scripts/validate-hook-preflight.sh 2>&1); then
        pass "Hook preflight checks passed"
    else
        fail "Hook preflight checks failed"
        echo "$hook_preflight_output" | sed 's/^/    /'
    fi
else
    fail "Missing executable script: scripts/validate-hook-preflight.sh"
fi
echo ""

# 7. Test actual load with Claude CLI (if available)
echo "── Go Fast Race Check ──"
if [[ -x "scripts/validate-go-fast.sh" ]]; then
    if go_fast_output=$(./scripts/validate-go-fast.sh 2>&1); then
        pass "Changed-scope Go race checks passed"
        echo "$go_fast_output" | sed 's/^/    /'
    else
        fail "Changed-scope Go race checks failed"
        echo "$go_fast_output" | sed 's/^/    /'
    fi
else
    warn "Missing executable script: scripts/validate-go-fast.sh"
fi
echo ""

# 8. Test actual load with Claude CLI (if available)
echo "── Claude CLI ──"
if command -v claude &>/dev/null; then
    load_output=$(timeout 10 claude --plugin-dir . --help 2>&1) || true
    if echo "$load_output" | grep -qiE "invalid manifest|validation error|failed to load"; then
        fail "Claude CLI load failed"
        echo "$load_output" | grep -iE "invalid|failed|error" | head -3 | sed 's/^/    /'
    else
        pass "Claude CLI loads plugin"
    fi
else
    warn "Claude CLI not available for load test"
fi
echo ""

# Summary
echo "═══════════════════════════════════════════════════════"
if [[ $errors -gt 0 ]]; then
    echo -e "${RED}  VALIDATION FAILED: $errors errors${NC}"
    echo "═══════════════════════════════════════════════════════"
    exit 1
else
    echo -e "${GREEN}  ALL VALIDATIONS PASSED${NC}"
    echo "═══════════════════════════════════════════════════════"
    exit 0
fi
