---
id: PLAN-066
title: Nexus Brain — Project Context Intelligence Layer
status: approved
createdAt: 2026-04-13T00:00:00Z
goal: >
  Make nexusOrchestrator the "brain" for any AI agent working in a project.
  Instead of agents loading large CLAUDE.md files and scanning codebases
  themselves (thousands of tokens), the agent queries nexus and gets back
  exactly the minimal, targeted context it needs (200–500 tokens max).
  nexus stores, indexes, and serves project knowledge on demand via MCP tools,
  HTTP endpoints, and CLI commands.
taskIds:
  - TASK-509
  - TASK-510
  - TASK-511
  - TASK-512
  - TASK-513
  - TASK-514
  - TASK-515
  - TASK-516
  - TASK-517
  - TASK-518
  - TASK-519
  - TASK-520
---

## Problem

- `howto` and `howto_brief` are hardcoded static strings — not project-aware
- `get_project_context` and `get_focused_context` are mentioned in `howto_brief` but **not implemented**
- No SQLite tables exist for project knowledge, conventions, file maps, or learnings
- Agents must load entire CLAUDE.md and scan codebases themselves → thousands of wasted tokens
- Token waste degrades speed, increases cost, and reduces agent accuracy

## Solution

A `BrainService` — a parallel core service (same pattern as `ActivityService`) with:

1. **Knowledge store**: new `project_knowledge` SQLite table with FTS5 full-text search (BM25)
2. **Ingestion**: parse CLAUDE.md into typed knowledge entries (architecture, convention, file_map, learning, glossary)
3. **Token-budget assembly**: `get_project_context` fills a response in priority order until the token budget (default 400) is exhausted
4. **6 new MCP tools**: `brain_init`, `get_project_context`, `get_focused_context`, `ingest_knowledge`, `get_file_map`, `search_knowledge`
5. **HTTP API**: `/api/brain/*` endpoints mirroring MCP tools
6. **CLI**: `nexus brain init/ingest/search/list/status/context`

## Architecture Decision

`BrainService` is a separate service at `internal/core/services/brain_service.go` with its own port interface (`ports.BrainService`) and repository port (`ports.ProjectKnowledgeRepository`). It does NOT extend the existing Orchestrator port — that already has 50+ methods. This follows the exact precedent of `ActivityService`: standalone service, own repository, wired at daemon startup without touching the Orchestrator interface. The MCP and HTTP adapters receive `BrainService` as an additional dependency alongside `ports.Orchestrator`.

## Token Optimization Strategy

GetContext priority order within budget:

1. Architecture overview → ~80 tokens
2. Relevant conventions (filtered to focus area) → ~100 tokens
3. Suggested file map → ~60 tokens
4. Recent task learnings → ~80 tokens
5. Glossary/domain terms → ~60 tokens
   Total: ~380 tokens vs 3000+ for raw CLAUDE.md

## Implementation Waves

```
Wave 1 (parallel):  TASK-509 (domain types)
Wave 2 (parallel):  TASK-510 (ports)
Wave 3 (parallel):  TASK-511 (SQLite repo) + TASK-512 (repo tests)
Wave 4 (parallel):  TASK-513 (BrainService) + TASK-514 (service tests)
Wave 5 (parallel):  TASK-515 (MCP tools) + TASK-516 (HTTP handlers) + TASK-517 (CLI commands)
Wave 6 (parallel):  TASK-518 (daemon wiring)
Wave 7 (parallel):  TASK-519 (E2E tests) + TASK-520 (howto docs)
```

## Acceptance Criteria

- [ ] `brain_init "path/to/project"` ingests CLAUDE.md and returns BrainStatus
- [ ] `get_project_context` returns ≤400 tokens of ranked knowledge
- [ ] `get_focused_context "how does error handling work"` returns relevant conventions
- [ ] `ingest_knowledge` upserts by (project_path, kind, topic)
- [ ] FTS5 BM25 search returns ranked results for keyword queries
- [ ] All 6 MCP tools appear in `toolList()` and handle errors correctly
- [ ] All 9 HTTP endpoints respond with correct status codes
- [ ] `nexus brain` CLI subcommand tree fully functional
- [ ] Daemon starts without error with BrainService wired
- [ ] `go vet ./...` passes, `CGO_ENABLED=1 go test -race -count=1 ./...` passes
- [ ] `howto_brief` correctly references `brain_init` and `get_project_context` as active tools
