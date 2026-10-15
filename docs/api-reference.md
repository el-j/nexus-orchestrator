# API Reference

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

### Discovery

```
GET /.well-known/nexus.json
```

A public beacon (no token needed) that tells a tool or agent where everything is: `api.base_url`, `api.howto`,
`api.health`, `api.events`, and `mcp.endpoint`, the full URL of the MCP server on its separate port. The MCP
endpoint is built from `NEXUS_MCP_ADDR`; when that address has no host (`:63988`, `0.0.0.0:63988`) the host
you used to reach the REST API is reused. `mcp.endpoint` is omitted when the daemon does not know the address.

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

| Field           | Required | Description                                                                                                 |
| --------------- | -------- | ----------------------------------------------------------------------------------------------------------- |
| `projectPath`   | Yes      | Absolute path to the project directory                                                                      |
| `targetFile`    | Yes      | Relative path for the generated output file                                                                 |
| `instruction`   | Yes      | Natural language prompt for the LLM                                                                         |
| `contextFiles`  | No       | List of files to include as context                                                                         |
| `modelId`       | No       | Constrain to a specific model                                                                               |
| `providerHint`  | No       | Prefer a specific provider by name                                                                          |
| `command`       | No       | Task type: `plan`, `execute`, or `auto` (default: `auto`)                                                   |
| `providerName`  | No       | Lock routing to this exact provider                                                                         |
| `role`          | No       | Routing tier: `architect`/`techlead` prefer frontier providers, `coder`/`tester`/`linter` prefer local ones |
| `priority`      | No       | 1 = high, 2 = medium (default), 3 = low                                                                     |
| `tags`          | No       | Free-form labels                                                                                            |
| `agentic`       | No       | Run as a multi-turn tool loop (see below)                                                                   |
| `maxAgentTurns` | No       | Upper bound on model turns when `agentic` (default 8, max 25)                                               |

**Response:** `201 Created`

```json
{ "task_id": "a1b2c3d4-e5f6-...", "status": "QUEUED" }
```

| Status | Meaning                                                                                    |
| ------ | ------------------------------------------------------------------------------------------ |
| `400`  | Body is not valid JSON                                                                     |
| `422`  | `command` is `execute` but the project has no completed `plan` task                        |
| `429`  | The queue is full (`queueCap`). Transient: honour the `Retry-After` header and retry later |

#### Agentic tasks

