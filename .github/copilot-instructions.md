# nexusOrchestrator – Project Guidelines

## Architecture

Hexagonal architecture (Ports & Adapters). The dependency rule flows strictly inward:

```
inbound adapters → core services → ports ← outbound adapters
```

- **`internal/core/domain/`** — Pure domain types (`Task`, `TaskStatus`). No framework imports.
- **`internal/core/ports/`** — Go interfaces only: `LLMClient`, `TaskRepository`, `FileWriter`, `Orchestrator`. Nothing concrete here.
- **`internal/core/services/`** — Business logic. Depends only on ports. Never import adapters directly.
- **`internal/adapters/inbound/`** — CLI (Cobra), HTTP API (chi), Wails GUI binding, system tray.
- **`internal/adapters/outbound/`** — LM Studio (`llm_lmstudio`), Ollama (`llm_ollama`), SQLite (`repo_sqlite`), filesystem (`fs_writer`).

Entry points:
| Binary | Path | Purpose |
|--------|------|---------|
| Desktop GUI | `main.go` + `app.go` | Wails window + embedded HTTP API on `:9999` |
| Headless daemon | `cmd/nexus-daemon/main.go` | HTTP API only (for servers / background) |
| CLI client | `cmd/nexus-cli/main.go` | Thin HTTP client → daemon at `127.0.0.1:9999` |

## Build & Test

```sh
# Build
go build ./cmd/nexus-cli/...
go build ./cmd/nexus-daemon/...

# Run desktop app (requires Wails)
wails dev          # hot-reload dev
go run main.go     # production build

# Test
go test ./...
go test ./internal/core/services/...   # unit tests only

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

### Configuration
- Prefer environment variables over flags for daemon config:
  - `NEXUS_DB_PATH` — SQLite database file path (default: `nexus.db`)
  - `NEXUS_LISTEN_ADDR` — HTTP listen address (default: `:9999`)
- Provider base URLs (LM Studio `127.0.0.1:1234`, Ollama `127.0.0.1:11434`) are currently hardcoded in outbound adapters.

### Adding a new LLM provider
1. Create `internal/adapters/outbound/llm_<name>/adapter.go`.
2. Implement the `ports.LLMClient` interface (`Ping`, `ProviderName`, `GetAvailableModels`, `GenerateCode`).
3. Register the adapter in `DiscoveryService` (pass it alongside existing clients).
4. Wire it up in all three entry points (`main.go`, `cmd/nexus-daemon/main.go`).

### Adding a new inbound interface
1. Create `internal/adapters/inbound/<name>/`.
2. Accept `ports.Orchestrator` as a dependency — never a concrete service type.

## Key Reference Files

- Domain model: [internal/core/domain/task.go](../internal/core/domain/task.go)
- Port contracts: [internal/core/ports/ports.go](../internal/core/ports/ports.go)
- Worker loop pattern: [internal/core/services/orchestrator.go](../internal/core/services/orchestrator.go)
- LLM adapter example: [internal/adapters/outbound/llm_lmstudio/adapter.go](../internal/adapters/outbound/llm_lmstudio/adapter.go)
- HTTP API: [internal/adapters/inbound/httpapi/server.go](../internal/adapters/inbound/httpapi/server.go)
- Wails binding: [app.go](../app.go)

## Potential Pitfalls

- `go-sqlite3` requires CGO — ensure `CGO_ENABLED=1` and a C compiler (`gcc`/`clang`) are available.
- The frontend JS source is **not** in this repo; `frontend/dist/` contains pre-compiled assets only. Don't run `npm install` here.
- The CLI binary is a **remote client** — it makes HTTP calls to a running daemon/desktop app. It does not link core services directly.
- HTTP timeout on LM Studio is 60 s; Ollama is 120 s. For large prompts, be aware of these limits when testing adapters.
