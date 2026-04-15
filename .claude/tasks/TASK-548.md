---
id: TASK-548
planId: PLAN-070
title: 'Write brain_client_test.go — all 13 methods'
role: qa
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-548 — brain_client.go test coverage

## Context

`internal/adapters/outbound/httpapi_client/brain_client.go` has **0% test coverage**.
All 13 methods (8 brain + `NewBrainClient`, `newRequest`, `do` + 2 helpers) are untested.
This is the HTTP client used by CLI brain subcommands and transitively by the VS Code extension.
Any HTTP serialisation bug (wrong verb, wrong URL, wrong response decode) is invisible until
runtime.

## Work Required

Create `internal/adapters/outbound/httpapi_client/brain_client_test.go`.

Use `httptest.NewServer` to create a mock daemon server for each test. Follow the existing
`client_test.go` pattern in the same package.

Test cases:

1. `TestBrainClient_GetStatus_OK` — mock GET returns `200 BrainStatus{EntryCount:3}`; verify decoded
2. `TestBrainClient_GetStatus_ServerError` — mock returns `500`; verify error returned
3. `TestBrainClient_IngestFromFile_OK` — mock POST `/api/brain/ingest` returns `{"ingestedSections":2}`
4. `TestBrainClient_IngestFromFile_BadResponse` — mock returns `400`; verify error
5. `TestBrainClient_InitProject_OK` — mock POST `/api/brain/init` returns BrainStatus
6. `TestBrainClient_ListKnowledge_OK` — mock GET returns `[{"id":"1",...}]`
7. `TestBrainClient_ListKnowledge_Empty` — mock returns `null`; verify empty slice returned (not nil error)
8. `TestBrainClient_DeleteKnowledge_OK` — mock DELETE returns 204; verify nil error
9. `TestBrainClient_DeleteKnowledge_NotFound` — mock returns 404; verify error
10. `TestBrainClient_GetFileMap_OK` — mock GET returns `{"filePaths":["a.go","b.go"]}`
11. `TestBrainClient_SearchKnowledge_OK` — mock GET returns `{"results":[...]}`
12. `TestBrainClient_GetContext_OK` — mock POST returns ContextResponse
13. `TestBrainClient_GetFocusedContext_OK` — mock POST returns ContextResponse
14. `TestBrainClient_IngestKnowledge_ReturnsError` — verify descriptive error (not panic)

Test the correct HTTP method and URL path for each case. Use `r.Method` and `r.URL.Path` assertions
in the mock handler.

## File Targets

- `internal/adapters/outbound/httpapi_client/brain_client_test.go` (new)

## Acceptance Criteria

- `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/httpapi_client/...` green
- At least 14 test functions
- HTTP method and path verified for each client method
- `brain_client.go` coverage ≥ 80%
