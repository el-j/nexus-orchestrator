---
layout: default
title: API Reference
nav_order: 3
---

# API Reference

{: .no_toc }

## Table of contents

{: .no_toc .text-delta }

1. TOC
   {:toc}

---

## HTTP REST API

Base URL: `http://localhost:63987`

### Authentication and local-access protection

By default the daemon trusts local callers. Three layers protect it:

- **Bearer token (optional).** Set `NEXUS_API_TOKEN` (or rotate one via `PUT /api/config`) and every `/api/*`
  request except `/api/health`, `/api/howto` and `/.well-known/nexus.json` must send
  `Authorization: Bearer <token>`. The MCP server uses `NEXUS_MCP_TOKEN` the same way.
- **Browser origin check (always on).** A request carrying an `Origin` header that is not a local origin
  (`localhost`, `127.0.0.1`, `[::1]`, `wails://wails.localhost`) is rejected with `403`. This stops a web page
  you happen to visit from driving the daemon with a cross-site request. CLI tools, the MCP stdio proxy and
  `curl` send no `Origin` and are unaffected. Add trusted origins with `NEXUS_ALLOWED_ORIGINS`
  (comma-separated, e.g. `https://dash.example.com`).
- **Host check (loopback binds).** When `NEXUS_LISTEN_ADDR` is a loopback address, the `Host` header must be
  `localhost` or an IP literal, which defeats DNS-rebinding. Add host names with `NEXUS_ALLOWED_HOSTS`. A
  daemon deliberately bound to all interfaces (containers, LAN) skips the Host check - set `NEXUS_API_TOKEN` there.

Error bodies are always `{"error": "<message>"}`. Server faults (`5xx`) never include internal details.

### Submit Task

```
POST /api/tasks
```

Submit a new code-generation task to the queue.

**Request Body:**

```json
{
  "projectPath": "/path/to/project",
  "targetFile": "output.go",
  "instruction": "Write a function that sorts strings",
  "contextFiles": ["main.go", "utils.go"],
  "modelId": "codellama",
  "providerHint": "LM Studio",
  "command": "execute"
}
```

| Field          | Required | Description                                               |
| -------------- | -------- | --------------------------------------------------------- |
| `projectPath`  | Yes      | Absolute path to the project directory                    |
| `targetFile`   | Yes      | Relative path for the generated output file               |
| `instruction`  | Yes      | Natural language prompt for the LLM                       |
| `contextFiles` | No       | List of files to include as context                       |
| `modelId`      | No       | Constrain to a specific model                             |
| `providerHint` | No       | Prefer a specific provider by name                        |
| `command`      | No       | Task type: `plan`, `execute`, or `auto` (default: `auto`) |

**Response:** `201 Created`

```json
{ "task_id": "a1b2c3d4-e5f6-...", "status": "QUEUED" }
```

| Status | Meaning                                                                                    |
| ------ | ------------------------------------------------------------------------------------------ |
| `400`  | Body is not valid JSON                                                                     |
| `422`  | `command` is `execute` but the project has no completed `plan` task                        |
| `429`  | The queue is full (`queueCap`). Transient: honour the `Retry-After` header and retry later |

---

### List Tasks

```
GET /api/tasks
```

Returns all pending (QUEUED or PROCESSING) tasks.

**Response:** `200 OK`

```json
[
  {
    "id": "...",
    "status": "QUEUED",
    ...
  }
]
```

---

### Get Task

```
GET /api/tasks/{id}
```

Retrieve a single task by ID.

**Response:** `200 OK` or `404 Not Found`

```json
{
  "id": "a1b2c3d4-...",
  "status": "COMPLETED",
  "logs": "generated code output..."
}
```

---

### Cancel Task

```
DELETE /api/tasks/{id}
```

Cancel a queued task before it is processed.

**Response:** `204 No Content` on success, `404 Not Found` if task doesn't exist or already processed.

---

### List Providers

```
GET /api/providers
```

Returns all registered LLM providers with their liveness status.

