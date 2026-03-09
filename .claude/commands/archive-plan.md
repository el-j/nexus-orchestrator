You are the plan archivist for the **figma-vue-bridge** monorepo. Your role is to seal a completed plan by moving it out of the active registry into a permanent archive file — keeping `orchestrator.json` slim permanently.

> **When to run this**: After `/execute-plan` or `/execute-task` reports "🎉 Plan PLAN-NNN complete" — all tasks in the plan are done.

## Plan to archive

$ARGUMENTS

---

## Step 1 — Validate the plan is ready to archive

1. Read `.claude/orchestrator.json`.
2. Resolve the plan ID from `$ARGUMENTS` (e.g., `PLAN-018`).
3. Look up the plan in `orchestrator.json` `plans["PLAN-NNN"]`.
   - If not found: stop — "PLAN-NNN not found in active registry. It may already be archived in `.claude/plans/PLAN-NNN.json`."
4. Collect all task IDs from `plans["PLAN-NNN"].taskIds`.
5. For each task ID, check `orchestrator.json` `tasks["TASK-NNN"].status`:
   - If the task entry exists in the map: it must be `done`, `deferred`, or `superseded`. If any task is `todo`, `in-progress`, or `blocked`, stop:
     ```
     ❌ Cannot archive PLAN-NNN — tasks still pending:
       TASK-NNN: <title> (status: <status>)
     Complete or defer those tasks first.
     ```
   - If the task entry does **not** exist in the map (was already deleted from `orchestrator.json`), proceed to step 6 to validate via the .md file instead.

6. **Physical .md file validation (authoritative check):** For each task ID, read `.claude/tasks/TASK-NNN.md`.
   - Extract the `status:` field from the YAML front-matter (the line that starts with `status:` inside the `---` fences).
   - Acceptable values: `done`, `deferred`, `superseded`.
   - If the status is `todo`, `in-progress`, or `blocked` (or any other non-done value), **stop immediately**:
     ```
     ❌ Cannot archive PLAN-NNN — task .md file is not marked done:
       .claude/tasks/TASK-NNN.md: status: <status>
     This means Phase E (Mark complete) was skipped during task execution.
     Fix: set `status: done` in the front-matter and add an ## Execution Log section.
     ```
   - If the .md file does not exist: this is a warning only — log it but do not block archiving.
   - If orchestrator.json showed `done` but .md shows `todo`: this is a **conflict**. Stop with:
     ```
     ❌ Status conflict for TASK-NNN:
       orchestrator.json: done
       .claude/tasks/TASK-NNN.md: todo
     Update the .md file to status: done before archiving.
     ```

---

## Step 2 — Build the archive document

Construct the archive data:

```json
{
  "id": "PLAN-NNN",
  "goal": "<goal from plans entry>",
  "status": "done",
  "createdAt": "<from plan entry>",
  "completedAt": "<completedAt from plan, or current ISO 8601 if missing>",
  "archivedAt": "<current ISO 8601 timestamp>",
  "note": "<any note from plan entry, or empty string>",
  "criticalPath": ["TASK-NNN", "..."],
  "tasks": {
    "TASK-NNN": { "<full task object from orchestrator.json tasks map>" },
    "...": {}
  }
}
```

Include **all** task objects for this plan (from the `tasks` map in `orchestrator.json`). Include the full object — not just IDs.

---

## Step 3 — Write the archive file

Write the archive JSON to `.claude/plans/PLAN-NNN.json` (create the `.claude/plans/` directory if it does not exist).

Do **not** overwrite an existing archive file. If `.claude/plans/PLAN-NNN.json` already exists, stop:

```
❌ .claude/plans/PLAN-NNN.json already exists. Archive is already sealed.
```

---

## Step 4 — Update orchestrator-index.md

Read `.claude/orchestrator-index.md`. If it does not exist, create it with this header:

```markdown
# Orchestrator Plan Index

_This file is for human reference only. Commands do not load it._

| Plan | Goal | Status | Tasks | Completed |
| ---- | ---- | ------ | ----- | --------- |
```

Append a new row for the archived plan:

```
| PLAN-NNN | <goal truncated to 70 chars> | done | <task count> | <completedAt date YYYY-MM-DD> |
```

Write the updated file.

---

## Step 5 — Trim orchestrator.json

Remove the archived plan and its tasks from `orchestrator.json`:

1. Delete `plans["PLAN-NNN"]` from the `plans` map.
2. Delete every `tasks["TASK-NNN"]` that belonged to this plan.
3. If `activePlanId === "PLAN-NNN"`, set `activePlanId = null`.
4. Set `updatedAt = <current ISO 8601 timestamp>`.
5. Update `notes` to: `"PLAN-NNN archived in .claude/plans/. See orchestrator-index.md for history."`.
6. Write the updated `orchestrator.json`.

---

## Step 6 — Output confirmation

Print:

```
✅ PLAN-NNN archived

Archive:    .claude/plans/PLAN-NNN.json  (<task count> tasks)
Index:      .claude/orchestrator-index.md  (updated)
Active registry trimmed: orchestrator.json

orchestrator.json now contains:
  Plans:  <count remaining>
  Tasks:  <count remaining>
```

---

## Constraints

- NEVER archive a plan with non-done tasks — validate BOTH `orchestrator.json` AND the physical `.claude/tasks/TASK-NNN.md` front-matter (Steps 5–6).
- NEVER overwrite an existing `.claude/plans/PLAN-NNN.json`.
- Do NOT modify task `.md` files during archiving — if they show a non-done status, stop and instruct the caller to fix them first.
- The archive is a permanent, immutable record. Do not add interpretation or summaries beyond what is already in the plan entry.
- The `.md` front-matter is the **authoritative source of truth** for task completion. `orchestrator.json` may be stale or manually edited; the `.md` file can only reach `status: done` via deliberate Phase E execution.
