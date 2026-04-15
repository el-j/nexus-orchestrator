# /state -- Project Status and Process Monitoring

Shows the current state of all running processes, tasks, and branches at a glance.
Read-only -- changes no files.

---

## Section 1: Task Overview

Read `.claude/orchestrator.json`. Summarize `tasks` and `plans` sections.

Output table grouped by status:

```
=== Task Status ===

In Progress ({N}):
  {ID}  {Title}  Branch: {branch if known}  Plan: {planId}

Todo ({N}):
  {ID}  {Title}  Deps: {ok/blocked by X}

Blocked ({N}):
  {ID}  {Title}  Reason: {deps}

Done ({N} total, last 5):
  {ID}  {Title}

Backlog: {N}  |  Skipped: {N}  |  Failed: {N}
```

For each "todo" task: run dep check (are all deps "done"?).
For each "in-progress" task: show planId and last updated time.

---

## Section 2: Plan Status

Read `plans` block from `.claude/orchestrator.json`:

- Active plan ID and goal
- Status: active / completed
- Task completion: {done}/{total} tasks

---

## Section 3: Orchestrator Build Health

Run inline:

```bash
go vet ./... 2>&1 | tail -5
CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/... 2>&1 | tail -5
```

Report: OK or BROKEN (with first error line).

---

## Section 4: Open Branches

List open feature/task branches:
`git branch --list "feat/*" "fix/*" "task/*" "copilot/*"`

Per branch:

- Last commit: `git log -1 --format="%h %s (%cr)" {branch}`
- Merged to main: yes/no

---

## Output Format

All as compact, structured output:

```
========================================
  nexusOrchestrator -- Status
========================================

Active Plan:    {PLAN-NNN or none}
Build Health:   OK | BROKEN

--- Tasks ---
In Progress:    {N}
Todo:           {N}  ({IDs})
Blocked:        {N}
Done:           {N}
Total:          {N}

--- Active Plan Progress ---
  {PLAN-NNN}: {done}/{total} tasks done

--- Open Branches ---
  feat/TASK-XXX_name  (not merged, 2h ago)

========================================
```

---

## Rules

- No sub-agent needed (everything directly in main agent)
- Read-only -- change no files
- Handle missing files/directories gracefully (don't abort)
- Compact output, no prose
- State path: `.claude/orchestrator.json`

## Learnings

(populated by /learn after task cycles)
