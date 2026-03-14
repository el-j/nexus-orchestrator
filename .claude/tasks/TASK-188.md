---
id: TASK-188
title: QA — verify build-all, extension v0.2.0, full test suite
role: qa
planId: PLAN-025
status: todo
dependencies: [TASK-184, TASK-185, TASK-186, TASK-187]
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
Final QA gate for PLAN-025: verify all cross-compilation targets succeed, the extension packages correctly and includes all PLAN-022 functionality, and the full Go test suite is green and race-free.

## Files to Read
- `Makefile` — build targets
- `vscode-extension/package.json` — verify version + contributes.mcpServers
- `vscode-extension/dist/extension.js` — verify SessionMonitor is included
- `.claude/plans/PLAN-025.md` — acceptance criteria

## Implementation Steps

1. **Go baseline**:
   ```bash
   CGO_ENABLED=1 go vet ./...
   CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...
   ```
   Both must exit 0.

2. **Cross-compilation tests** (run serially):
   ```bash
   make build-linux-amd64
   make build-linux-arm64
   ```
   Both must exit 0. Note expected state of each binary:
   - `dist/linux_amd64/nexus-cli` (ELF 64-bit, statically linked)
   - `dist/linux_arm64/nexus-cli` (ELF 64-bit ARM, statically linked)

3. **Windows cross-compilation**:
   ```bash
   make build-windows-amd64
   ```
   Must exit 0.

4. **Darwin amd64 cross-compilation** (from darwin arm64 host):
   ```bash
   make build-darwin-amd64
   ```
   If this is documented as unsupported, verify the Makefile prints a clear warning instead of failing silently.

5. **Full Go test suite**:
   ```bash
   CGO_ENABLED=1 go test -race -count=1 ./...
   ```
   All packages must pass. No data races.

6. **Extension verification**:
   ```bash
   grep -c "SessionMonitor\|selectChatModels\|registerSession" vscode-extension/dist/extension.js
   grep '"version"' vscode-extension/package.json
   grep '"mcpServers"' vscode-extension/package.json
   ls -lh vscode-extension/nexus-orchestrator-0.2.0.vsix
   ```
   All checks must pass.

7. **MCP live endpoint validation** (if daemon is running):
   ```bash
   curl -s -X POST http://127.0.0.1:63988/mcp \
     -H 'Content-Type: application/json' \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
     python3 -c "
   import json,sys
   d=json.load(sys.stdin)
   tools=d['result']['tools']
   print(f'total tools: {len(tools)}')
   for t in tools:
     for k,v in t.get('inputSchema',{}).get('properties',{}).items():
       if v.get('type')=='array':
         assert 'items' in v, f'FAIL: {t[\"name\"]}.{k} missing items'
         print(f'OK: {t[\"name\"]}.{k} has items')
   "
   ```

8. **Record results** in orchestrator.json: mark TASK-184 through TASK-188 as `done`, set PLAN-025 `status: completed`, clear `activePlanId`.

## Acceptance Criteria
- [ ] `make build-linux-amd64` exits 0
- [ ] `make build-linux-arm64` exits 0
- [ ] `make build-windows-amd64` exits 0 OR documented skip
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `vscode-extension/nexus-orchestrator-0.2.0.vsix` exists
- [ ] `dist/linux_amd64/nexus-cli` exists and is ELF binary
- [ ] `vscode-extension/package.json` contains `contributes.mcpServers`
- [ ] `dist/extension.js` contains SessionMonitor references

## Anti-patterns to Avoid
- NEVER mark a plan completed without running the full test suite
- NEVER skip the race detector (`-race`) — PLAN-025 changes no concurrency but baselines must be clean
