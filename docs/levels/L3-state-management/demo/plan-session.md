# Demo: /plan Session

A session showing goal decomposition into tracked issues.

---

## Before

- Beads initialized (`bd init` run previously)
- Goal: Add email notifications to the app
- No existing issues for this feature

---

## Session

```
You: /plan "add email notifications for order updates"

Claude: I'll research the notification requirements and create a plan.

[Research] Looking at existing order system...

[Read] src/orders/processor.ts
[Read] src/notifications/ (doesn't exist)
[Grep] "email|notify|send" in src/

**Analysis:**

Orders are processed in processor.ts. No notification system exists.
Need to:
1. Set up email service
2. Add notification triggers to order events
3. Create email templates
4. Add user preference handling

**Creating issues:**

$ bd create --title "Set up email service (SendGrid)" --type task --priority 1
Created: agentops-a1b

$ bd create --title "Add notification triggers to order processor" --type task --priority 1
Created: agentops-c2d

$ bd create --title "Create order email templates" --type task --priority 2
Created: agentops-e3f

$ bd create --title "Add email preference settings" --type feature --priority 2
Created: agentops-g4h

$ bd create --title "Write notification tests" --type task --priority 3
Created: agentops-i5j

**Setting dependencies:**

$ bd dep add agentops-c2d agentops-a1b
(triggers depend on email service)

$ bd dep add agentops-e3f agentops-a1b
(templates depend on email service)

$ bd dep add agentops-i5j agentops-c2d
$ bd dep add agentops-i5j agentops-e3f
(tests depend on triggers and templates)

**Plan saved to:** .agents/plans/2025-01-15-email-notifications.md

**Ready to work:**
$ bd ready
1. [P1] agentops-a1b: Set up email service (SendGrid)
2. [P2] agentops-g4h: Add email preference settings

Start with: `/implement agentops-a1b`
```

---

## After

```
.beads/
└── beads.db                    # Issues stored here

.agents/
└── plans/
    └── 2025-01-15-email-notifications.md
```

Issues created:
- 2 ready (no blockers)
- 3 blocked (have dependencies)

---

## What You Learned

1. `/plan` researches before creating issues
2. Issues get IDs like `agentops-xyz`
3. Dependencies control execution order
4. `bd ready` shows what can be worked now
5. Plan document captures the reasoning
