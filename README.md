# nexusOrchestrator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/el-j/nexus-orchestrator?include_prereleases&label=release)](https://github.com/el-j/nexus-orchestrator/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/el-j/nexus-orchestrator)](https://goreportcard.com/report/github.com/el-j/nexus-orchestrator)

Nexus runs coding tasks on the AI models you already have and keeps what it learns about each project. Give it a task (from the UI, the CLI, an MCP tool, or your editor) and it routes it to a local model, an API, your Claude or Copilot subscription, writes the result, runs your verification command, and remembers the outcome. Agents that work on their own report through the same tools, so one place knows what happened in every project.

## What it does (and what it does not)

| It does                                                                                                     | How work gets there                                                                                                         |
| ----------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| **Runs a task itself** on a provider, writes the file, verifies it, streams the text while the model writes | `submit_task` (MCP), `POST /api/tasks`, `nexus-submit`, the UI, the VS Code extension                                       |
| **Lets an agent report its own work**                                                                       | `register_session`, `claim_task`, `update_task_status` (MCP)                                                                |
| **Remembers a project** (the brain): architecture, conventions, file map, what was tried, the active plan   | `init_project`, `ingest_knowledge`, `get_onboarding_context`; finished tasks are remembered when the project policy says so |
| **Watches** what some agents do                                                                             | nothing to do: Claude Code, Continue and network activity are read                                                          |

Editing `.claude/orchestrator.json`, plan files or the database by hand is **not** a way in: Nexus shows plan files, but nothing is run or remembered because a file changed. [docs/how-work-reaches-nexus.md](docs/how-work-reaches-nexus.md) explains each way in and how to connect the agents on your machine (`nexus-cli connect --write`).

## Features

**Run tasks on any model**

- Local: LM Studio, Ollama, any OpenAI-compatible runtime (vLLM, LocalAI, text-generation-webui)
- Cloud: OpenAI, Anthropic, Google Gemini, GitHub Copilot (token), any OpenAI-compatible API
- **Your subscriptions, no API key**: Claude Pro/Max through the logged-in `claude` CLI; GitHub Copilot and other models of your editor through the VS Code model bridge ([docs/delegating-subscriptions.md](docs/delegating-subscriptions.md))
- Automatic provider discovery, health checks, rate-limit and network failover, role-based routing (frontier models for architecture, local ones for boilerplate)
- **Live output**: the reply streams to the UI and to VS Code while the model writes it (`task.chunk` events, MCP progress notifications); a failing provider hands over to the next one without partial text reaching the file ([docs/streaming.md](docs/streaming.md))

**A workflow with rules**

- `.nexus/policy.json` per project: default verification command, file-size limit, tests first, free-text rules, task memory ([docs/rulebased-workflow.md](docs/rulebased-workflow.md))
- Plan-first gate, deterministic verification with self-healing correction turns, an agentic mode where the model reads and edits several files
- Plans and tasks found in any folder and linked automatically (`PLAN-n.md` / `TASK-n.md`, `.claude`, `.nexus`, or the folders you configure) ([docs/plan-files.md](docs/plan-files.md))

**Project brain**

- SQLite FTS5 knowledge per project, learned from `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, Copilot instructions and what agents ingest
- A token-budgeted briefing (rules, stack, active plan, plan files, learnings) for every new session and every task

**Interfaces**

- Full web UI served by the daemon at `http://localhost:63987/app/` (Mission Control, Agents, Providers, Projects with Activity, Plans, History, Backlog and Brain tabs, Settings); a light dashboard stays at `/ui`
- Desktop app (Wails) for macOS, Windows and Linux
- REST API on `:63987` with an SSE event stream; MCP server on `:63988` (42 tools) for Claude Code, Claude Desktop, Cursor, Antigravity, VS Code Copilot; `nexus-mcp-stdio` for agents that start MCP servers as commands
- CLI (`nexus-cli`, `nexus-submit`), GitHub Action
- [VS Code extension](vscode-extension/README.md): task queue, live output, plan view, brain commands, editor models as providers, and it connects the editor's own agent to Nexus (VS Code 1.101+ through its MCP support; Antigravity through its MCP config)

**Built to be trusted**

- Hexagonal architecture (the core never imports adapters), real-daemon end-to-end tests, 100% coverage gates on the extension and the frontend, mutation testing (local), a cross-surface parity matrix
- Local first, zero telemetry, loopback-only by default with origin and host guards

## Quick Start

New here? `make` lists everything; [docs/using-nexus.md](docs/using-nexus.md) walks through running one orchestrator for all your projects.

### From source: one command

```sh
nvm use            # Node >= 22 builds the full UI; without it you get the light dashboard
make start         # builds and runs the daemon with ONE database for every project (~/.nexus/nexus.db)
# Full UI:  http://localhost:63987/app/    API: :63987    MCP: http://localhost:63988/mcp
make connect       # shows which agents on this machine can call Nexus (nexus-cli connect --write connects them)
make init-project PROJECT=/path/to/your/repo
```

### macOS (desktop app)

The easiest way is the installer. It downloads the app with `curl`, checks its checksum and puts it in `~/Applications` without the macOS quarantine flag, so it opens with a double click:

```sh
curl -sSfL https://raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh | sh -s -- --desktop
# newest beta instead of the last stable release:  ... | sh -s -- --desktop --beta
```

