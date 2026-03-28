# TASK-371: Task History — Add pipeline status summary bar

**Plan:** PLAN-054 | **Wave:** 1 | **Status:** done | **Role:** frontend

## Goal

Add a status distribution summary to HistoryView showing counts across ALL task statuses
(not just terminal ones), so users understand the full pipeline state at a glance.
Currently shows only 2 old completed tasks with no context.

## Context

- `GET /api/tasks` returns ALL tasks regardless of status
- HistoryView already calls `getAllTasks()` which fetches all tasks
- The view then filters to `COMPLETED | FAILED | CANCELLED` for display
- Status values: `QUEUED`, `PROCESSING`, `COMPLETED`, `FAILED`, `CANCELLED`, `DRAFT`, `BACKLOG`
- File: `frontend/src/views/HistoryView.vue`

## Implementation

### Add status summary computed:

```typescript
const statusSummary = computed(() => {
  const counts: Record<string, number> = {};
  for (const t of tasks.value) {
    counts[t.status] = (counts[t.status] ?? 0) + 1;
  }
  return counts;
});

const summaryBadges = computed(() =>
  [
    {
      label: 'Queued',
      status: 'QUEUED',
      color: 'bg-slate-500/20 text-slate-400',
      count: statusSummary.value['QUEUED'] ?? 0,
    },
    {
      label: 'Processing',
      status: 'PROCESSING',
      color: 'bg-yellow-500/20 text-yellow-300',
      count: statusSummary.value['PROCESSING'] ?? 0,
    },
    {
      label: 'Completed',
      status: 'COMPLETED',
      color: 'bg-emerald-500/20 text-emerald-300',
      count: statusSummary.value['COMPLETED'] ?? 0,
    },
    {
      label: 'Failed',
      status: 'FAILED',
      color: 'bg-red-500/20 text-red-300',
      count: statusSummary.value['FAILED'] ?? 0,
    },
    {
      label: 'Backlog',
      status: 'BACKLOG',
      color: 'bg-violet-500/20 text-violet-300',
      count: statusSummary.value['BACKLOG'] ?? 0,
    },
    {
      label: 'Draft',
      status: 'DRAFT',
      color: 'bg-blue-500/20 text-blue-300',
      count: statusSummary.value['DRAFT'] ?? 0,
    },
  ].filter((b) => b.count > 0),
);
```

### Add summary bar to header area (below the title):

```vue
<div v-if="summaryBadges.length > 0" class="flex flex-wrap gap-2 px-5 py-2 border-b border-white/5">
  <span
    v-for="badge in summaryBadges"
    :key="badge.status"
    class="text-[11px] px-2 py-1 rounded-lg font-semibold cursor-pointer transition-colors"
    :class="[badge.color, selectedStatus === badge.status ? 'ring-1 ring-white/30' : '']"
    :title="`Show ${badge.label} tasks`"
    @click="selectedStatus = selectedStatus === badge.status ? 'ALL' : badge.status"
  >
    {{ badge.count }} {{ badge.label }}
  </span>
</div>
```

### Make the filter tab state work with the new badges too:

- Clicking a badge sets `selectedStatus` to that status (or 'ALL' to deselect)
- Existing tab buttons (All / Completed / Failed / Cancelled) remain functional
- `selectedStatus` drives the filter, unifying both tab buttons and badge clicks

## Status

todo
