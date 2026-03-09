# Tasks

Task files are created here by `/orchestrator` and `/new-task` commands.
Each file is named `TASK-NNN.md` and executed with `/execute-task TASK-NNN`.

## Commands

| Command | Usage | Description |
|---------|-------|-------------|
| `/orchestrator <goal>` | `/orchestrator "Add Playwright e2e tests"` | Decompose a goal into a plan of ordered tasks (v2: parallel discovery, role assignments) |
| `/new-task <description>` | `/new-task "Fix TypeScript errors in API tests"` | Create a single focused task (v2: uses counters, requires role field) |
| `/execute-task TASK-NNN` | `/execute-task TASK-093` | Implement one task (adopts agent persona, parallel context read, marks done) |
| `/execute-plan PLAN-NNN` | `/execute-plan PLAN-018` | Execute all tasks in a plan as parallel waves (maximum throughput) |
| `/archive-plan PLAN-NNN` | `/archive-plan PLAN-017` | Seal a completed plan into `.claude/plans/PLAN-NNN.json`, trim orchestrator.json |

## Workflow

```
/orchestrator "my big goal"
  → creates PLAN-018 + TASK-093, TASK-094, ... in orchestrator.json
  → writes .claude/tasks/TASK-NNN.md files

# Option A — sequential execution
/execute-task TASK-093
  → implements TASK-093, verifies build+tests, marks done

/execute-task TASK-094
  → implements TASK-094 (only if TASK-093 is done), marks done

# Option B — parallel execution (faster)
/execute-plan PLAN-018
  → builds dependency graph
  → runs Wave 0 tasks in parallel, then Wave 1, etc.
  → marks all tasks done

# When plan is complete
/archive-plan PLAN-018
  → writes .claude/plans/PLAN-018.json (permanent archive)
  → updates .claude/orchestrator-index.md
  → removes PLAN-018 and its tasks from orchestrator.json (keeps it slim)
```

## State

All active plan/task state is persisted in `.claude/orchestrator.json` (v2).
Completed plans are archived to `.claude/plans/PLAN-NNN.json`.
Full plan history is indexed in `.claude/orchestrator-index.md`.

## orchestrator.json v2 structure

```json
{
  "version": "2",
  "counters": { "nextTaskId": 93, "nextPlanId": 18 },
  "activePlanId": null,
  "notes": "...",
  "plans": { "PLAN-NNN": { ... } },
  "tasks": { "TASK-NNN": { ..., "role": "cli" } }
}
```

Key v2 improvements over v1:
- `counters.nextTaskId` / `counters.nextPlanId` — no scanning required
- `role:` field on every task — drives agent persona selection
- `/execute-plan` — wave-based parallel execution
- `/archive-plan` — keeps orchestrator.json slim after plan completion

