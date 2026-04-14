---
id: TASK-550
planId: PLAN-070
title: 'Brain service: test GetFileMap, ListKnowledge, DeleteKnowledge + CLI brain error paths'
role: qa
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-550 — Brain service missing coverage + CLI brain error paths

## Context

Two test coverage gaps:

### Gap 1: `brain_service.go` 3 methods at 0%

`GetFileMap`, `ListKnowledge`, and `DeleteKnowledge` are fully implemented in
`internal/core/services/brain_service.go` but have zero test coverage in
`brain_service_test.go`.

### Gap 2: CLI brain subcommands have no error path tests

`internal/adapters/inbound/cli/brain.go` — 8 subcommands at 43–62% coverage. All error
branches (service returns error, required flag missing) are untested. The CLI layer's
`root_test.go` only wires up the mock; no dedicated brain CLI test file exists.

## Work Required

### `internal/core/services/brain_service_test.go` — add 3 tests

**`TestBrainService_ListKnowledge`**

- Seed entries of different kinds in `mockKnowledgeRepo`
- Call `b.ListKnowledge(ctx, "/proj", "convention")`
- Verify only entries of the correct kind are returned

**`TestBrainService_DeleteKnowledge`**

- Seed one entry
- Call `b.DeleteKnowledge(ctx, entry.ID)`
- Verify `repo.GetByID` returns `domain.ErrNotFound`

**`TestBrainService_GetFileMap`**

- Seed file_map kind entries with paths in `Content`
- Call `b.GetFileMap(ctx, "/proj", "")`
- Verify returned paths slice is non-empty

### `internal/adapters/inbound/cli/brain_test.go` (new file)

Create a dedicated test file for the CLI brain layer. Use `cobra` command execution via
`cmd.Execute()` with a mock `BrainService`.

Test cases:

1. `TestBrainCLI_Status_ServiceError` — mock returns error; verify non-zero exit
2. `TestBrainCLI_Ingest_OK` — mock returns count=3; verify stdout contains "3"
3. `TestBrainCLI_Search_OK` — mock returns entries; verify JSON in stdout
4. `TestBrainCLI_Init_OK` — mock returns BrainStatus; verify JSON in stdout
5. `TestBrainCLI_Delete_OK` — mock returns nil; verify "deleted" in stdout
6. `TestBrainCLI_Context_OK` — mock returns ContextResponse; verify JSON in stdout

## File Targets

- `internal/core/services/brain_service_test.go` (extend)
- `internal/adapters/inbound/cli/brain_test.go` (new)

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/core/services/...` green
- `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/inbound/cli/...` green
- `brain_service.go` coverage ≥ 75% (all methods have ≥1 test)
