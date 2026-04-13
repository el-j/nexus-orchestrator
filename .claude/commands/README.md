# .claude/commands — Slash Commands

All commands are invoked as `/command-name [arguments]`.

---

## Task Lifecycle

| Command                  | Purpose                                                           |
| ------------------------ | ----------------------------------------------------------------- |
| `/new-task`              | Create a new task file + register in orchestrator.json            |
| `/state`                 | Read-only project status overview (tasks, build health, branches) |
| `/validate`              | Deep plan validation against codebase before implementation       |
| `/execute-task TASK-NNN` | Execute a single task: branch → impl → review → test → merge      |
| `/orchestrator`          | Autonomous pipeline: process entire queue unattended              |
| `/review`                | White-box code review against task plan                           |
| `/test`                  | Black-box acceptance tests (no implementation knowledge)          |
| `/testfix`               | Intelligent test failure analysis and fix                         |
| `/learn`                 | Post-task learning agent — writes insights into agent MDs         |
| `/resolve [TASK-NNN]`    | Detect merge conflicts and create resolution tasks                |

## Agent Routing

| Command       | Purpose                                                |
| ------------- | ------------------------------------------------------ |
| `/hire-agent` | Route a task description to the right specialist agent |

## Pipeline Artifacts

| Command              | Purpose                                                 |
| -------------------- | ------------------------------------------------------- |
| `/execute-plan`      | Execute all tasks in the active plan (parallel capable) |
| `/execute-via-nexus` | Push entire plan to nexus daemon for remote execution   |
| `/push-to-nexus`     | Push tasks from project to nexus daemon                 |
| `/sync-from-nexus`   | Poll nexus daemon for completed tasks                   |
| `/archive-plan`      | Archive a completed plan                                |

## Project Bootstrap

| Command            | Purpose                                       |
| ------------------ | --------------------------------------------- |
| `/dogfood-plan002` | Submit PLAN-002 implementation tasks to nexus |

---

## Quality Pipeline Order

```
/new-task → /validate → /execute-task (→ /review → /test → /learn)
```

For the full queue unattended:

```
/orchestrator
```

---

## Agents (`.claude/agents/`)

Specialist agents invoked by `/execute-task` and `/hire-agent`:

| Agent                               | Best for                                       |
| ----------------------------------- | ---------------------------------------------- |
| `agent-employer`                    | Task routing and multi-agent wave planning     |
| `engineering-go-specialist`         | Go hexagonal arch, CGO/sqlite, Cobra, chi      |
| `engineering-senior-developer`      | Cross-stack full-stack implementation          |
| `engineering-devops-automator`      | CI/CD, GitHub Actions, release pipelines       |
| `engineering-backend-architect`     | API design, DB schema, Go service architecture |
| `engineering-software-architect`    | System design, ADRs, domain modeling           |
| `engineering-code-reviewer`         | PR reviews, code quality                       |
| `engineering-security-engineer`     | Threat modeling, secure code review            |
| `engineering-sre`                   | SLOs, observability, incident response         |
| `engineering-git-workflow-master`   | Branch strategy, conventional commits          |
| `engineering-frontend-developer`    | Vue 3, Pinia, Tailwind                         |
| `engineering-typescript-specialist` | TypeScript libs, Node.js, Zod                  |
| `design-ui-designer`                | Tailwind, dark mode, accessible components     |
| `design-ux-architect`               | User flows, information architecture           |
| `project-manager-senior`            | Planning, 3+ agent coordination                |
| `testing-qa-engineer`               | Test strategy, coverage, CI verification       |

---

## Follow-Up Queue (`.build/followup_queue.json`)

The pipeline accumulates findings during execution (tech debt, VERIFY items, ideas).
Items are surfaced after each task completes. VERIFY/high items block pre-merge.
