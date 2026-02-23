Discovery summary
- Failing goal: evolve-cycle-logging
- Root cause: cycle-history entries with `target` but missing required `goal_id` field.
- Scope: normalize historical entries and enforce goal_id-only logging in this session.
