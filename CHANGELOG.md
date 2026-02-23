# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.16.0] - 2026-02-23

### Added
- Evolve idle hardening — disk-derived stagnation detection, 60-minute circuit breaker, rolling fitness files, no idle commits
- Evolve `--quality` mode — findings-first priority cascade that prioritizes post-mortem findings over goals
- Evolve cycle-history.jsonl canonical schema standardization and artifact-only commit gating
- `heal-skill` checks 7-10 with `--strict` CI gate for automated skill maintenance
- 6-phase E2E validation test suite for RPI lifecycle (gate retries, complexity scaling, phase summaries, promise tags)
- Fixture-based CLI regression and parity tests
- `ao goals migrate` command for v1→v2 GOALS.yaml migration with deprecation warning (#48)
- Goal failure taxonomy script and tests

### Changed
- CLI taxonomy, shared resolver, skill versioning, and doctor dogfooding improvements (6 architecture concerns)
- GoReleaser action bumped from v6 to v7
- Evolve build detection generalized from hardcoded Go gate to multi-language detection

### Fixed
- `ao pool list --wide` flag and `pool show` prefix matching (#47)
- Consistent artifact counts across `doctor`, `badge`, and `metrics` (#46)
- Double multiplication in `vibe-check` score display (#45)
- Skills installed as symlinks now detected and checked in both directories (#44)
- Learnings resolved by frontmatter ID; `.md` file count in maturity scan (#43)
- JSON output truncated at clean object boundaries (#42)
- Misleading hook event count removed from display (#41)
- Post-mortem schema `model` field and resolver `DiscoverAll` migration
- 15+ missing skills added to catalog tables in `using-agentops`
- Handoff example filename format corrected to `YYYYMMDDTHHMMSSZ` spec
- Quickstart step numbering corrected (7 before 8)
- OpenAI docs skill: added Claude Code MCP alternative to Codex-only fallback
- Dead link to `conflict-resolution-algorithm.md` removed from post-mortem
- `ao forge search` → `ao search` in provenance and knowledge skills
- OSS docs: root-level doc path checks, removed golden-init reference
- Reverse-engineer-rpi fixture paths and contract refs corrected
- Crank: removed missing script refs, moved orphans to references
- Codex-team: removed vaporware Team Runner Backend section
- Security skill: bundled `security-gate.sh`, fixed `security-suite` path
- Evolve oscillation detection and TodoWrite→Task tools migration
- Wired `check-contract-compatibility.sh` into GOALS.yaml
- Synced embedded skills and regenerated CLI docs
