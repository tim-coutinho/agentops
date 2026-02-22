# Release Notes Generation

Release notes are **not the changelog**. The changelog is comprehensive and developer-facing. Release notes are what users see on the GitHub Release page — they should be approachable and highlight what matters.

**Audience:** People scrolling their GitHub feed. They haven't read your commit log, don't know your internal architecture names, and will spend 10 seconds deciding if this release matters to them. Write for THEM, not for contributors.

**Quality bar — the feed test:**

- Would a first-time visitor understand every bullet?
- Are internal code names explained or omitted? (e.g., "Gate 4" means nothing — say "retry loop after validation failure")
- Does the Highlights paragraph answer "why should I care?" in plain English?
- Is every bullet about user-visible impact, not implementation detail?

**Structure:**

```markdown
## Highlights

<2-4 sentence plain-English summary of what's new and why it matters.
Written for users, not contributors. No jargon. No internal architecture names.
Answer: "What can I do now that I couldn't before?">

## What's New

<Top 3-5 most important changes, each as a short bullet.
Pick from Added/Changed/Fixed — prioritize user-visible impact.
Explain internal terms or don't use them.>

## All Changes

<CONDENSED version of the changelog — NOT a raw copy-paste from CHANGELOG.md.
Strip: issue IDs (ag-xxx), file paths, internal tool names, architecture jargon.
Keep: what changed, described in plain English.
Each bullet: one sentence, no bold lead-in, no parenthetical issue refs.
End with a link to the full CHANGELOG.md for those who want raw detail.>

[Full changelog](../../CHANGELOG.md)
```

**The All Changes section is NOT the CHANGELOG.** The CHANGELOG is for contributors who want file paths, issue IDs, and implementation detail. The release page condenses that into plain-English bullets a user can scan in 15 seconds. When in doubt, leave it out — the link is there for the curious.

**Condensing rules:**
- Remove issue IDs: `(ag-ab6)` → gone
- Remove file paths: `skills/council/scripts/validate-council.sh` → "council validation script"
- Remove internal terms: "progressive-disclosure reference files" → "reference content loaded on demand"
- Collapse related items: "5 broken links" + "7 doc inaccuracies" → "12 broken links and doc inaccuracies fixed"
- Bold sparingly: only in What's New section, not in All Changes

**Example:**

```markdown
## Highlights

Three security vulnerabilities patched in hook scripts. The validation lifecycle
now retries automatically when validation fails instead of stopping — no manual
intervention needed. The five largest skills now load significantly faster by
loading reference docs only when needed.

## What's New

- **Self-healing validation** — Failed code reviews now retry automatically with failure context, instead of stopping and waiting for you
- **Faster skill loading** — Five core skills restructured to load reference content on demand instead of all at once
- **3 security fixes** — Command injection, regex injection, and JSON injection vulnerabilities patched in hook scripts

## All Changes

### Added

- Self-healing retry loop for the validation lifecycle
- Security and documentation sections for Go, Python, Rust, JSON, YAML standards
- 26 hook integration tests (injection resistance, kill switches, allowlist enforcement)
- Monorepo-friendly quickstart detection

### Fixed

- Command injection in task validation hook (now allowlist-based)
- Regex injection in task validation hook (now literal matching)
- JSON injection in prompt nudge hook (now safe escaping)
- 12 broken links and doc inaccuracies fixed across the project

### Removed

- Deprecated `/judge` skill (use `/council` instead)

[Full changelog](https://github.com/example/project/blob/main/CHANGELOG.md#version)
```

**Always write release notes to a file immediately after generating:**

```bash
mkdir -p .agents/releases
```

Write to `.agents/releases/YYYY-MM-DD-v<version>-notes.md` — this is the **public-facing** file that CI uses for the GitHub Release page. It contains ONLY the Highlights + What's New + All Changes structure above, ending with a link to the full CHANGELOG.md. No internal metadata, no pre-flight results, no next steps, no issue IDs, no file paths.

**IMPORTANT:** This file must be committed in the release commit (before tagging). CI checks out the tagged commit and reads this file from the repo. `.agents/releases/` must NOT be gitignored. If it is, CI falls back to CHANGELOG-only mode (no curated highlights).

**Show the release notes to the user** as part of Step 8 review, alongside the changelog and version bumps.

## How Release Notes Reach GitHub

The CI release pipeline (`scripts/extract-release-notes.sh`) generates the GitHub Release body:

1. It looks for curated notes at `.agents/releases/YYYY-MM-DD-v<version>-notes.md`
2. If found: curated highlights up top, full CHANGELOG section in a collapsible `<details>` block
3. If not found: raw CHANGELOG section only (no commit dump — missing CHANGELOG entry is an error)
4. GoReleaser consumes the generated `release-notes.md` via `--release-notes`

**Always write curated notes before tagging.** The curated file is what makes the difference between a wall of jargon and a readable release page. The CHANGELOG is for contributors; the curated notes are for everyone else.
