# YAML/Helm Standards

<!-- Canonical source: gitops/docs/standards/yaml-helm-standards.md -->
<!-- Last synced: 2026-01-19 -->

> **Purpose**: YAML and Helm chart standards for this repository.

## Scope

This document covers: YAML formatting, Helm chart conventions, Kustomize patterns, and validation tools.

**Related:**
- [Python Style Guide](./python-style-guide.md) - Python scripting conventions
- [Shell Script Standards](./shell-script-standards.md) - Bash conventions

---

## Quick Reference

| Aspect | Standard |
|--------|----------|
| **Indentation** | 2 spaces |
| **Line length** | 120 characters max |
| **Linter** | yamllint |
| **Helm** | helm lint, helm template |
| **Kustomize** | kustomize build --enable-helm |

---

## yamllint Configuration

Create `.yamllint.yml` at repo root:

```yaml
extends: default
rules:
  line-length:
    max: 120
    allow-non-breakable-inline-mappings: true
  indentation:
    spaces: 2
    indent-sequences: consistent
  truthy:
    check-keys: false
  comments:
    min-spaces-from-content: 1
  document-start: disable
  empty-lines:
    max: 2
```

**Usage:**
```bash
# Lint all YAML files
yamllint .

# Lint specific directory
yamllint apps/
```

---

## Helm Chart Conventions

### Chart Structure

```text
charts/<chart-name>/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── deployment.yaml
│   └── ...
└── charts/           # Nested charts (if needed)
```

### Validation Commands

```bash
# Lint chart
helm lint charts/<chart-name>/

# Template with values (dry-run)
helm template <release> charts/<chart-name>/ -f values.yaml

# Validate rendered output
helm template <release> charts/<chart-name>/ | kubectl apply --dry-run=client -f -
```

### values.yaml Conventions

```yaml
# GOOD - Commented sections, logical grouping
# =============================================================================
# Application Configuration
# =============================================================================

app:
  name: my-app
  replicas: 3

# Resource limits (adjust for environment)
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi

# BAD - No structure, no comments
app:
  name: my-app
replicas: 3
cpu: 100m
memory: 128Mi
```

---

## Kustomize Patterns

### Overlay Structure

```text
apps/<app>/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   └── service.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    ├── staging/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

### kustomization.yaml Template

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
  - service.yaml

# Environment-specific patches
patches:
  - path: ./patches/replicas.yaml
    target:
      kind: Deployment
      name: my-app
```

### Patch Types

| Type | Use Case | Example |
|------|----------|---------|
| **Strategic Merge** | Add/modify fields | `patches/extend-rbac.yaml` |
| **JSON Patch** | Precise operations | `patches/remove-field.yaml` |
| **Delete** | Remove resources | `$patch: delete` annotation |

**Strategic Merge Patch:**
```yaml
# patches/extend-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: my-role
rules:
  - apiGroups: ["custom.io"]
    resources: ["widgets"]
    verbs: ["get", "list"]
```

**Delete Patch:**
```yaml
# patches/delete-resource.yaml
$patch: delete
apiVersion: v1
kind: ConfigMap
metadata:
  name: unused-config
```

---

## Formatting Rules

### Quoting Strings

```yaml
# GOOD - Quote strings that look like other types
enabled: "true"      # String, not boolean
port: "8080"         # String, not integer
version: "1.0"       # String, not float

# GOOD - No quotes for actual typed values
enabled: true        # Boolean
port: 8080           # Integer
replicas: 3          # Integer

# BAD - Ambiguous without context
version: 1.0         # Parsed as float, might lose precision
```

### Multi-line Strings

```yaml
# GOOD - Literal block scalar (preserves newlines)
script: |
  #!/bin/bash
  set -euo pipefail
  echo "Hello"

# GOOD - Folded block scalar (folds newlines to spaces)
description: >
  This is a long description that will be
  folded into a single line with spaces.

# BAD - Escaped newlines (hard to read)
script: "#!/bin/bash\nset -euo pipefail\necho \"Hello\""
```

### Comments

```yaml
# Section header (full line)
# =============================================================================
# Database Configuration
# =============================================================================

database:
  host: localhost      # Inline comment (1 space before #)
  port: 5432
  # Subsection comment
  credentials:
    username: admin
```

---

## Template Best Practices

### Use include for Reusable Snippets

```yaml
# templates/_helpers.tpl
{{- define "app.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

# templates/deployment.yaml
metadata:
  labels:
    {{- include "app.labels" . | nindent 4 }}
```

