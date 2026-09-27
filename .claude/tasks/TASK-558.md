---
id: TASK-558
planId: PLAN-071
title: 'Inbound Adapters & Wiring — MCP, CLI, Daemon & Main Entry Points'
role: backend
status: done
createdAt: 2026-09-24T15:58:00Z
---

# TASK-558 — Inbound Adapters & Wiring

## Context

All entry points and adapters need to wire the `cmd_runner` adapter into the orchestrator and expose the new verification fields via MCP, HTTP, and CLI.

## Work Required

1. In `main.go` and `cmd/nexus-daemon/main.go`:
   - Instantiate `cmd_runner.New()` and pass via `services.WithCommandRunner(...)`.
2. In `internal/adapters/inbound/mcp/`:
   - Update `submit_task` tool schema to accept `verificationCommand` and `maxCorrectionTurns`.
   - Update tool handler to pass these fields into `domain.Task`.
3. In `internal/adapters/inbound/cli/`:
   - Update `submit` command to add `--verify` flag for `VerificationCommand`.
4. Validate full system:
   - Run `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...`
   - Run `go vet ./...`
