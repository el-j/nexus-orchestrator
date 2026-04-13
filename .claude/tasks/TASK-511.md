---
id: TASK-511
title: Implement project knowledge SQLite repository
role: backend
planId: PLAN-066
status: done
dependencies: [TASK-510]
createdAt: 2026-04-13T00:00:00Z
---

## Context

The knowledge repository stores and queries `ProjectKnowledge` entries in SQLite. FTS5 (built into mattn/go-sqlite3 with CGO) provides BM25 full-text search. The repository follows the same pattern as existing repos: shares the `*sql.DB` from the parent `Repository`, adds its own migration, and is constructed via `NewKnowledgeRepo(repo *Repository)`.

## Files to Read

- `internal/adapters/outbound/repo_sqlite/repo.go` — understand Repository struct, `migrate()` pattern, `filepath.Clean` usage, helper query patterns
- `internal/adapters/outbound/repo_sqlite/ai_session_repo.go` — model for a secondary repo sharing the DB
- `internal/core/ports/brain.go` — the interface to implement (TASK-510)
- `internal/core/domain/brain.go` — the types to persist (TASK-509)

## Implementation Steps

1. Create `internal/adapters/outbound/repo_sqlite/knowledge_repo.go`

2. Define `KnowledgeRepo` struct with `db *sql.DB` field

3. Constructor: `func NewKnowledgeRepo(repo *Repository) *KnowledgeRepo` — extracts `repo.db` (check field name in repo.go)

4. Add migration function `migrateKnowledge(db *sql.DB) error` that runs these statements in order:

   ```sql
   CREATE TABLE IF NOT EXISTS project_knowledge (
       id TEXT PRIMARY KEY,
       project_path TEXT NOT NULL,
       kind TEXT NOT NULL,
       topic TEXT NOT NULL,
       content TEXT NOT NULL,
       source TEXT NOT NULL DEFAULT '',
       token_count INTEGER NOT NULL DEFAULT 0,
       relevance_score REAL NOT NULL DEFAULT 0.5,
       created_at INTEGER NOT NULL,
       updated_at INTEGER NOT NULL
   );
   CREATE INDEX IF NOT EXISTS idx_pk_project ON project_knowledge(project_path);
   CREATE INDEX IF NOT EXISTS idx_pk_project_kind ON project_knowledge(project_path, kind);
   CREATE UNIQUE INDEX IF NOT EXISTS idx_pk_upsert ON project_knowledge(project_path, kind, topic);
   CREATE VIRTUAL TABLE IF NOT EXISTS project_knowledge_fts USING fts5(
       topic, content,
       content='project_knowledge',
       content_rowid='rowid'
   );
   CREATE TRIGGER IF NOT EXISTS pk_ai AFTER INSERT ON project_knowledge BEGIN
       INSERT INTO project_knowledge_fts(rowid, topic, content) VALUES (new.rowid, new.topic, new.content);
   END;
   CREATE TRIGGER IF NOT EXISTS pk_ad AFTER DELETE ON project_knowledge BEGIN
       INSERT INTO project_knowledge_fts(project_knowledge_fts, rowid, topic, content)
       VALUES ('delete', old.rowid, old.topic, old.content);
   END;
   CREATE TRIGGER IF NOT EXISTS pk_au AFTER UPDATE ON project_knowledge BEGIN
       INSERT INTO project_knowledge_fts(project_knowledge_fts, rowid, topic, content)
       VALUES ('delete', old.rowid, old.topic, old.content);
       INSERT INTO project_knowledge_fts(rowid, topic, content) VALUES (new.rowid, new.topic, new.content);
   END;
   CREATE TABLE IF NOT EXISTS context_query_log (
       id TEXT PRIMARY KEY,
       project_path TEXT NOT NULL,
       query_json TEXT NOT NULL,
       tokens_used INTEGER NOT NULL DEFAULT 0,
       section_count INTEGER NOT NULL DEFAULT 0,
       truncated INTEGER NOT NULL DEFAULT 0,
       created_at INTEGER NOT NULL
   );
   CREATE INDEX IF NOT EXISTS idx_cql_project ON context_query_log(project_path);
   ```

5. Call `migrateKnowledge(db)` from the parent `migrate()` function in `repo.go` (add one line: `if err := migrateKnowledge(db); err != nil { return err }`).

6. Implement `SaveKnowledge`: INSERT OR REPLACE using all fields. Use `uuid.New().String()` for new IDs. `filepath.Clean` on ProjectPath. Store timestamps as Unix epoch int64.

7. Implement `GetByID`: SELECT by id, scan all fields, return `ErrNotFound` wrapped error when no rows.

8. Implement `UpdateKnowledge`: UPDATE all mutable fields (kind, topic, content, source, token_count, relevance_score, updated_at) WHERE id = ?

9. Implement `DeleteKnowledge`: DELETE WHERE id = ?. Return `domain.ErrNotFound` if 0 rows affected.

10. Implement `GetByProject`: SELECT WHERE project_path = ? ORDER BY kind, relevance_score DESC

11. Implement `GetByProjectAndKind`: SELECT WHERE project_path = ? AND kind = ? ORDER BY relevance_score DESC

12. Implement `SearchFTS`:

    ```sql
    SELECT pk.id, pk.project_path, pk.kind, pk.topic, pk.content, pk.source,
           pk.token_count, pk.relevance_score, pk.created_at, pk.updated_at
    FROM project_knowledge pk
    JOIN project_knowledge_fts fts ON pk.rowid = fts.rowid
    WHERE project_knowledge_fts MATCH ?
      AND pk.project_path = ?
    ORDER BY bm25(project_knowledge_fts)
    LIMIT ?
    ```

13. Implement `GetStatus`:

    ```sql
    SELECT kind, COUNT(*), SUM(token_count), MAX(updated_at)
    FROM project_knowledge WHERE project_path = ? GROUP BY kind
    ```

    Aggregate into `domain.BrainStatus`: sum EntryCount, build KindCounts map, sum TotalTokens, set Initialized = EntryCount > 0.

14. Implement `DeleteByProject`: DELETE WHERE project_path = ?

15. Error wrapping: `fmt.Errorf("knowledge_repo: %s: %w", "operation name", err)` for all errors.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `KnowledgeRepo` implements `ports.ProjectKnowledgeRepository` (compiler check via `var _ ports.ProjectKnowledgeRepository = (*KnowledgeRepo)(nil)`)
- [ ] All 9 interface methods implemented
- [ ] FTS5 triggers created (INSERT/UPDATE/DELETE stay synchronized)
- [ ] `migrateKnowledge` called from parent `migrate()` in repo.go

## Anti-patterns to Avoid

- NEVER import core services from this adapter
- NEVER skip `filepath.Clean` on project paths
- NEVER use string interpolation to build SQL — always use `?` placeholders
- NEVER open a new DB connection — reuse the parent repo's `db`
