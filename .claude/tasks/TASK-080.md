---
id: TASK-080
title: "Pages: Architecture + API Reference docs"
role: devops
planId: PLAN-009
status: todo
dependencies: [TASK-078]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
Two critical documentation pages: the architecture overview explaining the hexagonal design, and a comprehensive API reference for both HTTP REST and MCP endpoints.

## Files to Read
- `internal/core/ports/ports.go`
- `internal/adapters/inbound/httpapi/server.go`
- `internal/adapters/inbound/mcp/server.go`
- `internal/core/domain/task.go`
- `internal/core/domain/session.go`
- `internal/core/domain/provider.go`
- `.github/copilot-instructions.md`

## Implementation Steps
1. Create `docs/architecture.md`:
   ```yaml
   ---
   layout: default
   title: Architecture
   nav_order: 2
   ---
   ```
   Content:
   - **Hexagonal Architecture** overview with ASCII diagram showing: `inbound adapters → core services → ports ← outbound adapters`
   - **Domain Layer**: Task, Session, Message, ProviderConfig types
   - **Port Contracts**: Orchestrator, LLMClient, TaskRepository, FileWriter, SessionRepository
   - **Inbound Adapters**: HTTP API, MCP Server, CLI, Wails GUI, System Tray
   - **Outbound Adapters**: LM Studio, Ollama, OpenAI-compat, Anthropic, SQLite, Filesystem
   - **Entry Points**: Desktop GUI, Headless Daemon, CLI Client — table with binary, path, purpose
   - **Concurrency Model**: Mutex-protected queue, background workers, stopCh shutdown

2. Create `docs/api-reference.md`:
   ```yaml
   ---
   layout: default
   title: API Reference
   nav_order: 3
   ---
   ```
   Content:
   - **HTTP REST API** (base: `http://localhost:9999`)
     - `POST /api/tasks` — Submit task (request/response JSON)
     - `GET /api/tasks` — List all tasks
     - `GET /api/tasks/{id}` — Get task by ID
     - `DELETE /api/tasks/{id}` — Cancel task
     - `GET /api/providers` — List providers
     - `POST /api/providers` — Register cloud provider
     - `DELETE /api/providers/{name}` — Remove provider
     - `GET /api/providers/{name}/models` — List provider models
     - `GET /api/events` — SSE event stream
     - `GET /api/health` — Health check
   - **MCP Server** (base: `http://localhost:9998/mcp`)
     - JSON-RPC 2.0 protocol, version "2024-11-05"
     - Tools: submit_task, get_task, get_queue, cancel_task, get_providers, health
     - Request/response examples for each tool
   - **Task Lifecycle** states: QUEUED → PROCESSING → COMPLETED/FAILED/CANCELLED/TOO_LARGE/NO_PROVIDER
   - **Environment Variables** table

## Acceptance Criteria
- [ ] `docs/architecture.md` exists with hexagonal architecture overview
- [ ] `docs/api-reference.md` exists with all HTTP and MCP endpoints documented
- [ ] ASCII architecture diagram included
- [ ] Request/response JSON examples for key endpoints
- [ ] Task lifecycle states documented
- [ ] No Go source files modified

## Anti-patterns to Avoid
- NEVER modify any Go source files
- NEVER include inaccurate endpoint paths — check server.go
