# Tag Vocabulary

<!-- Canonical source: gitops/docs/standards/tag-vocabulary.md -->
<!-- Last synced: 2026-01-19 -->

> **Purpose:** Controlled vocabulary for tagging `.agents/` documents to enable consistent categorization and retrieval.

## Scope

This document covers: tag selection rules, category definitions, and usage patterns for agent-generated artifacts.

**Related:**
- [Markdown Style Guide](./markdown-style-guide.md) - Document formatting conventions
- [Shell Script Standards](./shell-script-standards.md) - Bash scripting conventions
- [YAML/Helm Standards](./yaml-helm-standards.md) - Configuration file standards
- [Python Style Guide](./python-style-guide.md) - Python coding standards

---

## Quick Reference

| Rule | Value |
|------|-------|
| **Tag Count** | 3-5 tags per document |
| **First Tag** | MUST be document type |
| **Format** | lowercase, hyphenated |
| **New Tags** | Check existing before creating |

---

## Tag Selection Decision Tree

```text
Step 1: What type of document is this?
├─ Exploration/analysis → research
├─ Implementation roadmap → plan
├─ Reusable solution → pattern
├─ Post-work reflection → retro
└─ Extracted insight → learning

Step 2: What domain does it cover? (pick 1-2)
├─ AI/ML topics → agents, observability, evaluation, orchestration
├─ Infrastructure → infra, helm, kustomize, kubernetes, docker
├─ Operations → ops, ci-cd, workflow, docs, validation
└─ This project → 12-factor, context, deployment, governance

Step 3: What language/tech is involved? (if applicable)
├─ Shell/Bash → shell
├─ Python → python
├─ Go → go
├─ TypeScript → typescript
└─ Config files → yaml

Step 4: Any status modifier? (optional)
├─ Urgent → critical
├─ Waiting on something → blocked
├─ Out of date → deprecated
└─ Tested and confirmed → validated
```

---

## Tag Categories

### Document Type (Required - ALWAYS First)

| Tag | Directory | Use When |
|-----|-----------|----------|
| `research` | `.agents/research/` | Exploring a topic, gathering context |
| `plan` | `.agents/plans/` | Defining implementation steps |
| `pattern` | `.agents/patterns/` | Documenting a reusable solution |
| `retro` | `.agents/retros/` | Reflecting on completed work |
| `learning` | `.agents/learnings/` | Capturing extracted insights |

**Decision Table:**

| You Are Doing | Tag | Example |
|---------------|-----|---------|
| Reading docs, exploring code | `research` | "Understanding auth flow" |
| Breaking down a feature into tasks | `plan` | "Implement caching layer" |
| Describing how to solve recurring problem | `pattern` | "Error retry pattern" |
| Analyzing what went well/wrong | `retro` | "Sprint 5 retrospective" |
| Noting something to remember | `learning` | "K8s pod scheduling quirks" |

---

### AgentOps Domains

| Tag | Use When Document Covers | Example Topics |
|-----|--------------------------|----------------|
| `12-factor` | 12-factor methodology | Factor compliance, methodology patterns |
| `observability` | Monitoring, tracing | Metrics, logging, tracing setup |
| `agents` | AI agent architecture | Agent patterns, tool use, reasoning |
| `orchestration` | Multi-agent coordination | Workflows, handoffs, state machines |
| `context` | Context management | Window management, summarization |
| `evaluation` | Agent benchmarking | Metrics, testing, quality assessment |
| `deployment` | Production deployment | Release, rollout, scaling |
| `governance` | Safety, compliance | Guardrails, policies, auditing |

---

### Infrastructure Domains

| Tag | Use When Document Covers | Example Topics |
|-----|--------------------------|----------------|
| `infra` | General infrastructure | Cloud resources, provisioning |
| `helm` | Helm charts | Chart structure, values, templating |
| `kustomize` | Kustomize overlays | Patches, bases, transformations |
| `kubernetes` | Kubernetes resources | Pods, services, deployments |
| `docker` | Container images | Dockerfile, builds, registries |

---

### Operations Domains

| Tag | Use When Document Covers | Example Topics |
|-----|--------------------------|----------------|
| `ops` | Operations, runbooks | Incident response, maintenance |
| `ci-cd` | Pipelines, builds | GitHub Actions, GitLab CI, Tekton |
| `workflow` | Automation, orchestration | Scripts, scheduled tasks |
| `docs` | Documentation | READMEs, guides, references |
| `validation` | Testing, verification | Test suites, quality gates |

---

### Language Tags

| Tag | Use When Code Is | File Extensions |
|-----|------------------|-----------------|
| `shell` | Bash/shell scripts | `.sh`, `.bash` |
| `python` | Python code | `.py` |
| `go` | Go code | `.go` |
| `typescript` | TypeScript code | `.ts`, `.tsx` |
| `yaml` | YAML configurations | `.yaml`, `.yml` |

**ONLY add language tag if:**
- Document contains significant code examples
- Language choice is relevant to the topic
- Implementation is language-specific

---

### Status Tags (Optional Modifiers)

