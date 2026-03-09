You are a task-definition writer for the **figma-vue-bridge** monorepo. Given the description below, produce one precise, self-contained task file and register it in `.claude/orchestrator.json`.

> **v2 Architecture**: `orchestrator.json` is a slim active registry. Use `counters.nextTaskId` for the new task ID — do **not** scan task keys to compute the max.

## Task description

$ARGUMENTS

---

## Step 1 — Load orchestrator state and determine the next task ID

Read `.claude/orchestrator.json`.

- If the file does not exist, initialise it with the v2 empty structure:
  ```json
  {
    "version": "2",
    "updatedAt": "<ISO 8601>",
    "counters": { "nextTaskId": 1, "nextPlanId": 1 },
    "activePlanId": null,
    "notes": "",
    "plans": {},
    "tasks": {}
  }
  ```
- Read `counters.nextTaskId` — this is the ID number to use (zero-padded to 3 digits, e.g. `93` → `TASK-093`).
- If there is an `activePlanId`, note it — the new task will be linked to that plan.
  If there is no active plan, set `planId` to `null` in the registry entry.
- Scan the active `tasks` map keys only (not plans) to confirm no identical task already exists.

---

## Step 2 — Gather codebase context

Before writing the task, read the files most relevant to the description. Use CLAUDE.md as your map. At minimum:

- If the task involves a new shared type or schema → read `packages/shared/src/types/` and `packages/shared/src/schemas/` to check if it already exists.
- If the task involves the CLI → read `packages/cli/src/core/` for the relevant service(s).
- If the task involves the API → read `packages/api/src/routes/` and `packages/api/src/services/`.
- If the task involves the Web UI → read `packages/web-ui/src/views/` and `packages/web-ui/src/stores/`.
- If the task involves the Figma Plugin → read `packages/figma-plugin/src/code.ts` and `packages/figma-plugin/src/extractors.ts`.
- If the task involves the VS Code extension → read `packages/vscode-extension/src/extension.ts`.

Do **not** skip this. The task file must reference real file paths and real symbol names.

---

## Step 3 — Write the task file

Create `.claude/tasks/TASK-NNN.md` using this **exact** template:

```
---
id: TASK-NNN
title: "<concise imperative title, max 60 chars>"
status: todo
priority: <critical|high|medium|low>
role: <backend|frontend|schema|api|cli|figma-plugin|vscode|devops|ui|ux|qa|verify|planning>
dependencies: [<TASK-NNN, ...> or "none"]
estimated_effort: <XS 15min | S 30min | M 1h | L 2h | XL 4h+>
---

## Goal

One sentence: what this task achieves and why it is needed.

## Context

Key facts the executing agent must know (type names, service names, existing patterns to follow, pitfalls to avoid). Be specific. Reference exact file paths and the correct path aliases.

## Scope

### Files to modify
- `path/to/file.ts` — what change and why

### Files to create
- `path/to/new-file.ts` — what it contains

### Files to delete
- `path/to/old-file.ts` — reason

## Implementation

Step-by-step instructions in order. Be explicit about:
- Which package/path alias to use for imports
- Which existing types/schemas to import from `@figma-vue-bridge/shared`
- Which existing utilities/services to reuse
- What NOT to do

## Acceptance Criteria

- [ ] `npm run build` exits 0 with no new TypeScript errors
- [ ] `npm run test --workspace=@figma-vue-bridge/<pkg>` exits 0
- [ ] <specific functional criterion 1>
- [ ] <specific functional criterion 2>
- [ ] No `any` types introduced
```

---

### Role field → specialist agent

| role           | Agent file                                                  | Use for                                          |
| -------------- | ----------------------------------------------------------- | ------------------------------------------------ |
| `backend`      | `.github/agents/engineering-senior-developer.agent.md`      | CLI core, transformers, shared utilities         |
| `frontend`     | `.github/agents/web-ui.agent.md`                            | Vue components, Pinia stores, composables        |
| `schema`       | `.github/agents/shared.agent.md`                            | Zod schemas, shared types in `packages/shared`   |
| `api`          | `.github/agents/api.agent.md`                               | API routes, WebSocket, Express handlers          |
| `cli`          | `.github/agents/cli.agent.md`                               | CLI commands, generators, transformers           |
| `figma-plugin` | `.github/agents/figma-plugin.agent.md`                      | Figma Plugin code, extractors, JSON exports      |
| `vscode`       | `.github/agents/vscode-extension.agent.md`                  | VS Code extension commands, activation           |
| `devops`       | `.github/agents/engineering-senior-developer.agent.md`      | Docker, CI, build configs, Turborepo             |
| `ui`           | `.github/agents/design-ui-designer.agent.md`                | Visual design, component styling, design tokens  |
| `ux`           | `.github/agents/design-ux-architect.agent.md`               | User flows, information architecture             |
| `qa`           | `.github/agents/testing-evidence-collector.agent.md`        | Writing tests, documenting issues                |
| `verify`       | `.github/agents/testing-reality-checker.agent.md`           | Integration/build/test certification             |
| `planning`     | `.github/agents/project-manager-senior.agent.md`            | Breaking down ambiguous goals                    |

---

## Step 4 — Register in orchestrator.json

After saving the task file, update `.claude/orchestrator.json`:

1. Add an entry to `tasks`:

```json
"TASK-NNN": {
  "id": "TASK-NNN",
  "planId": "<activePlanId or null>",
  "title": "<title from task file>",
  "role": "<role from task file>",
  "status": "todo",
  "priority": "<priority>",
  "dependencies": ["TASK-NNN"],
  "estimatedEffort": "<XS|S|M|L|XL>",
  "createdAt": "<ISO 8601 timestamp>",
  "startedAt": null,
  "completedAt": null,
  "filesChanged": [],
  "buildStatus": null,
  "testStatus": null,
  "notes": ""
}
```

2. Increment `counters.nextTaskId` by 1.
3. If there is an `activePlanId`, append `TASK-NNN` to that plan's `taskIds` array.
4. Set `"updatedAt"` to the current ISO 8601 timestamp.
5. Write the full updated JSON back to `.claude/orchestrator.json`.

---

## Step 5 — Output confirmation

Print:

```
Created:      .claude/tasks/TASK-NNN.md
Registered:   .claude/orchestrator.json → tasks.TASK-NNN (nextTaskId now NNN+1)
Title:        <title>
Role:         <role> → will use <agent file>
Priority:     <priority>
Effort:       <effort>
Dependencies: <list or none>
Plan:         <PLAN-NNN or unlinked>

Run `/execute-task TASK-NNN` to implement this task.
```

---

## Constraints

- One task file only — do not create multiple tasks.
- If the description is too vague to write a scoped task, state what information is missing and stop. Do not invent scope.
- NEVER add `any` types to implementation instructions.
- NEVER point shared types to `packages/api` or `packages/cli` — always `packages/shared/src/`.
- NEVER use `npx vitest` — always `npm run test --workspace=@figma-vue-bridge/<pkg>`.
- NEVER use `console.log` in implementation instructions — always the `logger` from `@figma-vue-bridge/shared`.
- ALWAYS assign a `role:` field to every task.
- ALWAYS check existing `tasks` in `orchestrator.json` before creating a duplicate — identical task descriptions must be rejected with an explanation.
- ALWAYS follow project coding standards from CLAUDE.md.
