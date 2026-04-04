---
id: TASK-025
title: Verify — cross-platform build matrix + update README
role: verify
planId: PLAN-002
status: todo
dependencies: [TASK-024]
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

PLAN-002 adds a full Wails 2 GUI, orchestrator hardening, and a writeback system. Before the plan can be declared complete, we must verify that all binaries build cleanly on all target platforms, the test suite passes from scratch (fresh DB), and the README is updated to reflect the new architecture, commands, and writeback workflow.

## Files to Read

- `README.md` — current content to know what to update
- `go.mod` — Go version and Wails version
- `cmd/nexus-daemon/main.go` — daemon entry point
- `cmd/nexus-cli/main.go` — CLI entry point
- `main.go` — Wails desktop entry point
- `.github/copilot-instructions.md` — project conventions for README updates
- `frontend/package.json` — frontend build command

## Implementation Steps

### Step 1: Local build verification (macOS — development machine)

```sh
# Go toolchain checks
go vet ./...
CGO_ENABLED=1 go build ./cmd/nexus-cli/...
CGO_ENABLED=1 go build ./cmd/nexus-daemon/...
CGO_ENABLED=1 go test -race -count=1 -timeout 120s ./...

# Frontend build
cd frontend && npm ci && npm run build && cd ..
test -f frontend/dist/index.html

# Wails desktop build (macOS)
wails build -platform darwin/arm64 -o build/nexusOrchestrator-darwin-arm64
```

### Step 2: Cross-platform headless binary builds

Document these commands for CI (GitHub Actions or manual). All require CGO + appropriate cross-compiler:

```sh
# Linux (requires gcc)
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o build/nexus-daemon-linux-amd64 ./cmd/nexus-daemon/
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o build/nexus-cli-linux-amd64 ./cmd/nexus-cli/

# macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o build/nexus-daemon-darwin-amd64 ./cmd/nexus-daemon/
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o build/nexus-cli-darwin-amd64 ./cmd/nexus-cli/

# Windows (requires mingw64 cross-compiler on Linux/macOS):
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
  go build -o build/nexus-daemon-windows-amd64.exe ./cmd/nexus-daemon/
```

Note: Wails desktop binary (`main.go`) requires Wails toolchain per platform — use `wails build -platform <os/arch>` on each target natively or in CI.

### Step 3: Smoke test the daemon end-to-end

After starting the daemon:
```sh
./build/nexus-daemon-darwin-arm64 &
sleep 1

# Submit a task
curl -s -X POST http://127.0.0.1:63987/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"projectPath":"/tmp/smoke","prompt":"hello world"}' | jq .id

# List queue
curl -s http://127.0.0.1:63987/api/tasks | jq length

# MCP health
curl -s -X POST http://127.0.0.1:63988/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | jq '.result.tools | length'
# Expected: 6

kill %1
```

### Step 4: Update README.md

Sections to add or update:
1. **Architecture diagram** — update to include writeback system and new GUI
2. **Quick Start** — add `wails dev` for GUI development and production `wails build`
3. **CLI Commands** — add `queue submit` and `sessions get/clear` from TASK-016
4. **MCP Tools** — add `source_project_path` / `source_task_id` parameters to `submit_task`
5. **Writeback Workflow** — explain the push/sync cycle with code example using new `.claude/commands/`
6. **Cross-Platform Build Matrix** — table of supported platforms + required toolchain
7. **Environment Variables** — add `NEXUS_MAX_QUEUE` from TASK-014

### Step 5: Update `.github/copilot-instructions.md`

Sections to update:
- **Configuration** — add `NEXUS_MAX_QUEUE`
- **Adding a new LLM provider** — note that `GetPending()` must be handled if provider startup affects pending tasks
- **Key Reference Files** — add `app.go` Wails GUI bindings, `frontend/src/App.tsx`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 -timeout 120s ./...` exits 0
- [ ] `cd frontend && npm ci && npm run build` exits 0
- [ ] `wails build -platform darwin/arm64` exits 0 (on macOS arm64 dev machine)
- [ ] `frontend/dist/index.html` is NOT a stub — contains bundled React app (`<script type="module">`)
- [ ] README.md updated with writeback workflow section
- [ ] README.md updated with CLI `queue submit` usage
- [ ] Cross-platform build table present in README.md
- [ ] `NEXUS_MAX_QUEUE` documented in README.md and copilot-instructions.md

## Anti-patterns to Avoid

- NEVER commit `frontend/node_modules/` — ensure it is in `.gitignore`
- NEVER commit `build/` output binaries — ensure it is in `.gitignore`
- NEVER run `npm install` during `go build` — frontend build is a separate step
- NEVER mark PLAN-002 complete until all 19 tasks are `"done"` in orchestrator.json
