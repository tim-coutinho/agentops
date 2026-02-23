---
name: beads
description: 'This skill should be used when the user asks to "track issues", "create beads issue", "show blockers", "what''s ready to work on", "beads routing", "prefix routing", "cross-rig beads", "BEADS_DIR", "two-level beads", "town vs rig beads", "slingable beads", or needs guidance on git-based issue tracking with the bd CLI.'
skill_api_version: 1
metadata:
  tier: library
  dependencies: []
  internal: true
---

# Beads - Persistent Task Memory for AI Agents

Graph-based issue tracker that survives conversation compaction.

## Overview

**bd (beads)** replaces markdown task lists with a dependency-aware graph stored in git.

**Key Distinction**:
- **bd**: Multi-session work, dependencies, survives compaction, git-backed
- **Task tools (TaskCreate/TaskUpdate/TaskList)**: Single-session tasks, status tracking, conversation-scoped

**Decision Rule**: If resuming in 2 weeks would be hard without bd, use bd.

## Prerequisites

- **bd CLI**: Version 0.34.0+ installed and in PATH
- **Git Repository**: Current directory must be a git repo
- **Initialization**: `bd init` run once (humans do this, not agents)

## Examples

### Skill Loading from /vibe

**User says:** `/vibe`

**What happens:**
1. Agent loads beads skill automatically via dependency
2. Agent calls `bd show <id>` to read issue metadata
3. Agent links validation findings to the issue being checked
4. Output references issue ID in validation report

**Result:** Validation report includes issue context, no manual bd lookups needed.

### Skill Loading from /implement

**User says:** `/implement ag-xyz-123`

**What happens:**
1. Agent loads beads skill to understand issue structure
2. Agent calls `bd show ag-xyz-123` to read issue body
3. Agent checks dependencies with bd output
4. Agent closes issue with `bd close ag-xyz-123` after completion

**Result:** Issue lifecycle managed automatically during implementation.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| bd command not found | bd CLI not installed or not in PATH | Install bd: `brew install bd` or check PATH |
| "not a git repository" error | bd requires git repo, current dir not initialized | Run `git init` or navigate to git repo root |
| "beads not initialized" error | .beads/ directory missing | Human runs `bd init --prefix <prefix>` once |
| Issue ID format errors | Wrong prefix or malformed ID | Check rigs.json for correct prefix, follow `<prefix>-<tag>-<num>` format |

## Reference Documents

- [references/ANTI_PATTERNS.md](references/ANTI_PATTERNS.md)
- [references/BOUNDARIES.md](references/BOUNDARIES.md)
- [references/CLI_REFERENCE.md](references/CLI_REFERENCE.md)
- [references/DEPENDENCIES.md](references/DEPENDENCIES.md)
- [references/INTEGRATION_PATTERNS.md](references/INTEGRATION_PATTERNS.md)
- [references/ISSUE_CREATION.md](references/ISSUE_CREATION.md)
- [references/MOLECULES.md](references/MOLECULES.md)
- [references/PATTERNS.md](references/PATTERNS.md)
- [references/RESUMABILITY.md](references/RESUMABILITY.md)
- [references/ROUTING.md](references/ROUTING.md)
- [references/STATIC_DATA.md](references/STATIC_DATA.md)
- [references/TROUBLESHOOTING.md](references/TROUBLESHOOTING.md)
- [references/WORKFLOWS.md](references/WORKFLOWS.md)
