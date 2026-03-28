# TASK-359: Security — Origin validation & CORS headers

**Plan:** PLAN-052 | **Wave:** 4 | **Status:** done

## Description

Per MCP spec security requirements:

1. Validate `Origin` header on all incoming connections to prevent DNS rebinding
2. Add CORS headers for browser-based MCP clients
3. Ensure proper Content-Type enforcement

## Implementation

- Allow requests with no Origin header (CLI clients, desktop apps)
- Allow `Origin: http://127.0.0.1:*`, `Origin: http://localhost:*`
- Block other origins unless configured via env var `NEXUS_MCP_ALLOWED_ORIGINS`
- Add `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`
- Handle OPTIONS preflight

## Files to modify

- `internal/adapters/inbound/mcp/server.go` — add middleware

## Acceptance criteria

- Requests without Origin pass through
- localhost Origin passes through
- Foreign Origin blocked with 403
- CORS preflight returns proper headers