**Response:** `200 OK`

```json
[
  {
    "name": "LM Studio",
    "active": true,
    "activeModel": "codellama",
    "models": ["codellama", "deepseek-coder"]
  },
  {
    "name": "Ollama",
    "active": false
  }
]
```

---

### Register Provider

```
POST /api/providers
```

Dynamically register a new cloud LLM provider.

**Request Body:**

```json
{
  "name": "My OpenAI",
  "kind": "openai-compat",
  "baseURL": "https://api.openai.com/v1",
  "apiKey": "sk-...",
  "model": "gpt-4o-mini"
}
```

| Field     | Required | Description                                                       |
| --------- | -------- | ----------------------------------------------------------------- |
| `name`    | Yes      | Display name for the provider                                     |
| `kind`    | Yes      | Provider type: `lmstudio`, `ollama`, `openai-compat`, `anthropic` |
| `baseURL` | Yes      | API endpoint URL                                                  |
| `apiKey`  | Depends  | Required for cloud providers                                      |
| `model`   | No       | Default model to use                                              |

**Response:** `201 Created`

---

### Remove Provider

```
DELETE /api/providers/{name}
```

Deregister a provider by name.

**Response:** `204 No Content` or `404 Not Found`

---

### Get Provider Models

```
GET /api/providers/{name}/models
```

List available models from a specific provider.

**Response:** `200 OK`

```json
["codellama", "deepseek-coder", "llama3"]
```

---

### SSE Event Stream

```
GET /api/events
```

Server-Sent Events stream for real-time task lifecycle updates.

**Event Types:**

| Event              | Description                        |
| ------------------ | ---------------------------------- |
| `task.queued`      | Task was added to the queue        |
| `task.processing`  | Task is being processed by an LLM  |
| `task.completed`   | Task completed successfully        |
| `task.failed`      | Task processing failed             |
| `task.cancelled`   | Task was cancelled                 |
| `task.too_large`   | Task exceeded context window       |
| `task.no_provider` | No provider available for the task |

**Event Format:**

```
event: task.completed
data: {"type":"task.completed","taskId":"abc-123","status":"COMPLETED"}
```

---

### Health Check

```
GET /api/health
```

**Response:** `200 OK`

```json
{ "status": "ok" }
```

---

### Ingest Project Knowledge

```
POST /api/brain/ingest
```

Parses and ingests a markdown file into the project context brain.

**Request Body:**

```json
{
  "projectPath": "/path/to/project",
  "filePath": "CLAUDE.md"
}
```

**Response:** `200 OK`

```json
{
  "ingestedSections": 5
}
```

---

### Get Brain Status

```
GET /api/brain/status?projectPath=/path/to/project
```

Returns the context token size, entry count, and initialization state.

**Response:** `200 OK`

```json
{
  "projectPath": "/path/to/project",
  "initialized": true,
  "entryCount": 12,
  "kindCounts": { "feature": 8, "architecture": 4 },
  "totalTokens": 850
}
```

---

### Get Project Context

```
POST /api/brain/context
```

Aggregates the top-level macro context for LLM system prompts.

**Request Body:**

```json
{
  "projectPath": "/path/to/project",
  "maxTokens": 400
}
```

**Response:** `200 OK`

```json
{
  "projectPath": "/path/to/project",
  "sections": [...],
  "totalTokens": 380,
  "truncated": false
}
```

---

### Search Knowledge

```
GET /api/brain/search?projectPath=/path/to/project&q=auth&limit=5
```

Performs BM25 search against the SQLite FTS5 index.

**Response:** `200 OK`

```json
[
  {
    "title": "Authentication Provider",
    "content": "...",
    "source": "auth.md"
  }
]
```

---

### Get Focused Context

```
POST /api/brain/focused-context
```

Get focused context for a project based on a specific question. Returns context sections most relevant to the question, bounded by a token budget.

**Request Body:**

```json
{
  "projectPath": "/path/to/project",
  "question": "How does authentication work?",
  "maxTokens": 400
}
```

