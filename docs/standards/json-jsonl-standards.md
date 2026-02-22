# JSON/JSONL Standards

<!-- Canonical source: gitops/docs/standards/json-jsonl-standards.md -->
<!-- Last synced: 2026-01-19 -->

> **Purpose:** Standards for JSON configuration files and JSONL data formats used in this repository.

## Scope

This document covers: JSON formatting, JSONL conventions (beads, logs), schema validation, and tool configuration patterns.

**Related:**
- [YAML/Helm Standards](./yaml-helm-standards.md) - Alternative configuration format
- [Markdown Style Guide](./markdown-style-guide.md) - Documentation conventions
- [Python Style Guide](./python-style-guide.md) - Python JSON handling

---

## Quick Reference

| Standard | Value | Validation |
|----------|-------|------------|
| **Indentation** | 2 spaces | `jq .` or Prettier |
| **Trailing Newline** | Required | Editor config |
| **Trailing Commas** | Not allowed (JSON spec) | JSON parser |
| **Comments** | Not allowed (JSON spec) | Use JSONC for config |
| **Line Length** | No hard limit | Keep readable |
| **JSONL Delimiter** | Newline (`\n`) | One object per line |

---

## Format Decision Tree

```text
What is the file's purpose?
├─ Configuration (human-edited)
│   ├─ Needs comments? → Use JSONC (*.jsonc) or YAML
│   └─ No comments needed? → Use JSON with descriptive keys
├─ Data interchange
│   ├─ Single record/object → JSON
│   └─ Multiple records (append-only) → JSONL
├─ Log/event data → JSONL
└─ Schema definition → JSON Schema (*.schema.json)
```

---

## JSON Formatting

### Standard Format

```json
{
  "name": "example",
  "version": "1.0.0",
  "config": {
    "timeout": 30,
    "retries": 3,
    "enabled": true
  },
  "items": [
    "first",
    "second",
    "third"
  ]
}
```

### Formatting Rules

| Rule | Example | Why |
|------|---------|-----|
| 2-space indent | `  "key": "value"` | Readability, smaller files |
| Double quotes only | `"key"` not `'key'` | JSON spec requirement |
| No trailing commas | `["a", "b"]` not `["a", "b",]` | JSON spec requirement |
| Trailing newline | File ends with `\n` | POSIX standard, git diffs |
| UTF-8 encoding | Always | Universal compatibility |

### Key Naming

| Convention | Use For | Example |
|------------|---------|---------|
| `camelCase` | JavaScript/TypeScript config | `"apiVersion"` |
| `snake_case` | Python config, beads | `"issue_type"` |
| `kebab-case` | Avoid (quoting issues) | - |
| `UPPER_CASE` | Environment variables only | `"DATABASE_URL"` |

**ALWAYS:** Be consistent within a file. Match the ecosystem convention.

---

## JSONL Format

### What is JSONL?

JSON Lines: one valid JSON object per line, newline-delimited.

```jsonl
{"id": "abc-123", "status": "open", "title": "First issue"}
{"id": "abc-124", "status": "closed", "title": "Second issue"}
{"id": "abc-125", "status": "open", "title": "Third issue"}
```

### When to Use JSONL

| Use JSONL | Use JSON |
|-----------|----------|
| Append-only data (logs, events) | Single configuration object |
| Streaming ingestion | Nested hierarchical data |
| Line-by-line processing | Small datasets (<100 records) |
| Beads issues tracking | API responses |
| Large datasets | Human-edited files |

### JSONL Rules

| Rule | Rationale |
|------|-----------|
| One object per line | Enables `grep`, `head`, `tail` |
| No trailing comma | Each line is complete JSON |
| No array wrapper | Not `[{...}, {...}]` |
| Newline after last record | POSIX, append-friendly |
| UTF-8, no BOM | Universal compatibility |

### Processing JSONL

```bash
# Count records
wc -l issues.jsonl

# Filter by field
jq -c 'select(.status == "open")' issues.jsonl

# Extract field
jq -r '.title' issues.jsonl

# Pretty-print one record
head -1 issues.jsonl | jq .

# Append new record
echo '{"id": "new", "status": "open"}' >> issues.jsonl
```

---

## Beads JSONL Format

The `.beads/issues.jsonl` file uses a specific schema:

### Issue Record Schema

```json
{
  "id": "prefix-xxxx",
  "title": "Issue title",
  "status": "open",
  "priority": 2,
  "issue_type": "task",
  "owner": "user@example.com",
  "created_at": "2026-01-15T08:18:34.317984-05:00",
  "created_by": "User Name",
  "updated_at": "2026-01-15T08:42:39.253689-05:00",
  "closed_at": null,
  "close_reason": null,
  "dependencies": []
}
```

### Field Reference

