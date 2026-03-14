# PLAN-018 — Provider Visibility, Dynamic Config & VS Code Extension

## Goal

Fix provider discovery so local LM Studio and Docker-hosted Ollama instances appear in the GUI, add runtime provider configuration (add/edit/remove from the UI without env vars), and create a VS Code extension that routes Copilot-style code tasks through nexusOrchestrator.

## Problem Statement

1. **Providers invisible in GUI**: LM Studio runs on `127.0.0.1:1234` and Ollama in Docker (likely on a non-default host/port). The current discovery hardcodes `127.0.0.1:11434` for Ollama — Docker-mapped ports or remote hosts are missed. Cloud providers (OpenAI, Anthropic, GitHub) require env vars set *before* app launch and can't be added at runtime from the UI.

2. **No runtime provider management**: Users must restart the app with new env vars to add/remove cloud providers. The `RegisterCloudProvider` API exists but the GUI provider-management flow doesn't persist or surface it well.

3. **No VS Code integration**: There's no way for VS Code / Copilot users to route coding tasks through nexusOrchestrator. A thin VS Code extension that submits tasks to the daemon's HTTP API would let any Copilot-using workspace leverage the orchestrator's multi-provider routing.

## Workstreams

### A — Fix Provider Discovery & Display (TASK-125 → TASK-128)
- Make Ollama base URL configurable (env var + runtime)
- Make LM Studio base URL configurable (env var + runtime)  
- Show *all* registered providers in the GUI (both alive and unreachable), not just alive ones
- Add a "Refresh Providers" button and improve error messaging

### B — Runtime Provider CRUD in GUI (TASK-129 → TASK-131)
- Persist provider configs in SQLite so they survive restarts
- Build a provider-management panel in the Wails GUI (add/edit/remove providers with base URL + API key)
- Wire the Wails bindings and HTTP API for full CRUD lifecycle

### C — VS Code Extension (TASK-132 → TASK-136)
- Scaffold the VS Code extension (TypeScript, package.json, activation)
- Implement commands: submit task, view queue, pick provider/model
- Connect to daemon HTTP API at `127.0.0.1:63987` (configurable)
- Add status bar item showing daemon connection + active task count
- Extension documentation and README

## Dependencies

```
TASK-125 ─┐
TASK-126 ─┤─→ TASK-128 (display all providers)
TASK-127 ─┘
TASK-129 ──→ TASK-130 ──→ TASK-131
TASK-132 ──→ TASK-133 ──→ TASK-134 ──→ TASK-135 ──→ TASK-136
```

Workstreams A, B, and C are independent and can execute in parallel.
