---
layout: default
title: MCP Integration
nav_order: 5
---

# MCP Integration

{: .no_toc }

## Table of contents

{: .no_toc .text-delta }

1. TOC
   {:toc}

---

## What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) is an open standard for connecting AI assistants to external tools and data sources. nexusOrchestrator implements an MCP server using JSON-RPC 2.0, making it compatible with Claude Desktop and any MCP-aware client.

## Claude Desktop Setup

Add the following to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "nexusOrchestrator": {
      "url": "http://localhost:63988/mcp"
    }
  }
}
```

Restart Claude Desktop after editing the configuration. The nexusOrchestrator tools will appear in Claude's tool palette.

{: .note }
Make sure the nexus-daemon is running before starting Claude Desktop.

## VS Code Setup

Add the following to your VS Code MCP configuration file:

**macOS/Linux**: `~/Library/Application Support/Code/User/mcp.json`  
**Windows**: `%APPDATA%\Code\User\mcp.json`

```json
{
  "servers": {
    "Nexus Orchestrator": {
      "type": "http",
      "url": "http://127.0.0.1:63988/mcp"
    }
  }
}
```

{: .important }
Use `"type": "http"` (Streamable HTTP), **not** `"type": "sse"`. The SSE transport holds a session in memory; a daemon restart invalidates it, causing `400` errors and `"terminated / Failed to parse message"` noise in the VS Code output panel. Streamable HTTP is stateless and reconnects cleanly every time.

Reload the VS Code window after saving the configuration.

## Available Tools

### Tasks (13)

| Tool                   | Description                                                                                           |
| ---------------------- | ----------------------------------------------------------------------------------------------------- |
| `submit_task`          | Submit a code-generation task with project path, target file, instruction, verification command, etc. |
| `get_task`             | Retrieve the status and output of a task by its ID (`id`)                                             |
| `get_queue`            | List all pending (QUEUED or PROCESSING) tasks                                                         |
| `get_all_tasks`        | Return every task regardless of status                                                                |
| `cancel_task`          | Cancel a pending task before it is processed (`id`)                                                   |
| `update_task`          | Update mutable fields on an existing task (instruction, priority, provider, tags, status)             |
| `create_draft`         | Create a draft idea for a project without entering the execution queue                                |
| `get_backlog`          | List draft and backlog items for a project, ordered by priority                                       |
| `promote_task`         | Promote a draft or backlog task to the execution queue (`id`)                                         |
| `claim_task`           | Claim a QUEUED task for execution by an external AI session                                           |
| `update_task_status`   | Report task completion or failure from the executing AI session                                       |
| `heartbeat_task`       | Keep a PROCESSING task alive (prevents watchdog from marking it failed)                               |
| `terminate_ai_session` | Terminate an external AI agent session (SIGTERM or SIGKILL)                                           |

### AI Sessions (5)

| Tool                          | Description                                                            |
| ----------------------------- | ---------------------------------------------------------------------- |
| `register_session`            | Announce external AI agent session for visualization and orchestration |
| `get_ai_sessions`             | Return all external AI agent sessions registered with this daemon      |
| `deregister_ai_session`       | Soft-disconnect an AI agent session without killing process            |
| `heartbeat_ai_session`        | Refresh last-activity timestamp of an AI session to keep it alive      |
| `purge_disconnected_sessions` | Delete all disconnected AI sessions inactive for > 2 hours             |

### Providers (7)

| Tool                     | Description                                                      |
| ------------------------ | ---------------------------------------------------------------- |
| `get_providers`          | List all registered LLM providers and their liveness status      |
| `discover_providers`     | Scan the local system for installed AI providers/agents          |
| `promote_provider`       | Promote a discovered provider to an active LLM backend           |
| `list_provider_configs`  | List all persisted LLM provider configuration records            |
| `add_provider_config`    | Add a new LLM provider configuration and register when enabled   |
| `update_provider_config` | Update an existing LLM provider configuration by ID              |
| `remove_provider_config` | Delete a persisted provider configuration and deregister adapter |

### Brain / Knowledge (9)

| Tool                  | Description                                                                                |
| --------------------- | ------------------------------------------------------------------------------------------ |
| `get_brain_status`    | Retrieve indexing status and entry count of project knowledge brain                        |
| `ingest_knowledge`    | Parse and inject knowledge from markdown files into project brain storage                  |
| `get_project_context` | Obtain macro context (Architectures, Conventions, File Maps) bounded by token budget       |
| `get_focused_context` | Query task-specific micro context (Learning, Definitions, Gotchas) bounded by token budget |
| `search_knowledge`    | Full-text search across the project's knowledge base via FTS5 BM25 matching                |
| `init_project`        | Auto-ingest CLAUDE.md and initialize a project's knowledge base in one step                |
| `list_knowledge`      | List all knowledge documents stored for a project, optionally filtered by kind             |
| `delete_knowledge`    | Delete a knowledge document by ID from the project repository                              |
| `get_file_map`        | Retrieve cached file path map knowledge document for a project                             |

### Discovery & System (6)

| Tool                    | Description                                                                       |
| ----------------------- | --------------------------------------------------------------------------------- |
| `get_discovered_agents` | Return AI agent tools detected on the local system (Claude CLI, Copilot, etc.)    |
| `delegate_to_nexus`     | Delegate an AI agent session to the nexus task queue                              |
| `get_discovered_plans`  | Scan for plan/task/orchestration files in a project directory                     |
| `howto`                 | Return a complete integration guide — all tools, workflow patterns, and endpoints |
| `howto_brief`           | Ultra-compact integration guide (~200 tokens) for small-context models            |
| `health`                | Check if the orchestrator daemon is running and responsive                        |

## Usage Examples

### Submit a Task

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

### Get Task Status

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

### Check Available Providers

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_providers"
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\"name\":\"LM Studio\",\"active\":true,\"activeModel\":\"codellama\"},{\"name\":\"Ollama\",\"active\":false}]"
      }
    ]
  }
}
```