| Field         | Required | Description                                     |
| ------------- | -------- | ----------------------------------------------- |
| `projectPath` | Yes      | Absolute path to the project directory          |
| `question`    | Yes      | Question to focus the context retrieval around  |
| `maxTokens`   | No       | Token budget for returned context (default 400) |

**Response:** `200 OK`

```json
{
  "projectPath": "/path/to/project",
  "sections": [...],
  "totalTokens": 390,
  "truncated": false
}
```

---

### Initialize Knowledge Base

```
POST /api/brain/init
```

Auto-ingest CLAUDE.md and initialize a project's knowledge base. Discovers and ingests the project's CLAUDE.md automatically.

**Request Body:**

```json
{
  "projectPath": "/path/to/project",
  "claudeMDPath": "/path/to/project/CLAUDE.md"
}
```

| Field          | Required | Description                                             |
| -------------- | -------- | ------------------------------------------------------- |
| `projectPath`  | Yes      | Absolute path to the project directory                  |
| `claudeMDPath` | No       | Explicit path to CLAUDE.md (auto-discovered if omitted) |

**Response:** `200 OK`

```json
{
  "ingestedSections": 8
}
```

---

### List Knowledge Entries

```
GET /api/brain/knowledge?projectPath=/path/to/project&kind=architecture
```

List all knowledge entries for a project, with optional filtering by kind.

**Query Parameters:**

| Parameter     | Required | Description                                               |
| ------------- | -------- | --------------------------------------------------------- |
| `projectPath` | Yes      | Absolute path to the project directory                    |
| `kind`        | No       | Filter by knowledge kind (e.g. `feature`, `architecture`) |

**Response:** `200 OK`

```json
[
  {
    "id": "1",
    "title": "Hexagonal Architecture",
    "kind": "architecture",
    "source": "CLAUDE.md",
    "tokens": 120
  }
]
```

---

### Delete Knowledge Entry

```
DELETE /api/brain/knowledge/{id}
```

Delete a single knowledge entry by its ID.

**Response:** `204 No Content` on success, `404 Not Found` if the entry does not exist.

---

### Get File Map

```
GET /api/brain/file-map?projectPath=/path/to/project&focusArea=authentication
```

Get file path map for a project, optionally filtered to a focus area.

**Query Parameters:**

| Parameter     | Required | Description                                      |
| ------------- | -------- | ------------------------------------------------ |
| `projectPath` | Yes      | Absolute path to the project directory           |
| `focusArea`   | No       | Narrow the map to files relevant to a focus area |

**Response:** `200 OK`

```json
{
  "projectPath": "/path/to/project",
  "files": ["internal/auth/handler.go", "internal/auth/service.go"]
}
```

---

### Dashboard

```
GET /ui
```

Serves the embedded web dashboard with real-time task monitoring, task submission form, and provider management.

---

## MCP Server

Base URL: `http://localhost:63988`

### Protocol

- **Standard**: JSON-RPC 2.0
- **Version**: `2024-11-05`
- **Endpoint**: `POST /mcp`
- **Health**: `GET /health`
- **Default Port**: 63988 (configurable via `NEXUS_MCP_ADDR`)

### Available Tools

#### Tasks (13)

