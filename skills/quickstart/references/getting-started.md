# Getting Started with AgentOps

## What is AgentOps?

AgentOps is a knowledge flywheel for AI coding agents. It adds persistent memory, multi-model validation, and structured workflows to Claude Code. Instead of each session starting from scratch, AgentOps captures learnings, tracks issues, and builds on previous work.

## What Happens When You Run /quickstart?

The quickstart walks you through a mini **RPI cycle** (Research-Plan-Implement) on your actual codebase:

1. **Pre-flight** — Checks your environment (git, CLI tools, directories)
2. **Detection** — Identifies your project language, framework, and existing setup
3. **Mini Research** — Explores your *current* work (dirty working tree if present; otherwise last commit) to show what `/research` does
4. **Mini Plan** — Suggests one concrete improvement to show what `/plan` does
5. **Mini Vibe** — Quick validation of those same “recent files” to show what `/vibe` does
6. **Orientation** — Shows available skills organized by workflow phase
7. **Next Steps** — Personalized recommendation based on your project state

## What Each Phase Does

### Research (`/research`)

Deep codebase exploration. Reads code, understands patterns, maps architecture. Use this before planning significant work. Output goes to `.agents/research/`.

### Plan (`/plan`)

Decomposes a goal into trackable issues with dependencies and execution waves. Produces a structured plan that `/crank` or `/implement` can execute. Output goes to `.agents/plans/`.

### Vibe (`/vibe`)

Code validation combining complexity analysis (cyclomatic metrics) with multi-model council review. Answers: "Is this code ready to ship?" Output goes to `.agents/council/`.

## Expected Output at Each Step

| Step | What You See |
|------|-------------|
| Pre-flight | Environment status (git, ao, directories) |
| Detection | Language/framework detection (often includes the path that triggered detection), existing AgentOps state |
| Mini Research | Summary of active code area (dirty-tree-first), patterns, quality observation |
| Mini Plan | One specific improvement suggestion with rationale |
| Mini Vibe | Quick PASS/WARN/FAIL assessment of the same recent files |
| Orientation | Skill reference table organized by workflow phase |
| Next Steps | One personalized recommendation for what to try next |

## What to Do After Quickstart

Based on what quickstart found, pick your next action:

- **Want to understand the codebase?** Run `/research` for a deep dive
- **Have a goal to accomplish?** Run `/plan "your goal"` to decompose it
- **Ready to validate code?** Run `/vibe recent` for a full review
- **Have a multi-step epic?** Run `/crank` for hands-free execution
- **Want cross-model validation?** Run `/council --mixed` for Claude + Codex review
- **Want to cut a release?** Run `/release` for changelog, version bumps, and tagging
- **Want a full lifecycle?** Run `/rpi` for Research → Plan → Implement → Validate in one command
- **Want persistent knowledge?** Install the `ao` CLI: `brew tap boshu2/agentops https://github.com/boshu2/homebrew-agentops && brew install agentops && ao init --hooks`

## The RPI Lifecycle

```
Research → Plan → Implement → Validate
    ^                            |
    └──── Knowledge Flywheel ────┘
```

Each cycle captures learnings that improve the next cycle. Over time, the knowledge base grows and agents make better decisions.
