---
id: TASK-518
title: Wire BrainService into daemon and CLI startup
role: backend
planId: PLAN-066
status: done
dependencies: [TASK-515, TASK-516, TASK-517]
createdAt: 2026-04-13T00:00:00Z
---

## Context

All components exist but are not yet wired together. This task connects everything at the entry points: the daemon constructs KnowledgeRepo + BrainService and injects them into the HTTP and MCP servers. The CLI gets the brain subcommand. Also update `/.well-known/nexus.json` to advertise the new capability.

## Files to Read

- `cmd/nexus-daemon/main.go` — ALL OF IT. Understand the exact wiring sequence (repo → services → HTTP server → MCP server → start)
- `cmd/nexus-cli/main.go` — how root command is constructed
- `internal/adapters/inbound/mcp/server.go` — how MCP server is started, constructor signature
- `internal/adapters/inbound/httpapi/server.go` — Server constructor and WithBrainService setter
- `internal/adapters/inbound/httpapi/howto.go` — handleWellKnownNexus for capabilities

## Implementation Steps

1. **In `cmd/nexus-daemon/main.go`**, after the line that constructs `repo` (SQLite repository):

   ```go
   knowledgeRepo := repo_sqlite.NewKnowledgeRepo(repo)
   brainSvc := services.NewBrainService(knowledgeRepo, repo)
   ```

   Check exact variable names for repo and services in the file. Follow existing naming style.

2. **Wire BrainService into HTTP server**. Find the line that constructs or configures the HTTP server. Add:

   ```go
   httpServer.WithBrainService(brainSvc)
   ```

   (Or pass to constructor if that's the pattern used — check server.go)

3. **Wire BrainService into MCP server**. Find how the MCP server is started in main.go (look for `StartMCPServer` or equivalent). Add `SetBrainService` call:
   - If MCP server is started via a function call and returns nothing useful: need to either pass opts or use a returned `*Server` handle.
   - Check if `StartMCPServer` returns `*Server` or is fire-and-forget.
   - If fire-and-forget: modify `StartMCPServer` signature to accept `...ServerOption` options, add `WithBrain(ports.BrainService)` option. Update the call site.
   - If it returns `*Server`: call `mcpServer.SetBrainService(brainSvc)` before the blocking start.
   - Simplest safe approach: add `brainSvc ports.BrainService` as an explicit parameter to `StartMCPServer(ctx, orch, brainSvc, addr)` and have it call `server.SetBrainService(brainSvc)` internally.

4. **Update `/.well-known/nexus.json`** capabilities. Find `handleWellKnownNexus` in `howto.go`. Add `"project-brain"` to the `Capabilities` array/slice.

5. **In `cmd/nexus-cli/main.go`**: Find where subcommands are added. Register `brain` subcommand — follow the existing pattern. If brain CLI needs the base URL, it should be available at the same level other HTTP-calling commands get it.

6. **Verify the build** by running the actual build commands and fixing any import cycle or compilation errors that arise from the new wiring.

7. **Smoke check**: The daemon should start without error. If a CLAUDE.md is present in the working directory, optionally log "Brain service ready" at startup (but don't auto-ingest — let the user trigger that explicitly via MCP or CLI).

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Daemon starts without panic (test manually or via existing E2E test)
- [ ] `GET /.well-known/nexus.json` response includes `"project-brain"` in capabilities
- [ ] MCP `toolList()` includes all 6 brain tools when server is queried
- [ ] HTTP `GET /api/brain/status?project=/tmp` returns 200 (not 501) when daemon is running

## Anti-patterns to Avoid

- NEVER create circular imports (brain service → mcp adapter → brain service)
- NEVER start background goroutines for brain in the service layer
- NEVER panic on nil BrainService — guard everywhere with nil checks
- NEVER auto-ingest CLAUDE.md at startup without user action (could be slow for large files)
