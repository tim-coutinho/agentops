# Installing AgentOps for OpenCode

## Quick Install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/boshu2/agentops/main/scripts/install-opencode.sh | bash
```

This clones the repo, installs the plugin, and symlinks skills. Restart OpenCode after.

## Manual Install

### 1. Clone AgentOps

```bash
git clone https://github.com/boshu2/agentops.git ~/.config/opencode/agentops
```

### 2. Install Plugin Dependency

```bash
cd ~/.config/opencode/agentops/.opencode && bun install && cd -
```

### 3. Register the Plugin

```bash
mkdir -p ~/.config/opencode/plugins
ln -sf ~/.config/opencode/agentops/.opencode/plugins/agentops.js ~/.config/opencode/plugins/agentops.js
```

### 4. Symlink Skills

```bash
mkdir -p ~/.config/opencode/skills
ln -sfn ~/.config/opencode/agentops/skills ~/.config/opencode/skills/agentops
```

### 5. Restart OpenCode

Verify by asking: "do you have agentops?"

## What Gets Installed

| Component | Location | Purpose |
|-----------|----------|---------|
| Plugin (agentops.js) | `~/.config/opencode/plugins/` | 7 hooks: system prompt injection, tool enrichment, audit logging, CLI paths, compaction resilience |
| Skills (full set) | `~/.config/opencode/skills/agentops/` | Full AgentOps skill set (research, plan, council, vibe, crank, etc.) |

## Plugin Hooks

The plugin enriches Devstral and other models with prescriptive AgentOps guidance:

| Hook | What It Does |
|------|-------------|
| `system.transform` | Injects AgentOps bootstrap + per-skill tool mapping |
| `tool.execute.before` | Guards against task tool crash (missing subagent_type) |
| `tool.definition` | Enriches task + skill tool descriptions with usage guidance |
| `command.execute.before` | Redirects slashcommand skill calls to skill tool |
| `tool.execute.after` | Audit logs all tool calls to `.agents/audit/` |
| `shell.env` | Injects bd/ao/gt CLI paths into shell environment |
| `session.compacting` | Preserves AgentOps context through session compaction |

## Updating

```bash
cd ~/.config/opencode/agentops && git pull
```

Or re-run the install script — it pulls if the repo already exists.

## Usage

### Loading a Skill

Use OpenCode's native `skill` tool:

```
use skill tool to load agentops/research
```

**Important:** OpenCode's skill tool is **read-only**. It loads content into your context. Read the instructions and follow them inline.

### Key Differences from Claude Code

| Feature | Claude Code | OpenCode |
|---------|------------|----------|
| Skill invocation | `Skill(skill="X")` executes | `skill` tool loads (read-only) |
| Parallel agents | `TeamCreate` + `SendMessage` | `task(subagent_type="general")` |
| Slash commands | `/council` works directly | Use `skill` tool instead |

## Troubleshooting

### Plugin not loading

```bash
ls -l ~/.config/opencode/plugins/agentops.js  # Check symlink
ls ~/.config/opencode/agentops/.opencode/plugins/agentops.js  # Check source
```

### Skills not found

```bash
ls -l ~/.config/opencode/skills/agentops  # Check symlink
ls ~/.config/opencode/agentops/skills/    # Check source
```

### Known Limitation

OpenCode has a bug where `Locale.titlecase(undefined)` crashes when the model calls `task` without `subagent_type` (sst/opencode#13933). The plugin mitigates this but can't prevent the UI rendering crash entirely. Skills that spawn subagents may crash ~30% of the time until the upstream fix lands.

## Getting Help

- Report issues: https://github.com/boshu2/agentops/issues
