---
id: TASK-653
planId: PLAN-079
title: 'Standalone LLM Gateway & OpenAI-compatible proxy (nexus-router)'
role: gateway
status: done
createdAt: 2026-11-19
---

# TASK-653 — Standalone LLM Gateway & OpenAI-compatible proxy (nexus-router)

Source: PLAN-079 / GitHub Issue #144.

Goals:

1. Provide a standalone binary `cmd/nexus-router` that wraps model routing, cooldown tracking, and token budgets.
2. Serve an OpenAI-compatible reverse proxy (`/v1/chat/completions`) on `nexus-router serve`.
3. Support model ladder failover across API keys and subscription CLIs (Claude, Copilot, Agy).
4. Provide CLI prompt runner (`nexus-router prompt --role <role> <prompt>`).
5. Ensure 100% statement and doc-comment test coverage.
