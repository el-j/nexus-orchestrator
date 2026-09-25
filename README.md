# nexusOrchestrator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v0.10.0-blue)](https://github.com/el-j/nexusOrchestrator/releases/tag/v0.10.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/el-j/nexusOrchestrator)](https://goreportcard.com/report/github.com/el-j/nexusOrchestrator)

A universal control plane, context broker, and execution gatekeeper that coordinates AI coding workflows across diverse local models and frontier cloud providers, with per-project SQLite FTS5 brain memory and an extensive 41-tool Model Context Protocol (MCP) server.

## Features

**Core & Control Plane**

- Hexagonal architecture (Ports & Adapters) — core never imports adapters
- Multi-turn AI session isolation per project path (SQLite-backed conversation history)
- Deterministic Execution Gates (`writeAndVerifyTaskOutput`) with feedback-driven self-healing loops
- Smart Provider Fallback Chains with role-driven routing (`architect`, `developer`, `tester`, `linter`)
- Real-time workspace change invalidation watcher (`fsnotify`) with 300ms debouncing and auto-indexing
- Token-budgeted Tier 0/1 project onboarding context (`get_onboarding_context` < 800 tokens)

**LLM Backends**

- Google Gemini (native cloud adapter: `gemini-2.5-pro`, `gemini-2.5-flash`, etc.)
- Anthropic Claude (cloud: `claude-3-5-sonnet`, `claude-3-7-sonnet`, etc.)
- OpenAI (`gpt-4o`, `gpt-4o-mini`, etc.)
- LM Studio (local, `127.0.0.1:1234`)
- Ollama (local, `127.0.0.1:11434`)
- Any OpenAI-compatible runtime (vLLM, LocalAI, TextGenUI, Antigravity)

**Provider Discovery & Routing**

- Automatic port + process scanner detects running LLM runtimes at startup
- Role-based routing prioritization: frontier models for architecture/reasoning; local models for formatting/boilerplate
- Automatic rate-limit (HTTP 429) and network failover to secondary providers
- Live availability and health circuit-breaking per provider

**Task Management & Staging**

- Submit / queue / cancel with per-task context files and target file writeback
- Backlog / draft workflow — stage tasks before queuing
- Automated verification command gates (`--verify "go test ./..."`, `--turns 2`)
- Queue cap (configurable, default 50) with crash-recovery on restart

**Interfaces**

- HTTP REST API on `:63987` with SSE events stream (`/api/events`) and Onboarding endpoint (`/api/brain/onboarding`)
- MCP JSON-RPC 2.0 server on `:63988` (41 tools, VS Code Copilot, Claude Code, and Cursor compatible)
- Stdio transport proxy (`nexus-mcp-stdio`) for local agent CLI integration
- Desktop GUI (Wails + Vue 3) with Dashboard, Provider Management, Brain Knowledge, Activity Observatory
- System Tray with quick-stats and task notifications
- [VS Code Extension](vscode-extension/README.md) — auto-registers MCP server via `contributes.mcpServers` (VS Code 1.99+)
- GitHub Action for CI/CD task submission

**Observability & Brain**

- Project Brain with SQLite FTS5 BM25 search and markdown section ingestion
- Real-time AI session registry (track Copilot / Cursor / Claude Desktop / Continue sessions)
- SSE events stream for live task state machines and activity telemetry
- Structured operational logs with non-destructive multi-stage failover tracing

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

| Binary         | Purpose                                            |
| -------------- | -------------------------------------------------- |
| `nexus-daemon` | Headless background daemon (HTTP API + MCP server) |
| `nexus-cli`    | Interactive CLI client for a running daemon        |
| `nexus-submit` | One-shot task submission from a task-file          |

```sh
# Start the daemon
./nexus-daemon
# HTTP API:  http://localhost:63987
# MCP server: http://localhost:63988/mcp
```

### Build from source

```sh
# Build the headless daemon
CGO_ENABLED=1 go build ./cmd/nexus-daemon/...

# Run it
./nexus-daemon
# HTTP API:  http://localhost:63987
# MCP server: http://localhost:63988/mcp
```

### VS Code Extension

Install `nexus-orchestrator-0.2.0.vsix` from the [releases page](https://github.com/el-j/nexusOrchestrator/releases) or build from source:

```sh
cd vscode-extension && npm install && npm run package
code --install-extension nexus-orchestrator-0.2.0.vsix
```

The extension auto-registers the `nexus-orchest` MCP server in VS Code 1.99+ — no manual configuration required. See [vscode-extension/README.md](vscode-extension/README.md) for details.

## MCP Integration (Claude Desktop)

Add the following to your Claude Desktop `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "nexusOrchestrator": {
      "url": "http://localhost:63988/mcp"
    }
  }
}
```

### Available MCP Tools

| Tool            | Description                                                          |
| --------------- | -------------------------------------------------------------------- |
| `submit_task`   | Submit a code-generation task (projectPath, targetFile, instruction) |
| `get_task`      | Get status and output of a task by ID                                |
| `get_queue`     | List all pending tasks                                               |
| `cancel_task`   | Cancel a queued task by ID                                           |
| `get_providers` | List available LLM backends and their status                         |
| `health`        | Check daemon connectivity                                            |

## VS Code Extension

A companion extension routes tasks from VS Code directly to the running daemon:

- **Submit tasks** via Command Palette or right-click context menu
- **Task Queue** sidebar shows live status of all tasks
- **Status bar** shows daemon health and active task count
- **Provider picker** lets you select LM Studio, Ollama, or cloud backends per task

See [`vscode-extension/README.md`](vscode-extension/README.md) for installation and usage.

## Environment Variables

| Variable            | Default    | Description               |
| ------------------- | ---------- | ------------------------- |
| `NEXUS_DB_PATH`     | `nexus.db` | SQLite database path      |
| `NEXUS_LISTEN_ADDR` | `:63987`   | HTTP API listen address   |
| `NEXUS_MCP_ADDR`    | `:63988`   | MCP server listen address |

## Build & Test

```sh
CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go build ./...
CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...
go vet ./...
```

## Dogfooding

nexusOrchestrator is developed using itself. To use it on this codebase:

1. Start the daemon: `./nexus-daemon` (or run the Wails desktop app)
2. Browse to `http://localhost:63987` or open the VS Code extension
3. Submit a task pointing to this repo — e.g.:
   `nexus-submit --project . --target internal/core/services/orchestrator.go --task-file .claude/tasks/TASK-562.md --verify "go test ./..." --turns 2`
4. Monitor progress in the GUI Dashboard or via `nexus-cli queue`

Task definitions live in `.claude/tasks/` and plans in `.claude/plans/`.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