With `"agentic": true` the model no longer answers once and has the answer written to
`targetFile`. Instead it works over several turns, replying each time with fenced
` ```nexus-tool ` blocks holding one JSON call each, and the daemon answers with the results:

| Tool         | Arguments            | Effect                                                                                                |
| ------------ | -------------------- | ----------------------------------------------------------------------------------------------------- |
| `read_file`  | `path`               | Returns the file (truncated to 16 KiB)                                                                |
| `write_file` | `path`, `content`    | Creates or replaces a whole file                                                                      |
| `edit_file`  | `path`, `old`, `new` | Replaces the single exact occurrence of `old`; refused when `old` is missing or occurs more than once |
| `verify`     | none                 | Runs the task's `verificationCommand`                                                                 |
| `finish`     | `summary`            | Ends the task; refused while the verification command still fails                                     |

Safety: all file access is confined to `projectPath` (absolute paths, `..` and symlinks
that leave the project are rejected, and the model is told so). There is no tool that runs
an arbitrary command; only the `verificationCommand` the task was submitted with can run.
At most 8 calls of one reply are executed, the loop stops after `maxAgentTurns` turns
(status `FAILED`), the transcript must fit the provider's context window (otherwise
`TOO_LARGE`, or the next provider in the chain is used), and `DELETE /api/tasks/{id}`
aborts a turn that is in flight. A task re-queued by a daemon restart starts again from
the first turn; files already written remain.

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

Cancel a task. Which states can be cancelled:

| Task status                                     | What happens                                                                                                                                                                                 |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `QUEUED`, `DRAFT`, `BACKLOG`, `NO_PROVIDER`     | Moved to `CANCELLED` immediately.                                                                                                                                                            |
| `PROCESSING` (running in this daemon)           | The in-flight provider request is **aborted**; the task becomes `CANCELLED` a moment later (watch the `task.cancelled` event). It is not retried and does not fall back to another provider. |
| `PROCESSING` (claimed by an external agent)     | Taken off the board as `CANCELLED`; the external agent cannot be interrupted.                                                                                                                |
| `COMPLETED`, `FAILED`, `CANCELLED`, `TOO_LARGE` | Not cancellable.                                                                                                                                                                             |

When the daemon shuts down, a task that was running goes back to `QUEUED` (it is not marked failed).

**Response:** `204 No Content` on success, `404 Not Found` if the task doesn't exist, `409 Conflict` if its status cannot be cancelled.

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

Server-Sent Events stream for real-time updates. Every frame is a plain `data:` line carrying a JSON object
whose `type` field says what happened (there is no SSE `event:` field, except for `log` frames, see below).
The first frame is `{"type":"connected"}`.

**Task events** (`{"type": "...", "taskId": "...", "status": "..."}`):

| `type`             | Description                                            |
| ------------------ | ------------------------------------------------------ |
| `task.queued`      | Task was added to the queue                            |
| `task.processing`  | Task is being processed by an LLM                      |
| `task.completed`   | Task completed successfully                            |
| `task.failed`      | Task processing failed                                 |
| `task.cancelled`   | Task was cancelled (also after aborting a running one) |
| `task.too_large`   | Task exceeded the context window                       |
| `task.no_provider` | No provider available for the task                     |
| `task.draft`       | A draft was created                                    |
| `task.backlog`     | A task was moved to the backlog                        |
| `task.updated`     | A task's fields were edited                            |

**Other events:**

| `type`                  | Payload                              | Description                                    |
| ----------------------- | ------------------------------------ | ---------------------------------------------- |
| `ai_session_changed`    | `aiSessionId`, `status`, `timestamp` | A session registered, heartbeated or went idle |
| `ai_session_terminated` | `aiSessionId`, `status`, `timestamp` | A session was terminated                       |
| `ai_activity_new`       | `activity` (an AI activity record)   | A new agent activity was observed              |
| `provider_discovered`   | `count`                              | A provider scan finished (refresh your list)   |

**Log frames** use a named event so they can be told apart:

```
event: log
data: {"timestamp":"2026-01-01T12:00:00Z","level":"info","source":"httpapi","message":"..."}
```

**Frame format:**

```
data: {"type":"task.completed","taskId":"abc-123","status":"COMPLETED"}
```

Slow clients are skipped rather than blocking the daemon, so a client that reconnects should re-read the
state it cares about (for example `GET /api/tasks`) instead of relying on having seen every frame.

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

`filePath` is resolved relative to `projectPath` when it is not absolute. Because the daemon reads the file
on the caller's behalf, the request is confined (the same rules apply to the MCP `ingest` tool, the CLI, and
the desktop app):

- the file, **after resolving symlinks**, must be inside `projectPath` (a symlink that points out is refused);
- it must be a regular file of at most **1 MiB**;
- anything below `.ssh`, `.gnupg`, `.aws`, `.kube`, `.azure`, `.docker` or `.git`, and files named `.env*`,
  `.netrc`, `.npmrc`, `id_rsa` or `id_ed25519` are refused even inside the project;
- `projectPath` must be an existing directory other than the filesystem root;
- to ingest shared templates that live elsewhere, list their directories in `NEXUS_INGEST_ALLOWED_DIRS`
  (separated like `PATH`).

**Response:** `200 OK`

```json
{
  "ingestedSections": 5
}
```

**Errors:** `403` when the path is refused (`{"error": "path not allowed: … is outside the project"}`), `404`
when the file does not exist, `400` when `projectPath` or `filePath` is missing.

---

### Ingest Uploaded Content

```
POST /api/brain/ingest-content
```

Ingests markdown sent in the request instead of read from disk. This is what the dashboard and the desktop
app use for "Ingest File…": a browser file has no path the daemon could read, so its content is uploaded.
Nothing is read from the daemon's filesystem, so no path rules apply; the size limit does.

**Request Body:**

```json
{
  "projectPath": "/path/to/project",
  "name": "notes.md",
  "content": "# Notes\n\n## Architecture\n..."
}
```

`name` is only a label (stored as the source `upload:notes.md`; any directory part is dropped). `content`
is parsed like a `CLAUDE.md`: each `## ` section becomes a knowledge entry.