| Tool                   | Description                                                     | Parameters                                                                                                             |
| ---------------------- | --------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `submit_task`          | Submit a code-generation task to the orchestrator               | `projectPath`, `targetFile`, `instruction`, `contextFiles?`, `command?`, `verificationCommand?`, `maxCorrectionTurns?` |
| `get_task`             | Get task status and output by ID                                | `id`                                                                                                                   |
| `get_queue`            | List all pending (QUEUED or PROCESSING) tasks                   | —                                                                                                                      |
| `get_all_tasks`        | Return every task regardless of status                          | —                                                                                                                      |
| `cancel_task`          | Cancel a pending task by ID                                     | `id`                                                                                                                   |
| `update_task`          | Update mutable fields on an existing task                       | `id`, `instruction?`, `priority?`, `providerName?`, `modelId?`, `tags?`, `status?`                                     |
| `create_draft`         | Create a draft idea without entering the execution queue        | `projectPath`, `instruction`, `targetFile?`, `providerName?`, `modelId?`, `priority?`, `tags?`                         |
| `get_backlog`          | List draft and backlog items for a project, ordered by priority | `projectPath`                                                                                                          |
| `promote_task`         | Promote a draft or backlog task to the execution queue          | `id`                                                                                                                   |
| `claim_task`           | Claim a QUEUED task for execution by the specified AI session   | `task_id`, `session_id`                                                                                                |
| `update_task_status`   | Report task completion or failure from executing AI session     | `task_id`, `session_id`, `status`, `logs?`                                                                             |
| `heartbeat_task`       | Keep a PROCESSING task alive (prevents watchdog failure)        | `task_id`, `session_id`                                                                                                |
| `terminate_ai_session` | Terminate an external AI agent session (SIGTERM or SIGKILL)     | `session_id`, `force?`                                                                                                 |

#### AI Sessions (5)

| Tool                          | Description                                                          | Parameters                                    |
| ----------------------------- | -------------------------------------------------------------------- | --------------------------------------------- |
| `register_session`            | Announce external AI agent session for visualization & orchestration | `agent_name`, `project_path?`, `external_id?` |
| `get_ai_sessions`             | Return all external AI agent sessions registered with this daemon    | —                                             |
| `deregister_ai_session`       | Soft-disconnect an AI agent session without killing process          | `session_id`                                  |
| `heartbeat_ai_session`        | Refresh last-activity timestamp of an AI session to keep it alive    | `session_id`                                  |
| `purge_disconnected_sessions` | Delete all disconnected AI sessions inactive for > 2 hours           | —                                             |

#### Providers (7)

| Tool                     | Description                                                      | Parameters                                                  |
| ------------------------ | ---------------------------------------------------------------- | ----------------------------------------------------------- |
| `get_providers`          | List available LLM providers and models                          | —                                                           |
| `discover_providers`     | Scan local system for installed AI providers/agents              | —                                                           |
| `promote_provider`       | Promote a discovered provider to an active LLM backend           | `id`                                                        |
| `list_provider_configs`  | List all persisted LLM provider configuration records            | —                                                           |
| `add_provider_config`    | Add a new LLM provider configuration and register when enabled   | `kind`, `name`, `base_url?`, `api_key?`, `enabled?`         |
| `update_provider_config` | Update an existing LLM provider configuration by ID              | `id`, `kind?`, `name?`, `base_url?`, `api_key?`, `enabled?` |
| `remove_provider_config` | Delete a persisted provider configuration and deregister adapter | `id`                                                        |

#### Brain / Knowledge (9)

| Tool                  | Description                                                                     | Parameters                              |
| --------------------- | ------------------------------------------------------------------------------- | --------------------------------------- |
| `get_brain_status`    | Retrieve indexing status and entry count of project knowledge brain             | `projectPath`                           |
| `ingest_knowledge`    | Parse and ingest a markdown file (e.g. CLAUDE.md) into project knowledge base   | `projectPath`, `filePath`               |
| `get_project_context` | Obtain base macro context representation of the project bounded by token budget | `projectPath`, `maxTokens?`             |
| `get_focused_context` | Query bounded context sections specific to a reasoning question                 | `projectPath`, `question`, `maxTokens?` |
| `search_knowledge`    | Search project intelligence via BM25 matching                                   | `projectPath`, `query`, `limit?`        |
| `init_project`        | Auto-ingest CLAUDE.md and initialize a project's knowledge base in one step     | `projectPath`                           |
| `list_knowledge`      | List all knowledge documents stored for a project, optionally filtered by kind  | `projectPath`, `kind?`                  |
| `delete_knowledge`    | Delete a knowledge document by ID from the project repository                   | `projectPath`, `id`                     |
| `get_file_map`        | Retrieve cached file path map knowledge document for a project                  | `projectPath`                           |

