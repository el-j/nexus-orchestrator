---
id: TASK-186
title: Rebuild VS Code extension dist, bump to v0.2.0, repackage VSIX
role: devops
planId: PLAN-025
status: todo
dependencies: [TASK-185]
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
The current `dist/extension.js` (696 lines) was compiled before PLAN-022 added `SessionMonitor` — meaning users running the packaged VSIX don't get AI session auto-registration with the daemon. The extension version should be bumped to 0.2.0 to reflect all PLAN-022 additions (SessionMonitor, AI session API, session status bar) and the new MCP auto-registration from TASK-185.

## Files to Read
- `vscode-extension/package.json` — version field, scripts.build, scripts.package
- `vscode-extension/src/extension.ts` — verify SessionMonitor import was added in PLAN-022
- `vscode-extension/src/sessionMonitor.ts` — exists from PLAN-022
- `vscode-extension/tsconfig.json` — compiler settings

## Implementation Steps

1. **Verify `node_modules` exists**: Run `ls vscode-extension/node_modules/ | head -5`. If empty, run `cd vscode-extension && npm install` first.

2. **Bump version**: In `vscode-extension/package.json`, change `"version": "0.1.0"` to `"version": "0.2.0"`.

3. **Rebuild the extension bundle**:
   ```bash
   cd vscode-extension && npm run build
   ```
   This runs `esbuild src/extension.ts --bundle --outfile=dist/extension.js --external:vscode --format=cjs --platform=node`.
   
   After build, verify `dist/extension.js` is > 1000 lines (should include SessionMonitor, nexusClient, all commands).

4. **Verify the compiled bundle includes SessionMonitor**:
   ```bash
   grep -c "SessionMonitor\|selectChatModels\|registerSession" vscode-extension/dist/extension.js
   ```
   Must return > 0 matches.

5. **Package the VSIX**:
   ```bash
   cd vscode-extension && npm run package
   ```
   This runs `vsce package --no-dependencies`. This creates `nexus-orchestrator-0.2.0.vsix`.

6. **Verify the VSIX exists and is < 5MB**:
   ```bash
   ls -lh vscode-extension/nexus-orchestrator-0.2.0.vsix
   ```

7. **Clean up old VSIX**: Remove `vscode-extension/nexus-orchestrator-0.1.0.vsix` to avoid confusion:
   ```bash
   rm vscode-extension/nexus-orchestrator-0.1.0.vsix
   ```

## Acceptance Criteria
- [ ] `vscode-extension/package.json` version is `"0.2.0"`
- [ ] `vscode-extension/dist/extension.js` is rebuilt and contains `SessionMonitor` / `selectChatModels` references
- [ ] `vscode-extension/nexus-orchestrator-0.2.0.vsix` exists
- [ ] Old `nexus-orchestrator-0.1.0.vsix` is removed
- [ ] `go vet ./...` exits 0 (Go project unaffected)

## Anti-patterns to Avoid
- NEVER run `npm install` without first checking if node_modules exists — it takes time and may alter lock file
- NEVER manually edit `dist/extension.js` — always rebuild from source
- NEVER forget to bump the version BEFORE packaging — vsce uses the version from package.json as the filename
