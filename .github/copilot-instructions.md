# nexus-orchestrator – Project Guidelines

## Architecture

Hexagonal architecture (Ports & Adapters). The dependency rule flows strictly inward:

```
inbound adapters → core services → ports ← outbound adapters
```

- **`internal/core/domain/`** — Pure domain types (`Task`, `TaskStatus`, `Session`, `Message`). No framework imports.
- **`internal/core/ports/`** — Go interfaces only: `LLMClient`, `TaskRepository`, `FileWriter`, `SessionRepository`, `Orchestrator`. Nothing concrete here.
- **`internal/core/services/`** — Business logic. Depends only on ports. Never import adapters directly.
- **`internal/adapters/inbound/`** — CLI (Cobra), HTTP API (chi), MCP JSON-RPC 2.0 (`mcp`), Wails GUI binding, system tray.
- **`internal/adapters/outbound/`** — LM Studio (`llm_lmstudio`), Ollama (`llm_ollama`), SQLite (`repo_sqlite`), filesystem (`fs_writer`).

Entry points:
| Binary | Path | Purpose |
|--------|------|---------|
| Desktop GUI | `main.go` + `app.go` | Wails window + embedded HTTP API on `:63987` + MCP on `:63988` |
| Headless daemon | `cmd/nexus-daemon/main.go` | HTTP API on `:63987` + MCP on `:63988` (for servers / background) |
| CLI client | `cmd/nexus-cli/main.go` | Thin HTTP client → daemon at `127.0.0.1:63987` |

## Build & Test

```sh
# Build
go build ./cmd/nexus-cli/...
go build ./cmd/nexus-daemon/...

# Run desktop app (requires Wails)
wails dev          # hot-reload dev
go run main.go     # production build

# Test (CGO required — sqlite3)
CGO_ENABLED=1 go test -race ./...
CGO_ENABLED=1 go test ./internal/core/services/...   # unit tests only

# Lint / vet
go vet ./...
```

No Makefile — use plain `go` toolchain commands.

## Conventions

### Error handling

- Wrap errors with `fmt.Errorf("package: ...: %w", err)` — prefix with the package name.
  ```go
  return fmt.Errorf("orchestrator: process task: %w", err)
  return fmt.Errorf("sqlite: save task: %w", err)
  ```
- Use `log.Printf` for operational logging; `fmt.Fprintln(os.Stderr, ...)` for fatal startup errors.

### Concurrency

- Protect shared state with `sync.Mutex`. The `OrchestratorService` queue is a canonical example.
- Background workers communicate shutdown via a `stopCh chan struct{}` channel.
- Do not use goroutines inside core services — that is an infrastructure concern (inbound adapters own goroutine lifecycle).

### HTTP API

- Router: `github.com/go-chi/chi/v5` with `middleware.Logger` and `middleware.Recoverer`.
- All task endpoints live under `/api/tasks`: `POST`, `GET`, `DELETE /api/tasks/{id}`.
- JSON in/out. Return proper HTTP status codes (`201 Created`, `404 Not Found`, etc.).
- **Discovery beacon**: `GET /.well-known/nexus.json` — lightweight JSON doc with schema_version, API URLs, MCP info, and capabilities. First stop for any AI or tool discovering the server.
- **Integration guide**: `GET /api/howto` — full machine-readable doc: quick-start steps, connection URLs (dynamically resolved to request host), AI workflow patterns, all HTTP endpoints list, and cURL examples. Call this before doing anything else.

### MCP Server

- Adapter: `internal/adapters/inbound/mcp/server.go`, `sse.go`, `tools.go`
- Protocol: JSON-RPC 2.0, version `"2024-11-05"`, with Streamable HTTP (2025-03-26+) compatibility
- Default address: `:63988` (override with `NEXUS_MCP_ADDR`)
- Transports:
  - **HTTP POST** (`POST /mcp`): Direct JSON-RPC request/response (Streamable HTTP compatible)
  - **Legacy SSE** (`GET /sse` + `POST /messages?sessionId=...`): 2024-11-05 HTTP+SSE transport for Continue IDE, Cursor, etc.
  - **stdio proxy** (`cmd/nexus-mcp-stdio`): Subprocess bridge for Claude Desktop and other stdio-only clients
