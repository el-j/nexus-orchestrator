---
id: TASK-168
title: Frontend LogPanel and useLogs composable
role: frontend
planId: PLAN-023
status: todo
dependencies: [TASK-163]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The user wants an in-app log console instead of a separate terminal window. The LogPanel appears at the bottom of the app (like VS Code's terminal panel), is collapsible/resizable, and streams live log entries via SSE. This replaces the need for the visible console window.

## Files to Read
- `frontend/src/App.vue` — root layout to add LogPanel
- `frontend/src/composables/useTasks.ts` — SSE pattern reference (EventSource + fallback)
- `frontend/src/types/domain.ts` — existing type patterns
- `internal/core/domain/log_entry.go` — `LogEntry`, `LogLevel` from TASK-159

## Implementation Steps

1. Add TypeScript types in `frontend/src/types/domain.ts`:
   ```typescript
   export type LogLevel = 'info' | 'warn' | 'error' | 'debug'

   export interface LogEntry {
     timestamp: string
     level: LogLevel
     source: string
     message: string
   }
   ```

2. Create `frontend/src/composables/useLogs.ts`:
   - `const logs = ref<LogEntry[]>([])`
   - `const connected = ref(false)`
   - Max buffer: 2000 entries (ring buffer — shift oldest when full)
   - On mount: fetch `GET /api/logs` for initial buffer
   - Open `EventSource('http://127.0.0.1:9999/api/events')` and listen for `event.type === 'log'`
   - Parse each SSE data as `LogEntry`, push to buffer
   - `clear()` method to empty the buffer
   - `onUnmounted` → close EventSource
   - Export `{ logs, connected, clear }`

3. Create `frontend/src/components/LogPanel.vue`:
   - Fixed bottom panel, default height 200px
   - Drag-resizable top border (min 100px, max 600px)
   - Collapse/expand toggle button (chevron icon)
   - Header bar: "Logs" title, connection status dot (green=connected, red=disconnected), filter dropdowns (level, source), clear button, auto-scroll toggle
   - Log list: monospace font, each line shows `[timestamp] [level] [source] message`
   - Color by level: info=default, warn=yellow, error=red, debug=grey
   - Auto-scroll to bottom on new entries (unless user has scrolled up)
   - When collapsed, show just the header bar with entry count badge

4. Update `frontend/src/App.vue`:
   - Wrap the main content area in a flex column
   - Add `<LogPanel />` below the `<main>` router-view area
   - LogPanel persists across route changes (it's at the app root level)

## Acceptance Criteria
- [ ] LogPanel appears at the bottom of the app window
- [ ] SSE connection streams live log entries
- [ ] Log entries are color-coded by level
- [ ] Panel is collapsible and resizable
- [ ] Auto-scroll works (scrolls to bottom on new entries, stops if user scrolls up)
- [ ] Clear button empties the log buffer
- [ ] Ring buffer caps at 2000 entries
- [ ] `GET /api/logs` loads initial buffer on mount

## Anti-patterns to Avoid
- NEVER use polling for logs — SSE is required for real-time feel
- NEVER render all 2000 log entries in DOM — use virtual scrolling or limit visible rows
- NEVER block the main thread with log processing
