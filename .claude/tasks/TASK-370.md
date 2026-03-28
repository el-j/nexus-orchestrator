# TASK-370: Live AI view — Session grouping and conversation thread display

**Plan:** PLAN-054 | **Wave:** 1 | **Status:** done | **Role:** frontend

## Goal

Group Live AI activities by `sessionId` into conversation threads showing request/response
pairs, instead of a flat raw event list. Activities from the same session should appear
visually connected.

## Context

- `AIActivity` has `sessionId: string` field (maps to Claude JSONL session UUID)
- Activity types: `message` (user prompt) + `generation`/`tool_use` (Claude response)
- Current display: flat chronological list — "User prompt" and "Responding (546 tokens)" as separate unconnected rows
- File: `frontend/src/views/LiveActivityView.vue`
- Composable: `frontend/src/composables/useActivities.ts`

## Implementation

### Group activities by session in the view:

```typescript
// In LiveActivityView.vue computed section:
const sessionGroups = computed(() => {
  const groups: Map<string, AIActivity[]> = new Map();
  for (const a of filtered.value) {
    const key = a.sessionId || `solo-${a.id}`;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key)!.push(a);
  }
  // Convert to array sorted by most recent activity desc
  return Array.from(groups.entries())
    .map(([sid, acts]) => ({
      sessionId: sid,
      agentName: acts[0].agentName,
      projectPath: acts[0].projectPath,
      model: acts.find((a) => a.model)?.model ?? '',
      latestTime: acts[0].timestamp,
      totalTokens: acts.reduce((sum, a) => sum + (a.tokensOut ?? 0) + (a.tokensIn ?? 0), 0),
      activities: acts,
    }))
    .sort((a, b) => new Date(b.latestTime).getTime() - new Date(a.latestTime).getTime());
});
```

### Thread display template (replace flat list with grouped threads):

```vue
<div v-for="group in sessionGroups" :key="group.sessionId"
     class="mb-4 rounded-xl border border-white/[0.06] bg-white/[0.01] overflow-hidden">
  <!-- Session header -->
  <div class="flex items-center gap-3 px-4 py-2 bg-white/[0.03] border-b border-white/[0.06]">
    <span class="w-2 h-2 rounded-full bg-emerald-400 flex-shrink-0"></span>
    <span class="text-xs font-semibold text-slate-300">{{ group.agentName }}</span>
    <span v-if="group.model" class="text-[10px] text-slate-500 font-mono">· {{ group.model }}</span>
    <span class="ml-auto text-[10px] text-slate-600">{{ group.totalTokens }} tokens · {{ timeAgo(group.latestTime) }}</span>
    <span class="text-[10px] text-slate-600 font-mono truncate max-w-xs">{{ projectTail(group.projectPath) }}</span>
  </div>
  <!-- Activities in thread -->
  <div class="divide-y divide-white/[0.03]">
    <AIActivityCard v-for="a in group.activities" :key="a.id" :activity="a" compact />
  </div>
</div>
```

### Add `compact` prop to AIActivityCard to reduce padding in threaded mode

## Status

done
