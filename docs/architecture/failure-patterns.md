# The 12 Failure Patterns

Reference guide for debugging and retrospectives. Based on [Vibe Coding](https://itrevolution.com/product/vibe-coding-book/) by Gene Kim & Steve Yegge.

---

## Overview

Failure patterns are recurring ways that AI-assisted development goes wrong. Recognizing them early prevents compounding mistakes and wasted time.

| Category | Patterns | Timeframe |
|----------|----------|-----------|
| **Inner Loop** | Fix Spiral, Confident Hallucination, Context Amnesia, Tests Passing Lie | Seconds-Minutes |
| **Middle Loop** | Eldritch Horror, Silent Deletion, Zombie Resurrection, Gold Plating | Hours-Days |
| **Outer Loop** | Cargo Cult, Premature Abstraction, Security Theater, Documentation Mirage | Days-Weeks |

---

## Inner Loop Patterns

### 1. The Fix Spiral

**Description:** The AI makes a fix that breaks something else, then fixes that break which causes another issue, creating a cascading chain of changes that circles back without resolving the original problem. Each iteration adds complexity without progress.

**Symptoms:**
- More than 3 fix attempts without convergence
- Changes oscillating between two states
- Error messages changing but not disappearing
- Growing scope of modified files
- "This should work" appearing in explanations

**Prevention:**
- Set a hard limit: 3 attempts then STOP
- Before each fix, state explicit prediction of outcome
- Use tracer tests to validate assumptions before fixing
- Keep changes minimal and atomic
- Checkpoint working state before each attempt

**Recovery:**
1. Stop immediately - do not make another change
2. Revert to last known working state (git stash or git checkout)
3. Validate all assumptions about the problem
4. Create a minimal reproduction case
5. Fix root cause, not symptoms

---

### 2. The Confident Hallucination

**Description:** The AI generates plausible-sounding but factually incorrect information about APIs, libraries, syntax, or behavior. The confidence of the delivery makes it easy to trust without verification.

**Symptoms:**
- Code references non-existent methods or parameters
- API usage that "looks right" but fails at runtime
- Documentation citations that don't match actual docs
- Version-specific features applied to wrong versions
- Overly specific technical claims without evidence

**Prevention:**
- Verify all API calls against actual documentation
- Test code in isolation before integration
- Use "I don't know" as a valid response
- Check import statements actually resolve
- Run type checkers and linters early

**Recovery:**
1. Do not trust any related code from the same generation
2. Read actual documentation for the API in question
3. Create minimal test to verify correct behavior
4. Replace hallucinated code with verified implementation
5. Note the hallucination source for future reference

---

### 3. The Context Amnesia

**Description:** As the context window fills, the AI loses track of earlier constraints, requirements, or decisions. Work from the beginning of a session gets overwritten by work from the end, causing regressions and contradictions.

**Symptoms:**
- Reintroducing previously fixed bugs
- Contradicting earlier decisions without explanation
- Forgetting project-specific conventions
- Losing track of which files were modified
- Repeating work that was already completed

**Prevention:**
- Keep context utilization below 40%
- Save progress frequently to files (progress.json, bundles)
- Use explicit checkpoints: "Completed X, moving to Y"
- Start fresh sessions for distinct work items
- Front-load critical constraints in prompts

**Recovery:**
1. Stop and review what was accomplished
2. Save current state to persistent files
3. Start fresh context with saved state loaded
4. Explicitly re-state all constraints and requirements
5. Continue from checkpoint, not from memory

---

### 4. The Tests Passing Lie

**Description:** Tests pass but the code doesn't actually work. The tests may be too narrow, test the wrong thing, mock away the actual behavior, or the AI may have modified tests to pass rather than fixing the code.

**Symptoms:**
- Green test suite but broken functionality
- Tests that test mocks instead of real behavior
- Coverage looks good but edge cases fail
- Tests modified in same PR as code they test
- "All tests pass" but users report bugs

**Prevention:**
- Run tests yourself; don't trust reported results
- Separate test changes from code changes
- Use integration tests with real dependencies
- Check test assertions actually verify behavior
- Manual smoke test after test suite passes

**Recovery:**
1. Run manual end-to-end verification
2. Identify which tests are actually validating behavior
3. Add missing tests that catch the real bug
4. Fix code without modifying test assertions
5. Review test coverage for gaps

---

## Middle Loop Patterns

### 5. The Eldritch Horror

**Description:** AI-generated code becomes incomprehensible - functions spanning hundreds of lines, deeply nested logic, unclear naming, and tangled dependencies. The code works but no one can maintain or debug it.

**Symptoms:**
- Functions exceeding 200 lines
- Nesting depth beyond 4 levels
- Variable names like `temp2`, `data3`, `result_final_v2`
- Circular dependencies between modules
- Comments that don't match code behavior

**Prevention:**
- Enforce hard limits: <200 lines per function
- Review generated code before accepting
- Require meaningful names (nouns for data, verbs for functions)
- Break work into small, composable units
- Use explicit interfaces between components

**Recovery:**
1. Do not add more code to the horror
2. Write characterization tests to capture current behavior
3. Extract functions methodically, one at a time
4. Rename variables to reflect actual purpose
5. Document the intended architecture

---

### 6. The Silent Deletion

**Description:** The AI removes code that appears unused but is actually necessary - handling edge cases, supporting legacy integrations, or providing fallback behavior. The deletion isn't noticed until production fails.

**Symptoms:**
- "Cleanup" commits that remove "dead code"
- Features that worked yesterday now fail
- Error handling mysteriously missing
- Integration tests that used to exist are gone
- Comments about "why" deleted along with code

**Prevention:**
- Never delete code without understanding why it exists
- Check git history before removal
- Trace all references including dynamic calls
- Get human approval for any deletion
- Keep deleted code in comments initially

**Recovery:**
1. Use git to identify what was deleted and when
2. Restore deleted code with `git checkout` or cherry-pick
3. Add tests that would have caught the deletion
4. Document why the code exists (add comments)
5. Flag similar code as "do not delete" in review

---

### 7. The Zombie Resurrection

**Description:** Previously fixed bugs return because the AI regenerates similar code without the fix, or reverts changes during refactoring. The same issues keep coming back from the dead.

**Symptoms:**
- Bug reports for issues marked "fixed"
- Same error appearing in different code paths
- Fixes getting lost during refactoring
- Pattern of re-introducing the same mistakes
- "I thought we fixed this" conversations

**Prevention:**
- Add regression tests for every bug fix
- Document root cause, not just fix
- Use automated checks for known anti-patterns
- Review diffs for accidental reversions
- Keep a "lessons learned" file per project

**Recovery:**
1. Verify the fix still exists in codebase
2. If reverted, restore from git history
3. Add regression test that fails without fix
4. Document in prominent location why fix is needed
5. Add CI check if pattern is automatable

---

### 8. The Gold Plating

**Description:** The AI adds unrequested features, extra error handling, additional configurability, or "improvements" beyond what was asked. The extra work introduces bugs, delays delivery, and increases maintenance burden.

**Symptoms:**
- PR larger than expected for the task
- New config options no one asked for
- "While I was here, I also..." explanations
- Abstraction layers for single use cases
- Extra error handling for impossible states

**Prevention:**
- Define explicit scope before starting
- Review against original requirements
- Reject changes outside stated scope
- Ask "was this requested?" for each addition
- Prefer boring, obvious solutions

**Recovery:**
1. Identify what was actually requested
2. Revert unrequested additions
3. Keep only essential changes
4. If additions are valuable, make separate PR
5. Adjust prompts to emphasize minimalism

---

## Outer Loop Patterns

### 9. The Cargo Cult

**Description:** The AI copies patterns from examples without understanding why they work. The copied code may be inappropriate for the context, outdated, or solving a different problem entirely.

**Symptoms:**
- Copy-pasted code with irrelevant portions
- Patterns from different frameworks mixed together
- "Best practices" applied where they don't fit
- Configuration copied without understanding
- Stack Overflow answers used verbatim

**Prevention:**
- Ask "why does this pattern exist?" for each adoption
- Verify example matches your context
- Test copied code in isolation first
- Adapt patterns to local conventions
- Trace examples to their source

**Recovery:**
1. Identify which copied portions are actually needed
2. Remove cargo cult code that doesn't apply
3. Rewrite to match actual requirements
4. Document why remaining patterns are appropriate
5. Add tests that verify the pattern's purpose

---

### 10. The Premature Abstraction

**Description:** The AI creates generic abstractions before concrete use cases exist. The abstractions don't match actual needs, leading to awkward workarounds or complete rewrites when real requirements emerge.

**Symptoms:**
- Generic interfaces with one implementation
- Factory patterns for single classes
- Configuration for cases that don't exist
- "Future-proofing" that never gets used
- Abstractions that make simple things complex

**Prevention:**
- Require at least 3 concrete use cases before abstracting
- Write concrete implementations first
- Extract abstractions only when duplication appears
- Prefer duplication over wrong abstraction
- Ask "what problem does this abstraction solve?"

**Recovery:**
1. Identify actual current use cases
2. Inline the abstraction into concrete usage
3. Remove unused generic capabilities
4. Wait for real patterns to emerge
5. Abstract only when duplication becomes painful

---

### 11. The Security Theater

**Description:** Code appears secure but isn't - validation that doesn't cover edge cases, encryption with hardcoded keys, authentication that can be bypassed, or sanitization that misses injection vectors.

**Symptoms:**
- Security measures easily circumvented
- Validation on client but not server
- Hardcoded credentials or keys
- "Security by obscurity" approaches
- Logging sensitive data

**Prevention:**
- Use established security libraries, not custom code
- Security review by qualified humans
- Penetration testing for critical paths
- Threat modeling before implementation
- Static analysis for common vulnerabilities

**Recovery:**
1. Conduct proper security audit
2. Identify actual attack vectors
3. Replace theater with real protection
4. Use industry-standard approaches
5. Add security regression tests

---

### 12. The Documentation Mirage

**Description:** Documentation exists but doesn't match reality - outdated READMEs, incorrect API docs, comments that describe what code used to do, or auto-generated docs that miss critical details.

**Symptoms:**
- Following docs leads to errors
- Comments contradict adjacent code
- Examples that don't compile
- API docs missing required parameters
- Setup instructions that don't work

**Prevention:**
- Treat docs as code: test them
- Update docs in same PR as code changes
- Use executable documentation (doctests, notebooks)
- Review docs during code review
- Version docs with code

**Recovery:**
1. Test documentation by following it literally
2. Identify all discrepancies
3. Update docs to match current behavior
4. Add CI checks for doc accuracy where possible
5. Remove documentation that can't be maintained

---

## Quick Reference Card

| # | Pattern | Key Symptom | First Action |
|---|---------|-------------|--------------|
| 1 | Fix Spiral | >3 attempts | STOP, revert |
| 2 | Confident Hallucination | Non-existent APIs | Verify docs |
| 3 | Context Amnesia | Forgotten constraints | Save state |
| 4 | Tests Passing Lie | Green but broken | Manual test |
| 5 | Eldritch Horror | >200 line functions | Extract/refactor |
| 6 | Silent Deletion | Missing code | Check git history |
| 7 | Zombie Resurrection | Bugs return | Add regression test |
| 8 | Gold Plating | Unrequested features | Revert extras |
| 9 | Cargo Cult | Copied patterns | Understand why |
| 10 | Premature Abstraction | Generic w/ one use | Inline |
| 11 | Security Theater | Bypassable security | Audit |
| 12 | Documentation Mirage | Docs don't work | Test docs |

---

## Pattern Frequency Tracking

Use this table in retrospectives to track which patterns hit most often:

| Pattern | Occurrences | Total Hours Lost | Prevention Implemented |
|---------|-------------|------------------|----------------------|
| Fix Spiral | | | |
| Confident Hallucination | | | |
| Context Amnesia | | | |
| Tests Passing Lie | | | |
| Eldritch Horror | | | |
| Silent Deletion | | | |
| Zombie Resurrection | | | |
| Gold Plating | | | |
| Cargo Cult | | | |
| Premature Abstraction | | | |
| Security Theater | | | |
| Documentation Mirage | | | |

---

## See Also

- `/retro` - Session retrospective command
- `/vibe-validate` - Semantic code validation
- `/vibe-prescan` - Static pre-scan for patterns
- [Vibe Coding Book](https://itrevolution.com/product/vibe-coding-book/)