**Response:** `200 OK` with `{"ingestedSections": 3}`.

**Errors:** `400` when `projectPath` or `content` is missing, `413` when `content` is larger than 1 MiB, `501`
when the running brain service cannot ingest uploads.

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
  "sections": [{ "kind": "architecture", "topic": "Layers", "content": "...", "tokens": 120 }],
  "suggestedFiles": ["internal/core/services/orchestrator.go"],
  "tokensUsed": 380,
  "tokenBudget": 400,
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
{
  "results": [
    { "kind": "architecture", "topic": "Authentication Provider", "content": "...", "tokens": 85 }
  ]
}
```

`limit` is the maximum number of results (default 5).

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
  "tokensUsed": 390,
  "tokenBudget": 400,
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

### Backlog and planning

| Route                                 | Purpose                                                                 |
| ------------------------------------- | ----------------------------------------------------------------------- |
| `GET /api/tasks/all`                  | Every task, drafts and cancelled ones included. Optional `projectPath`. |
| `POST /api/tasks/draft`               | Create a backlog draft that does not enter the execution queue.         |
| `GET /api/tasks/backlog?projectPath=` | Drafts and backlog items for a project, by priority.                    |
| `POST /api/tasks/{id}/promote`        | Move a draft into the active queue.                                     |
| `PUT /api/tasks/{id}`                 | Update a task's instruction, priority, provider, tags or status.        |

### Provider discovery and configuration

| Route                                 | Purpose                                                                                                                                                                                      |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET /api/providers/discovered`       | Local and cloud backends found by the last scan.                                                                                                                                             |
| `POST /api/providers/discovered/scan` | Run a fresh scan.                                                                                                                                                                            |
| `POST /api/providers/promote/{id}`    | Turn a discovered provider into an active backend.                                                                                                                                           |
| `GET /api/providers/config`           | Stored provider configurations.                                                                                                                                                              |
| `POST /api/providers/config`          | Add a provider configuration (kind `lmstudio`, `ollama`, `openaicompat`, `anthropic`, `gemini`, `localai`, `vllm` or `cli`, see [delegating-subscriptions.md](delegating-subscriptions.md)). |
| `PUT /api/providers/config/{id}`      | Update a configuration.                                                                                                                                                                      |
| `DELETE /api/providers/config/{id}`   | Remove a configuration.                                                                                                                                                                      |

### Task coordination (for AI agents)

Agents that pull work from the queue instead of submitting it use these routes.
Each takes the agent's `sessionId` (see _AI sessions_ below).

| Route                            | Purpose                                                                                             |
| -------------------------------- | --------------------------------------------------------------------------------------------------- |
| `POST /api/tasks/{id}/claim`     | Claim a queued task. Body `{"sessionId"}`. `404` when the task or session is unknown.               |
| `PUT /api/tasks/{id}/status`     | Report the outcome. Body `{"sessionId","status","logs"}` where `status` is `COMPLETED` or `FAILED`. |
| `POST /api/tasks/{id}/heartbeat` | Keep a claimed task alive. Body `{"sessionId"}` (`session_id` is accepted for older clients).       |

### AI sessions

