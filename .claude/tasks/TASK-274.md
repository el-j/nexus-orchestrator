# TASK-274 — Rebuild VSIX + install

**Plan**: PLAN-043  
**Status**: done  
**Role**: build

## What

The VSIX at `build/vscode/nexus-orchestrator.vsix` was built on 2026-03-13 17:57, before:
- Port migration 9999 → 63987 (the bundle has `"http://127.0.0.1:9999"` inside)
- TASK-272 and TASK-273 changes

Rebuilding from current source fixes the port in `dist/extension.js` and packages all new features.

## Steps

```sh
cd vscode-extension
npm run build          # regenerates dist/extension.js
cd ..
cd vscode-extension && npx vsce package --no-dependencies --out ../build/vscode/nexus-orchestrator.vsix
code --install-extension build/vscode/nexus-orchestrator.vsix --force
```

## Verification

After install, reload VS Code window and confirm:
- `dist/extension.js` contains `"http://127.0.0.1:63987"` (not 9999)
- Extension activates without errors
- Task Queue view shows "Nexus daemon offline" when daemon is not running
