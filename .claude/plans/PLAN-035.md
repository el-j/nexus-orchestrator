---
id: PLAN-035
goal: "Add deterministic release-gate E2E coverage for daemon, frontend views, and VS Code extension"
status: completed
createdAt: 2026-03-13T02:00:00.000Z
completedAt: 2026-03-13T08:31:20.000Z
---

## Problem

The backend test surface is relatively strong, but recent regressions escaped through the UI and adapter boundary. The repo needs a deterministic release gate that exercises the real daemon, the backlog/history frontend surfaces, and the VS Code extension command/status path.

## Fix Strategy

### Wave 1 — Deterministic daemon E2E (TASK-239)
- Expand the binary smoke flow to cover draft/backlog/history/all-tasks, AI sessions, MCP tools/list, and MCP parity tools.
- Keep the scenarios provider-independent so the gate is stable in CI.

### Wave 2 — Frontend smoke coverage (TASK-240)
- Add Vitest-based smoke tests for BacklogView and HistoryView.
- Verify empty-array safety and correct data-source wiring.

### Wave 3 — VS Code extension smoke coverage (TASK-241)
- Add Vitest-based tests for the explicit queue workflow and route-aware status/queue surfaces.
- Verify the extension logic without depending on live Copilot.

### Wave 4 — CI gate and verification (TASK-242)
- Run Go, frontend, extension, and E2E checks in CI.
- Fail the pipeline when any release-gate surface regresses.

## Tasks
- TASK-239: Deterministic daemon E2E coverage
- TASK-240: Frontend backlog/history smoke tests
- TASK-241: VS Code extension queue/visibility smoke tests
- TASK-242: CI gate and verification