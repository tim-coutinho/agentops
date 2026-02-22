# Example Session: Software Development

**Profile**: software-dev
**Scenario**: Build a REST API endpoint with tests
**Duration**: ~2 hours

---

## Session Flow

### 1. Research Phase (20 min)

```
User: /research "REST API authentication patterns"
```

**Skills loaded**: research, languages

**Actions**:
- Explore existing auth patterns in codebase
- Find similar implementations
- Document constraints and requirements

**Output**: Research bundle with recommended approach

---

### 2. Planning Phase (15 min)

```
User: /plan "implement JWT auth endpoint"
```

**Skills loaded**: development, validation

**Actions**:
- Design API endpoint structure
- Define request/response schemas
- Plan middleware integration
- Specify file locations with line numbers

**Output**: Implementation plan with file:line specs

---

### 3. Implementation Phase (60 min)

```
User: /implement
```

**Skills loaded**: languages, development

**Actions**:
- Create auth middleware
- Implement JWT token generation
- Add route handlers
- Write error handling

**Skills used**:
1. `languages` - Python patterns
2. `development` - API architecture
3. `validation` - Tracer bullet deployment

---

### 4. Quality Phase (20 min)

```
User: Run code review and generate tests
```

**Skills loaded**: code-quality

**Actions**:
- Review implementation for security
- Check for edge cases
- Generate unit tests
- Generate integration tests

**Output**: Review feedback + test files

---

### 5. Commit Phase (5 min)

```
User: Commit these changes
```

**Actions**:
- Stage files
- Create semantic commit message
- Push to branch

---

## Skills Used Summary

| Skill | When | Purpose |
|-------|------|---------|
| research | Research | Understand existing patterns |
| languages | Plan, Implement | Python code patterns |
| development | Plan, Implement | API architecture |
| validation | Plan | Tracer bullet validation |
| code-quality | Quality | Review and testing |

---

## Session Outcome

- ✅ JWT auth endpoint implemented
- ✅ Middleware integrated
- ✅ Tests passing
- ✅ Code reviewed
- ✅ Committed with semantic message

**Time**: ~2 hours
