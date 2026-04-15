---
id: TASK-536
planId: PLAN-069
title: 'Write SQLite knowledge repository tests'
role: qa
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-536 — SQLite knowledge repository tests

## Context

`internal/adapters/outbound/repo_sqlite/knowledge_repo.go` implements
`ports.ProjectKnowledgeRepository` using SQLite FTS5. There are zero test files for this
repository. Key behaviors are entirely untested:

- FTS5 BM25 ranking (higher relevance items returned first)
- Upsert dedup (re-ingesting the same topic+project updates rather than inserts a duplicate)
- Project isolation (`SearchFTS` for project A must not return project B results)
- `GetStatus` entry count and token sum accuracy
- `DeleteKnowledge` removes from both main table and FTS shadow tables
- `DeleteByProject` bulk delete

Without these tests, regressions in FTS5 compilation flags or schema changes go undetected.

## Work Required

Create `internal/adapters/outbound/repo_sqlite/knowledge_repo_test.go`.

Use `t.TempDir()` for each test's SQLite file path. Bootstrap the DB via `NewSQLiteRepo` and
confirm `NewKnowledgeRepo` returns without error with `CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"`.

Test cases to cover:

1. `SaveKnowledge` + `GetByID` round-trip
2. `SaveKnowledge` same project+kind+topic → `UpdateKnowledge` path (upsert dedup)
3. `SearchFTS` returns results in BM25 order (most relevant first)
4. `SearchFTS` project isolation (two projects, query returns only correct project)
5. `GetStatus` entry count + token sum
6. `DeleteKnowledge` removes entry, subsequent `GetByID` returns `domain.ErrNotFound`
7. `DeleteByProject` removes all entries for one project without touching another

## File Targets

- `internal/adapters/outbound/repo_sqlite/knowledge_repo_test.go` (new)

## Acceptance Criteria

- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/adapters/outbound/repo_sqlite/...` all green
- All 7 test cases implemented
