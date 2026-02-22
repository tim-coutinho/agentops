# vibe-kit Plugin

**Comprehensive code validation and quality assurance for AI-assisted development**

The vibe-kit plugin provides advanced code validation capabilities through the `/vibe` skill and related validation components. It combines static analysis, complexity measurement, and multi-model judgment to ensure your code is production-ready.

## Overview

vibe-kit is a validation-focused plugin that answers one critical question: **"Is this code ready to ship?"**

It integrates two phases:
1. **Complexity analysis** - Identifies code quality hotspots
2. **Multi-model council validation** - Human-quality code review

## Components

### Core Skill

🔹 **`/vibe`** — The main validation skill

Performs comprehensive validation including:
- Cyclomatic complexity analysis
- Cognitive complexity measurement  
- Multi-judge code review
- Security pattern detection
- Architecture validation

### Validation Pipeline

1. **Prescan analysis** (fast, no LLM)
   - Phantom modifications detection
   - Hardcoded secrets scanning
   - SQL injection patterns
   - Cargo cult error handling

2. **Complexity metrics** (language-aware)
   - Python: `radon` (cyclomatic + maintainability index)
   - Go: `gocyclo` 
   - TypeScript/JS: Complexity analysis

3. **Council judgment** (multi-model validation)
   - 3+ independent judges review code
   - Focus areas: correctness, security, edge cases, quality
   - Optional: Spec-compliance validation

4. **Decision support**
   - PASS: Ready to ship
   - WARN: Review concerns before shipping
   - FAIL: Not production-ready

## Quick Start

```bash
# Validate recent changes
/vibe recent

# Validate specific directory
/vibe src/auth/

# Deep validation (3 judges + complexity)
/vibe --deep src/auth/

# Security audit with spec compliance
/vibe --preset=security-audit authentication/
```

## Installation & Configuration

### Prerequisites

```bash
# Python complexity analysis
pip install radon

# Go complexity analysis  
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

# Security scanning
brew install gitleaks
```

### Plugin Structure

```
skills/vibe/
├── SKILL.md               # Main skill documentation
├── references/
│   ├── patterns.md        # Detection patterns
│   ├── standards/
│   │   ├── python-standards.md
│   │   ├── go-standards.md
│   │   └── ...
├── scripts/
│   ├── validate.sh        # Automation scripts
│   └── prescan.sh         # Pre-scan tooling
└── README.md              # This file
```

## Usage Examples

### Validate Recent Commits

```bash
/vibe recent
```

Runs complexity analysis on last 3 commits, followed by multi-model council review.

### Validate Specific Path

```bash
/vibe src/validation-layer/
```

Validates all files in the validation-layer directory with default analysis.

### Deep Validation Mode

```bash
/vibe --deep api/controllers/
```

Uses 3 judges instead of 2 for more thorough review. Includes spec compliance if a relevant issue ID is found.

### Fast Inline Check

```bash
/vibe --quick utils/helpers.ts
```

Performs a single-agent inline validation without spawning sub-agents (60-80% faster).

## Pattern Detection

vibe-kit detects over 100 code quality, security, and architectural patterns:

**Prescan (Static):**
- P1-P10: Hardcoded secrets, SQL injection, dead code

**Semantic (LLM-powered):**
- QUAL-001-007: Dead code, poor naming, magic numbers
- SEC-001-010: Injection, auth bypass, data exposure  
- ARCH-001-010: Circular dependencies, god classes

**Complexity Metrics:**
- CMPLX-001-008: Cyclomatic, nesting depth, parameter count

See `references/patterns.md` for full pattern catalog.

## Integration

### With `/rpi` (Research-Plan-Implement)

```bash
/rpi "Implement rate limiting"
  ├─ Research phase
  ├─ Planning phase  
  ├─ Implementation phase
  └─ Vibe validation ← vibe-kit integration point
```

### With CI/CD

```yaml
# .github/workflows/validate.yml
name: Code Validation
on: [push, pull_request]

jobs:
  vibe-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: /vibe recent || (echo "❌ Vibe failed" && exit 1)
```

## Severity & Decision Making

vibe-kit uses a tiered severity system with clear action recommendations:

**Critical:**
- Block merging
- Security vulnerabilities
- Data loss potential

**High:**
- Fix before merge
- Significant quality gaps
- Architectural concerns

**Medium:**
- Follow-up issue creation
- Technical debt tracking

**Low:**
- Optional improvements
- Style preferences

## Validation Artifacts

All validation reports are stored in `.agents/council/` with format:
```
YYYYMMDDTHHMMSS-vibe-<target>.md
```

Sample report contents:
- Complexity scores and hotspots
- Judge verdicts and findings
- Shared concerns
- Recommendation (SHIP/FIX/REFACTOR)

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `COMPLEXITY SKIPPED: radon not installed` | `pip install radon` |
| `gocyclo not found` | `go install github.com/fzipp/gocyclo/cmd/gocyclo@latest` |
| No files detected | Specify target path explicitly: `/vibe src/auth/` |
| Constrain tests failing | Check constraint tests in `internal/constraints/` |

## Advanced Features

### Spec-Compliance Validation

When vibe detects a relevant issue ID or spec, it adds a spec-compliance judge:

```bash
/vibe --spec my-communication-flow-v1.2.2
```

- Compares implementation vs specification
- Validates API contracts
- Checks boundary conditions

### Developer Experience Review

With `PRODUCT.md` present, vibe automatically includes DX perspectives:

```bash
/vibe recent
# Auto-detects PRODUCT.md and adds:
# --perspectives="api-clarity,error-experience,discoverability"
```
  
### Adversarial Debate Mode

```bash
/vibe --debate src/core/
```

Two-round validation with opposing judges to surface hidden issues.

## Performance Optimization

For large codebases, use selective validation:

```bash
# Validate only changed files
/vibe $(git diff --name-only HEAD~1)

# Skip complexity analysis for quick feedback
/vibe --quick-no-complexity auth/middleware/

# Disable constraint tests for experimental work
/vibe --skip-constraints experimental/"```

## History & Evolution

vibe-kit evolved through several architectural phases:

- **v0.1**: Basic standalone complexity checker
- **v1.0**: Multi-model council integration
- **v1.3**: Spec compliance capabilities
- **v2.0**: Developer experience perspectives
- **Current**: Full validation ecosystem with 100+ patterns

## Relation to Other Skills

- **council** - Multi-model judgment engine
- **complexity** - Standalone complexity analysis
- **bug-hunt** - Specialized issue investigation
- **standards** - Language-specific coding conventions

## Contributing

Help improve vibe-kit by:

1. **Adding new patterns** - Submit pattern proposals
2. **Extending language support** - Add complexity analyzers
3. **Refining judge criteria** - Improve validation accuracy
4. **Adding preset configurations** - Domain-specific validation

## Support

For questions or issues:
- Report bugs: `bd create "vibe-kit issue: description"`
- Check patterns: Browse `skills/vibe/references/patterns.md`
- See examples: `skills/vibe/references/examples.md`

## License

MIT - Part of AgentOps ecosystem

---

> "Validation is not about perfection, but about confidence. vibe-kit gives you data-driven confidence in your code quality."