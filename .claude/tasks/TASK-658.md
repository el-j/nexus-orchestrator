---
id: TASK-658
planId: PLAN-080
title: 'The OpenAI-compatible gateway inside the daemon, with brain context'
role: architecture
status: todo
createdAt: 2026-11-24
---

# TASK-658 — The OpenAI-compatible gateway inside the daemon, with brain context

Mount the gateway (`/v1/chat/completions`, `/v1/models`) on the daemon's REST port behind the same token and origin guards, route through the daemon's providers (subscription clients included), record usage in the ledger and, when the request names a project (`X-Nexus-Project` header), prepend the project's onboarding and focused context from the brain. Any OpenAI-compatible tool (Cursor, Cline, Continue, Aider) can then use Nexus as its model endpoint. Tests first; e2e with a fake provider; surface matrix, API reference, changelog.
