#!/usr/bin/env bash
# skill-assertions.sh — Per-skill content assertions for OpenCode headless tests
#
# Each function is named assert_skill_<skillname> and receives:
#   $1 = logfile path
#   $2 = repo root path
#
# Returns 0 if all assertions pass, 1 if any fail.
# These are sourced by run-headless-tests.sh and called after each attempt.

# Tier 1 skills (read-only, should always pass)

assert_skill_status() {
    local logfile="$1"
    assert_contains_any "$logfile" "status output has workflow terms" \
        "Dashboard" "RPI" "Progress" "Status" "PROGRESS" "ratchet" "epic" "flywheel" "SUGGESTED NEXT ACTION"
}

assert_skill_knowledge() {
    local logfile="$1"
    assert_contains_any "$logfile" "knowledge output has results" \
        "learning" "pattern" "knowledge" "found" "result" ".agents/" ".md" ".agents/learnings"
}

assert_skill_complexity() {
    local logfile="$1"
    assert_contains_any "$logfile" "complexity output has metrics" \
        "complexity" "score" "rating" "cyclomatic" "lines" "function" "SKILL.md" "section" "nesting"
}

assert_skill_doc() {
    local logfile="$1"
    assert_contains_any "$logfile" "doc output has coverage info" \
        "documentation" "coverage" "missing" "found" "skills/" "SKILL.md" "coverage:"
}

assert_skill_handoff() {
    local logfile="$1"
    assert_contains_any "$logfile" "handoff output has session info" \
        "handoff" "session" "summary" "context" "continue" ".agents/handoff"
}

assert_skill_retro() {
    local logfile="$1"
    assert_contains_any "$logfile" "retro output has learnings" \
        "learning" "retro" "went well" "improved" "pattern" "lesson" "What"
}

# Tier 2 skills (degraded mode)

assert_skill_research() {
    local logfile="$1"
    local repo_root="$2"
    # Either produces research output or at least discusses findings
    assert_contains_any "$logfile" "research produced findings" \
        "finding" "research" "discovered" "explored" "investigation" ".agents/research"
}

assert_skill_plan() {
    local logfile="$1"
    assert_contains_any "$logfile" "plan output has structure" \
        "issue" "wave" "plan" "dependency" "acceptance" "task"
}

assert_skill_pre-mortem() {
    local logfile="$1"
    assert_contains_any "$logfile" "pre-mortem has verdict" \
        "PASS" "WARN" "FAIL" "verdict" "pre-mortem" "concern" "proceed"
}

assert_skill_implement() {
    local logfile="$1"
    assert_contains_any "$logfile" "implement checked for work" \
        "implement" "issue" "ready" "beads" "bd" "task" "work"
}

assert_skill_vibe() {
    local logfile="$1"
    assert_contains_any "$logfile" "vibe has verdict" \
        "PASS" "WARN" "FAIL" "verdict" "vibe" "complexity" "review" "quality"
}

assert_skill_bug-hunt() {
    local logfile="$1"
    assert_contains_any "$logfile" "bug-hunt investigated" \
        "bug" "test" "failure" "investigate" "found" "issue" "error"
}

assert_skill_learn() {
    local logfile="$1"
    assert_contains_any "$logfile" "learn captured knowledge" \
        "learn" "saved" "knowledge" "insight" "captured" "remember"
}

assert_skill_trace() {
    local logfile="$1"
    assert_contains_any "$logfile" "trace found history" \
        "trace" "decision" "history" "commit" "session" "council" "architecture"
}

# Tier 3 skills (expected failure — these assertions are lenient)

assert_skill_council() {
    local logfile="$1"
    assert_contains_any "$logfile" "council attempted multi-model" \
        "council" "judge" "verdict" "consensus" "spawn" "agent" "validate"
}

assert_skill_crank() {
    local logfile="$1"
    assert_contains_any "$logfile" "crank checked epic" \
        "crank" "epic" "wave" "issue" "implement" "autonomous"
}

assert_skill_swarm() {
    local logfile="$1"
    assert_contains_any "$logfile" "swarm described workers" \
        "swarm" "worker" "spawn" "parallel" "task" "agent"
}

assert_skill_rpi() {
    local logfile="$1"
    assert_contains_any "$logfile" "rpi described lifecycle" \
        "rpi" "research" "plan" "implement" "vibe" "post-mortem" "lifecycle"
}

assert_skill_codex-team() {
    local logfile="$1"
    assert_contains_any "$logfile" "codex-team described agents" \
        "codex" "agent" "spawn" "team" "worker" "parallel"
}
