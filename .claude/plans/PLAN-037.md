---
id: PLAN-037
goal: "Execute Wave 1 release-readiness hardening: queued-task authority, publish-gate parity, and durable provider promotion"
status: completed
createdAt: 2026-03-13T10:17:51.000Z
---

## Problem

The release-readiness audit identified three Wave 1 ship blockers that still make the repository unsafe to release: queued-task recovery is not authoritative, publish is weaker than CI, and provider promotion is not durable across restart.

## Fix Strategy

### Wave 1 — Queue authority and admission-path fix (TASK-247)
- Make persisted `QUEUED` rows authoritative for execution and recovery.
- Centralize all transitions into `QUEUED` behind the same admission checks used by `SubmitTask`.

### Wave 2 — Publish workflow parity (TASK-248)
- Make the artifact-producing publish workflow run the same release-critical checks as CI before any build jobs start.

### Wave 3 — Provider promotion durability (TASK-249)
- Persist promoted providers as enabled so they survive restart and stay available.

### Wave 4 — Verification (TASK-250)
- Re-run the full validation stack and verify the specific Wave 1 fixes with targeted evidence.

## Tasks
- TASK-247: Backend queue authority and admission hardening
- TASK-248: DevOps publish workflow full-gate parity
- TASK-249: Backend durable provider promotion persistence
- TASK-250: Verification for PLAN-037 Wave 1 hardening

## Result
- Completed the Wave 1 release-readiness hardening identified in the audit.
- Persisted queued tasks are now the authoritative runnable queue, publish is gated by the release-critical validation stack, and provider promotion is durable across restart.
- Verification passed across focused Go validation, full `go test -race`, daemon E2E, frontend coverage, and VS Code extension coverage.