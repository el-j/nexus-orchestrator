---
id: TASK-567
title: CLI expansion — providers config CRUD and discovery subcommands
role: cli
planId: PLAN-071
status: todo
dependencies: [TASK-566]
createdAt: 2026-04-15T00:00:00Z
---

## Context

The `providers` CLI group currently only exposes a minimal subset of the providers API. Missing: config CRUD (add, update, remove, list persisted configs), discovered providers listing, scan trigger, and promote. These are essential for headless/scripted provider management.

## Files to Read

- `internal/adapters/inbound/cli/root.go`
- `internal/adapters/outbound/httpapi_client/client.go`
- `internal/core/domain/provider.go`

## Implementation Steps

1. Extend the existing `providers` cobra command group with:
   - `providers config list` — GET /api/providers/config; table: ID, kind, name, baseURL, enabled
   - `providers config add --kind <k> --name <n> --base-url <u> [--api-key <k>]` — POST /api/providers/config
   - `providers config update <id> [--name <n>] [--base-url <u>] [--enabled]` — PUT /api/providers/config/{id}
   - `providers config remove <id>` — DELETE /api/providers/config/{id}
   - `providers discovered` — GET /api/providers/discovered; table: name, kind, baseURL, status
   - `providers scan` — POST /api/providers/discovered/scan; print count of discovered providers
   - `providers promote <name>` — POST /api/providers/{name}/promote
2. All subcommands support `--json` for machine-readable output
3. `providers config add` and `providers config update` must NOT prompt for `--api-key` on stdout — accept it only via flag or `NEXUS_API_KEY` env var (avoid leaking to shell history)

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All 7 new providers subcommands exist and make correct HTTP calls
- [ ] API key is never echoed to stdout in normal output
- [ ] `--json` flag works on all subcommands

## Anti-patterns to Avoid

- NEVER read API keys from stdin interactively — use flag or env var only
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