### Health Check

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "health"
  }
}
```

## Brain / Project Knowledge Tools

nexusOrchestrator's brain layer indexes your project's documentation into a SQLite FTS5 store. AI agents can query it for token-budgeted, semantically-ranked context before working on tasks.

### Get Brain Status

Check the indexing state and token count for a project's knowledge base.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": { "name": "get_brain_status", "arguments": { "projectPath": "/your/project" } }
}
```

### Ingest Knowledge

Parse and ingest a markdown file (e.g. CLAUDE.md) into the project knowledge store.

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "ingest_knowledge",
    "arguments": { "projectPath": "/your/project", "filePath": "/your/project/CLAUDE.md" }
  }
}
```

### Get Project Context

Retrieve the macro context for a project, bounded by a token budget, for use in LLM system prompts.

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_project_context",
    "arguments": { "projectPath": "/your/project", "maxTokens": 800 }
  }
}
```

### Search Knowledge

Full-text BM25 search across all ingested knowledge entries for a project.

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "search_knowledge",
    "arguments": { "projectPath": "/your/project", "query": "architecture", "limit": 5 }
  }
}
```

### Get Focused Context

Retrieve task-specific micro context (Learning, Definitions, Gotchas) bounded by a token budget for a specific reasoning query.

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "get_focused_context",
    "arguments": {
      "projectPath": "/your/project",
      "question": "How does authentication middleware work?",
      "maxTokens": 400
    }
  }
}
```

### Initialize Project Brain

Initialize the knowledge repository schema and auto-ingest `CLAUDE.md` in one call.

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "init_project",
    "arguments": { "projectPath": "/your/project" }
  }
}
```

## Protocol Details

- **Protocol**: JSON-RPC 2.0
- **Version**: `2024-11-05`
- **Endpoint**: `POST /mcp`
- **Health**: `GET /health`
- **Default Port**: 63988 (configurable via `NEXUS_MCP_ADDR`)

The MCP server supports both `initialize` and `tools/list` lifecycle methods, and all tool invocations via `tools/call`.

## Troubleshooting

{: .warning }
**Connection refused**: Make sure the nexus-daemon is running and the MCP port (default 63988) is not blocked by a firewall.

{: .note }
**No tools appearing**: Verify the URL in `claude_desktop_config.json` ends with `/mcp` (not just the host:port).

| Issue                | Solution                                                                |
| -------------------- | ----------------------------------------------------------------------- |
| Connection refused   | Start nexus-daemon first: `./nexus-daemon`                              |
| Port conflict        | Use `NEXUS_MCP_ADDR=:9090` to change the MCP port                       |
| No tools in Claude   | Check URL ends with `/mcp`, restart Claude Desktop                      |
| Task stuck in QUEUED | Check `GET /api/providers` — ensure at least one LLM provider is active |
