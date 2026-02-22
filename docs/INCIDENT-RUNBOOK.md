# AgentOps Incident Runbook — Consumer Recovery

> **Audience:** Anyone responding to a broken AgentOps installation.
> **Assumption:** You are stressed and need copy-pasteable commands. Each section is self-contained.

---

## Table of Contents

1. [Emergency Kill Switches](#1-emergency-kill-switches)
2. [Scenario A: Broken Skills After Update](#2-scenario-a-broken-skills-after-update)
3. [Scenario B: Evolve Pushed Bad Code to Main](#3-scenario-b-evolve-pushed-bad-code-to-main)
4. [Scenario C: Hook Scripts Fail at Session Start](#4-scenario-c-hook-scripts-fail-at-session-start)
5. [Rollback Options](#5-rollback-options)
6. [Root Cause Analysis](#6-root-cause-analysis)
7. [Prevention Checklist](#7-prevention-checklist)

---

## 1. Emergency Kill Switches

**Do these FIRST if sessions are broken. Restore functionality, then investigate.**

```bash
# Disable ALL AgentOps hooks globally (instant, affects all sessions)
export AGENTOPS_HOOKS_DISABLED=1

# Stop evolve from running (persistent across sessions)
mkdir -p ~/.config/evolve
echo "incident $(date -Iseconds)" > ~/.config/evolve/KILL
```

To make the hook disable persistent, add to your shell profile:
```bash
echo 'export AGENTOPS_HOOKS_DISABLED=1' >> ~/.zshrc
```

### Per-Hook Kill Switches

If you know which hook is broken, disable just that one:

| Hook | Kill Switch Env Var |
|------|-------------------|
| session-start.sh | `AGENTOPS_SESSION_START_DISABLED=1` |
| pre-mortem-gate.sh | `AGENTOPS_SKIP_PRE_MORTEM_GATE=1` |
| task-validation-gate.sh | `AGENTOPS_TASK_VALIDATION_DISABLED=1` |
| pending-cleaner.sh | `AGENTOPS_PENDING_CLEANER_DISABLED=1` |
| precompact-snapshot.sh | `AGENTOPS_PRECOMPACT_DISABLED=1` |
| All hooks (global) | `AGENTOPS_HOOKS_DISABLED=1` |

Every hook script checks `AGENTOPS_HOOKS_DISABLED=1` at the top and exits immediately if set.

---

## 2. Scenario A: Broken Skills After Update

**Symptom:** Consumer ran `npx skills add boshu2/agentops --all -g` and now Claude sessions are broken — hooks error, skills don't load, or sessions hang at start.

### Triage (< 5 min)

```bash
# 1. Kill hooks immediately
export AGENTOPS_HOOKS_DISABLED=1

# 2. Check what version was installed
cat ~/.claude/skills/agentops/plugin.json 2>/dev/null | jq -r '.version'
# Or check the marketplace cache
cat ~/.claude/plugins/marketplaces/agentops-marketplace/plugin.json 2>/dev/null | jq -r '.version'

# 3. Check if skills are symlinks (known failure mode)
ls -la ~/.claude/skills/ | head -20

# 4. Test a hook manually
bash ~/.claude/skills/agentops/hooks/session-start.sh
```

### Fix: Reinstall from a known-good version

```bash
# Remove broken installation
rm -rf ~/.claude/skills/agentops

# Remove any symlinks (known failure: npx skills can't write through symlinks)
find ~/.claude/skills -maxdepth 1 -type l -delete

# Reinstall from latest (if latest is fixed)
npx skills@latest add boshu2/agentops --all -g

# OR pin to a specific tag
npx skills@latest add boshu2/agentops@v2.5.0 --all -g
```

### Fix: Nuke and reinstall (if pinning doesn't work)

```bash
# Nuclear option: remove everything and reinstall
rm -rf ~/.claude/skills/agentops
rm -rf ~/.claude/plugins/marketplaces/agentops-marketplace

# Reinstall
npx skills@latest add boshu2/agentops --all -g
```

### Re-enable hooks after fix

```bash
unset AGENTOPS_HOOKS_DISABLED
# Remove from shell profile if you added it there
```

---

## 3. Scenario B: Evolve Pushed Bad Code to Main

**Symptom:** `/evolve` ran autonomously, committed code that breaks builds, tests, or other skills. The regression gate failed to catch it, or evolve committed before the gate ran.

### Triage (< 5 min)

```bash
# 1. Stop evolve immediately
mkdir -p ~/.config/evolve
echo "incident: bad code on main $(date -Iseconds)" > ~/.config/evolve/KILL

# Also set local stop in the repo
echo "emergency stop" > .agents/evolve/STOP

# 2. Check what evolve did
cat .agents/evolve/cycle-history.jsonl 2>/dev/null   # cycle outcomes
cat .agents/evolve/session-summary.md 2>/dev/null     # session wrap-up
ls -lt .agents/evolve/fitness-*.json 2>/dev/null      # fitness snapshots

# 3. Find evolve's commits
git log --oneline -20   # look for evolve/rpi commit messages
```

### Revert evolve's changes

```bash
# Find the last good commit (before evolve ran)
# Look at fitness snapshots for session_start_sha
jq -r '.cycle_start_sha' .agents/evolve/fitness-0.json 2>/dev/null

# Or find it manually
git log --oneline -30 | less

# Revert everything after the known-good SHA
GOOD_SHA="<paste sha here>"
git revert --no-commit ${GOOD_SHA}..HEAD
git commit -m "revert: evolve incident — rolling back to ${GOOD_SHA}"

# Verify
cd cli && go build ./cmd/ao && go test ./...
./tests/run-all.sh
```

### If evolve is mid-run (still executing)

```bash
# Kill switch stops it at the next cycle boundary
mkdir -p ~/.config/evolve
echo "emergency stop" > ~/.config/evolve/KILL

# If it's in a tmux session, also kill the process
# Find the session
tmux list-sessions | grep -i evolve
# Kill it
tmux kill-session -t <session-name>
```

### Re-enable evolve after fix

```bash
rm ~/.config/evolve/KILL
rm .agents/evolve/STOP 2>/dev/null
```

---

## 4. Scenario C: Hook Scripts Fail at Session Start

**Symptom:** Claude session hangs, errors on start, or prints garbage JSON. The session-start hook or one of the other 11 hook scripts has a bug.

### Triage (< 5 min)

```bash
# 1. Disable hooks to get a working session
export AGENTOPS_HOOKS_DISABLED=1

# 2. Check hook error log
cat .agents/ao/hook-errors.log 2>/dev/null | tail -20

# 3. Test each hook manually to find the broken one
for hook in ~/.claude/skills/agentops/hooks/*.sh; do
  echo "--- Testing: $(basename $hook) ---"
  timeout 5 bash "$hook" 2>&1 | tail -3
  echo "Exit: $?"
done

# 4. Check for syntax errors
for hook in ~/.claude/skills/agentops/hooks/*.sh; do
  bash -n "$hook" 2>&1 && echo "OK: $(basename $hook)" || echo "BROKEN: $(basename $hook)"
done
```

### Common hook failures

**Missing binary (ao, jq, shellcheck):**
```bash
# Check if ao CLI is installed
which ao    # should be /opt/homebrew/bin/ao
which jq    # required by several hooks

# If ao is missing, hooks degrade gracefully (|| true pattern)
# But session-start.sh may still produce broken JSON
```

**Infinite loop or timeout:**
```bash
# hooks.json defines timeouts per hook (2-120s)
# If a hook exceeds its timeout, Claude kills it
# Check hooks.json for timeout values
jq '.hooks | to_entries[] | .value[].hooks[] | {command: .command[:60], timeout: .timeout}' \
  ~/.claude/skills/agentops/hooks/hooks.json
```

**Bad JSON output from session-start.sh:**
```bash
# session-start.sh must output valid JSON with hookSpecificOutput
# Test it and validate output
bash ~/.claude/skills/agentops/hooks/session-start.sh 2>/dev/null | jq .
# If jq fails, the output is malformed
```

### Fix: Disable just the broken hook

Once you identify the broken hook, disable only that one instead of all hooks. Use the per-hook kill switches from Section 1.

If the broken hook doesn't have its own kill switch, you can patch it:
```bash
# Add a kill switch to the top of the hook (after the shebang)
# This is a temporary fix in the installed copy
HOOK_FILE=~/.claude/skills/agentops/hooks/<broken-hook>.sh
# Back it up first
cp "$HOOK_FILE" "${HOOK_FILE}.bak"
# Make it exit immediately
sed -i '' '2i\
exit 0  # TEMPORARY: disabled during incident\
' "$HOOK_FILE"
```

---

## 5. Rollback Options

### Option A: Reinstall from a specific git tag

```bash
# List available tags
gh release list -R boshu2/agentops --limit 10

# Install a specific version
npx skills@latest add boshu2/agentops@v2.5.0 --all -g
```

### Option B: Pin to a specific commit

```bash
# Clone and install from a known-good commit
cd /tmp
git clone https://github.com/boshu2/agentops.git agentops-recovery
cd agentops-recovery
git checkout <known-good-sha>

# Copy skills manually
rm -rf ~/.claude/skills/agentops
cp -r . ~/.claude/skills/agentops
```

### Option C: Sync from marketplace cache

```bash
# The marketplace cache may have a working version
cd ~/.claude/plugins/marketplaces/agentops-marketplace
git log --oneline -10   # find a good state
git checkout <good-sha>

# Then reinstall from cache
npx skills@latest add boshu2/agentops --all -g
```

### Option D: Nuclear reinstall

```bash
# Remove everything AgentOps-related
rm -rf ~/.claude/skills/agentops
rm -rf ~/.claude/plugins/marketplaces/agentops-marketplace
find ~/.claude/skills -maxdepth 1 -type l -delete   # remove symlinks

# Clear any cached state
rm -rf ~/.config/evolve/KILL 2>/dev/null

# Fresh install
npx skills@latest add boshu2/agentops --all -g

# Verify
cat ~/.claude/skills/agentops/.claude-plugin/plugin.json | jq -r '.version'
bash ~/.claude/skills/agentops/hooks/session-start.sh 2>/dev/null | jq .
```

---

## 6. Root Cause Analysis

After restoring service, investigate what went wrong.

### Was it an evolve regression?

```bash
# Check evolve history
cat .agents/evolve/cycle-history.jsonl | jq -s '.'

# Check fitness snapshots for regressions
for f in .agents/evolve/fitness-*-post.json; do
  echo "--- $f ---"
  jq '[.goals[] | select(.result == "fail") | .id]' "$f" 2>/dev/null
done

# Check GOALS.yaml for broken check commands
cat GOALS.yaml
```

### Was it a bad commit?

```bash
# Use git bisect to find the breaking commit
git bisect start
git bisect bad HEAD
git bisect good <last-known-good-sha>

# For each step, run the relevant test
./tests/run-all.sh && git bisect good || git bisect bad

# When done
git bisect reset
```

### Was it a hook script bug?

```bash
# Shellcheck all hooks
shellcheck -x -P SCRIPTDIR hooks/*.sh

# Check for recent hook changes
git log --oneline -20 -- hooks/

# Run the hook preflight validation
./scripts/validate-hook-preflight.sh

# Run full test suite
./tests/run-all.sh
```

### Was it a dependency issue?

```bash
# Check if ao CLI is working
ao status
ao flywheel status

# Check Go CLI builds
cd cli && go build ./cmd/ao && go test ./...

# Check for missing system tools
for cmd in jq shellcheck git ao; do
  which "$cmd" >/dev/null 2>&1 && echo "OK: $cmd" || echo "MISSING: $cmd"
done
```

---

## 7. Prevention Checklist

### Before releasing a new version

- [ ] `./tests/run-all.sh` passes (all tiers)
- [ ] `./tests/smoke-test.sh` passes
- [ ] `cd cli && go build ./cmd/ao && go test -race ./...` clean
- [ ] `shellcheck -x -P SCRIPTDIR hooks/*.sh` clean
- [ ] `./scripts/validate-hook-preflight.sh` passes
- [ ] All hook scripts test manually: `bash hooks/<name>.sh`
- [ ] Plugin and marketplace versions match: `jq -r '.version' .claude-plugin/plugin.json` equals `jq -r '.metadata.version' .claude-plugin/marketplace.json`
- [ ] `session-start.sh` output is valid JSON: `bash hooks/session-start.sh 2>/dev/null | jq .`

### Before running evolve

- [ ] GOALS.yaml check commands all work: run each `check:` value manually
- [ ] Evolve kill switch is clear: `test ! -f ~/.config/evolve/KILL && echo "clear"`
- [ ] Git working tree is clean: `git status --porcelain` is empty
- [ ] Know the current HEAD: `git rev-parse HEAD` (save this for revert)
- [ ] Set a reasonable cycle cap: `--max-cycles=3` for first run

### After evolve completes

- [ ] Review `cycle-history.jsonl` for any regressions
- [ ] Run `./tests/run-all.sh` manually (don't trust evolve's self-assessment)
- [ ] Check `git log --oneline -20` for reasonable commit messages
- [ ] Run `git diff <pre-evolve-sha>..HEAD --stat` to see total scope of changes

### Hook script authoring rules

- Every hook MUST check `AGENTOPS_HOOKS_DISABLED=1` at the top
- Every hook MUST fail open (`exit 0` on error, never `set -e`)
- Every hook MUST respect its timeout in `hooks.json`
- Guard all external commands: `command -v <tool> >/dev/null 2>&1 && ...`
- JSON output must be valid — test with `jq .` before committing
- Log failures to `.agents/ao/hook-errors.log`, never to stderr

---

## Quick Reference Card

```
STOP EVERYTHING:     export AGENTOPS_HOOKS_DISABLED=1
STOP EVOLVE:         echo "stop" > ~/.config/evolve/KILL
CHECK HOOK ERRORS:   cat .agents/ao/hook-errors.log | tail -20
TEST HOOKS:          for h in hooks/*.sh; do bash -n "$h"; done
REINSTALL:           npx skills@latest add boshu2/agentops --all -g
NUCLEAR REINSTALL:   rm -rf ~/.claude/skills/agentops && npx skills@latest add boshu2/agentops --all -g
REVERT EVOLVE:       git revert --no-commit <good-sha>..HEAD && git commit -m "revert: evolve incident"
VERSION CHECK:       jq -r '.version' ~/.claude/skills/agentops/.claude-plugin/plugin.json
```
