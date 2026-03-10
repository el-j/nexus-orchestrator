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

Base URL: `http://localhost:9999`

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

| Field | Required | Description |
|-------|----------|-------------|
| `projectPath` | Yes | Absolute path to the project directory |
| `targetFile` | Yes | Relative path for the generated output file |
| `instruction` | Yes | Natural language prompt for the LLM |
| `contextFiles` | No | List of files to include as context |
| `modelId` | No | Constrain to a specific model |
| `providerHint` | No | Prefer a specific provider by name |
| `command` | No | Task type: `plan`, `execute`, or `auto` (default: `auto`) |

**Response:** `201 Created`
```json
{
  "id": "a1b2c3d4-e5f6-...",
  "projectPath": "/path/to/project",
  "targetFile": "output.go",
  "instruction": "Write a function that sorts strings",
  "status": "QUEUED",
  "command": "execute",
  "createdAt": "2025-01-01T00:00:00Z",
  "updatedAt": "2025-01-01T00:00:00Z"
}
```

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

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name for the provider |
| `kind` | Yes | Provider type: `lmstudio`, `ollama`, `openai-compat`, `anthropic` |
| `baseURL` | Yes | API endpoint URL |
| `apiKey` | Depends | Required for cloud providers |
| `model` | No | Default model to use |

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

| Event | Description |
|-------|-------------|
| `task.queued` | Task was added to the queue |
| `task.processing` | Task is being processed by an LLM |
| `task.completed` | Task completed successfully |
| `task.failed` | Task processing failed |
| `task.cancelled` | Task was cancelled |
| `task.too_large` | Task exceeded context window |
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
{"status": "ok"}
```

---

### Dashboard

```
GET /ui
```

Serves the embedded web dashboard with real-time task monitoring, task submission form, and provider management.

---

## MCP Server

Base URL: `http://localhost:9998`

### Protocol

- **Standard**: JSON-RPC 2.0
- **Version**: `2024-11-05`
- **Endpoint**: `POST /mcp`
- **Health**: `GET /health`
- **Default Port**: 9998 (configurable via `NEXUS_MCP_ADDR`)

### Available Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `submit_task` | Submit a code-generation task | `projectPath`, `targetFile`, `instruction`, `contextFiles`, `command` |
| `get_task` | Get task by ID | `taskId` |
| `get_queue` | List all pending tasks | — |
| `cancel_task` | Cancel a queued task | `taskId` |
| `get_providers` | List LLM providers | — |
| `health` | Check daemon status | — |

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

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXUS_DB_PATH` | `nexus.db` | SQLite database file path |
| `NEXUS_LISTEN_ADDR` | `127.0.0.1:9999` | HTTP API listen address |
| `NEXUS_MCP_ADDR` | `127.0.0.1:9998` | MCP server listen address |
| `NEXUS_OPENAI_API_KEY` | — | OpenAI API key (enables OpenAI provider) |
| `NEXUS_OPENAI_MODEL` | `gpt-4o-mini` | Default OpenAI model |
| `NEXUS_ANTHROPIC_API_KEY` | — | Anthropic API key (enables Anthropic provider) |
| `NEXUS_ANTHROPIC_MODEL` | `claude-3-5-sonnet-20241022` | Default Anthropic model |
| `NEXUS_GITHUBCOPILOT_TOKEN` | — | GitHub Copilot token |
| `NEXUS_GITHUBCOPILOT_MODEL` | `gpt-4o` | Default GitHub Copilot model |