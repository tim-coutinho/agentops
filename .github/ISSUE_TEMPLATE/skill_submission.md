---
name: Skill Submission
about: Submit a new skill to the AgentOps skill catalog
title: '[SKILL] '
labels: skill-submission, needs-review
assignees: ''
---

## Skill Information

**Name:** [your-skill-name]
**Version:** [1.0.0]
**Description:** [Brief description of what your skill does]
**Author:** [Your Name]
**License:** [Apache-2.0 or other]

## Features

List the main features of your skill:

- [ ] Feature 1
- [ ] Feature 2
- [ ] Feature 3

## Components

### Agents

List your agents with brief descriptions:

- **agent-name** - Description of what this agent does

### Commands

List your commands:

- **/command-name** - Description of what this command does

### Skills

List your skills:

- **skill-name** - Description of what this skill provides

### Hooks

List any hooks (if applicable):

- **hook-name** - Description

### MCP Servers

List any MCP server integrations (if applicable):

- **server-name** - Description

## Token Budget

**Estimated tokens:** [e.g., 5,000]
**Percentage of context:** [e.g., 2.5%]

**How was this estimated?**
- [ ] Word count x 1.3
- [ ] Actual testing
- [ ] Approximation

## Dependencies

List any skill dependencies:

- [ ] core-workflow (required/optional)
- [ ] Other skills (list them)

## Testing Checklist

**Pre-submission testing:**

- [ ] Skill installs successfully locally
- [ ] All agents tested and working
- [ ] All commands tested and working
- [ ] All skills tested and working
- [ ] YAML frontmatter validated (all SKILL.md files)
- [ ] Token budget verified through testing
- [ ] Documentation complete (README.md)
- [ ] Examples provided and tested
- [ ] No hardcoded secrets or credentials
- [ ] No security vulnerabilities
- [ ] Links in documentation work

**Code quality:**

- [ ] Follows 12-Factor AgentOps principles (if applicable)
- [ ] Includes Laws of an Agent in agents
- [ ] Clear agent tool permissions
- [ ] Comprehensive error handling
- [ ] Anti-patterns documented

## Installation Instructions

Provide installation command:

```bash
npx skills@latest add your-repo/your-skill --all -g
```

## Usage Examples

Provide 2-3 concrete usage examples:

### Example 1: [Use Case]

```bash
# Steps to use
```

**Expected result:**
[What should happen]

### Example 2: [Use Case]

```bash
# Steps to use
```

**Expected result:**
[What should happen]

## Screenshots/Demos

If applicable, include screenshots or demo videos showing your skill in action.

[Attach images or provide links]

## Documentation

**README.md included?** [ ] Yes [ ] No

**Documentation covers:**
- [ ] Installation
- [ ] Quick start
- [ ] All components
- [ ] Examples
- [ ] Troubleshooting
- [ ] Known limitations

## Related Issues

Does this skill address any existing issues?

Closes #[issue-number]
Relates to #[issue-number]

## Additional Context

Add any other context, design decisions, or notes about the skill:

- Why was this skill created?
- What problems does it solve?
- Any special considerations?
- Future enhancements planned?

## Checklist for Maintainers

**Maintainers will verify:**

- [ ] Skill structure is correct
- [ ] SKILL.md has valid YAML frontmatter
- [ ] All required files present
- [ ] Documentation is comprehensive
- [ ] Token budget is reasonable
- [ ] No security issues
- [ ] Code quality is acceptable
- [ ] Examples work as described
- [ ] Follows contribution guidelines
- [ ] Ready to merge

---

**By submitting this skill, I confirm:**

- [ ] I have read and agree to the [Code of Conduct](../CODE_OF_CONDUCT.md)
- [ ] I have read the [Contributing Guidelines](../CONTRIBUTING.md)
- [ ] This is original work or properly attributed
- [ ] I have the rights to distribute this code
- [ ] This skill follows the [Security Policy](../SECURITY.md)
- [ ] I will respond to feedback and review comments
