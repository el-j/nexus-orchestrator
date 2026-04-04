# PLAN-032 — Fix Null-Array Crash on Startup

## Goal
Fix the `null is not an object (evaluating 'e.value.map')` crash that occurs when the built desktop app starts with an empty task queue / empty AI sessions table. The root cause is Go's `encoding/json` marshaling nil slices as `null`, which gets assigned to Vue reactive refs that are then `.map()`d or `.filter()`d.

## Root Cause Analysis
1. **Go side:** `repo_sqlite/repo.go:GetPending()` declares `var tasks []domain.Task` (nil) → JSON encodes as `null`. Same for `ai_session_repo.go:ListAISessions()` → `var sessions []domain.AISession`.
2. **Frontend side:** `useTasks.ts` assigns `tasks.value = await getQueue()` without null guard → `tasks.value` becomes `null` → `useProjectFilter.ts` calls `tasks.value.map(...)` → **crash**.
3. **Cascading:** `DashboardView.vue`, `HistoryView.vue`, `BacklogView.vue` all call `.filter()` on `tasks.value`.

## Fix Strategy
Two-layer defense:
- **Go layer (TASK-230):** Initialize slices with `make()` so they marshal to `[]` not `null`
- **Frontend layer (TASK-231):** Add `?? []` guards to all composable Wails call assignments
- **Verify (TASK-232):** Build and runtime verification

## Tasks

| ID | Title | Role | Wave | Dependencies |
|----|-------|------|------|--------------|
| TASK-230 | Go: Initialize nil slices in SQLite repo methods | backend | 1 | [] |
| TASK-231 | Frontend: Add null-coalescing guards to all composable Wails calls | backend | 1 | [] |
| TASK-232 | Verify: Build + type check + go test | verify | 2 | [TASK-230, TASK-231] |

## Status: active
Created: 2026-03-13