- Standard methods: `initialize`, `ping`, `tools/list`, `tools/call`, `resources/list`, `prompts/list`
- 36 tools: `howto` / `howto_brief` (call first!), task CRUD, provider management, AI session lifecycle, backlog planning, agent discovery, brain knowledge management (`ingest_knowledge`, `search_knowledge`, `get_project_context`, `get_focused_context`, `get_brain_status`)
- Security: Origin validation (localhost/127.0.0.1 allowed, foreign origins blocked), CORS preflight support
- `initialize` response includes `Mcp-Session-Id` header and `serverInfo.Instructions`
- `NewMcpServer(orch, brain) *Server` — registers `/mcp`, `/health`, `/sse`, `/messages` handlers; `*Server` implements `http.Handler`
- `StartMCPServer(ctx, orch, brain, addr)` — runs server with graceful shutdown on context cancellation (no WriteTimeout for SSE)

### Session Isolation

- Each project (`domain.Task.ProjectPath`) gets its own conversation history (`domain.Session`)
- `LLMClient.Chat([]domain.Message)` is the multi-turn method; `GenerateCode()` is the single-shot fallback
- `SessionRepository` (port) is implemented by `repo_sqlite.SessionRepo` (separate from `Repository` to avoid `Save()` conflict)
- `NewSessionRepo(r *Repository) *SessionRepo` — shares the same `*sql.DB` as the task repo
- `OrchestratorService` falls back to `GenerateCode()` when `sessionRepo == nil` (useful in tests)

### Configuration

- Prefer environment variables over flags for daemon config:
  - `NEXUS_DB_PATH` — SQLite database file path (default: `nexus.db`)
  - `NEXUS_LISTEN_ADDR` — HTTP API listen address (default: `:63987`)
  - `NEXUS_MCP_ADDR` — MCP server listen address (default: `:63988`)
- Provider base URLs are configurable via env vars (all default to localhost):
  - `NEXUS_LMSTUDIO_URL` — LM Studio API root (default: `http://127.0.0.1:1234/v1`)
  - `NEXUS_OLLAMA_URL` — Ollama API root (default: `http://127.0.0.1:11434`)
  - `NEXUS_ANTIGRAVITY_URL` — Antigravity (Google) API root (default: `http://127.0.0.1:4315/v1`)
  - `NEXUS_OPENAI_API_KEY` — enables OpenAI adapter; `NEXUS_OPENAI_MODEL` sets model (default `gpt-4o-mini`)
  - `NEXUS_GITHUBCOPILOT_TOKEN` — enables GitHub Copilot adapter
  - `NEXUS_ANTHROPIC_API_KEY` — enables Anthropic/Claude adapter; `NEXUS_ANTHROPIC_MODEL` sets model
- Antigravity is a Google-backed desktop AI app exposing an OpenAI-compatible API at port `4315`. It is always registered in `buildProviders()` (like LM Studio/Ollama) and wired via `llm_openaicompat`. `ProviderKindDesktopApp`, `ProviderKindLocalAI`, `ProviderKindVLLM`, and `ProviderKindTextGenUI` all route through `llm_openaicompat` in `buildProviderFromConfig`.

### Adding a new LLM provider

1. Create `internal/adapters/outbound/llm_<name>/adapter.go`.
2. Implement the `ports.LLMClient` interface (`Ping`, `ProviderName`, `GetAvailableModels`, `GenerateCode`, `Chat`).
3. Register the adapter in `DiscoveryService` (pass it alongside existing clients).
4. Wire it up in all three entry points (`main.go`, `cmd/nexus-daemon/main.go`).

### Adding a new inbound interface

1. Create `internal/adapters/inbound/<name>/`.
2. Accept `ports.Orchestrator` as a dependency — never a concrete service type.

## Key Reference Files

- Domain model: [internal/core/domain/task.go](../internal/core/domain/task.go)
- Session domain types: [internal/core/domain/session.go](../internal/core/domain/session.go)
- Port contracts: [internal/core/ports/ports.go](../internal/core/ports/ports.go)
- Worker loop pattern: [internal/core/services/orchestrator.go](../internal/core/services/orchestrator.go)
- LLM adapter example: [internal/adapters/outbound/llm_lmstudio/adapter.go](../internal/adapters/outbound/llm_lmstudio/adapter.go)
- HTTP API: [internal/adapters/inbound/httpapi/server.go](../internal/adapters/inbound/httpapi/server.go)
- MCP adapter: [internal/adapters/inbound/mcp/server.go](../internal/adapters/inbound/mcp/server.go)
- Session repository: [internal/adapters/outbound/repo_sqlite/session_repo.go](../internal/adapters/outbound/repo_sqlite/session_repo.go)
- Wails binding: [app.go](../app.go)

## AI Orchestrator Workflow (Copilot / Agents using the MCP)

