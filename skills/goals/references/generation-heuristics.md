# Goal Generation Heuristics

## Goal Quality Criteria

A good goal:

1. **Mechanically verifiable** — `check` is a shell command that exits 0 (pass) or non-zero (fail). No human judgment required.
2. **Descriptive** — `description` says what it measures, not how. "Go CLI compiles without errors" not "run go build".
3. **Weighted by impact** — 5 = build/test integrity, 3-4 = feature fitness, 1-2 = hygiene.
4. **Pillar-mapped** — Maps to one of: knowledge-compounding, validated-acceleration, goal-driven-automation, zero-friction-workflow. Infrastructure goals omit `pillar`.
5. **Not trivially true** — Check can actually fail in a realistic scenario. `test -f README.md` is trivially true.
6. **Not duplicative** — No two goals test the same thing. Check existing IDs before proposing.

## Scan Sources

| Source | What to look for | Goal type |
|--------|-----------------|-----------|
| `PRODUCT.md` | Value props, design principles, theoretical pillars without goals | Pillar |
| `README.md` | Claims, badges, features without verification | Pillar |
| `skills/*/SKILL.md` | Skills with no goal referencing them | Pillar or Infra |
| `tests/`, `hooks/` | Scripts not covered by goals | Infrastructure |
| `docs/` | Doc files referenced but not covered | Infrastructure |
| Existing goals | Checks referencing deleted paths | Prune candidates |

## Theoretical Pillar Coverage

Generate mode should check that all 4 theoretical pillars have goals:

### 1. Systems Theory (Meadows)

Targets leverage points #3-#6 (information flows, rules, self-organization, goals). Goals should verify that the system operates at these leverage points rather than lower ones (parameters, buffers).

### 2. DevOps (Three Ways)

- **Flow** maps to `zero-friction-workflow` and `goal-driven-automation`
- **Feedback** maps to `validated-acceleration`
- **Continual Learning** maps to `knowledge-compounding`

Goals should cover all three ways.

### 3. Brownian Ratchet

The pattern: chaos + filter + ratchet = directional progress from undirected energy. Goals should verify:
- Chaos source exists (agent sessions generate varied outputs)
- Filter exists (council validates, vibe checks)
- Ratchet exists (knowledge flywheel captures and persists gains)

### 4. Knowledge Flywheel

Escape velocity condition: `signal_rate x retrieval_rate > decay_rate` (informally: you learn faster than you forget). Goals should verify:
- Signal generation (extract, forge, retro produce learnings)
- Retrieval (inject loads learnings into sessions)
- Decay resistance (learnings are persisted, not just in-memory)

## Weight Guidelines

| Weight | Category | Examples |
|--------|----------|----------|
| 5 | **Critical** | Build passes, tests pass, manifests valid |
| 4 | **Important** | Full test suite, hook safety, mission alignment |
| 3 | **Feature fitness** | Skill behaviors, positioning, documentation |
| 2 | **Hygiene** | Lint, coverage floors, doc counts |
| 1 | **Nice to have** | Stubs, aspirational checks |

## ID Conventions

- Use kebab-case: `go-cli-builds`, `readme-compounding-hero`
- Prefix with domain: `readme-`, `go-`, `skill-`, `hook-`
- Keep under 40 characters
- Must be unique across all goals
