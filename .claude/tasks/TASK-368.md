# TASK-368: Plans view — Project Brain header card + group task files by parent plan

**Plan:** PLAN-054 | **Wave:** 2 | **Status:** done | **Role:** frontend

## Goal

Transform the Plans view from a flat file dump into a structured "project brain" overview:

1. Show `orchestrator.json` (kind=nexus) as a special "Project Brain" header card with project stats
2. Group `claude-task` files by their parent plan ID (parsed from summary/filename)
3. Show each plan group with a header showing plan ID, task count, completion status

## Context

- `DiscoveredPlanFile.kind` is `"nexus"` for orchestrator.json, `"claude-task"` for TASK-\*.md
- `DiscoveredPlanFile.summary` for task files starts: `# TASK-NNN: ... **Plan:** PLAN-052 | **Wave:** ...`
- The nexus kind summary (enriched by TASK-372) will contain: `plans: 52, tasks: 363, completedPlans: 42, activePlan: null`
- File: `frontend/src/views/DiscoveredPlansView.vue`

## Implementation

### Parsing plan parent from task summary

```typescript
function extractPlanId(file: DiscoveredPlanFile): string | null {
  const m = file.summary.match(/\*\*Plan:\*\*\s+(PLAN-\d+)/);
  return m ? m[1] : null;
}

function extractPlanIdFromPath(file: DiscoveredPlanFile): string | null {
  // fallback: TASK-NNN.md → look for numeric range correlating to plan
  const taskNum = parseInt(file.path.match(/TASK-(\d+)/)?.[1] ?? '0');
  if (!taskNum) return null;
  // Not reliable without parsing — prefer summary extraction
  return null;
}
```

### View structure

```
[Project Brain Card — nexus kind]
  nexusOrchestrator / .claude/orchestrator.json
  52 plans · 363 tasks · last modified today

[Other kind pills: 3 markdown · 2 cursor · 1 mcp-config]

[PLAN-052 group — 8 files]  ← collapsible
  TASK-356.md  TASK-357.md  TASK-358.md  ...

[PLAN-054 group — 4 files]  ← collapsible

[Ungrouped / Other]
  CLAUDE.md  ROADMAP.md  ...
```

### Key components to add in DiscoveredPlansView.vue:

1. `nexusBrain` computed — find kind===nexus file, parse its summary for stats
2. `planGroups` computed — group claude-task files by extracted plan ID, sort by plan number desc
3. `otherFiles` computed — all non-nexus, non-grouped files
4. Kind summary pills across the top
5. Plan group accordion/collapsible section
6. ProjectBrain card at top above the groups

## Status

todo
