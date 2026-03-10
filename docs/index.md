---
layout: default
title: Home
nav_order: 1
description: "nexusOrchestrator — Route AI code-generation tasks to any LLM backend with session memory and MCP support"
---

# nexusOrchestrator
{: .fs-9 }

Route AI code-generation tasks to any LLM backend — locally or in the cloud.
{: .fs-6 .fw-300 }

[Download](/nexusOrchestrator/downloads){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[Get Started](/nexusOrchestrator/getting-started){: .btn .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/nexus-orchestrator/nexusOrchestrator){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## What is nexusOrchestrator?

nexusOrchestrator is a **local AI task orchestrator** written in Go that manages code-generation tasks across multiple LLM providers. It provides per-project session isolation, automatic provider discovery with failover, a full REST API, and an MCP server compatible with Claude Desktop.

Whether you're running models locally with [LM Studio](https://lmstudio.ai/) or [Ollama](https://ollama.ai/), or using cloud providers like OpenAI and Anthropic, nexusOrchestrator routes your tasks intelligently to the right backend.

---

## Key Features

### Multi-Backend LLM Routing
Automatically discovers and routes tasks to LM Studio, Ollama, OpenAI, Anthropic, or any OpenAI-compatible endpoint. Per-task model selection and provider hints give you fine-grained control.

### Per-Project Session Memory
Every project path gets its own SQLite-backed conversation history. Multi-turn interactions maintain context across tasks, so your LLM always knows what came before.

### MCP Server (Model Context Protocol)
Built-in JSON-RPC 2.0 server on port 9998, compatible with Claude Desktop and any MCP client. Six tools: submit_task, get_task, get_queue, cancel_task, get_providers, and health.

### Full HTTP REST API
Complete task management on port 9999 — submit, list, cancel, and monitor tasks. Real-time updates via Server-Sent Events (SSE). Provider CRUD for dynamic backend management.

### Context-Window Guard
Pre-flight token estimation prevents prompt overflow before it reaches the LLM. Tasks exceeding the model's context limit are flagged as TOO_LARGE immediately.

### Smart Provider Discovery
Auto-detect running LLM backends with health checks and failover. If your preferred provider goes down, the orchestrator finds an alternative automatically.

### Command-Aware Task Routing
Classify tasks as "plan" (orchestration/documentation) or "execute" (code implementation). The orchestrator enforces that execution tasks have a prior plan, preventing uncoordinated changes.

### Desktop GUI + System Tray
Native desktop application powered by Wails with an embedded web dashboard, real-time task monitoring, and system tray integration.

---

## Quick Start

```sh
# Build the headless daemon
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...

# Start it
./nexus-daemon
# HTTP API:  http://localhost:9999
# MCP server: http://localhost:9998/mcp
# Dashboard:  http://localhost:9999/ui
```

```sh
# Submit a task via curl
curl -X POST http://localhost:9999/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "projectPath": "/path/to/your/project",
    "targetFile": "output.go",
    "instruction": "Write a function that sorts a slice of strings"
  }'
```

---

## Architecture

nexusOrchestrator follows a **hexagonal architecture** (ports & adapters) with a strict inward dependency rule:

```
Inbound Adapters  →  Core Services  →  Ports  ←  Outbound Adapters
(HTTP, MCP, CLI)     (Orchestrator)    (interfaces)  (LLM, SQLite, FS)
```

[Learn more about the architecture →](/nexusOrchestrator/architecture)

---

## Supported Providers

| Provider | Type | Default Address |
|----------|------|----------------|
| LM Studio | Local | `127.0.0.1:1234` |
| Ollama | Local | `127.0.0.1:11434` |
| OpenAI | Cloud | `api.openai.com` |
| Anthropic | Cloud | via SDK |
| GitHub Copilot | Cloud | `api.githubcopilot.com` |
| Any OpenAI-compatible | Cloud/Local | Configurable |

---

## What's Next?

- [Downloads](/nexusOrchestrator/downloads) — Desktop app and CLI binaries for all platforms
- [Getting Started](/nexusOrchestrator/getting-started) — Installation, configuration, and your first task
- [Architecture](/nexusOrchestrator/architecture) — Deep dive into the hexagonal design
- [API Reference](/nexusOrchestrator/api-reference) — Complete HTTP and MCP endpoint documentation
- [MCP Integration](/nexusOrchestrator/mcp-integration) — Connect with Claude Desktop
