---
id: TASK-554
planId: PLAN-071
title: 'Domain & Ports Extension — VerificationCommand, MaxCorrectionTurns, CommandRunner port'
role: backend
status: done
createdAt: 2026-09-24T15:58:00Z
---

# TASK-554 — Domain & Ports Extension

## Context

Tasks need to specify verification commands (e.g. `go test ./...` or `npm test`), limits on correction turns, and capture verification output. Additionally, hexagonal architecture requires an outbound port interface `CommandRunner` for running shell/verification commands.

## Work Required

1. `internal/core/domain/task.go`:
   - Add fields to `domain.Task`:
     - `VerificationCommand string` (`json:"verificationCommand,omitempty"`)
     - `MaxCorrectionTurns int` (`json:"maxCorrectionTurns,omitempty"`)
     - `VerificationOutput string` (`json:"verificationOutput,omitempty"`)
2. `internal/core/ports/ports.go`:
   - Add `CommandRunner` interface:
     ```go
     type CommandRunner interface {
         Run(ctx context.Context, dir string, command string) (string, error)
     }
     ```
3. `internal/core/domain/task_test.go`:
   - Test json serialization/deserialization and domain behavior with the new fields.
