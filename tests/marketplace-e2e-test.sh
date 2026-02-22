#!/bin/bash
# Comprehensive E2E test for AgentOps marketplace
# Tests: plugin structure, skill validation, cross-references, commands
# Usage: ./tests/marketplace-e2e-test.sh [--verbose]

# shellcheck disable=SC2015  # pass/fail functions are simple echos that won't fail
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERBOSE="${1:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

errors=0
warnings=0
tests_passed=0

log() { echo -e "${BLUE}[TEST]${NC} $1"; }
pass() { echo -e "${GREEN}  ✓${NC} $1"; tests_passed=$((tests_passed + 1)); }
fail() { echo -e "${RED}  ✗${NC} $1"; errors=$((errors + 1)); }
warn() { echo -e "${YELLOW}  !${NC} $1"; warnings=$((warnings + 1)); }
section() { echo -e "\n${CYAN}━━━ $1 ━━━${NC}"; }

cd "$REPO_ROOT"

# =============================================================================
section "1. JSON Validation"
# =============================================================================

log "Validating all JSON files..."

json_files=(
    ".claude-plugin/marketplace.json"
    ".claude-plugin/plugin.json"
)

for jf in "${json_files[@]}"; do
    if [[ ! -f "$jf" ]]; then
        fail "$jf - file not found"
        continue
    fi
    if python3 -m json.tool "$jf" > /dev/null 2>&1; then
        pass "$jf"
    else
        fail "$jf - invalid JSON"
    fi
done

# =============================================================================
section "2. Marketplace Schema Validation"
# =============================================================================

log "Validating marketplace structure..."

# Check marketplace has required fields (note: description is in metadata)
marketplace_valid=$(python3 << 'PYEOF'
import json
import sys

with open('.claude-plugin/marketplace.json') as f:
    mp = json.load(f)

errors = []

# Required top-level fields
for field in ['name', 'plugins']:
    if field not in mp:
        errors.append(f"Missing required field: {field}")

# Description can be in metadata or top-level
if 'description' not in mp and ('metadata' not in mp or 'description' not in mp.get('metadata', {})):
    errors.append("Missing description (either top-level or in metadata)")

# Each plugin must have name, description, source
for i, plugin in enumerate(mp.get('plugins', [])):
    for field in ['name', 'description', 'source']:
        if field not in plugin:
            errors.append(f"Plugin {i}: missing {field}")

if errors:
    for e in errors:
        print(f"ERROR: {e}")
    sys.exit(1)
else:
    print(f"OK: {len(mp.get('plugins', []))} plugins defined")
    sys.exit(0)
PYEOF
) && pass "Marketplace schema valid - $marketplace_valid" || fail "Marketplace schema invalid"

# =============================================================================
section "3. Plugin References"
# =============================================================================

log "Validating plugin sources exist..."

while IFS='|' read -r name source; do
    [[ -z "$name" ]] && continue

    if [[ "$source" == "." ]]; then
        pj=".claude-plugin/plugin.json"
    else
        pj="${source}/.claude-plugin/plugin.json"
    fi

    if [[ -f "$pj" ]]; then
        pass "$name -> $source"
    else
        fail "$name: source $source has no plugin.json"
    fi
