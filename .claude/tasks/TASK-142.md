---
id: TASK-142
title: Update docs/index.md with VS Code extension feature
role: docs
planId: PLAN-019
status: todo
dependencies: []
createdAt: 2026-03-11T18:00:00.000Z
---

## Context

`docs/index.md` is the Jekyll/static landing page markdown for the GitHub Pages site.
It lists project features and supported interfaces but makes no mention of the VS Code
extension. Since `docs/index.md` is not deployed by the current Vite build (only
`docs/dist` is deployed), this is documentation housekeeping — but keeping it accurate
is important for developers who browse the repo directly on GitHub.

## Files to Read

- `docs/index.md` (full file)
- `docs/getting-started.md` (skim for VS Code mention)

## Implementation Steps

1. Open `docs/index.md`. Find the "Features" or key highlights section.

2. **Add a VS Code Extension bullet** to the features list:
   ```markdown
   - 🔌 **VS Code Extension** — submit tasks, monitor queue, and switch providers without leaving your editor ([download .vsix](https://github.com/el-j/nexus-orchestrator/releases/latest/download/nexus-orchestrator-vscode.vsix))
   ```
   Insert this bullet after the "Desktop GUI" or "Desktop App" bullet.

3. If there is a "Interfaces" or "Client options" table/list, update it to include
   "VS Code Extension" alongside HTTP API, MCP, CLI.

4. Check `docs/getting-started.md`. If it has a section listing installation options or
   client interfaces, add a brief mention:
   ```markdown
   ### VS Code Extension
   
   Download the [`.vsix`](https://github.com/el-j/nexus-orchestrator/releases/latest/download/nexus-orchestrator-vscode.vsix)
   and install via `code --install-extension nexus-orchestrator-vscode.vsix`.
   Configure the daemon URL under `Nexus > Daemon URL` in VS Code settings.
   ```

## Acceptance Criteria

- [ ] `docs/index.md` mentions VS Code Extension in its features list
- [ ] Link to the VSIX download is correct
- [ ] `docs/getting-started.md` has at least a brief mention of the VS Code extension

## Anti-patterns to Avoid

- Do not rewrite large sections of the docs — surgical additions only
- Do not change the Vite/Vue SPA source (`docs/src/`) — that's TASK-140 and TASK-141
- Do not create new files
