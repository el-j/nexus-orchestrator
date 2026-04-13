---
id: TASK-509
title: Define brain domain types
role: backend
planId: PLAN-066
status: todo
dependencies: []
createdAt: 2026-04-13T00:00:00Z
---

## Context

All new brain/context-intelligence types live in a single new file `internal/core/domain/brain.go`. This is the foundation — ports, services, and adapters all depend on these types. No other files are modified in this task.

## Files to Read

- `internal/core/domain/task.go` — understand existing domain file style (doc comments, package name, import patterns)
- `internal/core/domain/session.go` — understand how secondary domain types are structured

## Implementation Steps

1. Create `internal/core/domain/brain.go` with package `domain`

2. Define `KnowledgeKind` as `type KnowledgeKind string` with 6 constants:

   ```
   KnowledgeArchitecture KnowledgeKind = "architecture"
   KnowledgeConvention   KnowledgeKind = "convention"
   KnowledgeFileMap      KnowledgeKind = "file_map"
   KnowledgeLearning     KnowledgeKind = "learning"
   KnowledgeGlossary     KnowledgeKind = "glossary"
   KnowledgeTaskSummary  KnowledgeKind = "task_summary"
   ```

3. Define `ProjectKnowledge` struct:

   ```
   ID, ProjectPath, Kind KnowledgeKind, Topic, Content, Source string
   TokenCount int
   RelevanceScore float64  // 0.0-1.0, default 0.5
   CreatedAt, UpdatedAt time.Time
   ```

   JSON tags: camelCase (id, projectPath, kind, topic, content, source, tokenCount, relevanceScore, createdAt, updatedAt). Source has `omitempty`.

4. Define `ContextQuery` struct:

   ```
   ProjectPath, TaskID, Question, FocusArea string
   MaxTokens int
   ```

   JSON tags: camelCase. TaskID, Question, FocusArea have `omitempty`.

5. Define `ContextSection` struct:

   ```
   Kind KnowledgeKind, Topic, Content string
   Tokens int
   ```

   JSON tags: camelCase.

6. Define `ContextResponse` struct:

   ```
   ProjectPath string
   Sections []ContextSection
   SuggestedFiles []string  // omitempty
   TokensUsed, TokenBudget int
   Truncated bool
   ```

   JSON tags: camelCase. SuggestedFiles has `omitempty`.

7. Define `BrainStatus` struct:

   ```
   ProjectPath string
   Initialized bool
   EntryCount int
   KindCounts map[string]int
   TotalTokens int
   LastUpdated time.Time  // omitempty
   ```

   JSON tags: camelCase. LastUpdated has `omitempty`.

8. Add doc comment to each type explaining its purpose (1 sentence).
9. Only import: `"time"`.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... 2>&1` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `internal/core/domain/brain.go` exists with all 7 types (1 type alias + 6 structs)
- [ ] All 6 KnowledgeKind constants defined
- [ ] No imports beyond `"time"`

## Anti-patterns to Avoid

- NEVER import adapters from domain
- NEVER add methods to domain structs — keep them pure data containers
- NEVER use `interface{}` or `any` — all fields are concrete types
