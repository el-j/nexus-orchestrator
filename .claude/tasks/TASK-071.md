---
id: TASK-071
title: Process-level E2E smoke test script (build+run daemon)
role: verify
planId: PLAN-008
status: todo
dependencies: [TASK-068, TASK-069, TASK-070]
createdAt: 2026-03-10T14:00:00.000Z
---

## Context
The old `scripts/dogfood-plan002.sh` is PLAN-002-specific and stale. We need a **generic, reusable** E2E smoke test script that builds the daemon binary, starts it as a real process, exercises the HTTP API + MCP endpoint, and verifies results. This script can be run by CI or developers to prove the system works as a real binary, not just in-process httptest.

## Files to Read
- `scripts/dogfood-plan002.sh` (existing script to replace)
- `cmd/nexus-daemon/main.go` (daemon entry point)
- `internal/adapters/inbound/httpapi/server.go` (HTTP endpoints)
- `internal/adapters/inbound/mcp/server.go` (MCP endpoint)

## Implementation Steps

1. Create `scripts/e2e-smoke.sh` (NOT replacing the old script — keep it for reference).
2. The script must:
   a. Build `nexus-daemon` to a temp path.
   b. Start the daemon with a temp DB (`NEXUS_DB_PATH=/tmp/nexus-e2e-$$.db`), binding to ephemeral or known ports (`:19999` for HTTP, `:19998` for MCP to avoid conflicts).
   c. Wait for health endpoint: `curl -sf http://127.0.0.1:19999/api/health`.
   d. **Test 1 — Health**: Verify `/api/health` returns `{"status":"ok","service":"nexus-orchestrator"}`.
   e. **Test 2 — Providers**: `GET /api/providers` — verify it returns a JSON array (may be empty if no LLM running, which is OK for CI).
   f. **Test 3 — Submit task**: `POST /api/tasks` with a test task body → verify 201 + `task_id` in response.
   g. **Test 4 — Get task**: `GET /api/tasks/{id}` → verify task exists with status QUEUED or PROCESSING (if no LLM, it may stay QUEUED or go to NO_PROVIDER).
   h. **Test 5 — List tasks**: `GET /api/tasks` → verify JSON array contains the submitted task.
   i. **Test 6 — Cancel task**: `DELETE /api/tasks/{id}` → verify 204.
   j. **Test 7 — MCP health**: `POST http://127.0.0.1:19998/mcp` with JSON-RPC health tool call → verify result.
   k. **Test 8 — MCP initialize**: JSON-RPC `initialize` → verify `protocolVersion` is `"2024-11-05"`.
   l. **Test 9 — Dashboard**: `GET /ui` → verify 200 + HTML content.
   m. **Final**: Stop daemon, clean up temp files.
3. Use `set -euo pipefail` and `trap` for cleanup.
4. Print clear PASS/FAIL for each test with coloured output.
5. Exit 0 if all pass, 1 if any fail.
6. Use env vars: `NEXUS_HTTP_PORT` (default 19999), `NEXUS_MCP_PORT` (default 19998) for port override.

## Acceptance Criteria
- [ ] `scripts/e2e-smoke.sh` exists and is executable (`chmod +x`)
- [ ] Script builds daemon, starts it, runs at least 9 tests, stops daemon, cleans up
- [ ] Script exits 0 when all tests pass (with or without a running LLM provider)
- [ ] Script uses `trap` to ensure daemon is killed on failure
- [ ] Script does not conflict with running daemon (uses different ports)
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0

## Anti-patterns to Avoid
- NEVER hardcode absolute paths — use `$PROJECT_ROOT` or `$(dirname "$0")/..`
- NEVER leave daemon processes running on failure — always `trap` cleanup
- NEVER test features that require a live LLM (set expectations accordingly: task may go NO_PROVIDER)