| Route                                  | Purpose                                                                                                                            |
| -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `POST /api/ai-sessions`                | Register an agent session (name, role, model, project).                                                                            |
| `GET /api/ai-sessions`                 | List registered sessions.                                                                                                          |
| `DELETE /api/ai-sessions`              | Purge sessions that are disconnected; returns `{"deleted": n}`.                                                                    |
| `GET /api/ai-sessions/discovered`      | Agents found on this machine (Claude, Cursor, Copilot, ...) that are not registered yet.                                           |
| `DELETE /api/ai-sessions/{id}`         | Deregister a session.                                                                                                              |
| `POST /api/ai-sessions/{id}/heartbeat` | Keep a session alive; sessions silent for five minutes are marked disconnected.                                                    |
| `POST /api/ai-sessions/{id}/terminate` | Disconnect a session and stop its agent process when Nexus discovered it. Body `{"force": true}` sends SIGKILL instead of SIGTERM. |
| `GET /api/ai-sessions/{id}/tasks`      | Tasks assigned to a session.                                                                                                       |
| `POST /api/ai-sessions/{id}/delegate`  | Hand a session's work over to Nexus.                                                                                               |

### Logs, activity and plans

| Route                                          | Purpose                                                                              |
| ---------------------------------------------- | ------------------------------------------------------------------------------------ |
| `GET /api/logs`                                | Recent daemon log lines (the same stream the SSE endpoint carries).                  |
| `GET /api/activities`                          | Agent activity records. Query `agent`, `since` (RFC 3339) and `limit`.               |
| `GET /api/activities/timeline`                 | Activity grouped over time; `since` defaults to the last 24 hours.                   |
| `GET /api/plans/discovered?projectPath=`       | Plan files (`PLAN-*.md`, `TASKS.md`, ...) discovered in a project.                   |
| `POST /api/plans/discovered/scan?projectPath=` | Rescan and return the discovered plan files.                                         |
| `GET /api/brain/onboarding?projectPath=`       | A compact orientation for an agent joining the project; `maxTokens` limits its size. |

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

#### Tasks (14)

| Tool                   | Description                                                                                                                                                                                                                               | Parameters                                                                                                             |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `submit_task`          | Submit a code-generation task to the orchestrator                                                                                                                                                                                         | `projectPath`, `targetFile`, `instruction`, `contextFiles?`, `command?`, `verificationCommand?`, `maxCorrectionTurns?` |
| `get_task`             | Get task status and output by ID                                                                                                                                                                                                          | `id`                                                                                                                   |
| `wait_for_task`        | Block until a task has finished and return it; streams the reply as `notifications/progress` when asked (see below)                                                                                                                       | `id`, `timeoutSeconds`                                                                                                 |
| `get_queue`            | List all pending (QUEUED or PROCESSING) tasks                                                                                                                                                                                             | —                                                                                                                      |
| `get_all_tasks`        | Return every task regardless of status                                                                                                                                                                                                    | —                                                                                                                      |
| `cancel_task`          | Cancel a pending task by ID                                                                                                                                                                                                               | `id`                                                                                                                   |
| `update_task`          | Update mutable fields on an existing task                                                                                                                                                                                                 | `id`, `instruction?`, `priority?`, `providerName?`, `modelId?`, `tags?`, `status?`                                     |
| `create_draft`         | Create a draft idea without entering the execution queue                                                                                                                                                                                  | `projectPath`, `instruction`, `targetFile?`, `providerName?`, `modelId?`, `priority?`, `tags?`                         |
| `get_backlog`          | List draft and backlog items for a project, ordered by priority                                                                                                                                                                           | `projectPath`                                                                                                          |
| `promote_task`         | Promote a draft or backlog task to the execution queue                                                                                                                                                                                    | `id`                                                                                                                   |
| `claim_task`           | Claim a QUEUED task for execution by the specified AI session                                                                                                                                                                             | `task_id`, `session_id`                                                                                                |
| `update_task_status`   | Report task completion or failure from executing AI session                                                                                                                                                                               | `task_id`, `session_id`, `status`, `logs?`                                                                             |
| `heartbeat_task`       | Keep a PROCESSING task alive (prevents watchdog failure)                                                                                                                                                                                  | `task_id`, `session_id`                                                                                                |
| `terminate_ai_session` | Disconnect a session and stop its agent process (SIGTERM or SIGKILL) — the process is only signalled when Nexus itself discovered it running (never PID ≤ 1, the daemon or its parent); otherwise the session is just marked disconnected | `session_id`, `force?`                                                                                                 |