| Tag | Use When | Signal |
|-----|----------|--------|
| `critical` | Urgent priority | Handle immediately |
| `blocked` | Waiting on dependency | Can't proceed yet |
| `deprecated` | No longer recommended | Use newer approach |
| `validated` | Tested and confirmed | Safe to rely on |

---

## Tag Format Rules

| Rule | Good | Bad |
|------|------|-----|
| Lowercase only | `kubernetes` | `Kubernetes`, `KUBERNETES` |
| Hyphenated compounds | `ci-cd` | `ci_cd`, `cicd` |
| No spaces | `12-factor` | `12 factor` |
| No special chars | `agents` | `agents!`, `#agents` |
| Singular form | `agent` | `agents` (exception: domain tags) |

---

## Examples

### Research Document

**File:** `.agents/research/auth-flow-analysis.md`

```yaml
---
tags: [research, agents, context, validated]
created: 2026-01-15
author: agent
---
```

**Reasoning:**
- `research` - Exploration document (type, first)
- `agents` - About agent architecture (primary domain)
- `context` - Involves context management (secondary domain)
- `validated` - Findings have been verified (status)

---

### Implementation Plan

**File:** `.agents/plans/caching-layer-implementation.md`

```yaml
---
tags: [plan, deployment, kubernetes, infra]
created: 2026-01-15
author: agent
---
```

**Reasoning:**
- `plan` - Implementation roadmap (type, first)
- `deployment` - About deploying a feature (primary domain)
- `kubernetes` - K8s-specific implementation (infrastructure)
- `infra` - Infrastructure-level work (domain)

---

### Pattern Document

**File:** `.agents/patterns/retry-with-backoff.md`

```yaml
---
tags: [pattern, python, agents, validated]
created: 2026-01-15
author: agent
---
```

**Reasoning:**
- `pattern` - Reusable solution (type, first)
- `python` - Python implementation (language)
- `agents` - Agent-related pattern (domain)
- `validated` - Pattern has been tested (status)

---

### Retrospective

**File:** `.agents/retros/sprint-5-auth-feature.md`

```yaml
---
tags: [retro, ci-cd, workflow]
created: 2026-01-15
author: agent
---
```

**Reasoning:**
- `retro` - Post-work reflection (type, first)
- `ci-cd` - CI/CD was major topic (domain)
- `workflow` - Workflow improvements discussed (domain)

---

## Common Errors

| Error | Example | Fix |
|-------|---------|-----|
| Missing type tag | `tags: [python, agents]` | Add `research`, `plan`, etc. first |
| Too many tags | `tags: [a, b, c, d, e, f, g]` | Limit to 3-5 most relevant |
| Too few tags | `tags: [research]` | Add at least one domain tag |
| Wrong tag order | `tags: [python, research]` | Type tag MUST be first |
| Inventing tags | `tags: [research, my-new-tag]` | Check vocabulary first |
| Uppercase | `tags: [Research, Python]` | Use lowercase only |

---

## Anti-Patterns

| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| Tag Explosion | Creating unique tags per doc | No consolidation, poor search | Reuse existing vocabulary |
| Over-tagging | 10+ tags per document | Dilutes meaning | 3-5 focused tags |
| Under-tagging | Only type tag | Can't filter by domain | Add 1-2 domain tags |
| Stale Tags | `deprecated` on active docs | Confusing signal | Remove or update |
| Inconsistent Format | Mix of `CI-CD`, `ci_cd`, `cicd` | Can't search reliably | Always `ci-cd` |

---

## AI Agent Guidelines

When AI agents tag documents:

| Guideline | Rationale |
|-----------|-----------|
| ALWAYS use type tag first | Consistent structure |
| ALWAYS check existing tags before inventing | Vocabulary control |
| ALWAYS include 1-2 domain tags | Enables filtering |
| NEVER exceed 5 tags | Keeps tags meaningful |
| NEVER use uppercase | Format consistency |
| PREFER specific over general | `kubernetes` over `infra` when K8s-specific |

---

## Validation

### Tag Validation Script

```bash
#!/usr/bin/env bash
# Validate tags in .agents/ documents

VALID_TYPE_TAGS="research plan pattern retro learning"
ERRORS=0

for file in .agents/**/*.md; do
    tags=$(grep -E "^tags:" "$file" 2>/dev/null | head -1)
    if [[ -z "$tags" ]]; then
        echo "WARN: $file missing tags"
        continue
    fi

    # Check first tag is a type tag
    first_tag=$(echo "$tags" | sed 's/tags: \[\([^,]*\).*/\1/')
    if ! echo "$VALID_TYPE_TAGS" | grep -qw "$first_tag"; then
        echo "ERROR: $file first tag '$first_tag' not a type tag"
        ((ERRORS++))
    fi
done

exit $ERRORS
```

---

## Summary

**Key Takeaways:**
1. Use 3-5 tags per document
2. First tag MUST be document type (`research`, `plan`, etc.)
3. Include 1-2 domain tags for filtering
4. Lowercase, hyphenated format only
5. Check existing vocabulary before creating tags
6. Use status tags sparingly for important signals
7. Follow decision tree for consistent selection
