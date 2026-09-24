---
id: TASK-557
planId: PLAN-071
title: 'Execution Engine Self-Healing Verification Loop'
role: backend
status: done
createdAt: 2026-09-24T15:58:00Z
---

# TASK-557 — Execution Engine Self-Healing Verification Loop

## Context

In `internal/core/services/execution_engine.go`, code generation writes to disk and marks the task `COMPLETED`. When `task.VerificationCommand` is present, Nexus must execute the verification command. If it fails, Nexus must enter a self-healing loop: feed the compiler/test error back to the model, re-generate the file, and re-verify until it passes or turns are exhausted.

## Work Required

1. In `internal/core/services/orchestrator.go`:
   - Add `commandRunner ports.CommandRunner` field to `OrchestratorService`.
   - Add `WithCommandRunner(runner ports.CommandRunner) Option`.
2. In `internal/core/services/execution_engine.go`:
   - Implement post-generation verification and self-healing logic:
     - If `task.VerificationCommand != ""` and `o.commandRunner != nil`:
       - Run command in `task.ProjectPath`.
       - If verification passes: record output, mark `StatusCompleted`.
       - If verification fails: loop up to `task.MaxCorrectionTurns` (default 2):
         - Construct feedback prompt with the exact command and error output.
         - Call `executeGeneration()` to obtain corrected code.
         - Write corrected code to `task.TargetFile`.
         - Re-run verification.
         - If it passes, mark `StatusCompleted`!
       - If turns exhausted and still failing: record error output in logs, mark `StatusFailed`.
3. In `internal/core/services/execution_engine_test.go`:
   - Test task with verification command passing on first attempt.
   - Test task with verification failing on turn 1, then self-healing and passing on turn 2.
   - Test task failing verification through all turns, transitioning to `StatusFailed`.
   - Verify backwards compatibility for tasks without verification command.
