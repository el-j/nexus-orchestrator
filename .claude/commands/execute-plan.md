You are the parallel plan executor for the **figma-vue-bridge** monorepo. Your role is to execute all tasks in the active plan as fast as possible by running independent tasks in parallel waves, with each task handled by a dedicated specialist sub-agent.

> **When to run this**: When you have a plan with multiple tasks and want maximum throughput. Sub-agents do the implementation work; this command orchestrates them.

## Plan to execute

$ARGUMENTS _(optional: a PLAN-NNN id; defaults to `activePlanId`)_

---

## Phase 0 — Load plan and build dependency graph

1. Read `.claude/orchestrator.json`.
2. Resolve the plan:
   - If `$ARGUMENTS` is a plan ID (e.g. `PLAN-018`), use it.
   - Otherwise use `activePlanId`. If neither is set, stop: "No active plan. Run `/orchestrator <goal>` first."
3. Look up the plan in `plans["PLAN-NNN"]`.
4. Collect all task IDs from `taskIds`.
5. For each task ID, read `orchestrator.json` `tasks[id]` and the corresponding `.claude/tasks/TASK-NNN.md` file.
6. **Build a dependency graph:**
   - Node = task ID
   - Edge = dependency relationship (A → B means B depends on A)
   - Skip tasks already `done` — they count as resolved for dependency purposes.
   - Skip tasks that are `blocked` — report them up-front and exclude from execution waves.
7. Compute **execution waves** by topological sort:
   - **Wave 0**: all non-done tasks with no unresolved dependencies
   - **Wave 1**: tasks whose only dependencies are in Wave 0
   - **Wave N**: tasks whose dependencies are all in Waves 0..N-1
8. Print the wave plan (see Output Format A below) and wait for user confirmation only if any wave has more than 4 tasks. Otherwise proceed automatically.

---

## Phase 1 — Confirm baseline

Run `npm run build 2>&1 | tail -8`. If the build fails (non-zero exit or `error TS`), stop and report errors. Do not execute tasks on a broken baseline.

---

## Phase 2 — Execute waves in order

For each wave, execute all tasks in that wave **in parallel** as sub-tasks.

### Sub-task prompt template for each task

Spawn one `Task` per task in the wave. Each sub-task prompt must be self-contained:

> **Prompt:**
>
> You are a precision implementation agent for the **figma-vue-bridge** monorepo.
>
> **Your role for this task:** `<role from task front-matter>`
> **Persona to adopt:** Read `.github/agents/<agent-file-for-role>` fully. Adopt that agent's identity, quality standards, and working style.
>
> **Task:** `TASK-NNN — <title>`
> **Task file:** `.claude/tasks/TASK-NNN.md`
>
> Execute the following phases with **minimal intermediate output** — only print start/end markers and your final result JSON.
>
> **PHASE A — Read task file**
> Read `.claude/tasks/TASK-NNN.md` completely.
>
> **PHASE B — Read context files in parallel**
> Spawn parallel sub-tasks (one per package group) to read all files listed in `## Scope`. Wait for all to complete, merge results.
>
> **PHASE C — Implement**
> Follow `## Implementation` steps exactly. Enforce all project rules:
>
> - NO `any` types → use `unknown` with Zod validation or explicit narrowing
> - NO raw `throw` from public service methods → use `Result<T,E>` pattern
> - Zod validation at ALL API boundaries (inline in route handlers)
> - `fs-extra` for ALL file operations (atomic: write `.tmp` then rename)
> - `Result<T,E>` for error handling in CLI core logic
> - `<script setup lang="ts">` in all Vue components
> - Pinia stores use setup function syntax only
> - `@figma-vue-bridge/shared` for all cross-package type imports
> - Path aliases: `@/` in web-ui; `@core/` `@config/` `@generators/` `@transformers/` in CLI
> - NO `console.log/error/warn` → use `logger` from `@figma-vue-bridge/shared`
> - NO `sleep` or arbitrary delays
>
> **PHASE D — Build & test**
> Run `npm run build 2>&1 | grep -E "(error|Error)" | head -20`
> Run `npm run test --workspace=@figma-vue-bridge/<most-relevant-pkg>`
> If build or tests fail, fix the errors before reporting done.
>
> **PHASE E — Mark complete (MANDATORY — do not skip)**
>
> 1. Update `.claude/tasks/TASK-NNN.md` front-matter: `status: done`.
> 2. Append `## Execution Log` to the task file with completed date, files changed, build ✅, tests ✅.
>
> Return ONLY this JSON when done (no other text):
>
> ```json
> {
>   "taskId": "TASK-NNN",
>   "status": "done" | "blocked" | "failed",
>   "filesChanged": ["path/to/file.ts", "..."],
>   "buildStatus": "clean" | "errors",
>   "testStatus": "passed" | "failed" | "passed-with-preexisting-failures",
>   "blockedReason": null,
>   "notes": "any relevant notes"
> }
> ```

---

### Role → agent file mapping

