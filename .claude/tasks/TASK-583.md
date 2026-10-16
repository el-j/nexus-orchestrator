---
id: TASK-583
planId: PLAN-074
title: 'Claude CLI driver: stream-json, usage, rate-limit event'
role: coder
wave: 1
status: todo
createdAt: 2026-10-16
dependsOn: [TASK-581]
---

# TASK-583 — Claude CLI driver: stream-json, usage, rate-limit event

In `llm_cli` add a driver for `claude`: `--output-format stream-json --include-partial-messages --tools "" --no-session-persistence --disable-slash-commands`; parse deltas, final text, usage, cost, `rate_limit_event` (status, rateLimitType, utilization, resetsAt), `terminal_reason`; drop `ANTHROPIC_API_KEY` from the child env; typed errors from 581. Fixture: a recorded real output (docs/subscription-bridges.md).

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
