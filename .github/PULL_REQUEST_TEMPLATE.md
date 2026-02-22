# Pull Request

## Description

**What does this PR do?**

[Provide a clear and concise description of the changes]

## Type of Change

**What type of change is this?**

- [ ] New plugin submission
- [ ] Plugin update/enhancement
- [ ] Bug fix
- [ ] Documentation update
- [ ] Infrastructure/tooling improvement
- [ ] Breaking change
- [ ] Other: [specify]

## Related Issues

**Does this PR address any issues?**

Closes #[issue-number]
Fixes #[issue-number]
Relates to #[issue-number]

---

## For Plugin Submissions

**If this is a new plugin or plugin update, complete this section:**

### Plugin Information

- **Name:** [plugin-name]
- **Version:** [1.0.0]
- **Description:** [Brief description]
- **Token Budget:** [~X tokens (X%)]

### Components Added/Changed

- [ ] Agents (list: )
- [ ] Commands (list: )
- [ ] Skills (list: )
- [ ] Hooks (list: )
- [ ] MCP servers (list: )

### Dependencies

- [ ] core-workflow (required/optional)
- [ ] Other plugins: [list]

### Testing Completed

**Pre-submission testing:**

- [ ] Plugin installs successfully locally
- [ ] All agents tested and working as expected
- [ ] All commands tested and working as expected
- [ ] All skills tested and working as expected
- [ ] JSON manifests validated (no syntax errors)
- [ ] YAML frontmatter validated (all agents)
- [ ] Token budget verified through actual testing
- [ ] Documentation is complete and accurate
- [ ] All examples tested and working
- [ ] No hardcoded secrets or credentials
- [ ] No security vulnerabilities identified
- [ ] All links in documentation work

**Installation test command:**

```bash
/plugin install file://$(pwd)/plugins/[plugin-name]
```

### Usage Examples

**Provide at least one working example:**

```bash
# Example usage
```

**Expected output:**
[What should happen]

---

## For Bug Fixes

**If this is a bug fix, complete this section:**

### Bug Description

[What bug does this fix?]

### Root Cause

[What was causing the bug?]

### Solution

[How does this PR fix it?]

### Testing

- [ ] Bug reproduced before fix
- [ ] Bug no longer occurs after fix
- [ ] No regression in other functionality
- [ ] Added test to prevent regression

---

## For Documentation Updates

**If this is a documentation update:**

### Changes Made

- [ ] README.md updated
- [ ] Plugin documentation updated
- [ ] Contributing guidelines updated
- [ ] Security policy updated
- [ ] Other: [specify]

### Reason for Update

[Why was this documentation update needed?]

---

## Changes Made

**Detailed breakdown of changes:**

### Files Added

- `path/to/file.ext` - [Purpose]

### Files Modified

- `path/to/file.ext` - [What changed and why]

### Files Deleted

- `path/to/file.ext` - [Why deleted]

## Testing Strategy

**How did you test these changes?**

1. [Test approach 1]
2. [Test approach 2]
3. [Test approach 3]

**Test results:**

```
[Include relevant test output or screenshots]
```

## Breaking Changes

**Does this PR introduce breaking changes?**

- [ ] Yes (explain below)
- [x] No

**If yes, describe the breaking changes:**

- [What breaks]
- [Migration path for users]
- [Documentation updates needed]

## Documentation

**Have you updated relevant documentation?**

- [ ] README.md (if user-facing changes)
- [ ] CHANGELOG.md (if version bump)
- [ ] Plugin README (if plugin changes)
- [ ] Agent documentation (if agent changes)
- [ ] Contributing guidelines (if process changes)
- [ ] N/A - No documentation needed

## Code Quality

**Self-review checklist:**

- [ ] Code follows project style guidelines
- [ ] Added comments for complex logic
- [ ] No unnecessary console.log or debug code
- [ ] No commented-out code (unless explained)
- [ ] Variable/function names are descriptive
- [ ] Error handling is appropriate
- [ ] Security best practices followed

## Security

**Security considerations:**

- [ ] No secrets or credentials in code
- [ ] Input validation where needed
- [ ] No SQL injection vulnerabilities
- [ ] No XSS vulnerabilities
- [ ] Dependencies are up to date
- [ ] Security policy reviewed
- [ ] N/A - No security implications

## Performance

**Performance impact:**

- [ ] No significant performance impact
- [ ] Performance improved (explain how)
- [ ] Performance regressed (explain why acceptable)
- [ ] Not applicable

## Deployment

**Deployment considerations:**

- [ ] No special deployment steps needed
- [ ] Requires configuration changes (documented)
- [ ] Requires database migration
- [ ] Requires dependency installation
- [ ] Other: [specify]

## Screenshots/Examples

**If applicable, add screenshots or examples:**

[Attach or describe visual changes]

## Checklist

**Before submitting, ensure:**

- [ ] I have read the [Contributing Guidelines](../CONTRIBUTING.md)
- [ ] I have read the [Code of Conduct](../CODE_OF_CONDUCT.md)
- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented complex code appropriately
- [ ] I have updated documentation as needed
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix/feature works
- [ ] New and existing tests pass locally
- [ ] Any dependent changes have been merged and published
- [ ] I have updated CHANGELOG.md (if applicable)

## Additional Notes

**Any additional information for reviewers:**

- Special considerations
- Known limitations
- Future enhancements planned
- Questions for reviewers

---

## For Reviewers

**Review checklist:**

- [ ] Code quality is acceptable
- [ ] Tests are comprehensive
- [ ] Documentation is complete
- [ ] No security issues identified
- [ ] Performance impact is acceptable
- [ ] Breaking changes are justified (if any)
- [ ] CHANGELOG.md updated (if needed)
- [ ] Ready to merge

**Review notes:**

[Space for reviewer comments]
