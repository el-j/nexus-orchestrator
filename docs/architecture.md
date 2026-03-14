---
layout: default
title: Architecture
nav_order: 2
---

# Architecture
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Hexagonal Architecture

nexusOrchestrator follows a **hexagonal architecture** (ports & adapters) with a strict inward dependency rule. The core business logic never imports adapter code — all external concerns are abstracted behind interfaces.

```
┌───────────────────────────────────────────────────────────┐
│                     Inbound Adapters                       │
│  HTTP API (chi)  │  MCP (JSON-RPC)  │  CLI  │  Wails GUI │
├───────────────────────────────────────────────────────────┤
│                      Core Services                         │
│         OrchestratorService   │   DiscoveryService         │
├───────────────────────────────────────────────────────────┤
│                     Port Interfaces                        │
│  Orchestrator │ LLMClient │ TaskRepo │ FileWriter │ Sess  │
├───────────────────────────────────────────────────────────┤
│                    Outbound Adapters                       │
│  LM Studio │ Ollama │ OpenAI │ Anthropic │ SQLite  │  FS  │
└───────────────────────────────────────────────────────────┘
```

The dependency flow is strictly **inward**:

```
inbound adapters → core services → ports ← outbound adapters
```

---

## Domain Layer

The domain layer contains pure Go types with no framework imports.

### Task

The central entity representing a single unit of AI work:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID, generated on submission |
| `ProjectPath` | `string` | Absolute path to the project |
| `TargetFile` | `string` | Relative path for generated code output |
| `Instruction` | `string` | Natural language prompt |
| `ContextFiles` | `[]string` | Files to include as context |
| `ModelID` | `string` | Constrain to a specific model (optional) |
| `ProviderHint` | `string` | Prefer a specific provider (optional) |
| `Command` | `CommandType` | Task classification: plan, execute, or auto |
| `Status` | `TaskStatus` | Lifecycle state |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |
| `Logs` | `string` | LLM output or error details |

### TaskStatus Lifecycle

```
          ┌──────────┐
          │  QUEUED   │
          └────┬─────┘
               │
          ┌────▼─────┐
          │PROCESSING │
          └────┬─────┘
               │
    ┌──────────┼──────────┬────────────┐
    ▼          ▼          ▼            ▼
COMPLETED   FAILED   TOO_LARGE   NO_PROVIDER

              CANCELLED (from QUEUED only)
```

| Status | Description |
|--------|-------------|
| `QUEUED` | Task is waiting in the queue |
| `PROCESSING` | Task is being processed by an LLM |
| `COMPLETED` | LLM generated output successfully |
| `FAILED` | LLM call failed |
| `CANCELLED` | Cancelled before processing |
| `TOO_LARGE` | Prompt exceeds model context window |
| `NO_PROVIDER` | No provider available for requested model |

### CommandType

Tasks can be classified to enable command-aware routing:

| Command | Description |
|---------|-------------|
| `plan` | Planning/orchestration work (creating plans, task documents) |
| `execute` | Code implementation (requires a prior completed plan) |
| `auto` | Let the orchestrator decide (default) |

### Session & Message

Per-project conversation history for multi-turn LLM interactions:

- **Session**: Identified by `ProjectPath`, contains a list of `Message` objects
- **Message**: Has `Role` (system/user/assistant) and `Content`

### ProviderConfig

Runtime configuration for dynamically registering cloud providers:

| Field | Description |
|-------|-------------|
| `Name` | Display name |
| `Kind` | Provider type: `lmstudio`, `ollama`, `openai-compat`, `anthropic` |
| `BaseURL` | API endpoint |
| `APIKey` | Authentication key |
| `Model` | Default model |

---

## Port Contracts

All external dependencies are abstracted behind Go interfaces in `internal/core/ports/ports.go`.

### Orchestrator (Inbound Port)

The primary interface that UI, CLI, and HTTP API call:

```go
type Orchestrator interface {
    SubmitTask(task domain.Task) (string, error)
    GetTask(id string) (domain.Task, error)
    GetQueue() ([]domain.Task, error)
    GetProviders() ([]ProviderInfo, error)
    CancelTask(id string) error
    RegisterCloudProvider(cfg domain.ProviderConfig) error
    RemoveProvider(providerName string) error
    GetProviderModels(providerName string) ([]string, error)
}
```

### LLMClient (Outbound Port)

Interface for any language model backend:

```go
type LLMClient interface {
    Ping() bool
    ProviderName() string
    ActiveModel() string
    GetAvailableModels() ([]string, error)
    ContextLimit() int
    GenerateCode(prompt string) (string, error)
    Chat(messages []domain.Message) (string, error)
}
```

### TaskRepository (Outbound Port)

Persistence for tasks:

```go
type TaskRepository interface {
    Save(t domain.Task) error
    GetByID(id string) (domain.Task, error)
    GetPending() ([]domain.Task, error)
    GetByProjectPath(projectPath string) ([]domain.Task, error)
    UpdateStatus(id string, status domain.TaskStatus) error
    UpdateLogs(id, logs string) error
}
```

### SessionRepository (Outbound Port)

Per-project conversation history:

```go
type SessionRepository interface {
    Save(s domain.Session) error
    GetByProjectPath(projectPath string) (domain.Session, error)
    AppendMessage(projectPath string, msg domain.Message) error
}
```

### FileWriter (Outbound Port)

Disk I/O for reading context and writing generated code:

```go
type FileWriter interface {
    WriteCodeToFile(projectPath, targetFile, code string) error
    ReadContextFiles(projectPath string, files []string) (string, error)
}
```

---

## Inbound Adapters

| Adapter | Package | Protocol | Default Port |
|---------|---------|----------|-------------|
| HTTP API | `httpapi` | REST + SSE | `:63987` |
| MCP Server | `mcp` | JSON-RPC 2.0 | `:63988` |
| CLI Client | `cli` | HTTP → daemon | — |
| Wails GUI | `wailsbind` | Native + embedded HTTP | `:63987` |
| System Tray | `tray` | OS native | — |

All inbound adapters accept `ports.Orchestrator` as a dependency — never a concrete service type.

---

## Outbound Adapters

| Adapter | Package | Backend |
|---------|---------|---------|
| LM Studio | `llm_lmstudio` | OpenAI-compatible API at `:1234` |
| Ollama | `llm_ollama` | Ollama API at `:11434` |
| OpenAI-compatible | `llm_openaicompat` | OpenAI, GitHub Copilot, Azure |
| Anthropic | `llm_anthropic` | Anthropic Messages API |
| SQLite | `repo_sqlite` | Local SQLite via `go-sqlite3` |
| Filesystem | `fs_writer` | Disk read/write |

---

## Entry Points

| Binary | Path | Purpose |
|--------|------|---------|
| Desktop GUI | `main.go` + `app.go` | Wails window + embedded HTTP API + MCP |
| Headless Daemon | `cmd/nexus-daemon/main.go` | HTTP API on `:63987` + MCP on `:63988` |
| CLI Client | `cmd/nexus-cli/main.go` | Thin HTTP client → daemon at `127.0.0.1:63987` |

---

## Concurrency Model

- **Shared state** is protected with `sync.Mutex` in `OrchestratorService`
- The **background worker** goroutine processes tasks sequentially — only one LLM call is ever in flight
- Shutdown is coordinated via a `stopCh chan struct{}` channel
- `Stop()` is idempotent via `sync.Once`
- **No goroutines inside core services** — goroutine lifecycle is an infrastructure concern owned by inbound adapters
- SSE broadcasting uses a separate Hub with its own mutex to avoid lock nesting