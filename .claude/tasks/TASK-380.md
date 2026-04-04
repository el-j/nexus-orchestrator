# TASK-380: Create mcp/tools_test.go — table-driven tests for all 31 MCP tool handlers

**Plan:** PLAN-055 | **Wave:** 4 | **Status:** done | **Role:** testing

## Goal

The MCP adapter has 31 tools but NO dedicated unit test file for tool handlers. `server_test.go` and `integration_test.go` test protocol-level behaviour; individual tool business logic (argument validation, error mapping, response shape) is untested.

## File to Create

`internal/adapters/inbound/mcp/tools_test.go`

## Required Reading Before Implementing

- `internal/adapters/inbound/mcp/tools.go` — read ALL tool handler signatures
- `internal/adapters/inbound/mcp/server_test.go` — understand mock orchestrator setup pattern
- `internal/core/services/orchestrator_test.go` — understand test double pattern
- `internal/adapters/inbound/mcp/integration_test.go` — understand existing test patterns

## Key Tools to Test (with focus areas)

| Tool                   | Key Cases                                                                                                           |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `create_draft`         | priority=string fails, priority=int ok; missing instruction fails; missing projectPath fails; happy path returns id |
| `promote_task`         | nonexistent ID → error; DRAFT → promoted + warning when no provider; already QUEUED → error                         |
| `cancel_task`          | QUEUED → cancelled ok; NO_PROVIDER → cancelled ok (after TASK-373 fix); PROCESSING → error                          |
| `claim_task`           | QUEUED task claimed ok; already CLAIMED by another → error                                                          |
| `update_task_status`   | wrong session → error; own session + COMPLETED → ok                                                                 |
| `get_backlog`          | empty projectPath returns all; with projectPath filters correctly                                                   |
| `get_queue`            | returns only QUEUED tasks                                                                                           |
| `get_task`             | valid ID → task JSON; invalid ID → error                                                                            |
| `register_session`     | new session created; same external_id → idempotent                                                                  |
| `heartbeat_ai_session` | valid session → ok; expired session → error                                                                         |
| `get_ai_sessions`      | returns registered sessions                                                                                         |
| `get_providers`        | returns provider list                                                                                               |
| `howto`                | returns non-empty string                                                                                            |
| `howto_brief`          | returns shorter string than howto                                                                                   |

## Structure Pattern

```go
package mcp_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    // ...
)

func TestMCPTool_CreateDraft(t *testing.T) {
    cases := []struct {
        name    string
        args    map[string]any
        wantErr bool
        check   func(t *testing.T, resp map[string]any)
    }{
        {
            name:    "priority as string fails",
            args:    map[string]any{"priority": "1", "instruction": "x", "projectPath": "/p"},
            wantErr: true,
        },
        {
            name: "happy path returns id",
            args: map[string]any{"priority": 1, "instruction": "do x", "projectPath": "/p"},
            check: func(t *testing.T, resp map[string]any) {
                require.Contains(t, resp, "id")
                require.Equal(t, "DRAFT", resp["status"])
            },
        },
        // ...
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            srv := newTestMCPServer(t)
            result := invokeToolHTTP(t, srv, "create_draft", tc.args)
            // ...
        })
    }
}
```

Use the existing `newTestServer()` helper from `server_test.go` (or create `newTestMCPServer()` that builds a real `*Server` with an in-memory orchestrator).

## Verification

- `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/... -v -run TestMCPTool` passes
- At least 30 test cases covering the 14 key tools above
- All error paths and argument validation paths covered

## Status

done
