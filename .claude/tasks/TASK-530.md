---
id: TASK-530
planId: PLAN-069
title: 'Implement 4 stub methods in brain_client.go'
role: backend
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-530 — Implement stub methods in brain_client.go

## Context

`internal/adapters/outbound/httpapi_client/brain_client.go` implements `ports.BrainService` for
CLI-to-daemon communication. Four methods always return hard-coded errors:

```go
func (r *BrainClient) IngestKnowledge(...)  { return ..., fmt.Errorf("IngestKnowledge not implemented in client") }
func (r *BrainClient) GetFileMap(...)       { return nil, fmt.Errorf("GetFileMap not implemented in client") }
func (r *BrainClient) InitProject(...)      { return ..., fmt.Errorf("InitProject not implemented in client") }
func (r *BrainClient) ListKnowledge(...)    { return nil, fmt.Errorf("ListKnowledge not implemented in client") }
func (r *BrainClient) DeleteKnowledge(...) { return fmt.Errorf("DeleteKnowledge not implemented in client") }
```

These methods are called by CLI subcommands. They must make real HTTP calls to the daemon.

## Work Required

Implement each method in `brain_client.go` to call the corresponding daemon routes added in
TASK-529:

- `IngestKnowledge` → `POST /api/brain/ingest` (use `IngestFromFile` pattern but with full `domain.ProjectKnowledge`)
- `InitProject` → `POST /api/brain/init`
- `ListKnowledge` → `GET /api/brain/knowledge?projectPath=&kind=`
- `DeleteKnowledge` → `DELETE /api/brain/knowledge/{id}`
- `GetFileMap` → `GET /api/brain/file-map?projectPath=&focusArea=`

Note: `IngestFromFile` is already implemented. `IngestKnowledge` is a low-level upsert — consider
whether it needs a separate route or can map to the same `/api/brain/ingest` endpoint.

## File Targets

- `internal/adapters/outbound/httpapi_client/brain_client.go`

## Acceptance Criteria

- All 5 stub methods replaced with real HTTP calls
- `go vet ./internal/adapters/outbound/httpapi_client/...` clean
- No method returns a hard-coded "not implemented" error
