# Plans and tasks: where the files live

An agent plans in markdown files. Nexus reads them so the VS Code **Workspace Agents** view, the
daemon's onboarding context and every new session know the active plan and what is left to do. The
files may sit in any folder and be written by any agent (Claude, Gemini, Copilot, a person); nothing
needs an `orchestrator.json`.

## What is found by itself

Without any configuration, in this order (the first file with an id wins):

1. `.claude/plans`, `.claude/tasks`, `.nexus/plans`, `.nexus/tasks`;
2. anywhere else in the project (up to six levels deep): a markdown file whose name is `PLAN-n…` or
   `TASK-n…`, whose folder is called `plans`/`plan` or `tasks`/`task`, or whose front matter says
   `type: plan` or `type: task`.

`node_modules`, `vendor`, build output and hidden folders (except `.claude`, `.nexus`, `.github`)
are not searched; `README.md`, `index.md` and `CHANGELOG.md` are never plans.

## Telling Nexus where they are

If the project keeps plans and tasks elsewhere under free file names, say so in `.nexus/policy.json`
(committed, so every tool and teammate agrees):

```json
{
  "planFiles": {
    "plans": ["design/specs"],
    "tasks": ["design/work"],
    "discover": true
  }
}
```

- `plans` / `tasks`: project-relative folders; every markdown file in them (three levels deep) is a
  plan or a task, whatever it is called. Paths must stay inside the project.
- `discover` (default `true`): also search the whole project as above. `false` limits the search to
  the listed folders (or to the defaults when none are listed).

In VS Code you can do the same without editing JSON: **Nexus: Choose Plan and Task Folders** (add a
folder with a folder picker, remove one, switch the project-wide search, open `policy.json`), or the
user settings `nexus.planFiles.plans`, `nexus.planFiles.tasks`, `nexus.planFiles.discover`, which are
added to the project's. The view shows which folders it read in the tooltip of the project node.

## What is read from a file

Front matter when present (`id`, `title`, `status`, `role`, `planId` / `plan` / `plan_id`, and for a
plan `tasks: [TASK-1, TASK-2]` or a `- item` list). Otherwise: the id from the file name
(`TASK-010-foo.md` → `TASK-010`, else the file name), the title from the first `#` heading (a leading
`TASK-010 —` is dropped), the status from a `Status:` or `**Status:**` line.

Active plan: the plan `orchestrator.json` names if there is one, else a plan whose status is
active / in progress / doing / wip, else the newest plan that is not done.

## How tasks are linked to plans

A task belongs to the first of these that applies:

1. the plan it names (`planId`, `plan` or `plan_id`, matched against a plan's id or file name),
2. the plan that lists it in its front matter,
3. the plan whose name its folder carries (`tasks/PLAN-2/TASK-5.md`),
4. the first `PLAN-n` its text mentions,
5. a plan whose text mentions the task id.

Tasks no plan claims show up under "Unassigned tasks"; a task a plan names but that has no file is
still listed under the plan.

The daemon and the extension implement the same rules and are tested against the same cases
(`testdata/planfiles/cases.json`), so they cannot drift apart.
