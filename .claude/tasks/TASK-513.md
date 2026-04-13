---
id: TASK-513
title: Implement BrainService
role: backend
planId: PLAN-066
status: done
dependencies: [TASK-511]
createdAt: 2026-04-13T00:00:00Z
---

## Context

`BrainService` is the core of the brain feature. It assembles token-budget-constrained context responses by pulling from the knowledge repository, implements BM25-driven focused search, and ingests CLAUDE.md-style markdown into structured knowledge entries. No goroutines — all methods are synchronous per project convention.

## Files to Read

- `internal/core/ports/brain.go` — the interface to implement (TASK-510)
- `internal/core/domain/brain.go` — all domain types (TASK-509)
- `internal/core/ports/ports.go` — TaskRepository interface (needed for task-aware context)
- `internal/core/services/orchestrator.go` — understand service constructor patterns, no goroutines in services

## Implementation Steps

1. Create `internal/core/services/brain_service.go`, package `services`

2. Define struct:

   ```go
   type BrainServiceImpl struct {
       repo     ports.ProjectKnowledgeRepository
       taskRepo ports.TaskRepository
   }
   ```

3. Constructor: `func NewBrainService(repo ports.ProjectKnowledgeRepository, taskRepo ports.TaskRepository) *BrainServiceImpl`

4. Compiler assertion: `var _ ports.BrainService = (*BrainServiceImpl)(nil)`

5. Private helper `estimateTokens(text string) int`:
   - Returns `max(1, len(text)/4)` — simple heuristic matching GPT-family tokenization
6. Private helper `classifySection(heading, content string) domain.KnowledgeKind`:
   - Normalize heading to lowercase
   - Keyword matching (in order of check):
     - Contains "architecture" or "design" or "hexagonal" or "layer" → KnowledgeArchitecture
     - Contains "convention" or "style" or "rule" or "pattern" or "standard" or "error" → KnowledgeConvention
     - Contains "file" or "structure" or "directory" or "map" or "path" or "layout" → KnowledgeFileMap
     - Contains "learning" or "lesson" or "discovered" or "pitfall" or "avoid" → KnowledgeLearning
     - Contains "glossary" or "term" or "definition" or "vocabulary" → KnowledgeGlossary
     - Default → KnowledgeConvention (most common for CLAUDE.md sections)

7. Implement `GetContext(ctx, q ContextQuery) (ContextResponse, error)`:
   - If q.MaxTokens == 0, set to 400
   - Load all entries: `repo.GetByProject(ctx, filepath.Clean(q.ProjectPath))`
   - Group by KnowledgeKind into a map
   - Priority order: Architecture, Convention, FileMap, Learning, Glossary, TaskSummary
   - For each kind group, sort by RelevanceScore descending (sort.Slice)
   - If q.FocusArea != "" and q.FocusArea != "all": filter Convention and FileMap entries to those whose Topic contains q.FocusArea (strings.Contains, case-insensitive)
   - Accumulate into response.Sections. For each entry: if tokensUsed + entry.TokenCount <= MaxTokens → add as ContextSection, add to tokensUsed. Else: set Truncated=true, break (stop adding any more entries)
   - Populate SuggestedFiles: collect Content from all FileMap entries that were included
   - Return ContextResponse

8. Implement `GetFocusedContext(ctx, q ContextQuery) (ContextResponse, error)`:
   - If q.MaxTokens == 0, set to 400
   - Call `repo.SearchFTS(ctx, filepath.Clean(q.ProjectPath), q.Question, 20)`
   - Accumulate results into Sections within token budget (same logic as GetContext accumulation)
   - Return ContextResponse

9. Implement `IngestKnowledge(ctx, k ProjectKnowledge) (ProjectKnowledge, error)`:
   - `filepath.Clean` on k.ProjectPath
   - If k.ID == "", generate: `k.ID = uuid.New().String()`
   - Compute: `k.TokenCount = estimateTokens(k.Content)`
   - If k.RelevanceScore == 0, set to 0.5
   - Set k.CreatedAt = time.Now() (if zero), k.UpdatedAt = time.Now()
   - Call `repo.SaveKnowledge(ctx, k)` — repo handles upsert via unique constraint
   - Return k (with populated ID and timestamps)

10. Implement `IngestFromFile(ctx, projectPath, filePath string) (int, error)`:
    - Read file: `os.ReadFile(filePath)`
    - Split on `\n## ` regex to get sections (first split gives preamble before first ##)
    - For each section: first line is the heading (before first \n), rest is content
    - Skip sections with empty content (< 20 chars)
    - Call `classifySection(heading, content)` for kind
    - topic = strings.TrimSpace(heading)
    - source = `"file:" + filepath.Base(filePath)`
    - Call `IngestKnowledge(ctx, domain.ProjectKnowledge{ProjectPath: projectPath, Kind: kind, Topic: topic, Content: content, Source: source})`
    - Count successfully ingested entries, return count

11. Implement `SearchKnowledge(ctx, projectPath, query string, maxTokens int) ([]ContextSection, error)`:
    - If maxTokens == 0, set to 400
    - Call `repo.SearchFTS(ctx, filepath.Clean(projectPath), query, 20)`
    - Accumulate ContextSection entries within maxTokens budget
    - Return sections

12. Implement `GetFileMap(ctx, projectPath, focusArea string) ([]string, error)`:
    - `repo.GetByProjectAndKind(ctx, filepath.Clean(projectPath), domain.KnowledgeFileMap)`
    - If focusArea != "": filter entries whose Topic contains focusArea (case-insensitive)
    - Return slice of Content values (each is a file path or file list)

13. Implement `InitProject(ctx, projectPath, claudeMDPath string) (BrainStatus, error)`:
    - If claudeMDPath == "": try `filepath.Join(projectPath, "CLAUDE.md")` and `filepath.Join(projectPath, ".claude/CLAUDE.md")`; use first one that exists (`os.Stat` check)
    - If a CLAUDE.md found: call `IngestFromFile(ctx, projectPath, claudeMDPath)` — wrap error
    - Return `repo.GetStatus(ctx, filepath.Clean(projectPath))`

14. Implement `GetStatus(ctx, projectPath string) (BrainStatus, error)`:
    - Delegates to `repo.GetStatus(ctx, filepath.Clean(projectPath))`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `BrainServiceImpl` implements `ports.BrainService` (compiler assertion)
- [ ] All 8 interface methods implemented
- [ ] GetContext respects MaxTokens (never exceeds budget)
- [ ] GetContext fills Truncated=true when budget is hit
- [ ] IngestFromFile splits on `## ` headings correctly
- [ ] No goroutines anywhere in the service

## Anti-patterns to Avoid

- NEVER import adapter packages from the service layer
- NEVER use goroutines or sync primitives in service methods
- NEVER bypass `filepath.Clean` on project paths
- NEVER build SQL in the service — delegate all storage to the repo