| role           | Agent file                                              |
| -------------- | ------------------------------------------------------- |
| `backend`      | `.github/agents/engineering-senior-developer.agent.md`  |
| `frontend`     | `.github/agents/web-ui.agent.md`                        |
| `schema`       | `.github/agents/shared.agent.md`                        |
| `api`          | `.github/agents/api.agent.md`                           |
| `cli`          | `.github/agents/cli.agent.md`                           |
| `figma-plugin` | `.github/agents/figma-plugin.agent.md`                  |
| `vscode`       | `.github/agents/vscode-extension.agent.md`              |
| `devops`       | `.github/agents/engineering-senior-developer.agent.md`  |
| `ui`           | `.github/agents/design-ui-designer.agent.md`            |
| `ux`           | `.github/agents/design-ux-architect.agent.md`           |
| `qa`           | `.github/agents/testing-evidence-collector.agent.md`    |
| `verify`       | `.github/agents/testing-reality-checker.agent.md`       |
| `planning`     | `.github/agents/project-manager-senior.agent.md`        |
| _(unknown)_    | `.github/agents/engineering-senior-developer.agent.md`  |

---

## Phase 3 — Process wave results

> ⛔ **BLOCKING GATE — this phase MUST run after every wave, without exception.**
> Skipping Phase 3 or processing results informally (e.g. just reading the sub-agent JSON without updating files) leaves the codebase in a broken state where task `.md` files still show `status: todo`. The `/archive-plan` command validates `.md` front-matter directly and will refuse to proceed if any file is non-done.

After all sub-tasks in a wave complete, process each returned JSON:

1. For each `"status": "done"` result:

   > ⚠️ **MANDATORY — update the task `.md` file FIRST, then `orchestrator.json`.**
   > The `.md` front-matter is the **canonical source of truth**. Even if the sub-agent marked the file done in its PHASE E, verify it. If not done, update it now — this is a non-optional correctness step, not a clerical formality.

   a. **Update `.claude/tasks/TASK-NNN.md`**:
      - Set front-matter `status: done`.
      - If there is no `## Execution Log` section, append one:
        ```markdown
        ## Execution Log

        - **Completed:** <ISO 8601 date>
        - **Files changed:** <result.filesChanged joined by comma>
        - **Build:** ✅ clean
        - **Tests:** ✅ <result.testStatus>
        - **Notes:** <result.notes>
        ```
      - Write the file.

   b. Update `orchestrator.json` `tasks["TASK-NNN"]`:
      - `status = "done"`, `completedAt = <now ISO 8601>`
      - `filesChanged = <result.filesChanged>`
      - `buildStatus = <result.buildStatus>`
      - `testStatus = <result.testStatus>`
      - `notes = <result.notes>`
   - Write `orchestrator.json`.

2. For each `"status": "blocked"` or `"failed"` result:
   - Update `orchestrator.json`: `tasks["TASK-NNN"].status = "blocked"`, `notes = <reason>`.
   - Mark any downstream tasks (in later waves) that depend on this task as `"skip"` in the wave plan.
   - Print a warning but **continue with other tasks in the wave** — do not stop the whole plan.

3. After processing all results for a wave, print the wave summary (see Output Format B).

4. Proceed to the next wave.

---

## Phase 4 — Check plan completion

After all waves are processed:

1. Check if all tasks in `plans["PLAN-NNN"].taskIds` are now `done` (or `blocked`/`deferred`).
2. If all done:
   - Set `plans["PLAN-NNN"].status = "done"`, `activePlanId = null`.
   - Write `orchestrator.json`.
   - Print the final summary (Output Format C).
   - Print: "🎉 Plan PLAN-NNN complete — run `/archive-plan PLAN-NNN` to seal it."
3. If any tasks are blocked:
   - Print which tasks are blocked and why.
   - Print: "Plan PLAN-NNN partially complete — resolve blocked tasks and re-run."

---

## Output formats

### Format A — Wave plan (printed before execution starts)

```
📋 PLAN-NNN execution plan — <goal>

Wave 0 (parallel, N tasks):
  TASK-NNN  [cli]      <title>
  TASK-NNN  [frontend] <title>

Wave 1 (parallel after Wave 0, N tasks):
  TASK-NNN  [qa]       <title>

Blocked (excluded):
  TASK-NNN  [backend]  <title> — reason

Total: N tasks across N waves
Proceeding automatically...
```

### Format B — Wave summary (after each wave)

```
✅ Wave N complete (N/N tasks done)
  ✅ TASK-NNN  [cli]      <title>  — 3 files changed
  ✅ TASK-NNN  [frontend] <title>  — 2 files changed
  ❌ TASK-NNN  [api]      <title>  — BLOCKED: missing dependency X
```

### Format C — Final plan summary

```
🎉 PLAN-NNN complete — <goal>

Results:
  ✅ Done:     N tasks
  ⏭️  Deferred: N tasks
  ❌ Blocked:  N tasks

Total files changed: N
Waves executed: N
All state saved to .claude/orchestrator.json

Run `/archive-plan PLAN-NNN` to seal this plan.
```

---

## Constraints

- **NEVER** spawn more than 6 parallel sub-tasks in a single wave — batch larger waves into groups of 6.
- **NEVER** start a wave until all sub-tasks from the previous wave have returned results.
- **NEVER** modify `orchestrator.json` from within a sub-task — only the orchestrating agent patches it.
- **ALWAYS** process blocked/failed results before starting the next wave.
- If a task has no `role:` in its front-matter, default to `backend`.
