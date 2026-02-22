// Package safety provides runtime guards and input validation that protect
// agent-driven workflows from accidental and adversarial misuse.
//
// AgentOps orchestrates autonomous AI agents that execute shell commands, modify
// files, and interact with git. The safety package centralizes the threat model
// and defensive patterns that keep these operations bounded and reversible.
//
// # Threat Model
//
// The safety package addresses the following threat categories:
//
// T1 - Command Injection: Agent-supplied metadata (task validation rules,
// test commands, lint commands) flows into shell execution. Without guards,
// crafted metadata could inject arbitrary commands via shell metacharacters
// such as semicolons, pipes, backticks, or command substitution. Mitigations
// include binary allowlists (only go, pytest, npm, make), shell metacharacter
// blocking, bare-name-only binary requirements (no path separators), and
// array-based execution that avoids shell interpretation entirely.
//
// T2 - Path Traversal: Candidate IDs, artifact paths, and file references
// provided by agents or task metadata could escape the repository root via
// ".." sequences, absolute paths, or symlink chains. Mitigations include
// regex-validated candidate IDs (alphanumeric, hyphens, underscores only),
// symlink-resolving path canonicalization (pwd -P), repo-root confinement
// checks, and rejection of tilde (~) prefixed paths.
//
// T3 - Destructive Git Operations: Autonomous agents may attempt force push,
// hard reset, force clean, checkout-dot, restore-dot, or force branch delete,
// any of which can destroy uncommitted work or rewrite shared history.
// Mitigations include a PreToolUse hook that pattern-matches git commands
// against a block-list, suggests safe alternatives (--force-with-lease, stash,
// soft reset), and provides a narrow allow-list for known-safe patterns.
//
// T4 - Worker Privilege Escalation: In parallel swarm execution, worker agents
// must write files but never commit or push. A worker that commits creates
// merge conflicts across parallel workers and can corrupt the shared branch.
// Mitigations include identity-based gating (CLAUDE_AGENT_NAME prefix check),
// fallback role-file detection (.agents/swarm-role), and blocking git commit,
// git push, and git add -A/--all for worker-prefixed identities.
//
// T5 - Unvalidated Code Push: Pushing code that has not passed quality
// validation (vibe check, post-mortem) risks shipping broken or unreviewed
// changes. Mitigations include a push gate hook that checks RPI ratchet state
// (phased-state.json verdicts or chain.jsonl entries) and blocks git push and
// git tag until the validation phase is recorded as complete.
//
// T6 - Runaway Autonomous Loops: Autonomous RPI (Research-Plan-Implement)
// loops can consume unbounded resources if not externally stoppable. Mitigations
// include file-based kill switches (.agents/rpi/KILL) checked at cycle
// boundaries, environment variable kill switches (AGENTOPS_HOOKS_DISABLED and
// per-hook variants), context budget tracking with configurable handoff
// thresholds, and deterministic MemRL policy escalation that forces loop
// termination after configurable retry limits.
//
// T7 - Policy Bypass and Retry Abuse: Without deterministic escalation rules,
// failing phases could retry indefinitely, masking systemic problems. The MemRL
// policy contract defines a closed rule set mapping (mode, failure class,
// attempt bucket) to actions (retry or escalate). Wildcard fallback rules
// ensure no input combination escapes policy evaluation. Rollback triggers
// define operator-actionable thresholds for escalation rate, unknown failure
// class ratio, and missing metadata detection.
//
// T8 - Malicious Repository Sourcing: Hook scripts that source helper libraries
// from the repository root could be tricked into executing attacker-controlled
// code checked into the repo. Mitigations include sourcing all helper libraries
// from the plugin install directory (SCRIPT_DIR-relative) rather than the
// repository root, ensuring hooks execute trusted code regardless of repository
// contents.
//
// # Design Principles
//
// Fail open on missing infrastructure: hooks exit 0 when jq, ao, or helper
// libraries are absent, preventing safety mechanisms from blocking legitimate
// work when the toolchain is incomplete.
//
// Hot-path exit: guards check the cheapest condition first (e.g., does the
// command contain "git"?) before performing expensive JSON parsing or file I/O,
// keeping overhead under 5ms for the 99th percentile of non-matching commands.
//
// Kill switches at every layer: global (AGENTOPS_HOOKS_DISABLED), per-hook
// (AGENTOPS_*_DISABLED), and per-run (.agents/rpi/KILL) to allow operators to
// disable enforcement without code changes.
//
// Structured failure output: all blocking hooks emit machine-readable failure
// records via write_failure() for downstream diagnostics and audit trails.
package safety
