---
id: TASK-546
planId: PLAN-070
title: 'ProjectBrainCard: multi-file ingest, user-visible feedback, init button'
role: frontend
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-546 — Improve ProjectBrainCard UX

## Context

`frontend/src/components/ProjectBrainCard.vue` has three UX gaps:

1. **Single-file ingest only**: `handleIngest()` always passes `nexusFile.path` — the path of the
   project's nexus config file. There is no way to ingest arbitrary files or directories.
   Worse: the "Sync Brain" button is hidden when `nexusFile` is absent, giving projects without a
   nexus file zero ingest path even though the backend accepts any markdown file.

2. **No user-visible feedback**: After ingest, the count of ingested sections is only
   `console.log`'d. The user sees nothing — no success toast, no count badge update, no error
   message if ingest fails.

3. **No "Init" button**: `brain.InitProject` auto-ingests the project's CLAUDE.md. The card has
   no entry point for this common first-time-setup action.

## Work Required

In `frontend/src/components/ProjectBrainCard.vue`:

1. **Ingest from arbitrary file**: Change `handleIngest()` to accept an optional `filePath`
   parameter. When `filePath` is undefined, show a text input or (in Wails context) a native file
   picker. The existing "Sync Brain" button becomes the primary ingest button always visible (not
   gated on `nexusFile`), with a secondary "Ingest File…" option.

2. **User-visible feedback**: After ingest:
   - Show a dismissible inline message: `"Ingested N sections"` or `"Ingest failed: <error>"`.
   - Update the brain status badge reactively (re-call `getBrainStatus` after ingest).

3. **Init button**: Add an "Initialize Brain" button shown only when `status.entryCount === 0`.
   Clicking calls `initProject(projectPath)` (available after TASK-544), shows result in an
   inline message, then refreshes the status display.

## File Targets

- `frontend/src/components/ProjectBrainCard.vue`

## Acceptance Criteria

- `cd frontend && vue-tsc --noEmit` clean
- "Sync Brain" / ingest button visible even when `nexusFile` is absent
- Ingest success shows section count; ingest failure shows error message
- "Initialize Brain" button visible when `status.entryCount === 0`
- After any operation, `BrainStatus` display updates reactively
