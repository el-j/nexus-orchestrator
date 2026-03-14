# TASK-287 — Frontend: Per-agent task timeline

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Vue · frontend  
**Depends on:** TASK-286  

## Objective

Clicking a session card in the AI Agents view expands an inline timeline of tasks owned by that session.

## Changes

### Inline panel in `AISessionCard.vue`

Add collapsible timeline section below the main card content. Triggered by a toggle button ("Show tasks" / "Hide tasks").

**On expand:** `GET /api/ai-sessions/{id}/tasks` → render tasks sorted by `updatedAt` desc.

If browser supports `EventSource` and the server advertises SSE (`Accept: text/event-stream`), open an SSE connection for live updates. On SSE event, prepend the new task to the list.

```typescript
let eventSource: EventSource | undefined;

function openSSE(sessionId: string) {
  eventSource = new EventSource(`${daemonUrl}/api/ai-sessions/${sessionId}/tasks`, {
    // EventSource doesn't support custom headers; rely on daemon defaulting to SSE
    // when Content-Type of the *response* is text/event-stream
  });
  eventSource.onmessage = (e) => {
    const event = JSON.parse(e.data);
    // Update or prepend task in local list
  };
}
```

Note: standard `EventSource` doesn't set `Accept` headers. The daemon handler should detect SSE by checking `r.Header.Get("Accept")` is absent or set by query param `?stream=1` as fallback. Add `?stream=1` support to `handleGetSessionTasks` in TASK-281.

### Timeline entry component (inline in card):

```html
<div class="timeline-entry" v-for="task in sessionTasks">
  <span class="status-chip" :class="task.status.toLowerCase()">{{ task.status }}</span>
  <span class="target-file">{{ task.targetFile }}</span>
  <span class="instruction-excerpt">{{ task.instruction.slice(0, 80) }}…</span>
  <span class="rel-time">{{ relativeTime(task.updatedAt) }}</span>
</div>
```

Maximum 50 tasks rendered. "Load more" button — deferred; placeholder only in this task.

## Acceptance Criteria

- Timeline appears on card click without page navigation
- Tasks rendered newest-first
- SSE connection opened when available (falls back to static list gracefully)
- `EventSource` is closed when the card collapses or the component is unmounted
