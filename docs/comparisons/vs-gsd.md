# AgentOps vs GSD (Get Shit Done)

> **GSD** is a lightweight meta-prompting and context engineering system for Claude Code, focused on shipping fast without enterprise overhead.

---

## At a Glance

| Aspect | GSD | AgentOps |
|--------|-----|----------|
| **Philosophy** | "Ship fast, no BS" | "Knowledge compounds over time" |
| **Core strength** | Lightweight, minimal overhead | Cross-session memory, learning |
| **GitHub** | glittercowboy/get-shit-done | boshu2/agentops |
| **Primary use** | Rapid prototyping, shipping | Ongoing codebase work |

---

## What GSD Does Well

### 1. Minimal Overhead
No enterprise complexity. Just structure to ship consistently:

```
GSD Loop:
  discuss → plan → execute → verify → (repeat until done)
```

### 2. Fast Project Setup
`/gsd:new-project` gets you started immediately with sensible defaults stored in `.planning/config.json`.

### 3. Model Flexibility
Configure which Claude model each phase uses. Balance quality vs token spend per task.

### 4. Human Input at Each Phase
- **Discuss:** Get your input
- **Plan:** Proper research
- **Execute:** Clean implementation
- **Verify:** Human verification

---

## Where GSD Falls Short

### No Persistence

```
┌─────────────────────────────────────────────────────────────────┐
│                         GSD                                     │
│                                                                 │
│  Session 1: discuss → plan → execute → verify → Done           │
│                                                    ↓            │
│                                               (session ends)    │
│                                                    ↓            │
│                                               (all gone)        │
│                                                                 │
│  Session 2: Start from scratch                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      AGENTOPS                                   │
│                                                                 │
│  Session 1: research → plan → crank → post-mortem              │
│                                              ↓                  │
│                                      (learnings extracted)      │
│                                              ↓                  │
│                                      (stored in .agents/)       │
│                                                                 │
│  Session 2: (inject prior knowledge) → better starting point   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**GSD is ephemeral by design.** Great for one-off tasks, not for repeated work.

### No Failure Prevention

GSD validates at the "verify" step — after implementation.

```
GSD:
  discuss → plan → execute → verify (catch issues here)

AgentOps:
  research → plan → pre-mortem (catch issues here) → crank → post-mortem
```

### Limited Validation

GSD's "verify" phase is human verification. No systematic multi-aspect validation.

| Validation | GSD | AgentOps |
|------------|:---:|:--------:|
| Human review | ✅ | ✅ |
| Semantic correctness | ❌ | ✅ |
| Security review | ❌ | ✅ |
| Architecture analysis | ❌ | ✅ |
| Complexity metrics | ❌ | ✅ |
| AI slop detection | ❌ | ✅ |

---

## Feature Comparison

| Feature | GSD | AgentOps | Winner |
|---------|:---:|:--------:|:------:|
| Setup speed | ✅ Instant | ⚠️ Requires init | GSD |
| Minimal overhead | ✅ Lightweight | ⚠️ More structure | GSD |
| Model flexibility | ✅ Per-phase config | ⚠️ Standard | GSD |
| Human-in-loop | ✅ Each phase | ✅ 4 gates | Tie |
| **Cross-session memory** | ❌ None | ✅ Git-persisted | **AgentOps** |
| **Knowledge compounding** | ❌ No | ✅ Escape velocity | **AgentOps** |
| **Pre-mortem simulation** | ❌ No | ✅ 10 failure modes | **AgentOps** |
| **8-aspect validation** | ❌ Human only | ✅ Semantic validator | **AgentOps** |
| **Issue tracking** | ❌ No | ✅ Beads integration | **AgentOps** |

---

## Workflow Comparison

### GSD Workflow

```
/gsd:new-project     →  Set up project config
         ↓
discuss              →  Get human input on requirements
         ↓
plan                 →  Research and create plan
         ↓
execute              →  Implement the plan
         ↓
verify               →  Human verification
         ↓
       Done          →  (nothing persists)
```

### AgentOps Workflow

```
/research     →  Explore codebase + inject prior knowledge
     ↓
/plan         →  Break into tracked issues
     ↓
/pre-mortem   →  Simulate 10 failure modes
     ↓
/crank        →  Implement → validate → commit
     ↓
/post-mortem  →  Validate + extract learnings
     ↓
  Done        →  (learnings persist for next session)
```

---

## Overhead Comparison

```
                    SETUP TIME              ONGOING OVERHEAD
                    ══════════              ════════════════

GSD:                ░░░░░░░░░░░░░░░░        ░░░░░░░░░░░░░░░░
                    (instant)               (minimal)

AgentOps:           ████████░░░░░░░░        ████████░░░░░░░░
                    (init + hooks)          (moderate)


                    SESSION VALUE           LONG-TERM VALUE
                    ═════════════           ═══════════════

GSD:                ████████████████        ░░░░░░░░░░░░░░░░
                    (fast shipping)         (no compounding)

AgentOps:           ████████████░░░░        ████████████████
                    (structured)            (compounds)
```

**Trade-off:** GSD optimizes for speed now. AgentOps optimizes for value over time.

---

## Use Case Fit

### GSD is Perfect For

| Use Case | Why |
|----------|-----|
| Hackathons | Speed is everything |
| Prototypes | Ship and throw away |
| One-off scripts | No need for memory |
| Side projects | Low overhead |
| Learning | Fast iteration |

### AgentOps is Perfect For

| Use Case | Why |
|----------|-----|
| Production codebases | Memory matters |
| Long-running projects | Knowledge compounds |
| Team codebases | Capture institutional knowledge |
| Complex systems | Failure prevention critical |
| Repeated maintenance | Same bugs, should remember |

---

## When to Choose GSD

- You're **prototyping** or building throwaway code
- You want **minimal setup** and overhead
- Sessions are **independent** (no prior context needed)
- You're doing a **hackathon** or **side project**
- **Speed** is more important than learning

## When to Choose AgentOps

- You work on the **same codebase** over many sessions
- You want your agent to **remember past work**
- You want **failure prevention** before building
- You want **systematic validation** beyond human review
- You value **compounding knowledge** over time

---

## Can They Work Together?

**Not really.** GSD's philosophy is "lightweight, no overhead." AgentOps adds structure and persistence — the opposite approach.

```
GSD:        "Ship fast, forget fast"
AgentOps:   "Ship smart, remember forever"
```

Choose based on your needs:
- **One-off work:** GSD
- **Ongoing work:** AgentOps

---

## The Bottom Line

| Dimension | GSD | AgentOps |
|-----------|-----|----------|
| **Philosophy** | Ship fast | Learn fast |
| **Overhead** | Minimal | Moderate |
| **Persistence** | None | Git-tracked |
| **Validation** | Human | 8-aspect |
| **Best for** | Prototypes | Production |

**GSD gets you to v0.1 fast.**
**AgentOps gets you to v10.0 smarter.**

---

## The Honest Assessment

**If you're building something you'll throw away:** Use GSD. It's lighter, faster, and doesn't pretend you need memory.

**If you're building something you'll maintain:** Use AgentOps. The overhead pays off by session 10.

```
Session 1:   GSD faster
Session 5:   About equal
Session 10:  AgentOps ahead
Session 50:  AgentOps way ahead
Session 100: AgentOps is domain expert, GSD still at day 1
```

---

<div align="center">

[← vs. SDD](vs-sdd.md) · [Back to Comparisons](README.md)

</div>
