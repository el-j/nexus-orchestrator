---
id: TASK-547
planId: PLAN-070
title: 'Fix processNextTask silent errors + IngestFromFile error aggregation + FTS sanitization'
role: backend
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-547 — Backend quality fixes: silent errors + FTS safety

## Context

Three backend correctness issues found in the audit:

### Issue 1: `processNextTask` silent error swallowing

`internal/core/services/execution_engine.go` — `processNextTask` calls `buildChatContext` and
`executeGeneration`. On error from either, the function returns `true` (indicating a task was
processed) but does NOT set the task status to failed. The task stays PROCESSING forever, consuming
a worker slot and blocking the queue.

Fix: on error from `buildChatContext` or `executeGeneration`, call
`repo.UpdateTaskStatus(ctx, task.ID, domain.StatusFailed, err.Error())` before returning.

### Issue 2: `IngestFromFile` silently drops per-section errors

`internal/core/services/brain_service.go` — Both the preamble and section loops use
`if err == nil { count++ }`. Individual `IngestKnowledge` errors are never surfaced. A caller
that sees `count=0, err=nil` cannot distinguish "nothing to ingest" from "everything failed".

Fix: aggregate errors (e.g. `var errs []error`). If all sections fail, return a wrapped
multi-error. If some succeed, log the errors (using `slog` if available, or return a wrapped
partial-error to the caller). At minimum: if `count == 0 && len(sectionErrors) > 0`, return
the first error.

### Issue 3: FTS5 raw query passed to MATCH

`internal/adapters/outbound/repo_sqlite/knowledge_repo.go` — `SearchFTS` passes the raw
`query` string directly to SQLite FTS5 `MATCH`. Unbalanced quotes or malformed FTS syntax
returns a runtime error that surfaces as a 500.

Fix: sanitize by escaping double-quotes and wrapping the query in double-quotes for phrase
matching: `query = `"`+ strings.ReplaceAll(query,`"`, `""`) + `"``

This makes all search inputs safe while still enabling FTS5 to rank by BM25. Optionally:
catch the SQLite error and return a helpful `"invalid search query"` message.

## File Targets

- `internal/core/services/execution_engine.go` (or `orchestrator.go` — wherever `processNextTask` lives)
- `internal/core/services/brain_service.go`
- `internal/adapters/outbound/repo_sqlite/knowledge_repo.go`

## Acceptance Criteria

- `go vet ./...` clean
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...` green
- Task that fails in `buildChatContext` gets status = failed (not stuck PROCESSING)
- `IngestFromFile` returns an error when all sections fail to save
- `SearchFTS` does not return a 500 for input containing unbalanced double-quotes
