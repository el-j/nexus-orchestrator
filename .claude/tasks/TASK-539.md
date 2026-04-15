---
id: TASK-539
planId: PLAN-069
title: 'SQLite quality: explicit BM25 ORDER BY + UpdateKnowledge token_count recompute'
role: quality
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-539 — SQLite knowledge repo quality fixes

## Context

Two quality issues in `internal/adapters/outbound/repo_sqlite/knowledge_repo.go`:

### Issue 1: Implicit BM25 ordering in SearchFTS

The FTS5 `bm25()` function returns a negative float (lower = more relevant). The current query
relies on SQLite's default row order when no `ORDER BY` is present — this is implementation-defined
and not guaranteed. It should be:

```sql
ORDER BY bm25(knowledge_fts) ASC
```

(ASC because bm25 scores are negative — closer to 0 is less relevant, more negative is more.)

### Issue 2: UpdateKnowledge does not recompute token_count

When `UpdateKnowledge` persists a modified `ProjectKnowledge` entry, it writes whatever
`TokenCount` value is in the struct. If the caller changes `Content` without updating `TokenCount`,
the stored count drifts. The `SaveKnowledge` path computes token count from `len(content)/4`.
`UpdateKnowledge` should apply the same recomputation: `k.TokenCount = len(k.Content) / 4`.

## Work Required

In `internal/adapters/outbound/repo_sqlite/knowledge_repo.go`:

1. Locate the `SearchFTS` method SQL query — add `ORDER BY bm25(knowledge_fts) ASC` to the query.

2. Locate the `UpdateKnowledge` method — add token count recomputation before the SQL UPDATE:
   ```go
   if len(k.Content) > 0 {
       k.TokenCount = len(k.Content) / 4
   }
   ```

## File Targets

- `internal/adapters/outbound/repo_sqlite/knowledge_repo.go`

## Acceptance Criteria

- `SearchFTS` SQL query contains explicit `ORDER BY bm25(...) ASC`
- `UpdateKnowledge` recomputes `TokenCount` from content length before persisting
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/adapters/outbound/repo_sqlite/...` green (requires TASK-536 tests to be present)
