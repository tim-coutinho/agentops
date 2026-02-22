# Markdown Style Guide

<!-- Canonical source: gitops/docs/standards/markdown-style-guide.md -->
<!-- Last synced: 2026-01-19 -->

> **Purpose:** AI-agent-optimized Markdown conventions for documentation, skills, and reference material.

## Scope

This document covers: structure optimization for AI parsing, link conventions, code block formatting, table standards, and progressive disclosure patterns.

**Related:**
- [Tag Vocabulary](./tag-vocabulary.md) - Document tagging standards
- [Python Style Guide](./python-style-guide.md) - Python coding conventions
- [YAML/Helm Standards](./yaml-helm-standards.md) - Configuration file standards

---

## Quick Reference

| Standard | Value | Validation |
|----------|-------|------------|
| **Line Length** | 100 chars soft limit | Visual check |
| **Heading Style** | ATX (`#`) | markdownlint |
| **List Marker** | `-` for unordered | markdownlint |
| **Code Fence** | Triple backtick with language | markdownlint |
| **Link Style** | Reference links for repeated URLs | Visual check |

---

## AI-Agent Optimization Principles

| Principle | Implementation | Why |
|-----------|----------------|-----|
| **Tables over prose** | Use tables for comparisons, options, mappings | Parallel parsing, scannable |
| **Explicit rules** | ALWAYS/NEVER, not "try to" | Removes ambiguity |
| **Decision trees** | If/then logic in lists or tables | Executable reasoning |
| **Named patterns** | Anti-patterns with names | Recognizable error states |
| **Progressive disclosure** | Quick ref → details JIT | Context window efficiency |
| **Copy-paste ready** | Complete examples, not fragments | Reduces inference errors |

---

## Document Structure

### SKILL.md Template

```markdown
# Skill Name

> **Triggers:** "phrase 1", "phrase 2", "phrase 3"

## Quick Reference

| Action | Command | Notes |
|--------|---------|-------|
| ... | ... | ... |

## When to Use

| Scenario | Action |
|----------|--------|
| Condition A | Do X |
| Condition B | Do Y |

## Workflow

1. Step one
2. Step two
3. Step three

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| Error message | Root cause | Solution |

## References

- [Reference 1](./references/detail1.md) - Load when needed
- [Reference 2](./references/detail2.md) - Load when needed
```

### Reference Doc Template

```markdown
# Reference: Topic Name

<!-- Load JIT when skill needs deep context -->

## Context

Brief overview of when this reference applies.

## Details

### Section 1

...

### Section 2

...

## Decision Tree

```text
Is X true?
├─ Yes → Do A
│   └─ Did A fail?
│       ├─ Yes → Try B
│       └─ No → Done
└─ No → Do C
```

## Anti-Patterns

| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| ... | ... | ... | ... |
```

---

## Heading Conventions

### Hierarchy Rules

| Level | Use For | Example |
|-------|---------|---------|
| `#` | Document title (one per file) | `# Style Guide` |
| `##` | Major sections | `## Installation` |
| `###` | Subsections | `### macOS Setup` |
| `####` | Minor divisions (sparingly) | `#### Homebrew Method` |

**NEVER:**
- Skip heading levels (`#` → `###`)
- Use bold text as fake headings
- Start with `##` (missing `#` title)

### Heading Text

```markdown
# Good - Title Case for Title
## Good - Sentence case for sections
### Good - Sentence case continues

# Bad - all lowercase title
## Bad - ALL CAPS SECTION
### Bad - Using: Colons: Everywhere
```

---

## Code Blocks

### Language Hints (Required)

ALWAYS specify language for syntax highlighting:

```markdown
# Good - Language specified
```python
def hello():
    print("world")
```

# Bad - No language hint
```
def hello():
    print("world")
```
```

### Common Language Hints

| Language | Fence | Use For |
|----------|-------|---------|
| `bash` | ` ```bash ` | Shell commands, scripts |
| `python` | ` ```python ` | Python code |
| `go` | ` ```go ` | Go code |
| `typescript` | ` ```typescript ` | TypeScript code |
| `yaml` | ` ```yaml ` | YAML configuration |
| `json` | ` ```json ` | JSON data |
| `text` | ` ```text ` | Plain text, ASCII diagrams |
| `diff` | ` ```diff ` | Code diffs |
| `toml` | ` ```toml ` | TOML configuration |

### Command Output

For commands with expected output:

```markdown
```bash
$ kubectl get pods
NAME         READY   STATUS    RESTARTS   AGE
my-pod       1/1     Running   0          5m
```
```

---

## Tables

### When to Use Tables

| Situation | Use Table? | Alternative |
|-----------|------------|-------------|
| Comparing 3+ items | Yes | - |
| Key-value mappings | Yes | - |
| Command reference | Yes | - |
| Step-by-step instructions | No | Numbered list |
| Narrative explanation | No | Paragraphs |
| Two items only | No | Inline comparison |

### Table Formatting

```markdown
# Good - Aligned, readable
| Column A | Column B | Column C |
|----------|----------|----------|
| Value 1  | Value 2  | Value 3  |
| Long value here | Short | Medium |

# Bad - Misaligned, hard to read
|Column A|Column B|Column C|
|-|-|-|
|Value 1|Value 2|Value 3|
```

### Table Cell Content

| Content Type | Formatting |
|--------------|------------|
| Code/commands | Backticks: `` `kubectl get pods` `` |
| Emphasis | Bold for key terms: `**required**` |
| Links | Inline: `[text](url)` |
| Long text | Keep under 50 chars, or use footnotes |

---

## Links

### Link Style Decision Tree

```text
How many times is this URL used?
├─ Once → Inline link: [text](url)
└─ Multiple times → Reference link: [text][ref]