done < <(python3 -c "
import json
with open('.claude-plugin/marketplace.json') as f:
    mp = json.load(f)
for p in mp.get('plugins', []):
    src = p['source'].lstrip('./')
    if not src: src = '.'
    print(f\"{p['name']}|{src}\")
")

# =============================================================================
section "4. Skill YAML Frontmatter Validation"
# =============================================================================

log "Validating skill frontmatter..."

skill_count=0
skill_errors=0

# This is a single-plugin repo with skills at root level
skills_dir="skills"
if [[ -d "$skills_dir" ]]; then
    for skill_dir in "$skills_dir"/*/; do
        [[ ! -d "$skill_dir" ]] && continue
        skill_file="${skill_dir}SKILL.md"
        skill_name=$(basename "$skill_dir")

        if [[ ! -f "$skill_file" ]]; then
            fail "$skill_name: SKILL.md missing"
            skill_errors=$((skill_errors + 1))
            continue
        fi

        skill_count=$((skill_count + 1))

        # Validate frontmatter - check for ---\n...\n--- pattern and required fields
        if ! head -1 "$skill_file" | grep -q "^---$"; then
            fail "$skill_name: No YAML frontmatter"
            skill_errors=$((skill_errors + 1))
            continue
        fi

        # Extract frontmatter and check required fields
        frontmatter=$(awk '/^---$/{if(++c==2)exit}c==1' "$skill_file")
        missing=""
        for field in name description; do
            if ! echo "$frontmatter" | grep -q "^${field}:"; then
                missing="$missing $field"
            fi
        done

        if [[ -n "$missing" ]]; then
            warn "$skill_name: Missing:$missing"
        else
            [[ "$VERBOSE" == "--verbose" ]] && pass "$skill_name"
        fi
    done
else
    warn "No skills directory found at $skills_dir"
fi

if [[ $skill_errors -eq 0 && $skill_count -gt 0 ]]; then
    pass "All $skill_count skills have valid frontmatter"
elif [[ $skill_count -eq 0 ]]; then
    warn "No skills found to validate"
else
    fail "$skill_errors skills with invalid frontmatter"
fi

# =============================================================================
section "5. Skill Cross-Reference Validation"
# =============================================================================

log "Validating skill dependencies..."

# Build skill inventory and check references
crossref_result=$(python3 << 'PYEOF'
import os
import re
from pathlib import Path

skills_dir = Path('skills')
all_skills = set()
skill_deps = {}

if not skills_dir.exists():
    print("OK: No skills directory")
    exit(0)

# Collect all skills
for skill_dir in skills_dir.iterdir():
    if not skill_dir.is_dir():
        continue
    skill_file = skill_dir / 'SKILL.md'
    if skill_file.exists():
        skill_name = skill_dir.name
        all_skills.add(skill_name)

        # Extract dependencies (under metadata: per Anthropic skills spec)
        with open(skill_file) as f:
            content = f.read()
        match = re.match(r'^---\s*\n(.*?)\n---', content, re.DOTALL)
        if match:
            yaml_content = match.group(1)
            in_deps = False
            deps = []
            for line in yaml_content.split('\n'):
                stripped = line.strip()
                if stripped.startswith('dependencies:'):
                    in_deps = True
                    continue
                if in_deps:
                    if stripped.startswith('- '):
                        dep = stripped[2:].strip().strip('"').strip("'")
                        # Strip inline comments
                        if '#' in dep:
                            dep = dep[:dep.index('#')].strip()
                        if dep:
                            deps.append(dep)
                    elif stripped and not stripped.startswith('-'):
                        in_deps = False
            if deps:
                skill_deps[skill_name] = deps

# Check for broken references
errors = []
for skill, deps in skill_deps.items():
    for dep in deps:
        if dep not in all_skills:
            errors.append(f"{skill} -> {dep} (not found)")

if errors:
    for e in errors:
        print(f"ERROR: {e}")
    exit(1)
else:
    print(f"OK: {len(skill_deps)} skills with {sum(len(d) for d in skill_deps.values())} total dependencies")
    exit(0)
PYEOF
) && pass "Cross-references valid - $crossref_result" || fail "Invalid cross-references: $crossref_result"

# =============================================================================
section "6. Commands Validation"
# =============================================================================

log "Validating commands..."

cmd_count=0
[[ -d "commands" ]] && cmd_count=$(find commands -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')

if [[ $cmd_count -gt 0 ]]; then
    pass "Found $cmd_count commands in commands/"
else
    warn "No commands found in commands/"
fi

# Check INDEX.md exists
if [[ -f "commands/INDEX.md" ]]; then
    pass "commands/INDEX.md exists"
else
    warn "commands/INDEX.md missing"
fi

# =============================================================================
section "7. Content Quality Checks"
# =============================================================================

log "Running content quality checks..."

# Check for placeholder patterns (actual placeholders, not docs about them)
placeholder_count=$( (grep -rn "\[your-email\]\|\[your-name\]" --include="*.md" . 2>/dev/null || true) | wc -l | tr -d '[:space:]')
placeholder_count=${placeholder_count:-0}
if [[ $placeholder_count -gt 0 ]]; then
    fail "$placeholder_count files with placeholder patterns"
else
    pass "No placeholder patterns"
fi

# Check for broken internal links (simple check)
broken_links=$( (grep -rhoE '\[.*\]\(\./[^)]+\)' --include="*.md" . 2>/dev/null || true) | \
    sed 's/.*(\.\///' | sed 's/).*//' | \
    while read -r link; do
        [[ ! -e "$link" ]] && echo "$link"
    done | wc -l | tr -d '[:space:]')
broken_links=${broken_links:-0}

if [[ "$broken_links" -gt 0 ]]; then
    warn "$broken_links potentially broken internal links"
else
    pass "No obvious broken internal links"
fi

# Check skill content is substantive (> 500 chars after frontmatter)
thin_skills=0
for skill_file in skills/*/SKILL.md; do
    [[ ! -f "$skill_file" ]] && continue
    content_after_frontmatter=$(sed '1,/^---$/d; 1,/^---$/d' "$skill_file" | wc -c)
    if [[ $content_after_frontmatter -lt 500 ]]; then
        [[ "$VERBOSE" == "--verbose" ]] && warn "$skill_file: only $content_after_frontmatter chars of content"
        thin_skills=$((thin_skills + 1))
    fi
done

if [[ $thin_skills -gt 0 ]]; then
    warn "$thin_skills skills with thin content (< 500 chars)"
else
    pass "All skills have substantive content"
fi

# =============================================================================
section "8. Security Checks"
# =============================================================================

log "Running security checks..."

# Check for potential secrets (simplified gitleaks-style)
# Exclude: examples, placeholders, variables ($VAR), environment var patterns, Go template vars
secret_patterns='(password|api[_-]?key|secret|token)\s*[:=]\s*["\x27][^\s"$]+["\x27]'
secret_count=$( (grep -riE "$secret_patterns" --include="*.md" --include="*.json" . 2>/dev/null || true) | \
    (grep -v 'example\|placeholder\|your-\|<\|>\|\$[A-Z_]\|from-literal\|standards\|{{\s*\.Env\.' || true) | wc -l | tr -d '[:space:]')
secret_count=${secret_count:-0}

if [[ $secret_count -gt 0 ]]; then
    fail "$secret_count potential secrets found"
    [[ "$VERBOSE" == "--verbose" ]] && grep -riE "$secret_patterns" --include="*.md" --include="*.json" . 2>/dev/null | grep -v "example\|placeholder\|{{\s*\.Env\." | head -3
else
    pass "No potential secrets detected"
fi

# Check for dangerous shell patterns
dangerous_patterns='rm\s+-rf\s+/|curl.*\|\s*bash|wget.*\|\s*sh'
dangerous_count=$( (grep -riE "$dangerous_patterns" --include="*.md" --include="*.sh" . 2>/dev/null || true) | wc -l | tr -d '[:space:]')
dangerous_count=${dangerous_count:-0}

if [[ "$dangerous_count" -gt 0 ]]; then
    warn "$dangerous_count potentially dangerous shell patterns"
else
    pass "No dangerous shell patterns"
fi

# =============================================================================
# Summary
# =============================================================================

echo ""
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}                    E2E TEST SUMMARY                        ${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  Tests passed:  ${GREEN}$tests_passed${NC}"
echo -e "  Warnings:      ${YELLOW}$warnings${NC}"
echo -e "  Errors:        ${RED}$errors${NC}"
echo ""

if [[ $errors -gt 0 ]]; then
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}  FAILED - $errors errors found                              ${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 1
elif [[ $warnings -gt 0 ]]; then
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}  PASSED WITH WARNINGS                                      ${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 0
else
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  ALL E2E TESTS PASSED                                      ${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 0
fi
