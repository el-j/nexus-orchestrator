# TASK-381: HTTP API test — NO_PROVIDER cancel pipeline + promote pre-flight

**Plan:** PLAN-055 | **Wave:** 4 | **Status:** done | **Role:** testing

## Goal

Add HTTP API integration tests to verify the state machine fixes from TASK-373 and TASK-374:

1. Task promoted with no provider → response includes `warning` field
2. Task in NO_PROVIDER state → DELETE /api/tasks/{id} succeeds (was 409, now 204)
3. Confirm PROCESSING tasks still cannot be cancelled (status quo preserved)
4. Confirm the full happy-path promote flow still works

## Files to Edit

- `internal/adapters/inbound/httpapi/server_test.go` — add test cases
- OR create `internal/adapters/inbound/httpapi/handlers_tasks_test.go` if separated

## Tests to Add

```go
func TestHandlePromoteTask_NoProvider_ReturnsWarning(t *testing.T) {
    _, srv := newTestServer(t)  // no LLM providers wired
    // Create a draft
    draft := createDraftViaAPI(t, srv, domain.Task{Instruction: "test", ProjectPath: "/p"})

    // Promote — should succeed but include warning
    w := httptest.NewRecorder()
    r := httptest.NewRequest(http.MethodPost, "/api/tasks/"+draft.ID+"/promote", nil)
    srv.ServeHTTP(w, r)

    require.Equal(t, http.StatusOK, w.Code)
    var body map[string]any
    require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
    require.Equal(t, true, body["promoted"])
    require.NotEmpty(t, body["warning"])  // warning field present
}

func TestHandleCancelTask_NoProvider_Succeeds(t *testing.T) {
    orch, srv := newTestServer(t)
    draft := createDraftViaAPI(t, srv, domain.Task{Instruction: "test", ProjectPath: "/p"})
    // Set task to NO_PROVIDER by directly modifying repo (test helper)
    orch.repo.UpdateStatus(draft.ID, domain.StatusNoProvider)

    w := httptest.NewRecorder()
    r := httptest.NewRequest(http.MethodDelete, "/api/tasks/"+draft.ID, nil)
    srv.ServeHTTP(w, r)

    require.Equal(t, http.StatusNoContent, w.Code)
    // Verify task is now CANCELLED
    task := getTaskViaAPI(t, srv, draft.ID)
    require.Equal(t, domain.StatusCancelled, task.Status)
}

func TestHandleCancelTask_Processing_Returns409(t *testing.T) {
    orch, srv := newTestServer(t)
    draft := createDraftViaAPI(t, srv, domain.Task{Instruction: "test", ProjectPath: "/p"})
    // Set to PROCESSING (shouldn't be cancellable)
    orch.repo.UpdateStatus(draft.ID, domain.StatusProcessing)

    w := httptest.NewRecorder()
    r := httptest.NewRequest(http.MethodDelete, "/api/tasks/"+draft.ID, nil)
    srv.ServeHTTP(w, r)

    require.Equal(t, http.StatusConflict, w.Code)
}
```

## Verification

- `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/httpapi/... -run TestHandlePromote,TestHandleCancel` passes
- All 4 test cases green
- No regression in existing httpapi tests

## Status

done