**Live progress from `wait_for_task`.** Send `_meta.progressToken` in the `tools/call` request and
`Accept: text/event-stream`: the response then is an event stream. Each piece of the model's reply
is a `notifications/progress` event (`progress` counts up, `message` holds the text; a retry or the
next provider sends `↻ new attempt`), and the last event is the normal JSON-RPC result. Without the
token or the header the reply is plain JSON after the task ended. The stdio proxy and the legacy
`/sse` transport return the final result only. `timeoutSeconds` defaults to 300 (max 3600); on a
timeout the task comes back as it is with `"timedOut": true`.

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

#### Brain / Knowledge (10)

| Tool                     | Description                                                                                                                   | Parameters                              |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| `get_brain_status`       | Retrieve indexing status and entry count of project knowledge brain                                                           | `projectPath`                           |
| `ingest_knowledge`       | Parse and ingest a markdown file (e.g. CLAUDE.md) into project knowledge base                                                 | `projectPath`, `filePath`               |
| `get_project_context`    | Obtain base macro context representation of the project bounded by token budget                                               | `projectPath`, `maxTokens?`             |
| `get_focused_context`    | Query bounded context sections specific to a reasoning question                                                               | `projectPath`, `question`, `maxTokens?` |
| `search_knowledge`       | Search project intelligence via BM25 matching                                                                                 | `projectPath`, `query`, `limit?`        |
| `init_project`           | Auto-ingest CLAUDE.md and initialize a project's knowledge base in one step                                                   | `projectPath`                           |
| `list_knowledge`         | List all knowledge documents stored for a project, optionally filtered by kind                                                | `projectPath`, `kind?`                  |
| `delete_knowledge`       | Delete a knowledge document by ID from the project repository                                                                 | `projectPath`, `id`                     |
| `get_file_map`           | Retrieve cached file path map knowledge document for a project                                                                | `projectPath`                           |
| `get_onboarding_context` | Token-budgeted (< 800 tokens) onboarding summary for agent bootstrap: stack, invariants, test commands, active plan and tasks | `projectPath`, `maxTokens?`             |

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
| `NEXUS_INGEST_ALLOWED_DIRS` | —                            | Extra directories brain ingestion may read from |
| `NEXUS_SCAN_INTERVAL`       | `30s`                        | Provider re-scan interval (must be > 0)         |
| `NEXUS_ADDR`                | `http://127.0.0.1:63987`     | Daemon URL used by `nexus` and `nexus-submit`   |
| `NEXUS_MCP_URL`             | `http://127.0.0.1:63988/mcp` | MCP endpoint used by `nexus-mcp-stdio`          |
| `NEXUS_OPENAI_API_KEY`      | —                            | OpenAI API key (enables OpenAI provider)        |
| `NEXUS_OPENAI_MODEL`        | `gpt-4o-mini`                | Default OpenAI model                            |
| `NEXUS_ANTHROPIC_API_KEY`   | —                            | Anthropic API key (enables Anthropic provider)  |
| `NEXUS_ANTHROPIC_MODEL`     | `claude-3-5-sonnet-20241022` | Default Anthropic model                         |
| `NEXUS_GITHUBCOPILOT_TOKEN` | —                            | GitHub Copilot token                            |
| `NEXUS_GITHUBCOPILOT_MODEL` | `gpt-4o`                     | Default GitHub Copilot model                    |
