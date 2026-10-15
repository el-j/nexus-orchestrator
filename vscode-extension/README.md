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

| Setting                 | Default                  | Description                |
| ----------------------- | ------------------------ | -------------------------- |
| `nexus.daemonUrl`       | `http://127.0.0.1:63987` | URL of running daemon      |
| `nexus.defaultProvider` | ``                       | Pre-selected provider name |
| `nexus.defaultModel`    | ``                       | Pre-selected model         |

Set these in **File → Preferences → Settings** (`⌘,`) and search for `nexus`.

## Commands

| Command                        | Description                            |
| ------------------------------ | -------------------------------------- |
| Nexus: Submit Task             | Submit a code task to the orchestrator |
| Nexus: View Task Queue         | Focus the Task Queue panel             |
| Nexus: Select Provider / Model | Choose default provider + model        |
| Nexus: Show Providers          | Same as selecting providers            |
| Nexus: Cancel Task             | Cancel a queued/running task           |

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

### Workspace Agents (plans and tasks)

The **Workspace Agents** view lists the plans and tasks of every open folder, whichever agent wrote
them: `.claude/plans/PLAN-*.md` and `.claude/tasks/TASK-*.md` (and the same under `.nexus/`), plus
`.claude/orchestrator.json` when a tool maintains one. Front matter (`id`, `title`, `status`,
`planId`, `tasks`) is used when present; otherwise the first heading, a `Status:` line and the first
`PLAN-n` mentioned. The active plan is the one `orchestrator.json` names, else one marked
active/in-progress, else the newest unfinished one. The view refreshes when those files change. Plans and tasks in other folders are found when they
are named `PLAN-n.md` / `TASK-n.md`; for any other layout run **Nexus: Choose Plan and Task Folders**
(or set `nexus.planFiles.*`). See [docs/plan-files.md](../docs/plan-files.md).
The daemon's onboarding context reads the same files, so a fresh session learns the active plan
even without an `orchestrator.json`.

### Connecting the editor's agent to Nexus

An agent that cannot call the Nexus MCP tools can only work around them. The extension connects the
agent of the editor it runs in:

- **VS Code 1.101+**: registers the daemon's MCP server with VS Code's MCP support (nothing is written;
  VS Code asks you to trust it the first time; see Chat → Configure Tools).
- **Antigravity**: its agent reads `~/.gemini/.../mcp_config.json`. The extension offers once to add the
  `nexus` server there (a bundled bridge, `dist/mcpProxy.js`, run with the editor's own Node) and keeps
  the entry current across updates. Setting `nexus.mcp.autoConnect`: `ask` (default), `always`, `never`.
  Existing servers stay and the old file is kept as a backup. Reload the window afterwards.
- Any editor: **Nexus: Connect This Editor's Agent** does it on demand, or run `nexus-cli connect --write`.

### Status bar

The bottom-left status bar item shows daemon connection status and active task count. Click it for quick actions.

## Using nexusOrchestrator as a Copilot proxy

Submit tasks to route Copilot-style requests through any configured provider — including local LM Studio/Ollama instances — giving you control over which model handles each task.

## Troubleshooting

| Symptom                          | Fix                                                                              |
| -------------------------------- | -------------------------------------------------------------------------------- |
| `⚠ Nexus: offline` in status bar | Start `nexus-daemon` or the desktop app                                          |
| Wrong daemon port                | Set `nexus.daemonUrl` in VS Code settings                                        |
| No providers in picker           | Add providers via the Nexus GUI or set `NEXUS_OLLAMA_URL` / `NEXUS_LMSTUDIO_URL` |

## Release

To publish a new version to the VS Code Marketplace and Open VSX Registry, the CI pipeline requires the following repository secrets:

| Secret     | Description                                                                                                                                                                                                                                  |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `VSCE_PAT` | Personal access token for the [VS Code Marketplace](https://marketplace.visualstudio.com/). Generate one at [dev.azure.com](https://dev.azure.com) under **User Settings → Personal Access Tokens** with the **Marketplace → Manage** scope. |
| `OVSX_PAT` | Personal access token for the [Open VSX Registry](https://open-vsx.org/). Generate one at `https://open-vsx.org/user-settings/tokens` after logging in.                                                                                      |

Both publish steps only run when a `refs/tags/v*` tag is pushed (or a GitHub Release is published). If a secret is absent the corresponding step is silently skipped, so the CI will still succeed.
