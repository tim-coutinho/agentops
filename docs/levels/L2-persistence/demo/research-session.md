# Demo: /research with Persistence

A session showing research that persists to `.agents/`.

---

## Before

- Fresh Claude Code session
- `.agents/` directory exists (or will be created)
- Previous research documents may exist

---

## Session

```
You: /research "how does the payment system work"

Claude: I'll explore the payment system and save findings.

[Glob] **/payment*.{ts,py}
Found: src/payments/processor.ts, src/payments/stripe.ts, tests/payments.test.ts

[Grep] "charge|refund|subscription" in src/payments/
Found 18 matches

[Read] src/payments/processor.ts (89 lines)
[Read] src/payments/stripe.ts (124 lines)

**Analysis:**

The payment system uses Stripe as the backend:

1. **Entry point**: `processor.ts` - orchestrates all payment operations
2. **Stripe integration**: `stripe.ts` - API wrapper with retry logic
3. **Supported operations**: charge, refund, subscription management

Key flows:
- One-time payments: processor.charge() → stripe.createPaymentIntent()
- Subscriptions: processor.subscribe() → stripe.createSubscription()
- Refunds: processor.refund() → stripe.createRefund() (partial supported)

**Writing to:** .agents/research/2025-01-15-payment-system.md

[Write] .agents/research/2025-01-15-payment-system.md

✓ Research saved. You can load this in future sessions.
```

---

## After

```
.agents/
└── research/
    └── 2025-01-15-payment-system.md   # <- New file
```

The research document contains:
- Full analysis with file references
- Code snippets of key functions
- Diagrams or flow descriptions
- Questions for follow-up

---

## What You Learned

1. L2 research creates persistent documents
2. Documents survive session clears and context limits
3. Future sessions can load research: `read .agents/research/...`
4. Research accumulates as institutional memory
