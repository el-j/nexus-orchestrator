---
id: TASK-520
title: Update howto, well-known, and agent workflow docs
role: backend
planId: PLAN-066
status: done
dependencies: [TASK-518]
createdAt: 2026-04-13T00:00:00Z
---

## Context

The static howto strings and the discovery endpoint need to reflect the new brain tools accurately. This is the final task — it makes nexus "self-documenting" about the brain feature so agents discover it automatically on first connection.

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — `toolHowto()` and `toolHowtoBrief()` static strings — read them in full
- `internal/adapters/inbound/httpapi/howto.go` — `buildHowToDoc()` structure, `AIWorkflows`, `Endpoints`, `Capabilities`
- `internal/adapters/inbound/httpapi/server.go` — `handleWellKnownNexus` (check if it's here or in howto.go)

## Implementation Steps

1. **Update `toolHowtoBrief()`** in `tools.go`:

   Replace or update the FIRST STEPS section to the brain-aware workflow:

   ```
   === FIRST STEPS (brain-aware) ===

   New session in a project:
   1. brain_init(project_path)               -- once per project, ingest CLAUDE.md
   2. get_project_context(project_path)       -- ~400 tokens of architecture + conventions
   3. register_session(agent_name, project)   -- register this agent
   4. claim_task(task_id, session_id)         -- pick up work
   5. [do the work]
   6. ingest_knowledge(...)                   -- optionally teach nexus what you learned
   7. update_task_status(task_id, session_id, "COMPLETED", logs)

   === BRAIN TOOLS (token-efficient context) ===
   get_project_context    get minimal project context (~400 tokens)
   get_focused_context    search knowledge for a specific question
   brain_init             bootstrap from CLAUDE.md
   ingest_knowledge       teach nexus about project conventions/learnings
   search_knowledge       BM25 keyword search with token budget
   get_file_map           get relevant files for a focus area
   ```

2. **Update `toolHowto()`** in `tools.go`:
   - Find the tool list section. Add all 6 brain tools with their parameter descriptions
   - Add a "Project Brain" section to the workflow overview:
     ```
     Project Brain: brain_init → get_project_context reduces agent token usage by 90%
     vs manually reading CLAUDE.md. Run brain_init once per project, then every agent
     session starts with get_project_context instead of loading files.
     ```
   - Update the total tool count in the header if it mentions "N tools"

3. **Update `buildHowToDoc()`** in `howto.go`:
   - Add 9 brain endpoints to the `Endpoints` slice:
     ```go
     {Method: "POST", Path: "/api/brain/init", Description: "Initialize project brain from CLAUDE.md"},
     {Method: "POST", Path: "/api/brain/context", Description: "Get minimal project context (ContextQuery → ContextResponse)"},
     {Method: "POST", Path: "/api/brain/focused", Description: "Focused knowledge search (question → ContextResponse)"},
     {Method: "POST", Path: "/api/brain/knowledge", Description: "Ingest a knowledge entry (upserts by project+kind+topic)"},
     {Method: "GET",  Path: "/api/brain/knowledge", Description: "List knowledge entries (project, kind query params)"},
     {Method: "DELETE", Path: "/api/brain/knowledge/{id}", Description: "Delete a knowledge entry"},
     {Method: "POST", Path: "/api/brain/search", Description: "BM25 full-text search with token budget"},
     {Method: "GET",  Path: "/api/brain/status", Description: "Brain status for a project (Initialized, EntryCount, TotalTokens)"},
     {Method: "GET",  Path: "/api/brain/files", Description: "File map entries for a focus area"},
     ```
   - Add a `BrainWorkflow` entry to `AIWorkflows` (or equivalent):
     ```
     Name: "Brain-first agent workflow"
     Steps: ["brain_init (once per project)", "get_project_context (per session)", "claim_task + execute", "ingest_knowledge (learnings)"]
     TokenSavings: "~90% reduction vs reading CLAUDE.md manually (~400 tokens vs 3000+)"
     ```

4. **Update `handleWellKnownNexus`**:
   - Find the `Capabilities` slice or map
   - Add `"project-brain"` if not already added in TASK-518
   - Verify the JSON output includes it by reading the handler carefully

5. **Update `CLAUDE.md`** at project root:
   - Add a "Brain / Context Intelligence" section describing:
     - What `nexus brain` does
     - Workflow: brain_init once, get_project_context per session
     - Available MCP tools and HTTP endpoints
     - CLI: `nexus brain init/status/context/search`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `toolHowtoBrief()` includes `brain_init` and `get_project_context` in FIRST STEPS
- [ ] `toolHowto()` lists all 6 brain tools
- [ ] `buildHowToDoc()` includes all 9 brain HTTP endpoints
- [ ] `/.well-known/nexus.json` capabilities includes `"project-brain"`
- [ ] CLAUDE.md project root has Brain section

## Anti-patterns to Avoid

- NEVER change the semantics of other tools while updating howto strings
- NEVER delete existing workflow documentation — only add/update
- NEVER hardcode endpoint counts ("N tools") if the real count changes — compute dynamically or update carefully
