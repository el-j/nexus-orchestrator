---
id: TASK-132
title: Scaffold VS Code extension
role: devops
planId: PLAN-018
status: todo
dependencies: []
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Create the VS Code extension scaffold under `vscode-extension/` with package.json, TypeScript config, activation entry point, and basic extension manifest. This extension will be a thin HTTP client that talks to the nexusOrchestrator daemon on `127.0.0.1:9999`.

## Files to Read
- `cmd/nexus-cli/main.go` (reference for how the CLI client talks to the daemon)
- `internal/adapters/inbound/httpapi/server.go` (API shape)

## Implementation Steps
1. Create `vscode-extension/` directory with:
   - `package.json` (name: `nexus-orchestrator`, publisher, activation events, contributes: commands)
   - `tsconfig.json`
   - `src/extension.ts` (activate/deactivate lifecycle)
   - `src/nexusClient.ts` (HTTP client class wrapping the daemon API)
   - `.vscodeignore`
2. Define commands in `package.json` contributes:
   - `nexus.submitTask` — submit a code task
   - `nexus.viewQueue` — show current task queue
   - `nexus.selectProvider` — pick provider/model
   - `nexus.showProviders` — show provider status
3. Register `nexus.daemonUrl` configuration setting (default: `http://127.0.0.1:9999`).
4. Implement `NexusClient` class with methods: `submitTask()`, `getTasks()`, `getTask(id)`, `getProviders()`, `cancelTask(id)`.
5. Add `esbuild` build script for bundling.

## Acceptance Criteria
- [ ] `cd vscode-extension && npm install && npm run build` succeeds
- [ ] Extension activates in VS Code without errors
- [ ] Commands are registered and appear in the command palette
- [ ] `NexusClient` can make HTTP requests to the daemon API

## Anti-patterns to Avoid
- NEVER bundle unnecessary dependencies — keep the extension lightweight
- NEVER hardcode URLs — use VS Code settings for configuration