### Whitespace Control

```yaml
# GOOD - Use {{- and -}} to control whitespace
{{- if .Values.enabled }}
apiVersion: v1
kind: ConfigMap
{{- end }}

# BAD - Extra blank lines in output
{{ if .Values.enabled }}

apiVersion: v1

{{ end }}
```

### Required Values

```yaml
# Fail fast if required value missing
image: {{ required "image.repository is required" .Values.image.repository }}
```

---

## Validation Workflow

### Pre-commit Checks

```bash
# 1. Lint YAML
yamllint .

# 2. Lint Helm charts
for chart in charts/*/Chart.yaml; do
    helm lint "$(dirname "$chart")"
done

# 3. Build Kustomize overlays
kustomize build apps/<app>/ --enable-helm > /dev/null
```

### CI Pipeline Example

```yaml
# .github/workflows/validate.yaml (example)
- name: Lint YAML
  run: yamllint .

- name: Lint Helm
  run: |
    for chart in charts/*/Chart.yaml; do
      helm lint "$(dirname "$chart")"
    done

- name: Validate Kustomize
  run: |
    for kust in apps/*/kustomization.yaml; do
      kustomize build "$(dirname "$kust")" --enable-helm > /dev/null
    done
```

---

## Format Decision Tree

```text
What are you configuring?
├─ Kubernetes manifests
│   ├─ Single environment? → Plain YAML
│   ├─ Multiple environments, small changes? → Kustomize overlays
│   └─ Complex templating needed? → Helm chart
├─ Application config
│   ├─ Needs comments? → YAML
│   └─ Machine-generated? → JSON
├─ CI/CD pipeline
│   └─ → YAML (platform-specific: GitHub Actions, GitLab CI)
└─ Data/records
    └─ → JSON or JSONL (see json-jsonl-standards.md)
```

### Helm vs Kustomize

| Use Helm When | Use Kustomize When |
|---------------|-------------------|
| Complex conditional logic | Simple overlay changes |
| Parameterized charts for distribution | Environment-specific patches |
| Need Helm ecosystem (repositories, hooks) | Already have plain manifests |
| Charts will be shared/published | Internal-only deployments |
| Need computed values | Static configuration changes |

---

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| `mapping values not allowed` | Missing space after colon | `key: value` not `key:value` |
| `found duplicate key` | Repeated YAML key | Remove duplicate |
| `could not find expected ':'` | Unquoted special chars | Quote the value |
| `helm: values don't align` | Incorrect indentation | Use 2 spaces |
| `Error: YAML parse error` | Tab characters | Convert tabs to spaces |
| `nil pointer evaluating` | Missing value in Helm | Use `default` or `required` |
| Kustomize "resource not found" | Wrong path in resources | Check relative paths |
| Boolean parsed as string | `"true"` vs `true` | Remove quotes for boolean |

---

## Anti-Patterns

| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| Tab Indentation | Using tabs | YAML spec requires spaces | 2 spaces |
| Anchor Abuse | Complex `<<: *anchor` chains | Hard to debug | Duplicate with comments |
| Unquoted Versions | `version: 1.0` | Parsed as float | `version: "1.0"` |
| Inline JSON | `data: {"key": "value"}` | Hard to read | Multi-line YAML |
| No Comments | values.yaml without docs | Users guess meanings | Section comments |
| Hardcoded Secrets | `password: hunter2` | Security risk | External secret reference |

---

## AI Agent Guidelines

When AI agents write YAML/Helm for this repo:

| Guideline | Rationale |
|-----------|-----------|
| ALWAYS run `yamllint` before committing | Catches formatting issues |
| ALWAYS quote version strings | Prevents float parsing |
| ALWAYS use `helm template` to verify | Catches rendering errors |
| NEVER use tabs | YAML spec violation |
| NEVER hardcode secrets in values | Security risk |
| PREFER `\|` for multi-line strings | Preserves formatting |
| PREFER `required` over silent defaults | Fail fast on missing values |

---

## Summary

1. Use 2-space indentation, 120 char line limit
2. Run `yamllint` on all YAML files
3. Use `helm lint` and `helm template` for charts
4. Use Kustomize patches for environment customizations
5. Quote strings that look like numbers or booleans
6. Use `|` for multi-line strings
7. Document values.yaml with section comments
8. Use `{{- include ... }}` for reusable templates
9. Choose Helm vs Kustomize based on decision tree
10. Check Common Errors table for troubleshooting