When the nexus-orchestrator MCP is available (`:63988`), **use it as the primary source of truth** for all planning and orchestration. The MCP tools replace ad-hoc task tracking.

> **VS Code MCP Connection** — In `~/Library/Application Support/Code/User/mcp.json` use
> `"type": "http"` (Streamable HTTP, stateless) **not** `"type": "sse"`. The SSE transport
> holds a session in memory; a daemon restart invalidates it causing `400` errors and
> "terminated / Failed to parse message" log noise. Correct config:
>
> ```json
> { "servers": { "Nexus Orchestrator": { "type": "http", "url": "http://127.0.0.1:63988/mcp" } } }
> ```

### Session Startup (every conversation)

```
1. mcp_nexus_orchest_register_session  — identify yourself (agent_name, external_id, project_path)
2. mcp_nexus_orchest_get_queue         — check for pending tasks to claim
3. mcp_nexus_orchest_get_discovered_plans  — get current plan file state for the project
4. mcp_nexus_orchest_get_backlog       — see drafts awaiting promotion
```

### Planning Workflow

```
1. Create .claude/plans/PLAN-NNN.md and .claude/tasks/TASK-NNN.md for each plan/task (source of truth)
2. mcp_nexus_orchest_create_draft      — mirror tasks into nexus queue as DRAFT (priority=int)
3. mcp_nexus_orchest_promote_task      — move DRAFT → QUEUED when ready to execute
4. mcp_nexus_orchest_update_task_status — mark COMPLETED or FAILED with logs when done
```

### Execution Workflow (AI Worker)

```
1. mcp_nexus_orchest_get_queue         — see claimable QUEUED tasks
2. mcp_nexus_orchest_claim_task        — take ownership (prevents duplicate work)
3. Execute with your own reasoning capabilities
4. mcp_nexus_orchest_update_task_status — COMPLETED | FAILED with result summary
5. mcp_nexus_orchest_heartbeat_ai_session — send periodically to stay visible
```

### Orchestration Commands

```
mcp_nexus_orchest_get_ai_sessions         — see all registered AI sessions
mcp_nexus_orchest_purge_disconnected_sessions — clean up stale idle sessions
mcp_nexus_orchest_get_providers           — check which LLM backends are active
mcp_nexus_orchest_get_discovered_agents   — list detected AI tools (Copilot, Continue, etc.)
mcp_nexus_orchest_delegate_to_nexus       — get delegation instructions for a session
```

### Key Conventions

- `external_id` for Copilot sessions: `"copilot-vscode-{project-basename}"` — ensures idempotent re-registration
- `priority` is an integer (1 = highest); the `create_draft` tool requires int, not string
- Always call `register_session` at conversation start AND as a periodic heartbeat (every ~5 min)
- `.claude/plans/PLAN-NNN.md` and `.claude/tasks/TASK-NNN.md` remain the human-readable source of truth; MCP queue is the machine-readable execution layer
- After completing work, call `mcp_nexus_orchest_update_task_status` with the draft ID returned by `create_draft`
- **IMPORTANT**: Only call `promote_task` for tasks that should be **executed by an LLM provider** (code generation). The nexus worker immediately dispatches promoted tasks. For Copilot's own planning/validation tasks, keep as DRAFT — record completion in `.claude/orchestrator.json` directly.

## Potential Pitfalls

- `go-sqlite3` requires CGO — ensure `CGO_ENABLED=1` and a C compiler (`gcc`/`clang`) are available.
- The frontend JS source is **not** in this repo; `frontend/dist/` contains pre-compiled assets only. Don't run `npm install` here.
- The CLI binary is a **remote client** — it makes HTTP calls to a running daemon/desktop app. It does not link core services directly.
- HTTP timeout on LM Studio is 60 s; Ollama is 120 s. For large prompts, be aware of these limits when testing adapters.
- `memRepo` / test stubs that share state with the orchestrator worker goroutine need `sync.Mutex` — omitting it causes data races under `-race`.
- MCP `create_draft` `priority` field must be an **integer** (not string). `tags` must be a JSON array of strings.
- SSE connections: `StartMCPServer` has a `ReadTimeout` that kills idle SSE after 15 s (TASK-404 fix pending). Use Streamable HTTP (`/mcp`) for reliable MCP connectivity until fixed.
- When a task files uses YAML frontmatter, `status` is the key (e.g. `status: todo`); when body-level, look for `**Status:** todo`. Both forms exist in `.claude/tasks/`.
- `counters.nextTaskId` in `orchestrator.json` is the source of truth for new task IDs — always read before creating new TASK files to avoid collisions.
