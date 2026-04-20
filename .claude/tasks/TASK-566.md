---
id: TASK-566
title: CLI expansion — daemon URL flag and full ai-sessions command group
role: cli
planId: PLAN-071
status: todo
dependencies: [TASK-557, TASK-558]
createdAt: 2026-04-15T00:00:00Z
---

## Context

`nexus-cli` has no way to override the daemon URL — it is hardcoded to `http://127.0.0.1:63987` with no `--daemon-url` flag or `NEXUS_ADDR` env var. The entire `ai-sessions` HTTP API group (list, register, deregister, heartbeat, terminate, delegate, purge) has zero CLI exposure despite being a core orchestration feature used by every AI agent.

## Files to Read

- `internal/adapters/inbound/cli/root.go`
- `cmd/nexus-cli/main.go`
- `internal/adapters/outbound/httpapi_client/client.go`

## Implementation Steps

1. In `root.go`: add a persistent root flag `--daemon-url` (string, default `""`) and read `NEXUS_ADDR` env var; in `PersistentPreRun`, resolve final URL: flag > env > default `http://127.0.0.1:63987`; pass through to the HTTP client
2. Add `ai-sessions` cobra subcommand with the following sub-subcommands:
   - `ai-sessions list` — GET /api/ai-sessions, table output (ID, agentName, projectPath, status, updatedAt)
   - `ai-sessions register --agent-name <n> --project-path <p> [--model <m>]` — POST /api/ai-sessions
   - `ai-sessions deregister <id>` — DELETE /api/ai-sessions/{id}
   - `ai-sessions heartbeat <id>` — POST /api/ai-sessions/{id}/heartbeat
   - `ai-sessions terminate <id> [--force]` — POST /api/ai-sessions/{id}/terminate
   - `ai-sessions purge` — DELETE /api/ai-sessions (purge disconnected)
   - `ai-sessions delegate <session-id>` — POST /api/ai-sessions/{id}/delegate; print returned instruction
3. Each subcommand must: call the HTTP client method, handle `404`/`403` with a clear message, and print JSON output with `--json` flag or human-readable table by default
4. Add the `ai-sessions` command to the root cobra command

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `NEXUS_ADDR=http://custom:9999 nexus-cli ai-sessions list` uses the custom URL
- [ ] `nexus-cli --daemon-url http://custom:9999 ai-sessions list` uses the custom URL
- [ ] All 7 ai-sessions subcommands exist and make the correct HTTP calls
- [ ] `--json` flag produces machine-readable output

## Anti-patterns to Avoid

- NEVER hardcode the daemon URL in command implementations — always use the resolved URL from root
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
