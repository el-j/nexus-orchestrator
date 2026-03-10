---
layout: default
title: Getting Started
nav_order: 4
---

# Getting Started
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Prerequisites

- **Go 1.24+** with CGO support (`CGO_ENABLED=1`)
- **C compiler** — `gcc` or `clang` (required by `go-sqlite3`)
- **At least one LLM provider**:
  - [LM Studio](https://lmstudio.ai/) running on `127.0.0.1:1234`, or
  - [Ollama](https://ollama.ai/) running on `127.0.0.1:11434`, or
  - Cloud API keys for OpenAI, Anthropic, or GitHub Copilot

## Installation

```sh
# Clone the repository
git clone https://github.com/el-j/nexusOrchestrator.git
cd nexusOrchestrator

# Build all binaries
CGO_ENABLED=1 go build ./...

# Or build specific binaries
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...
CGO_ENABLED=1 go build -o nexus-cli ./cmd/nexus-cli/...
```

## Starting the Daemon

```sh
# Start with default settings
./nexus-daemon
# HTTP API:  http://127.0.0.1:9999
# MCP server: http://127.0.0.1:9998/mcp
# Dashboard:  http://127.0.0.1:9999/ui
```

### Custom Configuration

```sh
# Use environment variables for custom settings
NEXUS_DB_PATH=/path/to/nexus.db \
NEXUS_LISTEN_ADDR=:8080 \
NEXUS_MCP_ADDR=:8081 \
./nexus-daemon
```

### Cloud Provider Configuration

```sh
# OpenAI
NEXUS_OPENAI_API_KEY=sk-... NEXUS_OPENAI_MODEL=gpt-4o-mini ./nexus-daemon

# Anthropic
NEXUS_ANTHROPIC_API_KEY=sk-ant-... NEXUS_ANTHROPIC_MODEL=claude-3-5-sonnet-20241022 ./nexus-daemon

# GitHub Copilot
NEXUS_GITHUBCOPILOT_TOKEN=ghu_... NEXUS_GITHUBCOPILOT_MODEL=gpt-4o ./nexus-daemon
```

## Submitting Your First Task

```sh
# Submit a code-generation task
curl -s -X POST http://localhost:9999/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "projectPath": "'$PWD'",
    "targetFile": "hello.go",
    "instruction": "Write a Go function that returns Hello World"
  }' | jq .
```

Response:
```json
{
  "id": "a1b2c3d4-...",
  "projectPath": "/path/to/project",
  "targetFile": "hello.go",
  "instruction": "Write a Go function...",
  "status": "QUEUED",
  "createdAt": "2025-01-01T00:00:00Z"
}
```

## Checking Task Status

```sh
# Get task by ID
curl -s http://localhost:9999/api/tasks/TASK_ID | jq .

# List all pending tasks
curl -s http://localhost:9999/api/tasks | jq .
```

## Cancelling a Task

```sh
curl -X DELETE http://localhost:9999/api/tasks/TASK_ID
# Returns 204 No Content on success
```

## Managing Providers at Runtime

```sh
# List all providers
curl -s http://localhost:9999/api/providers | jq .

# Register a new cloud provider
curl -s -X POST http://localhost:9999/api/providers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My OpenAI",
    "kind": "openai-compat",
    "baseURL": "https://api.openai.com/v1",
    "apiKey": "sk-...",
    "model": "gpt-4o-mini"
  }' | jq .

# Remove a provider
curl -X DELETE http://localhost:9999/api/providers/My%20OpenAI
```

## Using the Dashboard

Open `http://localhost:9999/ui` in your browser for a live dashboard that:
- Shows all tasks with real-time status updates via SSE
- Allows submitting new tasks
- Displays provider status and model information
- Auto-refreshes every 2 seconds

## Running Tests

```sh
# Full test suite with race detection
CGO_ENABLED=1 go test -race ./...

# Service tests only
CGO_ENABLED=1 go test ./internal/core/services/...

# Lint
go vet ./...
```

## Next Steps

- [API Reference](/nexusOrchestrator/api-reference) — Full HTTP and MCP endpoint docs
- [MCP Integration](/nexusOrchestrator/mcp-integration) — Connect with Claude Desktop
- [Architecture](/nexusOrchestrator/architecture) — Understand the hexagonal design