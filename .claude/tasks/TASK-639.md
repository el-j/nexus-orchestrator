---
id: TASK-639
planId: PLAN-077
title: 'Split the oversized functions'
role: refactor
status: done
createdAt: 2026-10-31
---

# TASK-639 — Split the oversized functions

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

handleToolCall (cyclomatic 48), buildOnboarding (40), migrate (187 lines, table-driven steps),
buildHowToDoc (135), the four *ToolDefs (123-125, one file per tool group or data), runAgentLoop,
Handler (route table per area), executeWithFallbackChain, and the five near-duplicate ChatStream
implementations (shared SSE reader). Enable funlen and gocyclo in .golangci.yml at the current
limits so none grows back.

Done: `funlen` (100 lines, 60 statements) and `gocyclo` (25) are enabled in `.golangci.yml` (tests and the three declarative `*ToolDefs` catalogues excluded), and nothing exceeds them. `handleToolCall` is a table (`tools_dispatch.go`), the REST routes are one function per area (`routes.go`), `migrate` runs data (`schema.go`, `schema_columns.go`), `buildOnboarding`, `writeAndVerifyTaskOutput` (a `healer`), `runAgentLoop`, `executeWithFallbackChain`, `resolveProviderWithFallback`, `linkPlanFiles`, `scanDir`, `probeClaudeSubAgentsDir`, `classifySection`, `looksLikePlanMarkdown`, `buildHowToDoc` and `ChatStream` are split into named steps. Not done: a shared SSE reader for the five `ChatStream` implementations, which differ in framing; none is over the limits.