#### Discovery & System (6)

| Tool                    | Description                                                                       | Parameters     |
| ----------------------- | --------------------------------------------------------------------------------- | -------------- |
| `get_discovered_agents` | Return AI agent tools detected on the local system (Claude CLI, Copilot, etc.)    | —              |
| `delegate_to_nexus`     | Delegate an AI agent session to the nexus task queue                              | `session_id`   |
| `get_discovered_plans`  | Scan for plan/task/orchestration files in a project directory                     | `projectPath?` |
| `howto`                 | Return a complete integration guide — all tools, workflow patterns, and endpoints | —              |
| `howto_brief`           | Ultra-compact integration guide (~200 tokens) for small-context models            | —              |
| `health`                | Check daemon reachable and operational                                            | —              |

### Example: Submit Task via MCP

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "submit_task",
    "arguments": {
      "projectPath": "/path/to/project",
      "targetFile": "handler.go",
      "instruction": "Add error handling to the HTTP handler",
      "command": "execute"
    }
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"id\":\"abc-123\",\"status\":\"QUEUED\"}"
      }
    ]
  }
}
```

### Example: Get Task Status via MCP

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_task",
    "arguments": {
      "taskId": "abc-123"
    }
  }
}
```

---

## Task Lifecycle

```
QUEUED → PROCESSING → COMPLETED
                    → FAILED
                    → TOO_LARGE (pre-flight)
                    → NO_PROVIDER (no backend)
QUEUED → CANCELLED (user cancellation)
```

A task moves through these states:

1. **QUEUED**: Submitted and waiting in the queue
2. **PROCESSING**: Picked up by the worker, LLM call in progress
3. **Terminal state**: One of COMPLETED, FAILED, TOO_LARGE, NO_PROVIDER, or CANCELLED

---

## Environment Variables

| Variable                    | Default                      | Description                                     |
| --------------------------- | ---------------------------- | ----------------------------------------------- |
| `NEXUS_DB_PATH`             | `nexus.db`                   | SQLite database file path                       |
| `NEXUS_LISTEN_ADDR`         | `127.0.0.1:63987`            | HTTP API listen address                         |
| `NEXUS_MCP_ADDR`            | `127.0.0.1:63988`            | MCP server listen address                       |
| `NEXUS_API_TOKEN`           | —                            | Require `Authorization: Bearer` on `/api/*`     |
| `NEXUS_MCP_TOKEN`           | —                            | Require `Authorization: Bearer` on the MCP API  |
| `NEXUS_ALLOWED_ORIGINS`     | —                            | Extra browser origins allowed (comma-separated) |
| `NEXUS_ALLOWED_HOSTS`       | —                            | Extra `Host` names allowed on loopback binds    |
| `NEXUS_SCAN_INTERVAL`       | `30s`                        | Provider re-scan interval (must be > 0)         |
| `NEXUS_ADDR`                | `http://127.0.0.1:63987`     | Daemon URL used by `nexus` and `nexus-submit`   |
| `NEXUS_MCP_URL`             | `http://127.0.0.1:63988/mcp` | MCP endpoint used by `nexus-mcp-stdio`          |
| `NEXUS_OPENAI_API_KEY`      | —                            | OpenAI API key (enables OpenAI provider)        |
| `NEXUS_OPENAI_MODEL`        | `gpt-4o-mini`                | Default OpenAI model                            |
| `NEXUS_ANTHROPIC_API_KEY`   | —                            | Anthropic API key (enables Anthropic provider)  |
| `NEXUS_ANTHROPIC_MODEL`     | `claude-3-5-sonnet-20241022` | Default Anthropic model                         |
| `NEXUS_GITHUBCOPILOT_TOKEN` | —                            | GitHub Copilot token                            |
| `NEXUS_GITHUBCOPILOT_MODEL` | `gpt-4o`                     | Default GitHub Copilot model                    |
