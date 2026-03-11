# nexusOrchestrator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/el-j/nexusOrchestrator)](https://github.com/el-j/nexusOrchestrator/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/el-j/nexusOrchestrator)](https://goreportcard.com/report/github.com/el-j/nexusOrchestrator)

A local AI orchestrator that routes code-generation tasks to LM Studio or Ollama, with per-project session memory and a full MCP server for use with Claude Desktop or other MCP clients.

## Features

- **Hexagonal architecture** — clean separation between domain, ports, and adapters
- **Multi-backend LLM routing** — detects and routes to LM Studio (`:1234`) or Ollama (`:11434`) automatically
- **Session isolation** — each project path gets its own multi-turn conversation history stored in SQLite
- **HTTP API** on `:9999` — REST endpoint for task management
- **MCP server** on `:9998` — JSON-RPC 2.0 endpoint compatible with Claude Desktop and any MCP client
- **Desktop GUI** via Wails + system tray

## Quick Start

### macOS (Desktop App)

Download `nexus-orchestrator-desktop-darwin-arm64.zip` (Apple Silicon) or `nexus-orchestrator-desktop-darwin-amd64.zip` (Intel) from [Releases](https://github.com/el-j/nexus-orchestrator/releases/latest).

> **First-run on macOS:** Because the app is not yet notarized with Apple, macOS Gatekeeper applies a quarantine attribute when you download it. Remove it before opening:
>
> ```sh
> xattr -dr com.apple.quarantine nexusOrchestrator.app
> open nexusOrchestrator.app
> ```

### CLI / Daemon (all platforms)

Download `nexus-orchestrator-darwin-arm64.tar.gz` (or your platform's archive) from [Releases](https://github.com/el-j/nexus-orchestrator/releases/latest). The archive contains three binaries:

| Binary | Purpose |
|--------|---------|
| `nexus-daemon` | Headless background daemon (HTTP API + MCP server) |
| `nexus-cli` | Interactive CLI client for a running daemon |
| `nexus-submit` | One-shot task submission from a task-file |

```sh
# Start the daemon
./nexus-daemon
# HTTP API:  http://localhost:9999
# MCP server: http://localhost:9998/mcp
```

### Build from source

```sh
# Build the headless daemon
CGO_ENABLED=1 go build ./cmd/nexus-daemon/...

# Run it
./nexus-daemon
# HTTP API:  http://localhost:9999
# MCP server: http://localhost:9998/mcp
```

## MCP Integration (Claude Desktop)

Add the following to your Claude Desktop `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "nexusOrchestrator": {
      "url": "http://localhost:9998/mcp"
    }
  }
}
```

### Available MCP Tools

| Tool | Description |
|------|-------------|
| `submit_task` | Submit a code-generation task (projectPath, targetFile, instruction) |
| `get_task` | Get status and output of a task by ID |
| `get_queue` | List all pending tasks |
| `cancel_task` | Cancel a queued task by ID |
| `get_providers` | List available LLM backends and their status |
| `health` | Check daemon connectivity |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXUS_DB_PATH` | `nexus.db` | SQLite database path |
| `NEXUS_LISTEN_ADDR` | `:9999` | HTTP API listen address |
| `NEXUS_MCP_ADDR` | `:9998` | MCP server listen address |

## Build & Test

```sh
CGO_ENABLED=1 go build ./...
CGO_ENABLED=1 go test -race ./...
go vet ./...
```

## Dogfooding — use nexusOrchestrator for its own development

nexusOrchestrator can submit its own PLAN-002 backlog tasks to itself for LLM implementation, validating the entire pipeline end-to-end.

### Prerequisites

- LM Studio running at `http://127.0.0.1:1234/v1` **or** Ollama at `http://127.0.0.1:11434`
- `CGO_ENABLED=1 go build ./...` passes

### Start the daemon and open the live dashboard

```sh
# Build and start the daemon
CGO_ENABLED=1 go build -o /tmp/nexus-daemon ./cmd/nexus-daemon
NEXUS_DB_PATH=/tmp/nexus-local.db /tmp/nexus-daemon &

# Open dashboard in browser
open http://localhost:9999/ui
```

The dashboard at `GET /ui` auto-refreshes every 2 seconds and streams live task status updates via Server-Sent Events.

### Submit a PLAN-002 implementation task

```sh
# Build nexus-submit once
CGO_ENABLED=1 go build -o /tmp/nexus-submit ./cmd/nexus-submit

# Submit TASK-013 (orchestrator hardening) to the running daemon
/tmp/nexus-submit \
  --task-file .claude/tasks/TASK-013.md \
  --project "$PWD" \
  --target internal/core/services/orchestrator.go \
  --context internal/core/services/orchestrator.go,internal/core/ports/ports.go \
  --wait
```

### Run all PLAN-002 tasks at once

```sh
./scripts/dogfood-plan002.sh
```

The script builds both binaries, starts a fresh daemon with an isolated DB, submits all 8 PLAN-002 tasks, and prints each task ID. It cleans up the daemon and DB file on exit.

### Track via MCP (for AI editors)

Point any MCP-compatible agent at `http://localhost:9998` and call:

```json
{"method": "tools/call", "params": {"name": "get_queue"}}
```

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
