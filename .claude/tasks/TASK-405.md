---
id: TASK-405
title: Add SSE transport tests — ping keepalive, reconnect resilience, timeout behaviour
role: testing
planId: PLAN-056
status: todo
dependencies: [TASK-404]
createdAt: 2026-03-28T22:00:00Z
---

## Goal

Add targeted tests to `internal/adapters/inbound/mcp/sse_test.go` covering the three
behaviours fixed in TASK-404.

## Tests to Add

### Test 1 — Ping events are sent

```
TestSSE_PingEventsAreSent
  1. Connect GET /sse in a goroutine
  2. Read events from the SSE stream with a short timeout
  3. Assert at least one event has empty name (SSE comment ": ping") within 3x ping interval
  NOTE: Use a test-only configurable tick duration to avoid 15s wait in unit tests
```

### Test 2 — Missing session returns parseable response

```
TestSSE_MissingSession_ReturnsParseable
  1. POST /messages?sessionId=nonexistent with valid JSON-RPC ping
  2. Assert response is either:
     - 204 No Content (empty body OK), OR
     - 2xx/4xx with Content-Type: application/json and valid JSON body
  3. Assert response body is NOT an empty string when status is 4xx
     (the bug: http.Error returns plain text "session not found\n"
      which client receives as unparseable)
```

### Test 3 — No ReadTimeout on SSE connections

```
TestSSE_NoReadTimeoutOnIdleConnection
  1. Start test MCP server
  2. Open SSE connection
  3. Sleep for longer than old ReadTimeout (15s) — use a mock or short duration
  4. Verify the SSE connection is still open and receives ping events
```

### Test 4 — Origin-less reconnect after server restart simulation

```
TestSSE_ReconnectAfterSessionLoss
  1. Connect GET /sse, get sessionId from endpoint event
  2. Manually clear the sseManager (simulate restart)
  3. POST /messages?sessionId=<old-id>
  4. Assert returns parseable response (not empty body + 400)
  5. Open new GET /sse, gets new sessionId
  6. POST initialize to new sessionId — assert 202 + valid SSE response
```

## File

`internal/adapters/inbound/mcp/sse_test.go` — append to existing test file

## Acceptance Criteria

- [ ] All 4 new tests pass with `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/...`
- [ ] No existing SSE tests broken
- [ ] Test ping interval is configurable (not hardcoded 15s) via a package-level `sseKeepaliveInterval` var
