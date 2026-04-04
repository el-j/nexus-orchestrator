---
id: TASK-487
plan: PLAN-063
status: done
wave: 2
priority: 1
---

# TASK-487: Wire `nexus.mcpPort` / `nexus.enableMCPPortSweep` settings

## Problem

`vscode-extension/package.json` `contributes.configuration` declares `nexus.mcpPort` (default `63988`) and `nexus.enableMCPPortSweep` (default `false`). Neither setting is read anywhere in the extension source — they are phantom settings with zero effect. Users who change these will see no behavior change and will file confusing bug reports.

## Checklist

- [ ] In `extension.ts`, read `nexus.mcpPort` via `vscode.workspace.getConfiguration('nexus').get<number>('mcpPort', 63988)` when constructing the `NexusClient` (or `MCPClient`) base URL; thread the port through the constructor
- [ ] In `nexusClient.ts`, change the hardcoded `63988` MCP port constant to accept the port as a constructor parameter with `63988` as the default
- [ ] Implement port sweep logic in `nexusClient.ts` behind a `tryConnect(ports: number[]): Promise<number>` method: probe each port in the list with a short `HEAD /health` or `GET /.well-known/nexus.json` request and return the first port that responds 200; throw if none respond
- [ ] In `extension.ts`, when `nexus.enableMCPPortSweep` is `true`, call `tryConnect` with the curated sweep list `[63988, 63987, 63989, 63990, 63986]` plus the user-configured `nexus.mcpPort` (deduplicated, user port tried first); use the resolved port for all subsequent client calls
- [ ] Listen for `vscode.workspace.onDidChangeConfiguration` and reconnect the client when either setting changes
- [ ] Add a manual sweep trigger: `nexus.reconnect` command that calls `tryConnect` and shows a status bar message with the discovered port

## Files to change

- `vscode-extension/src/extension.ts`
- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/src/commands/index.ts` (add `reconnect` command)
- `vscode-extension/package.json` (register `nexus.reconnect` command)

## Acceptance criteria

- [ ] Changing `nexus.mcpPort` in VS Code Settings causes the extension to reconnect to the new port
- [ ] With `nexus.enableMCPPortSweep: true`, the extension finds the daemon when it is running on any port in the sweep list without manual configuration
- [ ] No hardcoded `63988` literals remain in client construction paths
