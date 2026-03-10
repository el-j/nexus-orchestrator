---
id: TASK-072
title: New .claude/commands/execute-via-nexus.md (self-dogfood command)
role: planning
planId: PLAN-008
status: todo
dependencies: [TASK-068, TASK-069, TASK-070]
createdAt: 2026-03-10T14:00:00.000Z
---

## Context
Currently, `execute-plan.md` runs plan tasks locally via sub-agents. `push-to-nexus.md` and `sync-from-nexus.md` push/pull tasks to a running daemon individually. We need a **unified self-dogfood command** that orchestrates the full cycle: ensure daemon is running, push plan tasks in wave order, poll for completion via sync, and report results. This command lets the nexusOrchestrator process its own task backlog through its LLM pipeline.

## Files to Read
- `.claude/commands/push-to-nexus.md`
- `.claude/commands/sync-from-nexus.md`
- `.claude/commands/execute-plan.md`
- `.claude/commands/dogfood-plan002.md` (old approach to improve on)
- `.claude/orchestrator.json` (structure)

## Implementation Steps

1. Create `.claude/commands/execute-via-nexus.md`.
2. The command accepts `$ARGUMENTS`:
   - A plan ID (e.g. `PLAN-008`) — process that specific plan's tasks
   - `active` or empty — use `activePlanId` from orchestrator.json
3. **Step 1 — Prerequisites**:
   - Read `.claude/orchestrator.json` to get the plan and its tasks.
   - Verify daemon is running: `curl -sf http://127.0.0.1:9999/api/health`.
   - If not running: print instructions to start it and stop.
   - Verify at least one active provider: `GET /api/providers` has at least one entry with `active: true`.
4. **Step 2 — Build dependency graph**:
   - Parse task dependencies from orchestrator.json.
   - Group into waves (same approach as execute-plan.md).
   - Only select tasks with `status: "todo"` — skip done/pushed/in-progress.
5. **Step 3 — Push wave by wave**:
   - For each wave, push all tasks using the push-to-nexus.md logic (POST /api/tasks with task file content as instruction).
   - Record `nexusTaskId` in orchestrator.json.
   - Mark tasks as `"pushed"`.
6. **Step 4 — Poll for completion**:
   - After pushing a wave, poll `GET /api/tasks/{nexusTaskId}` every 5s until all tasks in the wave are COMPLETED, FAILED, or CANCELLED.
   - Timeout per wave: 5 minutes.
   - On completion: update orchestrator.json status to `"done"` with timestamps.
   - On failure: mark `"failed"`, report which task failed and why.
   - On timeout: report timeout, leave as `"pushed"`.
7. **Step 5 — Next wave**: Only proceed to next wave if all tasks in current wave are done.
8. **Step 6 — Summary**: Print final summary (tasks completed, failed, timed out) and whether the plan is fully done.

## Acceptance Criteria
- [ ] `.claude/commands/execute-via-nexus.md` exists
- [ ] Command checks daemon health before pushing
- [ ] Command respects task dependency waves
- [ ] Command updates orchestrator.json with nexusTaskId and status changes
- [ ] Command handles FAILED and timeout scenarios gracefully
- [ ] Command can be interrupted without leaving orchestrator.json in corrupt state

## Anti-patterns to Avoid
- NEVER modify any Go source files from this command
- NEVER push tasks whose dependencies are not yet done
- NEVER leave orphaned "pushed" status — always have a sync step
