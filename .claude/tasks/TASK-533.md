---
id: TASK-533
planId: PLAN-069
title: 'Add missing brain CLI subcommands (init, list, delete, context, file-map)'
role: cli
status: todo
createdAt: 2026-04-13T16:00:00Z
---

# TASK-533 — Add missing brain CLI subcommands

## Context

`internal/adapters/inbound/cli/brain.go` currently registers three subcommands under `nexus brain`:

- `status` — OK
- `ingest` — OK
- `search` — OK

The `ports.BrainService` interface exposes 4 additional operations that have no CLI entry points:

| Method            | Missing subcommand                                          |
| ----------------- | ----------------------------------------------------------- |
| `InitProject`     | `nexus brain init --project <path> [--file <claudeMDPath>]` |
| `ListKnowledge`   | `nexus brain list --project <path> [--kind <kind>]`         |
| `DeleteKnowledge` | `nexus brain delete --id <id>`                              |
| `GetContext`      | `nexus brain context --project <path> [--max-tokens 800]`   |
| `GetFileMap`      | `nexus brain file-map --project <path> [--focus <area>]`    |

**Depends on**: TASK-529 (HTTP routes) and TASK-530 (client stubs).

## Work Required

Add 5 new `cobra.Command` constructors to `internal/adapters/inbound/cli/brain.go` and register
them in `newBrainCmd()`:

1. `newBrainInitCmd(brain)` — calls `brain.InitProject`; outputs resulting `BrainStatus` as JSON
2. `newBrainListCmd(brain)` — calls `brain.ListKnowledge`; outputs `[]ProjectKnowledge` as JSON
3. `newBrainDeleteCmd(brain)` — calls `brain.DeleteKnowledge`; outputs success message
4. `newBrainContextCmd(brain)` — calls `brain.GetContext`; outputs `ContextResponse` as JSON
5. `newBrainFileMapCmd(brain)` — calls `brain.GetFileMap`; outputs paths one per line

## File Targets

- `internal/adapters/inbound/cli/brain.go`

## Acceptance Criteria

- `nexus brain --help` lists all 8 subcommands
- `go vet ./internal/adapters/inbound/cli/...` clean
- `CGO_ENABLED=1 go build ./cmd/nexus-cli/...` clean
