---
id: learn-2026-test-001
type: learning
created_at: "2026-01-15T10:00:00Z"
category: testing
confidence: high
tags: [testing, fixtures]
---

# Context Cancellation Pattern

When testing CLI commands that read from .agents/, always seed a complete fixture directory.
Incomplete fixtures cause false negatives in integration tests.