Is it an internal doc?
├─ Yes → Relative path: [Guide](./other-doc.md)
└─ No → Full URL: [Docs](https://example.com)
```

### Internal Links

```markdown
# Good - Relative paths for internal docs
See [Python Style Guide](./python-style-guide.md) for details.

# Good - Anchor links within document
See [Code Blocks](#code-blocks) above.

# Bad - Absolute paths (break on clone)
See [Guide](/Users/me/project/docs/guide.md).

# Bad - GitHub URLs for local docs
See [Guide](https://github.com/org/repo/blob/main/docs/guide.md).
```

### Reference Links

For repeated URLs:

```markdown
See the [official docs][k8s-docs] for more info.
The [Kubernetes documentation][k8s-docs] covers this in detail.

[k8s-docs]: https://kubernetes.io/docs/
```

---

## Lists

### Unordered Lists

Use `-` consistently (not `*` or `+`):

```markdown
# Good
- Item one
- Item two
  - Nested item
  - Another nested

# Bad - Mixed markers
* Item one
+ Item two
- Item three
```

### Ordered Lists

Use `1.` for all items (auto-renumbering):

```markdown
# Good - All 1s (auto-renumbers)
1. First step
1. Second step
1. Third step

# Acceptable - Explicit numbering
1. First step
2. Second step
3. Third step

# Bad - Wrong numbers
1. First step
3. Second step
2. Third step
```

### Task Lists

For checklists:

```markdown
- [ ] Incomplete task
- [x] Completed task
- [ ] Another incomplete
```

---

## Emphasis

| Purpose | Syntax | Example |
|---------|--------|---------|
| Important terms | `**bold**` | **required** |
| File names, commands | `` `backticks` `` | `config.yaml` |
| Titles, emphasis | `*italic*` | *optional* |
| Keyboard keys | `<kbd>` | <kbd>Ctrl</kbd>+<kbd>C</kbd> |

**NEVER use bold for:**
- Entire paragraphs
- Headings (use `#`)
- Code (use backticks)

---

## Blockquotes

### Callout Patterns

```markdown
> **Note:** Supplementary information.

> **Warning:** Something that could cause issues.

> **Important:** Critical information that must not be missed.

> **Tip:** Helpful suggestion or shortcut.
```

### Multi-line Quotes

```markdown
> This is a longer quote that spans
> multiple lines. Each line starts
> with the `>` character.
```

---

## Images

### Image Syntax

```markdown
# Basic image
![Alt text](./images/diagram.png)

# Image with title
![Alt text](./images/diagram.png "Optional title")

# Reference-style image
![Alt text][img-ref]

[img-ref]: ./images/diagram.png
```

### Alt Text Requirements

| Image Type | Alt Text Should Include |
|------------|------------------------|
| Screenshot | What it shows, key elements |
| Diagram | What it represents |
| Icon | Purpose/meaning |
| Decorative | Leave empty: `![](image.png)` |

---

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| Broken internal links | Wrong relative path | Use `./` prefix, check file exists |
| Code not highlighted | Missing language hint | Add language after opening fence |
| Table renders as text | Missing blank line before | Add blank line before table |
| List breaks mid-content | Missing blank line | Add blank line between list items with blocks |
| Heading not rendered | No space after `#` | Add space: `# Title` not `#Title` |
| Nested list wrong | Inconsistent indentation | Use exactly 2 spaces for nesting |

---

## Anti-Patterns

| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| Wall of Text | Long paragraphs without structure | Hard to scan, agents miss key info | Break into lists/tables |
| Implicit Logic | "You might want to..." | Ambiguous for agents | Explicit: "ALWAYS do X when Y" |
| Deep Nesting | 4+ levels of bullet points | Confusing hierarchy | Flatten or use headings |
| Link Rot | URLs without context | Agents can't verify broken links | Add descriptive text |
| Screenshot-Only | Instructions only in images | Agents can't read images | Text + optional screenshot |
| Fake Headings | `**Bold Text**` as section title | Breaks TOC, navigation | Use `##` headings |

---

## Validation

### markdownlint Configuration

Create `.markdownlint.yml` at repo root:

```yaml
# .markdownlint.yml
default: true

# Line length - soft limit
MD013:
  line_length: 100
  code_blocks: false
  tables: false

# Allow inline HTML for kbd, etc.
MD033:
  allowed_elements:
    - kbd
    - br
    - details
    - summary

# Allow bare URLs in some contexts
MD034: false

# Consistent list markers
MD004:
  style: dash

# Heading style
MD003:
  style: atx
```

### Validation Commands

```bash
# Lint Markdown files
npx markdownlint '**/*.md' --ignore node_modules

# Check links (optional)
npx markdown-link-check README.md

# Format with Prettier
npx prettier --write '**/*.md'
```

---

## Summary

**Key Takeaways:**

1. Optimize for AI parsing: tables, explicit rules, decision trees
2. ATX headings (`#`), never skip levels
3. Always specify language hints on code blocks
4. Use `-` for unordered lists, `1.` for ordered
5. Relative paths for internal links
6. Reference links for repeated URLs
7. Tables for comparisons, lists for sequences
8. Add blank lines before tables and code blocks
9. Validate with markdownlint
