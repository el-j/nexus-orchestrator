---
id: TASK-635
planId: PLAN-077
title: 'Go coverage to 100 %: inbound adapters and bootstrap'
role: testing
status: done
createdAt: 2026-10-31
---

# TASK-635 — Go coverage to 100 %: inbound adapters and bootstrap

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

httpapi server.go 13, handlers_providers 5, brain_handlers 4, dashboard 4, handlers_config 4, hub 4;
mcp tools_tasks 5, server 4, tools_wait 3; cli brain.go 6; bootstrap/runtime.go 13; connect.go 6.

Done: the files the merged profile listed for the inbound adapters, bootstrap and connect are covered (`httpapi` and `connect` at 100 % on their own tests). New tests: `coverage_gaps_test.go` and `coverage_gaps_internal_test.go` (httpapi), `coverage_gaps_test.go` (mcp), `brain_failures_test.go` (cli), `runtime_failures_test.go` (bootstrap), additions to `connect_test.go`. Bugs found and fixed on the way: `StartServer`/`StartServerWith` returned as soon as shutdown began, before the in-flight requests had drained (the shared `runHTTP` now waits for the drain); the server failure channel held two errors although it is meant to keep the first (capacity 1); `RunHTTP`-style duplication between `StartServer` and `StartServerWith` is gone. Branches that could never run were removed: the empty-id guards of the brain handlers (the route never matches an empty id) and the content-over-limit case (the 1 MiB body limit fires first), `maskAPIKey("")`, the nonce error (`crypto/rand.Read` cannot fail), the empty-token case in the MCP auth check, `requiresAuth` + `isAuthorized` merged into `authorized`, the four copies of the event broadcast in `Hub`, `MkdirAll` in `connect` (the config folder always exists when an agent counts as installed), the `MarshalIndent` error. Failures injectable for tests: the MCP shutdown grace, the workspace watcher, the final file move in `connect`.
