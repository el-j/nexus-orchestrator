---
id: TASK-658
planId: PLAN-080
title: 'The OpenAI-compatible gateway inside the daemon, with brain context'
role: architecture
status: done
createdAt: 2026-11-24
---

# TASK-658 — The OpenAI-compatible gateway inside the daemon, with brain context

Mount the gateway (`/v1/chat/completions`, `/v1/models`) on the daemon's REST port behind the same token and origin guards, route through the daemon's providers (subscription clients included), record usage in the ledger and, when the request names a project (`X-Nexus-Project` header), prepend the project's onboarding and focused context from the brain. Any OpenAI-compatible tool (Cursor, Cline, Continue, Aider) can then use Nexus as its model endpoint. Tests first; e2e with a fake provider; surface matrix, API reference, changelog.

Done: `gateway.Config` takes a `ContextProvider` and a `UsageRecorder`; `BrainContext` puts the onboarding summary and the entries that answer the last user message in front of the conversation as a system message, `X-Nexus-Project` names the project, answered calls enter the usage ledger. `httpapi.Server.WithGateway` / `Extras.Gateway` mount `Routes()` under `/v1` and the API token guards `/v1`; `bootstrap` wires the daemon's own discovery service, brain and ledger. Unit tests keep every package at 100 %; e2e (`gateway_test.go`) runs a chat with and without a project and the token guard against the real daemon; surface matrix row, API reference and changelog updated.
