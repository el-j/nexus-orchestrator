---
id: TASK-148
title: Replace polling in useTasks.ts with SSE consumer
role: api
planId: PLAN-021
status: todo
dependencies: []
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
`frontend/src/composables/useTasks.ts` currently polls `GET /api/tasks` every 2 seconds via
`setInterval`. The HTTP API already exposes a Server-Sent Events endpoint at `GET /api/events`
(hub.go broadcasts `data: {"type":"task_updated","taskId":"…","status":"…"}\n\n` on every lifecycle
change). Replacing the polling interval with an `EventSource` subscription gives instant task status
updates, eliminates unnecessary round-trips, and reduces GUI flicker.

The SSE event JSON shape is `{ type: string; taskId: string; status: string }`.
On any event with `type !== "connected"`, the composable should call `refresh()` to re-fetch the
full task list. This keeps the logic simple while getting instant updates.

The composable must fall back to polling (2 s) when `EventSource` is unavailable (i.e. in Wails
desktop mode, where Wails may intercept fetch but not EventSource). Detect availability via
`typeof EventSource !== 'undefined'`.

## Current file content of frontend/src/composables/useTasks.ts
```typescript
import { ref, onMounted, onUnmounted } from 'vue'
import type { Task } from '../types/domain'
import { getQueue } from '../types/wails'

export function useTasks() {
  const tasks = ref<Task[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let interval: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    try {
      tasks.value = await getQueue()
      error.value = null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load tasks'
    }
  }

  onMounted(async () => {
    loading.value = true
    await refresh()
    loading.value = false
    interval = setInterval(refresh, 2000)
  })

  onUnmounted(() => {
    if (interval) clearInterval(interval)
  })

  return { tasks, loading, error, refresh }
}
```

## SSE endpoint details
- URL: `http://127.0.0.1:63987/api/events` (but use relative path `/api/events` in the Vite dev
  proxy environment; in Wails build the wails.ts helpers rewrite base URL to `:63987` already)
- Event format: plain SSE, `data:` lines only, no `event:` field used
- Each `data:` payload is JSON: `{ "type": "task_updated" | "connected", "taskId": "...", "status": "..." }`
- Connection sends an initial `data: {"type":"connected"}` ping on connect

## Wails compatibility note
In Wails desktop builds, `getQueue()` is a native binding. However `EventSource` connecting to
`http://127.0.0.1:63987/api/events` must work because Wails exposes the HTTP API on `:63987`.
To detect the base URL, import and use `getBaseURL()` from `../types/wails` if it exists, otherwise
default to `http://127.0.0.1:63987`.

Actually: keep it simple — use a relative URL `/api/events` for browser (Vite proxy will handle it)
and `http://127.0.0.1:63987/api/events` for Wails. Check `window.__WAILS__` or
`typeof window !== 'undefined' && (window as any).__wails !== undefined` to detect Wails context.
If detection is complex, just use `http://127.0.0.1:63987/api/events` unconditionally — it works
in both environments.

## Implementation Steps
1. Keep all existing exports (`tasks`, `loading`, `error`, `refresh`) unchanged.
2. Replace `setInterval` with an `EventSource` connection when `typeof EventSource !== 'undefined'`:
   a. Create `new EventSource('http://127.0.0.1:63987/api/events')`.
   b. On `es.onmessage`: parse `event.data` as JSON, if `type !== 'connected'` call `refresh()`.
   c. On `es.onerror`: close EventSource, log a console.warn, fall back to polling every 2 s (set
      `interval = setInterval(refresh, 2000)`).
3. When `EventSource` is unavailable, fall back to `setInterval(refresh, 2000)` immediately.
4. In `onUnmounted`: close the EventSource (if open) and clear any fallback interval.
5. Output the complete updated file.

## Acceptance Criteria
- [ ] `EventSource` used when available — interval polling removed as primary mechanism
- [ ] On SSE error the composable falls back to `setInterval` polling (2 s)
- [ ] `onUnmounted` closes EventSource and clears interval
- [ ] All existing exported names (`tasks`, `loading`, `error`, `refresh`) preserved
- [ ] Output is the complete file (no truncation)

## Anti-patterns to Avoid
- Do NOT import anything new from Vue beyond what's already imported
- Do NOT change the function signature or return values
- Do NOT use `fetch` or `axios` — only `EventSource` for the SSE connection
- Do NOT add console.log — use `console.warn` only for the fallback error case
