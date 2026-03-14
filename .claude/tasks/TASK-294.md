---
id: TASK-294
title: 'VS Code extension VSIX rebuild for PLAN-044 features'
role: devops
planId: PLAN-045
status: todo
dependencies: [TASK-291, TASK-293]
createdAt: 2026-03-14T18:00:00.000Z
---

## Context

PLAN-044 (Universal AI Takeover) added significant new files to the VS Code extension:

- `agentDetector.ts` — 4-strategy AI agent detection with 30s polling
- `aiSessionsTreeProvider.ts` — new tree view (`nexus.aiSessions`)
- `commands/delegateToNexus.ts` — 3-path delegation command
- Extended `nexusClient.ts` with `DiscoveredAgent`, `DelegateResponse`, new methods
- Extended `package.json` with new view, commands, menus, configuration property

The last VSIX in `vscode-extension/` was built before PLAN-044. The `.vsix` must be
rebuilt so the extension can be locally installed/tested with all new functionality.

## Steps

1. `cd vscode-extension`
2. `npm run package` — produces `nexus-orchestrator-<version>.vsix`
3. Confirm the `.vsix` file was created successfully
4. Check `npx tsc --noEmit` still exits 0 (type-check before packaging)
5. Optionally: confirm file size is reasonable (previous was ~200KB range)

## Acceptance Criteria

- [ ] `npx tsc --noEmit` inside `vscode-extension/` exits 0
- [ ] `npm run package` exits 0
- [ ] A `nexus-orchestrator-*.vsix` file exists in `vscode-extension/`
- [ ] VSIX size > 50 KB (sanity check it includes bundled files)

## Notes

- The extension version in `package.json` should be bumped if this is to be
  distributed (but that is optional for local dev use)
- `npm run build` (esbuild bundle) runs automatically as part of `npm run package`
