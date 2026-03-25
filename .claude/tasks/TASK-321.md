# TASK-321: Frontend — AIAgentsView sub-agent tree + model badge + workingDir

**Plan:** PLAN-048
**Role:** frontend
**Dependencies:** TASK-318

## Goal

Enrich the existing AIAgentsView "Discovered Agents" section to show:

- A **model badge** (e.g., `claude-sonnet-4-6`) on each agent card
- **Sub-agents** nested under their parent with a connecting visual tree line
- **Working directory** shown as a short path (last 2 segments)
- **Session model** on registered AI sessions (from `AISession.modelId`)

## Changes

### `frontend/src/views/AIAgentsView.vue` (or the component that renders discovered agents)

**Sub-agent tree:**

- Group `DiscoveredAgent[]` into a tree: agents without `parentAgentId` are roots; agents with `parentAgentId` are nested under their parent.
- Render root agents normally; under each root, render sub-agents indented with a `│ ` or left-border line to visually indicate the hierarchy.
- Sub-agents show the same card but smaller (`text-sm`) with a "sub-agent" label.

**Model badge:**

- If `agent.modelId` is non-empty, show a small badge chip: `bg-slate-100 text-slate-700 rounded px-1.5 text-xs font-mono`
- Shorten model names: `claude-sonnet-4-6` → `sonnet-4-6`, `gpt-4o-mini` → `gpt-4o-mini` (strip leading `claude-` prefix if present)

**Working directory:**

- If `agent.workingDir` is non-empty, show the last 2 path segments as a `text-slate-500 text-xs` line below the agent name.

**Registered sessions model:**

- `AISession` now has `modelId` — show same model badge on active session cards.

### `frontend/src/types/domain.ts`

Update `DiscoveredAgent` interface:

```ts
export interface DiscoveredAgent {
  // ... existing fields ...
  modelId?: string;
  workingDir?: string;
  parentAgentId?: string;
  subAgentIds?: string[];
}
```

Update `AISession` interface:

```ts
export interface AISession {
  // ... existing fields ...
  modelId?: string;
}
```

## Acceptance Criteria

- [ ] Sub-agents nested under their parent visually (indented + border line)
- [ ] Model badge visible on agent/session cards when `modelId` is set
- [ ] Working dir last 2 segments shown
- [ ] Registered session cards show `modelId` from AISession
- [ ] `npm run build` succeeds with no type errors
- [ ] No regressions in existing card rendering