Or download `nexus-orchestrator-desktop-darwin-arm64.zip` (Apple Silicon) or `nexus-orchestrator-desktop-darwin-amd64.zip` (Intel) from [Releases](https://github.com/el-j/nexus-orchestrator/releases). The app is not notarized with Apple yet, so macOS refuses a browser-downloaded copy ("Apple could not verify nexusOrchestrator.app"). Either remove the quarantine flag first:

```sh
xattr -dr com.apple.quarantine nexusOrchestrator.app
open nexusOrchestrator.app
```

or, after the blocked attempt, open **System Settings → Privacy & Security**, scroll to the message about nexusOrchestrator and click **Open Anyway** (macOS 15 and later no longer offers right-click → Open). The app keeps its data in `~/.nexus/nexus.db`; do not run it next to a separately started daemon (both want `:63987`).

### CLI / daemon (all platforms)

Download `nexus-orchestrator-<os>-<arch>.tar.gz` (`.zip` on Windows) from [Releases](https://github.com/el-j/nexus-orchestrator/releases), or `curl -sSfL https://raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh | sh`. The archive contains:

| Binary            | Purpose                                                            |
| ----------------- | ------------------------------------------------------------------ |
| `nexus-daemon`    | Headless daemon: REST API, MCP server, light dashboard (`/ui`)     |
| `nexus-cli`       | CLI client: queue, drafts, brain, `connect`                        |
| `nexus-submit`    | One-shot task submission from a task file                          |
| `nexus-mcp-stdio` | stdio-to-HTTP bridge for agents that start MCP servers as commands |

```sh
./nexus-daemon     # database in ~/.nexus/nexus.db
# REST: http://localhost:63987   MCP: http://localhost:63988/mcp
```

The full web UI is embedded by `make start`/`make web` (Node >= 22) and is part of the desktop app.

### VS Code / Antigravity extension

Install `nexus-orchestrator.vsix` from [Releases](https://github.com/el-j/nexus-orchestrator/releases) (`code --install-extension nexus-orchestrator.vsix`), or build it: `make build-vscode`. It needs a running daemon (`nexus-daemon`, `make start` or the desktop app) and VS Code 1.99+. It also connects the editor's own agent to Nexus: VS Code 1.101+ gets the MCP server through its MCP support, Antigravity is offered a change to its `mcp_config.json` once (setting `nexus.mcp.autoConnect`). Details: [vscode-extension/README.md](vscode-extension/README.md).

## Connecting agents (MCP)

An agent that cannot call Nexus tools can only work around them. Connect the agents on your machine:

```sh
nexus-cli connect              # report only
nexus-cli connect --write      # Antigravity, Cursor, Claude Code: merges the nexus server into each config
```

Claude Desktop (`claude_desktop_config.json`):

```json
{ "mcpServers": { "nexusOrchestrator": { "url": "http://localhost:63988/mcp" } } }
```

### MCP tools (42)

| Group          | Tools                                                                                                                                                                                                                         |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Tasks          | `submit_task`, `get_task`, `wait_for_task` (blocks until done; streams the model's text as progress notifications), `get_queue`, `get_all_tasks`, `cancel_task`, `create_draft`, `get_backlog`, `promote_task`, `update_task` |
| Agent sessions | `register_session`, `claim_task`, `update_task_status`, `heartbeat_task`, `get_ai_sessions`, `delegate_to_nexus`, ...                                                                                                         |
| Providers      | `get_providers`, `health`, `discover_providers`, `promote_provider`, `add_provider_config`, ...                                                                                                                               |
| Brain          | `get_onboarding_context`, `get_project_context`, `get_focused_context`, `search_knowledge`, `ingest_knowledge`, `init_project`, ...                                                                                           |

Every tool with its arguments: [docs/api-reference.md](docs/api-reference.md).

## Environment variables

| Variable                                                                                               | Default             | Description                                                                              |
| ------------------------------------------------------------------------------------------------------ | ------------------- | ---------------------------------------------------------------------------------------- |
| `NEXUS_DB_PATH`                                                                                        | `~/.nexus/nexus.db` | SQLite database path                                                                     |
| `NEXUS_LISTEN_ADDR`                                                                                    | `127.0.0.1:63987`   | REST API listen address                                                                  |
| `NEXUS_MCP_ADDR`                                                                                       | `127.0.0.1:63988`   | MCP server listen address                                                                |
| `NEXUS_API_TOKEN` / `NEXUS_MCP_TOKEN`                                                                  | none                | Bearer tokens for the REST API / MCP server                                              |
| `NEXUS_OPENAI_API_KEY`, `NEXUS_ANTHROPIC_API_KEY`, `NEXUS_GEMINI_API_KEY`, `NEXUS_GITHUBCOPILOT_TOKEN` | none                | Cloud providers (each with a `..._MODEL`)                                                |
| `NEXUS_CLI_PROVIDERS`                                                                                  | none                | Extra executables a `cli` provider may run (default: claude, aichat, llm, gemini, codex) |
| `NEXUS_ALLOWED_ORIGINS` / `NEXUS_ALLOWED_HOSTS`                                                        | none                | Extra browser origins / host names the local servers accept                              |

## Build & test

```sh
make test          # Go (race), frontend, docs, extension
make nice          # format, vet, lint, type-check
CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...
```

## Dogfooding

Nexus is developed with itself: `make dogfood` starts the daemon and loads this repository's `CLAUDE.md`; see [CLAUDE.md](CLAUDE.md) for the workflow. Plans live in `.claude/plans/`, tasks in `.claude/tasks/`.

## Documentation

[How work reaches Nexus](docs/how-work-reaches-nexus.md) · [Using Nexus](docs/using-nexus.md) · [Delegating to your subscriptions](docs/delegating-subscriptions.md) · [VS Code model bridge](docs/vscode-model-bridge.md) · [Live output](docs/streaming.md) · [Rule-based workflow](docs/rulebased-workflow.md) · [Plan files](docs/plan-files.md) · [API reference](docs/api-reference.md) · [Architecture](docs/architecture.md) · [Changelog](CHANGELOG.md)

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
