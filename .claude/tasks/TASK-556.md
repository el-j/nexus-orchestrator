---
id: TASK-556
planId: PLAN-071
title: 'Outbound Command Runner Adapter — cmd_runner'
role: backend
status: done
createdAt: 2026-09-24T15:58:00Z
---

# TASK-556 — Outbound Command Runner Adapter

## Context

Hexagonal architecture requires an adapter implementing `ports.CommandRunner` to execute verification commands in specified directories with timeouts and output capture.

## Work Required

1. Create `internal/adapters/outbound/cmd_runner/cmd_runner.go`:
   - Implement `ports.CommandRunner`.
   - Execute commands via `sh -c` (Unix) or `cmd.exe /c` (Windows) using `os/exec.CommandContext`.
   - Combine stdout and stderr into output string.
   - Configure default timeout (e.g. 2 minutes) with context cancellation.
2. Create `internal/adapters/outbound/cmd_runner/cmd_runner_test.go`:
   - Test successful command (e.g. `echo hello`).
   - Test non-zero exit command (e.g. `exit 1` or invalid command).
   - Test timeout cancellation.
   - Test working directory setting.
