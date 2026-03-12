---
id: TASK-185
title: VS Code extension contributes.mcpServers + nexus.mcpPort config
role: architecture
planId: PLAN-025
status: todo
dependencies: []
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
Currently users must manually create `.vscode/mcp.json` to register the nexus-orchest MCP server — leading to the "Failed to validate tool mcp_nexus-orchest_submit_task" error until the file is created. VS Code 1.99+ supports `contributes.mcpServers` in `package.json`, which auto-registers the MCP server when the extension activates. Adding this makes setup zero-config.

## Files to Read
- `vscode-extension/package.json` — current contributes section (engines: `"vscode": "^1.85.0"`)
- `vscode-extension/src/extension.ts` — activation flow
- `.vscode/mcp.json` — the manual file we created as a workaround

## Implementation Steps

1. **Bump the minimum VS Code engine version** to `"^1.99.0"` in `package.json` to enable `contributes.mcpServers`.

2. **Add `nexus.mcpPort` configuration property** to `contributes.configuration.properties`:
   ```json
   "nexus.mcpPort": {
     "type": "number",
     "default": 9998,
     "description": "Port number of the nexusOrchestrator MCP server (JSON-RPC 2.0). Must match NEXUS_MCP_ADDR on the daemon."
   }
   ```

3. **Add `contributes.mcpServers`** to the `contributes` object:
   ```json
   "mcpServers": {
     "nexus-orchest": {
       "label": "Nexus Orchestrator",
       "type": "http",
       "url": "http://127.0.0.1:${config:nexus.mcpPort}/mcp"
     }
   }
   ```
   This uses VS Code's `${config:...}` substitution to pick up the user's `nexus.mcpPort` setting.

4. **Verify**: The `contributes.mcpServers` key must be at the same level as `contributes.commands`, `contributes.views`, etc. — NOT nested inside another key.

5. **Update `.vscode/mcp.json`** comment / README note: the `.vscode/mcp.json` file we created manually is now superseded by the extension's `contributes.mcpServers`. Delete it or leave it as a fallback for users without the extension installed. Best: keep `.vscode/mcp.json` as a fallback for Claude Desktop / Cursor users; the extension handles VS Code.

6. **Do NOT** modify `extension.ts` — the `contributes.mcpServers` mechanism is entirely declarative; no activation code is needed.

## Acceptance Criteria
- [ ] `vscode-extension/package.json` has `contributes.mcpServers.nexus-orchest` with `type: "http"` and `url: "http://127.0.0.1:${config:nexus.mcpPort}/mcp"`
- [ ] `contributes.configuration.properties` includes `nexus.mcpPort` with type `number` and default `9998`
- [ ] `engines.vscode` is `"^1.99.0"` or higher
- [ ] No TypeScript compilation errors (`cd vscode-extension && npm run build` exits 0)
- [ ] `go vet ./...` exits 0 (Go project unaffected)

## Anti-patterns to Avoid
- NEVER nest `mcpServers` inside `views` or `commands` — it must be a direct child of `contributes`
- NEVER hardcode the port `9998` in the URL — use `${config:nexus.mcpPort}` for user configurability
- NEVER change the server name from `nexus-orchest` — Copilot prefixes tools with this name (`mcp_nexus-orchest_*`)
