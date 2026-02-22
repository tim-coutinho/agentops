# Quickstart Troubleshooting

## Hooks Aren't Running

**Symptom:** AgentOps hooks don't fire on session start or tool use.

**Checks:**
```bash
# Verify hooks are installed
ao hooks test

# Check hooks.json exists and is valid (hooks live in hooks/, not .claude-plugin/)
cat hooks/hooks.json | jq . 2>/dev/null

# Check settings.json for ao hooks
cat ~/.claude/settings.json | jq '.hooks' 2>/dev/null

# Verify plugin is loaded
claude --plugin ./ --help
```

**Fixes:**
- Reinstall hooks: `ao hooks install` (minimal) or `ao init --hooks --full` (all 8 events)
- Check that `hooks/hooks.json` is not malformed JSON
- Restart Claude Code after hook changes

## Skills Not Showing Up

**Symptom:** `/quickstart`, `/vibe`, or other skills don't trigger.

**Checks:**
```bash
# Verify plugin installation
npx skills@latest list

# Check SKILL.md exists with valid frontmatter
head -5 skills/quickstart/SKILL.md
```

**Fixes:**
- Reinstall: `npx skills@latest add boshu2/agentops --all -g`
- Update: `npx skills@latest update`
- If update doesn't pick up changes, pull directly:
  ```bash
  cd ~/.claude/plugins/marketplaces/agentops-marketplace/
  git pull
  ```

## CLI Not Found (ao, bd, gt)

**Symptom:** `command not found: ao` (or `bd`, `gt`).

**Installation:**
```bash
# ao (AgentOps CLI) — requires Homebrew tap first
brew tap boshu2/agentops https://github.com/boshu2/homebrew-agentops
brew install agentops
ao init              # Create .agents/ dirs + .gitignore
ao init --hooks      # Also install flywheel hooks (SessionStart + Stop)

# bd (Beads issue tracking)
brew install boshu2/agentops/beads
bd init --prefix <your-prefix>

# gt (Gas Town workspace manager)
brew install boshu2/agentops/gastown
```

**Verify PATH:**
```bash
which ao bd gt 2>/dev/null
echo $PATH | tr ':' '\n' | grep -E 'homebrew|bin'
```

## Permission Denied

**Symptom:** Cannot create `.agents/` directory or write files.

**Checks:**
```bash
# Check directory permissions
ls -la . | head -3

# Check if .agents exists and is writable
ls -la .agents/ 2>/dev/null

# Try creating
mkdir -p .agents && echo "OK" || echo "PERMISSION DENIED"
```

**Fixes:**
- Check directory ownership: `ls -la .`
- Fix permissions: `chmod u+w .` (if you own the directory)
- If in a shared/mounted directory, work in a local clone instead

## Non-Git Directory

**Symptom:** Quickstart warns about missing git repo.

**Options:**

1. **Initialize git** (recommended):
   ```bash
   git init
   git add .
   git commit -m "Initial commit"
   ```
   Then re-run `/quickstart` for the full experience.

2. **Continue in manual mode:**
   Quickstart will skip git-dependent features (recent changes, vibe on diffs) and use file-browsing equivalents instead. You can still use `/research`, `/plan`, and other non-git skills.

## Language Detection Failed

**Symptom:** Quickstart says "I couldn't auto-detect a language."

**Why:** Quickstart couldn't find any recognized project files in its **shallow scan** (it avoids walking entire repos). In monorepos, the primary module might be deeper than the scan depth.

**Fix:**
- Run quickstart from your repo root (recommended), or `cd` into the primary module directory (e.g., `cli/`, `src/`).
- If detection still fails, tell quickstart your primary language when prompted and continue manually.

## Session Knowledge Not Persisting

**Symptom:** Each session starts fresh, no prior knowledge loaded.

**Checks:**
```bash
# Is ao CLI available?
which ao

# Is .agents/ directory present?
ls .agents/

# Check flywheel health
ao flywheel status
```

**Fixes:**
- Install ao: `brew tap boshu2/agentops https://github.com/boshu2/homebrew-agentops && brew install agentops && ao init`
- Install hooks: `ao init --hooks` (minimal) or `ao init --hooks --full` (all lifecycle events)
- Verify inject runs on session start: check `hooks/session-start.sh`
