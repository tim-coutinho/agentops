# Installing AgentOps for Codex

Quick setup to enable AgentOps skills in Codex.

## Installation

1. **Clone AgentOps repository**:
   ```bash
   mkdir -p ~/.codex/agentops
   cd ~/.codex/agentops
   git clone https://github.com/boshu2/agentops.git .
   ```

2. **Create personal skills directory**:
   ```bash
   mkdir -p ~/.codex/skills
   ```

3. **Update ~/.codex/AGENTS.md** to include this AgentOps section:
   ```markdown
   ## AgentOps System

   <EXTREMELY_IMPORTANT>
   You have AgentOps superpowers. AgentOps teaches you skills for AI-assisted development workflows. RIGHT NOW run: `~/.codex/agentops/.codex/agentops-codex bootstrap` and follow the instructions it returns.
   </EXTREMELY_IMPORTANT>
   ```

## Verification

Test the installation:
```bash
~/.codex/agentops/.codex/agentops-codex bootstrap
```

You should see skill listings and bootstrap instructions. The system is now ready for use.

## Codex Session-End Equivalent

Claude plugin hooks (`hooks/hooks.json`) are not consumed by Codex. Use this command at the end of a Codex session:

```bash
~/.codex/agentops/.codex/agentops-codex session-end
```

It runs the same flywheel close-loop used by the Claude Stop hook:
`ao flywheel close-loop --quiet` (with failure logging to `.agents/ao/hook-errors.log`).
