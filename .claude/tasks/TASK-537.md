---
id: TASK-537
planId: PLAN-069
title: 'Expand brain_service_test.go (SearchKnowledge, InitProject, GetStatus)'
role: qa
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-537 — Expand BrainService unit tests

## Context

`internal/core/services/brain_service_test.go` covers three paths:

- `GetContext` — budget truncation
- `IngestFromFile` — section count
- `GetFocusedContext` — basic result

Missing coverage:

- `SearchKnowledge` — delegates to `repo.SearchFTS`; verifies results returned and limit applied
- `InitProject` — auto-detects CLAUDE.md when `claudeMDPath` is empty; calls `IngestFromFile`
- `GetStatus` — delegates to `repo.GetStatus`
- `GetContext` error path — `repo.GetByProject` returns error → service returns wrapped error
- `IngestFromFile` file-not-found error path

## Work Required

Add test functions to `internal/core/services/brain_service_test.go` (extend existing file).
Use the existing `mockKnowledgeRepo` already defined in that file.

New test functions:

1. `TestBrainService_SearchKnowledge` — seeds 3 entries, calls `SearchKnowledge` with limit 2, verifies 2 results returned
2. `TestBrainService_InitProject_AutoDetect` — writes a CLAUDE.md to `t.TempDir()`, calls `InitProject` with empty `claudeMDPath`, verifies sections > 0
3. `TestBrainService_GetStatus` — sets `mockKnowledgeRepo.status`, calls `GetStatus`, verifies fields match
4. `TestBrainService_IngestFromFile_FileNotFound` — calls `IngestFromFile` with non-existent path, verifies error returned

## File Targets

- `internal/core/services/brain_service_test.go`

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/core/services/...` all green
- 4 new test functions added
