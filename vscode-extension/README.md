# Nexus Orchestrator for VS Code

> Route AI code-generation tasks from your editor through [nexusOrchestrator](../README.md) — leveraging LM Studio, Ollama, OpenAI, Anthropic, or GitHub Copilot backends.

## Prerequisites

- nexusOrchestrator daemon running (`nexus-daemon` binary or the desktop app)
- Daemon accessible at `http://127.0.0.1:63987` (default)

## Installation

Install via VSIX:

```sh
# In the vscode-extension/ directory
npm install
npm run build
npm run package          # produces nexus-orchestrator-0.1.0.vsix
code --install-extension nexus-orchestrator-0.1.0.vsix
```

> Marketplace listing coming soon.

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `nexus.daemonUrl` | `http://127.0.0.1:63987` | URL of running daemon |
| `nexus.defaultProvider` | `` | Pre-selected provider name |
| `nexus.defaultModel` | `` | Pre-selected model |

Set these in **File → Preferences → Settings** (`⌘,`) and search for `nexus`.

## Commands

| Command | Description |
|---------|-------------|
| Nexus: Submit Task | Submit a code task to the orchestrator |
| Nexus: View Task Queue | Focus the Task Queue panel |
| Nexus: Select Provider / Model | Choose default provider + model |
| Nexus: Show Providers | Same as selecting providers |
| Nexus: Cancel Task | Cancel a queued/running task |

All commands are accessible via the Command Palette (`⌘⇧P` / `Ctrl+Shift+P`).

## Usage

### Submitting a task

1. Select code in the editor (optional — used as instruction pre-fill)
2. Run **Nexus: Submit Task** from the Command Palette (`⌘⇧P`)
3. Enter your instruction
4. Pick provider/model (or choose Auto)
5. A progress notification shows while the task runs
6. When done, click **Open File** to see the result

### Task Queue sidebar

The Nexus activity bar icon (⚡) shows all tasks with live status. Right-click a task to cancel it.

### Status bar

The bottom-left status bar item shows daemon connection status and active task count. Click it for quick actions.

## Using nexusOrchestrator as a Copilot proxy

Submit tasks to route Copilot-style requests through any configured provider — including local LM Studio/Ollama instances — giving you control over which model handles each task.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `⚠ Nexus: offline` in status bar | Start `nexus-daemon` or the desktop app |
| Wrong daemon port | Set `nexus.daemonUrl` in VS Code settings |
| No providers in picker | Add providers via the Nexus GUI or set `NEXUS_OLLAMA_URL` / `NEXUS_LMSTUDIO_URL` |
