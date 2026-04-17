---
id: TASK-555
title: Fix all 12 silenced Go errors across session service, MCP server, and SSE
role: backend
planId: PLAN-071
status: todo
dependencies: []
createdAt: 2026-04-15T00:00:00Z
---

## Context

Twelve errors are silently discarded with `_ =` across three files. In `session_service.go` process termination errors are dropped (mask real signal failures). In `mcp/server.go` seven `json.NewEncoder(w).Encode()` write errors are dropped (client never learns of partial responses). In `mcp/tools.go` two `json.Unmarshal` errors are dropped (malformed MCP tool arguments produce silent zero-value behaviour instead of `-32602 invalid params`). In `mcp/sse.go` three `sendEvent` errors are dropped. All should either be logged or returned to the caller.

## Files to Read

- `internal/core/services/session_service.go`
- `internal/adapters/inbound/mcp/server.go`
- `internal/adapters/inbound/mcp/tools.go`
- `internal/adapters/inbound/mcp/sse.go`

## Implementation Steps

1. `session_service.go:137` — replace `_ = proc.Kill()` with `if err := proc.Kill(); err != nil { log.Printf("session_service: kill process %d: %v", proc.Pid, err) }`
2. `session_service.go:141` — same pattern for `proc.Signal(os.Interrupt)`
3. `session_service.go:250` — replace `_ = o.planFileRepo.UpsertPlanFile(ctx, f)` with `if err := ...; err != nil { log.Printf("session_service: upsert plan file %s: %v", f.Path, err) }` (do not abort the loop — log and continue)
4. `mcp/tools.go:187` (toolGetQueue) — replace `_ = json.Unmarshal(args, &p)` with: if unmarshal fails and `args` is non-empty, return `errInvalidParams("malformed arguments")` (use the existing helper pattern in tools.go)
5. `mcp/tools.go:214` (toolGetAllTasks) — same fix as step 4
6. `mcp/server.go` (7 Encode sites) — wrap each with `if err := json.NewEncoder(w).Encode(...); err != nil { log.Printf("mcp: encode response: %v", err) }` — do not return errors here as HTTP headers are already written
7. `mcp/sse.go` (3 sendEvent sites) — wrap with `if err := sendEvent(...); err != nil { log.Printf("mcp: sse send: %v", err) }` and close the connection if appropriate

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Zero remaining `_ = proc.Kill()` / `_ = proc.Signal()` patterns in session_service.go
- [ ] Zero remaining `_ = json.Unmarshal` patterns in mcp/tools.go
- [ ] Zero remaining `_ = json.NewEncoder(w).Encode` patterns in mcp/server.go
- [ ] MCP tools return `-32602` error for malformed JSON args instead of silently zero-valuing

## Anti-patterns to Avoid

- NEVER return an error after HTTP headers are written (encode errors in server.go must only log)
- NEVER abort the plan-file upsert loop on a single failure — log and continue
- NEVER import adapters from core services (hexagonal dependency rule)
