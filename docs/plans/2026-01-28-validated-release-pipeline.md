# Design: Validated Release Pipeline

**Date:** 2026-01-28
**Status:** Approved
**Author:** Claude + Ben

---

## Problem

Today's release process has no validation gate. Tagging triggers immediate release, which led to:

1. **Broken binaries shipped** — `ao version` showed `dev` instead of real version
2. **Tag pollution** — v1.0.9 → v1.0.10 → v1.0.11 in 30 minutes
3. **No rollback** — Once published, had to push forward with more releases

## Solution

Split release workflow into Build → Validate → Publish stages. Validation gates the release.

```
git tag v1.0.12
    ↓
┌─────────────────────────────────────────────────────────────────┐
│                        release.yml                              │
├─────────────────────────────────────────────────────────────────┤
│  Stage 1: BUILD                                                 │
│  ├─ GoReleaser builds 4 binaries (darwin/linux × amd64/arm64)   │
│  ├─ Creates tarballs + checksums                                │
│  └─ Uploads as workflow artifacts (NOT released yet)            │
├─────────────────────────────────────────────────────────────────┤
│  Stage 2: VALIDATE                                              │
│  ├─ Download darwin-arm64 artifact                              │
│  ├─ Check: ao version == v1.0.12 (not "dev")                    │
│  ├─ Check: ao --help exits 0                                    │
│  ├─ Check: ao status exits 0                                    │
│  └─ Validate generated Homebrew formula syntax                  │
├─────────────────────────────────────────────────────────────────┤
│  Stage 3: PUBLISH (only if Stage 2 passes)                      │
│  ├─ Create GitHub Release                                       │
│  ├─ Upload tarballs to release                                  │
│  └─ Push formula to boshu2/homebrew-agentops                    │
└─────────────────────────────────────────────────────────────────┘
```

## Validation Checks

| Check | Command | Pass Condition |
|-------|---------|----------------|
| Version injection | `ao version` | Output contains tag version (e.g., `1.0.12`) |
| Binary executes | `ao --help` | Exit code 0 |
| Basic functionality | `ao status` | Exit code 0 (or graceful error) |
| Formula syntax | `brew audit --strict` | No errors |

## Failure Modes

| Failure | Behavior |
|---------|----------|
| Build fails | No artifacts uploaded, workflow stops |
| Validation fails | Artifacts exist but release NOT created |
| Publish fails | Release partial, manual cleanup needed |

**On validation failure:**
1. Tag exists but no release is published
2. Fix the issue in code
3. Delete the tag: `git tag -d v1.0.12 && git push origin :refs/tags/v1.0.12`
4. Re-tag and push

## Implementation

### Phase 1: Update release.yml

Replace current workflow with three-job structure:

```yaml
jobs:
  build:
    # GoReleaser with --skip=publish
    # Upload artifacts

  validate:
    needs: build
    # Download artifacts
    # Run validation checks

  publish:
    needs: validate
    # Download artifacts
    # Create release + upload
    # Push Homebrew formula
```

### Phase 2: Validation Script

Create `scripts/validate-release.sh`:

```bash
#!/bin/bash
set -e

BINARY="$1"
EXPECTED_VERSION="$2"

# Check version
VERSION_OUTPUT=$("$BINARY" version 2>&1)
if ! echo "$VERSION_OUTPUT" | grep -q "$EXPECTED_VERSION"; then
  echo "FAIL: Version mismatch"
  echo "Expected: $EXPECTED_VERSION"
  echo "Got: $VERSION_OUTPUT"
  exit 1
fi

# Check help
"$BINARY" --help > /dev/null

# Check status (allow failure if not in repo)
"$BINARY" status 2>/dev/null || true

echo "PASS: All validation checks passed"
```

### Phase 3: Local Testing

Add `goreleaser` to dev dependencies for local validation:

```bash
# Test before tagging
goreleaser build --snapshot --clean --single-target
./dist/ao_*/ao version  # Should show version, not "dev"
```

## Files Changed

| File | Change |
|------|--------|
| `.github/workflows/release.yml` | Replace with 3-stage workflow |
| `scripts/validate-release.sh` | New validation script |
| `docs/RELEASING.md` | Document the release process |

## Testing Plan

1. Create test tag `v0.0.0-test`
2. Verify build stage produces artifacts
3. Verify validate stage catches intentionally broken binary
4. Verify publish stage only runs after validate passes
5. Delete test tag and release

## Future Enhancements

- [ ] Add Linux binary validation (via container)
- [ ] Add integration test suite
- [ ] Add changelog generation
- [ ] Add Slack/Discord notification on release

## Decision Log

| Decision | Rationale |
|----------|-----------|
| Single workflow, not separate | Simpler, artifacts stay together |
| Auto-approve if green | Minimal friction for frequent releases |
| Validate darwin-arm64 only | Most common dev platform, catches 90% of issues |
| Keep GoReleaser | Industry standard, handles Homebrew automatically |
