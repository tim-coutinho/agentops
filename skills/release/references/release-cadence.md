# Release Cadence Policy

AgentOps follows a **weekly release train** to avoid notification spam for watchers and stargazers. GitHub sends an email for every published release — there's no way to filter by release type.

## Schedule

| Type | When | Version | Example |
|------|------|---------|---------|
| **Regular release** | Weekly (Fridays) | Minor bump (`vX.Y.0`) | Features + fixes accumulated all week |
| **Security hotfix** | Immediately | Patch bump (`vX.Y.Z`) | Only for security vulnerabilities |
| **Skip week** | When nothing meaningful shipped | — | No release, no tag |

## Rules

- **One published release per week, max.** Accumulate changes in `[Unreleased]` in CHANGELOG.md.
- **Security hotfixes are the only exception.** Ship same day as patch version.
- **No 1-commit releases** for non-security fixes. Batch them.
- **No multiple releases in one day** unless one is a security hotfix.
- **Draft releases don't notify anyone.** Use drafts freely for CI testing.

## Pre-flight Cadence Check

During `/release` pre-flight, check the date of the last tag:

```bash
git log -1 --format=%ci $(git tag --sort=-version:refname -l 'v*' | head -1)
```

If another release was published within the last 7 days (and this is not a security hotfix), warn:

```
⚠ Release cadence: v2.9.2 was released 3 days ago.
  Weekly release train policy: batch non-security changes into one release per week.
  Continue only if this is a security hotfix. Otherwise, accumulate in [Unreleased].
```

Use AskUserQuestion to confirm:
- "This is a security hotfix — proceed"
- "Wait and batch into next week's release"

## Why

At scale (50+ watchers), 1.5 releases/day = 75 notification emails/day to every watcher. The GitHub community has a 4-year-old unresolved thread about this. Projects like Next.js are cited as worst offenders. Weekly batching prevents this before it becomes a problem.

## Curated Release Notes

Every published release should have curated notes at `.agents/releases/YYYY-MM-DD-v<version>-notes.md`. The CI pipeline (`scripts/extract-release-notes.sh`) uses these as the GitHub Release page highlights, with the full CHANGELOG in a collapsible `<details>` block.

See `references/release-notes.md` for the notes format and quality bar.
