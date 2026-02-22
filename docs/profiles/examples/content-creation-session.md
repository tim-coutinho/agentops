# Example Session: Content Creation

**Profile**: content-creation
**Scenario**: Extract pattern from production code, create documentation
**Duration**: ~3 hours

---

## Session Flow

### 1. Research Phase (45 min)

```
User: /research "authentication patterns in our codebase"
```

**Skills loaded**: research

**Actions**:
- Scan codebase for auth implementations
- Trace git history for evolution
- Find existing documentation
- Identify common patterns

**Findings**:
- 3 different auth approaches used
- JWT most common (12 services)
- OAuth for external integrations
- API keys for internal services

---

### 2. Pattern Extraction (30 min)

```
User: Extract the JWT auth pattern
```

**Skills loaded**: research, meta

**Actions**:
- Identify core components
- Document decision rationale
- Capture variations
- Note failure modes

**Output**: Pattern specification

---

### 3. Documentation Writing (60 min)

```
User: Create documentation for this pattern
```

**Skills loaded**: documentation

**Actions**:
- Write pattern overview
- Create usage examples
- Document configuration options
- Add troubleshooting section

**Outputs**:
1. `docs/patterns/jwt-authentication.md`
2. `docs/how-to/implement-jwt-auth.md`
3. `docs/reference/auth-api.md`

---

### 4. Quality Review (20 min)

```
User: Review docs for Diátaxis compliance
```

**Skills loaded**: documentation

**Actions**:
- Check document placement
- Verify content type matches location
- Ensure cross-references work
- Validate completeness

**Feedback**: Minor adjustments to how-to guide

---

### 5. Retrospective (25 min)

```
User: /retro "pattern extraction session"
```

**Skills loaded**: meta

**Actions**:
- Capture what worked
- Document challenges
- Extract reusable insights
- Update institutional memory

**Output**: Retrospective in `.agents/retros/`

---

## Skills Used Summary

| Skill | When | Purpose |
|-------|------|---------|
| research | Research, Extract | Find and analyze code |
| meta | Extract, Retro | Pattern synthesis, learning |
| documentation | Write, Review | Create and audit docs |

---

## Session Outcome

- ✅ Pattern extracted and documented
- ✅ 3 documentation files created
- ✅ Diátaxis compliant
- ✅ Retrospective captured

**Time**: ~3 hours (pattern now reusable for team)

---

## Artifacts Created

```
docs/
├── patterns/
│   └── jwt-authentication.md     # Pattern overview
├── how-to/
│   └── implement-jwt-auth.md     # Step-by-step guide
└── reference/
    └── auth-api.md               # API documentation

.agents/
├── research/
│   └── 2025-12-30-auth-patterns.md
├── patterns/
│   └── jwt-auth-pattern.md
└── retros/
    └── 2025-12-30-pattern-extraction.md
```
