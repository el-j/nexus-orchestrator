# TASK-385: Black-box E2E Go integration test — full system pipeline

**Plan:** PLAN-055 | **Wave:** 6 | **Status:** done | **Role:** testing

## Goal

Create a Go integration test that spins up a real `nexusd` daemon in-process (or via `httptest.NewServer`) and exercises the COMPLETE system pipeline:

1. All major HTTP API endpoints (task lifecycle, providers, sessions, plans)
2. All key MCP tools via JSON-RPC calls (same server, MCP port)
3. State machine: create draft → promote (with no-provider warning) → NO_PROVIDER → cancel (now works after TASK-373)
4. Session registration → heartbeat → deregister lifecycle
5. Plan file discovery (if `.claude/` dir exists)

## File to Create

`internal/e2e/blackbox_test.go`

## Architecture

Use `bootstrap.BuildOrchestrator()` to create a real orchestrator with SQLite in-memory (`:memory:`), then start it via `httpapi.NewServer()` + `mcp.NewServer()` on `httptest.NewServer`. No actual LLM needed — test NO_PROVIDER path.

## Structure

```go
//go:build integration
// +build integration

package e2e_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func setupTestSystem(t *testing.T) (httpAPIBase, mcpBase string, cleanup func()) {
    t.Helper()
    // Build orchestrator with in-memory SQLite
    orch := buildTestOrchestrator(t)  // see helpers section

    apiSrv := httptest.NewServer(httpapi.NewServer(orch))
    mcpSrv := httptest.NewServer(mcp.NewServer(orch))

    return apiSrv.URL, mcpSrv.URL, func() {
        apiSrv.Close()
        mcpSrv.Close()
    }
}

func TestE2E_TaskLifecycle_NoProvider(t *testing.T) {
    apiBase, _, cleanup := setupTestSystem(t)
    defer cleanup()

    // 1. Create draft via HTTP
    draft := createDraftHTTP(t, apiBase, map[string]any{
        "instruction": "write a hello world function",
        "projectPath": t.TempDir(),
        "priority": 2,
    })
    require.Equal(t, "DRAFT", draft["status"])
    id := draft["id"].(string)

    // 2. Promote — expect warning (no provider)
    result := promoteTaskHTTP(t, apiBase, id)
    require.True(t, result["promoted"].(bool))
    require.NotEmpty(t, result["warning"])

    // 3. Wait for worker to attempt execution → NO_PROVIDER
    var task map[string]any
    require.Eventually(t, func() bool {
        task = getTaskHTTP(t, apiBase, id)
        return task["status"] == "NO_PROVIDER"
    }, 5*time.Second, 200*time.Millisecond)

    // 4. Cancel NO_PROVIDER task (was broken before TASK-373)
    cancelTaskHTTP(t, apiBase, id)
    task = getTaskHTTP(t, apiBase, id)
    assert.Equal(t, "CANCELLED", task["status"])
}

func TestE2E_MCPToolchain_SessionLifecycle(t *testing.T) {
    _, mcpBase, cleanup := setupTestSystem(t)
    defer cleanup()

    // 1. Initialize MCP session
    initResp := mcpCall(t, mcpBase, "initialize", map[string]any{
        "protocolVersion": "2024-11-05",
        "clientInfo": map[string]any{"name": "e2e-test", "version": "0.0.1"},
    })
    require.Equal(t, "2024-11-05", initResp["protocolVersion"])

    // 2. Register AI session via MCP tool
    regResp := mcpToolCall(t, mcpBase, "register_session", map[string]any{
        "agent_name": "e2e-test-agent",
        "external_id": "e2e-test-001",
        "project_path": "/test/project",
    })
    sessionID := regResp["session_id"].(string)
    require.NotEmpty(t, sessionID)

    // 3. Create draft via MCP tool
    draftResp := mcpToolCall(t, mcpBase, "create_draft", map[string]any{
        "instruction": "e2e test task",
        "projectPath": "/test/project",
        "priority": 1,
    })
    taskID := draftResp["id"].(string)
    require.NotEmpty(t, taskID)

    // 4. Get queue (should be empty — task is DRAFT not QUEUED)
    queueResp := mcpToolCall(t, mcpBase, "get_queue", map[string]any{})
    require.IsType(t, []any{}, queueResp["tasks"])

    // 5. Promote → NO_PROVIDER warning
    promResp := mcpToolCall(t, mcpBase, "promote_task", map[string]any{"id": taskID})
    require.True(t, promResp["promoted"].(bool))

    // 6. Heartbeat (keep session alive)
    hbResp := mcpToolCall(t, mcpBase, "heartbeat_ai_session", map[string]any{
        "session_id": sessionID,
    })
    require.True(t, hbResp["ok"].(bool))

    // 7. Cancel task after NO_PROVIDER
    require.Eventually(t, func() bool {
        task := mcpToolCall(t, mcpBase, "get_task", map[string]any{"id": taskID})
        return task["status"] == "NO_PROVIDER"
    }, 5*time.Second, 200*time.Millisecond)

    cancelResp := mcpToolCall(t, mcpBase, "cancel_task", map[string]any{"id": taskID})
    require.True(t, cancelResp["cancelled"].(bool))

    // 8. Deregister session
    deregResp := mcpToolCall(t, mcpBase, "deregister_ai_session", map[string]any{
        "session_id": sessionID,
    })
    require.True(t, deregResp["ok"].(bool))
}

func TestE2E_AllHTTPEndpoints_Return2xx(t *testing.T) {
    apiBase, _, cleanup := setupTestSystem(t)
    defer cleanup()

    endpoints := []struct {
        method string
        path   string
        body   any
    }{
        {"GET", "/api/health", nil},
        {"GET", "/api/howto", nil},
        {"GET", "/.well-known/nexus.json", nil},
        {"GET", "/api/tasks", nil},
        {"GET", "/api/tasks/all", nil},
        {"GET", "/api/providers", nil},
        {"GET", "/api/providers/discovered", nil},
        {"GET", "/api/ai-sessions", nil},
        {"GET", "/api/ai-sessions/discovered", nil},
        {"GET", "/api/activities", nil},
        {"GET", "/api/activities/timeline", nil},
        {"GET", "/api/plans/discovered", nil},
    }

    for _, ep := range endpoints {
        t.Run(ep.method+" "+ep.path, func(t *testing.T) {
            resp := doHTTP(t, apiBase, ep.method, ep.path, ep.body)
            assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 300,
                "expected 2xx for %s %s, got %d", ep.method, ep.path, resp.StatusCode)
        })
    }
}
```

## Build Tag

Use `//go:build integration` so it doesn't run in normal `go test ./...`. Run with:

```bash
CGO_ENABLED=1 go test -tags integration -race ./internal/e2e/...
```

Add to CI as a separate job: `e2e-test`.

## Verification

- `CGO_ENABLED=1 go test -tags integration -race ./internal/e2e/... -v` passes
- All 3 test functions green
- E2E confirms NO_PROVIDER cancel fixed
- E2E confirms promote warning present
- All 12 health endpoints return 2xx

## Status

done
