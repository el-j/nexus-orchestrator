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
      "url": "http://localhost:9998/mcp"
    }
  }
}
```

Restart Claude Desktop after editing the configuration. The nexusOrchestrator tools will appear in Claude's tool palette.

{: .note }
Make sure the nexus-daemon is running before starting Claude Desktop.

## Available Tools

| Tool | Description |
|------|-------------|
| `submit_task` | Submit a code-generation task with project path, target file, and instruction |
| `get_task` | Retrieve the status and output of a task by its ID |
| `get_queue` | List all pending (QUEUED/PROCESSING) tasks |
| `cancel_task` | Cancel a queued task before it is processed |
| `get_providers` | List all registered LLM providers and their liveness status |
| `health` | Check if the orchestrator daemon is running and responsive |

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

## Protocol Details

- **Protocol**: JSON-RPC 2.0
- **Version**: `2024-11-05`
- **Endpoint**: `POST /mcp`
- **Health**: `GET /health`
- **Default Port**: 9998 (configurable via `NEXUS_MCP_ADDR`)

The MCP server supports both `initialize` and `tools/list` lifecycle methods, and all tool invocations via `tools/call`.

## Troubleshooting

{: .warning }
**Connection refused**: Make sure the nexus-daemon is running and the MCP port (default 9998) is not blocked by a firewall.

{: .note }
**No tools appearing**: Verify the URL in `claude_desktop_config.json` ends with `/mcp` (not just the host:port).

| Issue | Solution |
|-------|----------|
| Connection refused | Start nexus-daemon first: `./nexus-daemon` |
| Port conflict | Use `NEXUS_MCP_ADDR=:9090` to change the MCP port |
| No tools in Claude | Check URL ends with `/mcp`, restart Claude Desktop |
| Task stuck in QUEUED | Check `GET /api/providers` — ensure at least one LLM provider is active |