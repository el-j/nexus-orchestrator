---
id: TASK-515
title: Add brain MCP tools
role: mcp
planId: PLAN-066
status: done
dependencies: [TASK-513]
createdAt: 2026-04-13T00:00:00Z
---

## Context

Six new MCP tools expose BrainService via the JSON-RPC 2.0 MCP server. The pattern exactly mirrors existing tools: add to the `Server` struct, register in `toolList()`, dispatch in `handleToolCall()` switch, implement `tool*()` methods. Also update `howto_brief` to reflect that `get_project_context` and `brain_init` are now active tools (they were already mentioned as future tools).

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — ALL OF IT. Understand Server struct, toolList(), handleToolCall() switch, how existing tools unmarshal args and marshal results
- `internal/adapters/inbound/mcp/server.go` — Server struct definition, constructor, how services are injected (e.g. how activitySvc was added)
- `internal/core/ports/brain.go` — BrainService interface (TASK-510)
- `internal/core/domain/brain.go` — domain types to marshal/unmarshal (TASK-509)

## Implementation Steps

1. **Add brain field to Server struct** in `server.go` (or `tools.go` — check where Server is defined):

   ```go
   brain ports.BrainService
   ```

   Add method: `func (s *Server) SetBrainService(b ports.BrainService) { s.brain = b }`

2. **Add 6 entries to `toolList()`** (or equivalent tool registration function). Each entry needs: name, description, and inputSchema (JSON Schema object). Follow the exact format of existing tools:

   a. `brain_init` — params: `project_path` (string, required), `claude_md_path` (string, optional). Description: "Initialize the project brain by ingesting CLAUDE.md into the knowledge base. Call once per project to enable get_project_context. Auto-detects CLAUDE.md in project root if claude_md_path is omitted."

   b. `get_project_context` — params: `project_path` (string, required), `max_tokens` (integer, optional, default 400), `focus` (string, optional, enum: architecture/conventions/files/all). Description: "Get the minimal context needed to work in this project. Returns architecture, conventions, and relevant files within a tight token budget. Call FIRST when starting work on a project. Replaces reading CLAUDE.md manually."

   c. `get_focused_context` — params: `project_path` (string, required), `question` (string, required), `max_tokens` (integer, optional, default 400). Description: "Search project knowledge for information relevant to a specific question. BM25 full-text search within token budget."

   d. `ingest_knowledge` — params: `project_path` (string, required), `kind` (string, required, enum: architecture/convention/file_map/learning/glossary/task_summary), `topic` (string, required), `content` (string, required), `source` (string, optional). Description: "Teach nexus about this project: store a convention, architecture decision, learning, or file map. Upserts by (project_path, kind, topic)."

   e. `get_file_map` — params: `project_path` (string, required), `focus_area` (string, optional). Description: "Get the list of files relevant to a focus area. Returns file paths from the project knowledge base."

   f. `search_knowledge` — params: `project_path` (string, required), `query` (string, required), `max_tokens` (integer, optional, default 400). Description: "BM25 full-text search across all project knowledge within a token budget."

3. **Add 6 cases to `handleToolCall()` switch**:

   ```
   case "brain_init": return s.toolBrainInit(ctx, args)
   case "get_project_context": return s.toolGetProjectContext(ctx, args)
   case "get_focused_context": return s.toolGetFocusedContext(ctx, args)
   case "ingest_knowledge": return s.toolIngestKnowledge(ctx, args)
   case "get_file_map": return s.toolGetFileMap(ctx, args)
   case "search_knowledge": return s.toolSearchKnowledge(ctx, args)
   ```

4. **Implement 6 `tool*()` methods** on `*Server`. Each follows this pattern:
   - Guard: `if s.brain == nil { return nil, fmt.Errorf("brain service not initialized") }`
   - Unmarshal args into a local struct
   - Call `s.brain.MethodName(ctx, ...)`
   - Marshal result to JSON: `json.Marshal(result)`
   - Return as `[]mcp.Content` (check how existing tools return JSON results — follow same pattern)

   Key arg extraction:
   - `brain_init`: `projectPath` (required), `claudeMDPath` (optional, default "")
   - `get_project_context`: `projectPath` (required), `maxTokens` (optional, default 400 if 0), build `domain.ContextQuery{ProjectPath: projectPath, MaxTokens: maxTokens, FocusArea: focus}`
   - `get_focused_context`: `projectPath` (required), `question` (required), `maxTokens` (optional, default 400)
   - `ingest_knowledge`: `projectPath`, `kind`, `topic`, `content`, `source` (optional) → build `domain.ProjectKnowledge` and call `IngestKnowledge`
   - `get_file_map`: `projectPath`, `focusArea` (optional, default "")
   - `search_knowledge`: `projectPath`, `query`, `maxTokens` (optional, default 400)

5. **Update `toolHowtoBrief()`**: In the static string, find the section mentioning `get_project_context` and `get_focused_context` as "coming soon" or "not yet implemented" — update to mark them active. Add `brain_init` to FIRST STEPS. Make the call pattern explicit:
   ```
   FIRST STEPS (brain-aware workflow):
   1. brain_init({project_path})           ← once per project, ingest CLAUDE.md
   2. get_project_context({project_path})  ← get minimal context (~400 tokens)
   3. register_session + claim_task        ← pick up work
   4. update_task_status when done
   ```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All 6 tools appear in `toolList()` / tool registration
- [ ] All 6 cases present in `handleToolCall()` switch
- [ ] Each tool method guards against nil brain service
- [ ] `howto_brief` text accurately reflects brain tools as active

## Anti-patterns to Avoid

- NEVER call `s.brain.*` without nil guard
- NEVER swallow JSON marshal errors — wrap and return them
- NEVER add tool logic inline in the switch — always delegate to `tool*()` methods
