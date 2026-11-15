---
id: TASK-639
planId: PLAN-077
title: 'Split the oversized functions'
role: refactor
status: todo
createdAt: 2026-10-31
---

# TASK-639 — Split the oversized functions

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

handleToolCall (cyclomatic 48), buildOnboarding (40), migrate (187 lines, table-driven steps),
buildHowToDoc (135), the four *ToolDefs (123-125, one file per tool group or data), runAgentLoop,
Handler (route table per area), executeWithFallbackChain, and the five near-duplicate ChatStream
implementations (shared SSE reader). Enable funlen and gocyclo in .golangci.yml at the current
limits so none grows back.
