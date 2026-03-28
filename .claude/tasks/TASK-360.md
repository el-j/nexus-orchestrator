# TASK-360: Add stdio transport binary

**Plan:** PLAN-052 | **Wave:** 5 | **Status:** done

## Description

Create `cmd/nexus-mcp-stdio/main.go` — a thin binary that reads JSON-RPC from
stdin and writes responses to stdout. This enables `type: stdio` in Continue/Claude/Cursor.

The binary acts as a client that forwards MCP messages to the running daemon
at `127.0.0.1:63988`.

## Implementation

- Read newline-delimited JSON-RPC from stdin
- Forward each message via HTTP POST to `http://127.0.0.1:63988/mcp`
- Write response to stdout
- stderr for logging only
- Environment: `NEXUS_MCP_URL` to override daemon URL

## Files to create

- `cmd/nexus-mcp-stdio/main.go`

## Acceptance criteria

- Can be launched as subprocess by MCP clients
- JSON-RPC messages round-trip correctly
- Works with `command: ./nexus-mcp-stdio` in Continue config