| Field | Type | Required | Values |
|-------|------|----------|--------|
| `id` | string | Yes | `prefix-xxxx` format |
| `title` | string | Yes | Brief description |
| `status` | string | Yes | `open`, `in_progress`, `closed` |
| `priority` | integer | Yes | 0-4 (0=critical, 4=backlog) |
| `issue_type` | string | Yes | `task`, `bug`, `feature`, `epic`, `rig`, `agent` |
| `owner` | string | No | Email address |
| `created_at` | string | Yes | ISO 8601 timestamp |
| `created_by` | string | Yes | Creator name |
| `updated_at` | string | Yes | ISO 8601 timestamp |
| `closed_at` | string | No | ISO 8601 timestamp or null |
| `close_reason` | string | No | Reason for closing |
| `dependencies` | array | No | Dependency objects |

### Dependency Object

```json
{
  "issue_id": "prefix-child",
  "depends_on_id": "prefix-parent",
  "type": "parent-child",
  "created_at": "2026-01-15T08:19:32.440350-05:00",
  "created_by": "User Name"
}
```

---

## Configuration Files

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "strict": true,
    "outDir": "./dist"
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules"]
}
```

### package.json

```json
{
  "name": "package-name",
  "version": "1.0.0",
  "description": "Brief description",
  "main": "dist/index.js",
  "scripts": {
    "build": "tsc",
    "test": "jest",
    "lint": "eslint ."
  },
  "dependencies": {
    "dependency": "^1.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}
```

### VS Code settings.json

```json
{
  "editor.formatOnSave": true,
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "[json]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "files.insertFinalNewline": true,
  "files.trimTrailingWhitespace": true
}
```

---

## JSON Schema

### Defining Schemas

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/config.schema.json",
  "title": "Configuration",
  "type": "object",
  "required": ["name", "version"],
  "properties": {
    "name": {
      "type": "string",
      "description": "Project name",
      "minLength": 1
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$"
    },
    "enabled": {
      "type": "boolean",
      "default": true
    }
  },
  "additionalProperties": false
}
```

### Schema Validation

```bash
# Using ajv-cli
npx ajv validate -s schema.json -d config.json

# Using Python jsonschema
python -c "
import json
from jsonschema import validate

with open('schema.json') as s, open('config.json') as c:
    validate(json.load(c), json.load(s))
"
```

---

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| `Unexpected token` | Trailing comma | Remove comma after last item |
| `Unexpected token '` | Single quotes | Use double quotes only |
| `Unexpected token /` | Comments in JSON | Use JSONC or remove comments |
| `Invalid character` | BOM or wrong encoding | Save as UTF-8 without BOM |
| `Unexpected end of input` | Truncated file | Check for complete structure |
| JSONL parse error | Multi-line object | Ensure one object per line |
| Git merge conflict in JSONL | Concurrent edits | Append-only, resolve manually |

---

## Anti-Patterns

| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| Minified Config | `{"a":1,"b":2}` | Unreadable, hard to diff | Pretty-print with 2 spaces |
| Comments in JSON | `// comment` | Invalid JSON, breaks parsers | Use JSONC or key naming |
| Mixed Conventions | `camelCase` + `snake_case` | Inconsistent, confusing | Pick one per file |
| Nested Arrays of Arrays | `[[[]]]` | Hard to parse, validate | Flatten or use objects |
| Magic Numbers | `"priority": 2` | Meaning unclear | Document or use enums |
| No Schema | Large config without schema | No validation, drift | Add JSON Schema |

---

## Tooling

### Formatting

```bash
# jq - Format and validate
jq . config.json > formatted.json
cat config.json | jq .

# Prettier - Format with config
npx prettier --write '**/*.json'

# Python - Format
python -m json.tool config.json
```

### Validation

```bash
# jq - Check valid JSON
jq empty config.json && echo "Valid" || echo "Invalid"

# Python - Check valid JSON
python -c "import json; json.load(open('config.json'))"

# Node - Check valid JSON
node -e "require('./config.json')"
```

### JSONL Processing

```bash
# Validate each line
while read -r line; do
  echo "$line" | jq empty || echo "Invalid line: $line"
done < data.jsonl

# Convert JSON array to JSONL
jq -c '.[]' array.json > data.jsonl

# Convert JSONL to JSON array
jq -s '.' data.jsonl > array.json
```

---

## Editor Configuration

### .editorconfig

```ini
[*.json]
indent_style = space
indent_size = 2
insert_final_newline = true
trim_trailing_whitespace = true
charset = utf-8

[*.jsonl]
indent_style = space
indent_size = 0
insert_final_newline = true
```

### .prettierrc

```json
{
  "tabWidth": 2,
  "useTabs": false,
  "trailingComma": "none",
  "singleQuote": false
}
```

---

## Summary

**Key Takeaways:**

1. 2-space indentation, trailing newline, UTF-8 encoding
2. No trailing commas (JSON spec)
3. Use JSONL for append-only data, JSON for config
4. Match key naming to ecosystem (camelCase for JS, snake_case for Python)
5. Add JSON Schema for large/shared config files
6. Process JSONL with `jq -c` for streaming
7. Validate JSON with `jq empty` or language parser
8. Beads uses JSONL with specific issue schema
9. Use Prettier or jq for consistent formatting
