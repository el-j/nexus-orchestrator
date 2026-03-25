---
id: TASK-313
title: Split mcp/server.go into server.go (protocol) + tools.go (schemas + handlers)
role: architecture
planId: PLAN-047
status: todo
dependencies: [TASK-305, TASK-310]
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

`internal/adapters/inbound/mcp/server.go` is ~914 lines (will grow after TASK-305 adds 9 tools). It contains two fully separable concerns:

1. **MCP JSON-RPC protocol layer** — types, dispatch, `handleRPC`, `handleInitialize`, error helpers (~200 lines)
2. **Tool catalog + handlers** — `toolList()` (200-line schema declarations) + 29 `tool*` handler methods (~700 lines)

Separating them makes it trivial to add/remove tools without touching protocol code.

## Implementation Steps

1. Read the full `internal/adapters/inbound/mcp/server.go`.

2. **Create `internal/adapters/inbound/mcp/tools.go`** — same package `mcp`:
   - Move `toolList()` function
   - Move all `tool*` handler methods (e.g., `toolSubmitTask`, `toolGetTask`, ... all 29)
   - Keep all imports needed by those methods

3. **Trim `server.go`** to contain only:
   - Package declaration + imports
   - Types: `toolDef`, `callToolResult`, `mcpError`, `mcpRequest`, `mcpResponse`, `toolCall`, `Server` struct
   - `NewServer`, `ServeHTTP`, `StartMCPServer`, `handleRPC`, `handleInitialize`, `handleHealth`, `handleToolCall` (dispatch switch), `writeError`, `textResult`, `mcpError` constructor helper

4. Verify both files compile together (same package, no circular deps).

## Acceptance Criteria

- [ ] `server.go` ≤250 lines
- [ ] `tools.go` contains `toolList()` and all `tool*` methods
- [ ] `go vet ./internal/adapters/inbound/mcp/...` clean
- [ ] `go test ./internal/adapters/inbound/mcp/... -race -count=1` all pass
- [ ] `go build ./cmd/nexus-daemon/...` exits 0
