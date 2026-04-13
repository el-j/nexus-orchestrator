# /orchestrator -- Autonomous Task Pipeline

**Arguments:** $ARGUMENTS
(e.g. "only TASK-003", "from TASK-002", or empty for full queue)

## Principles

1. Full autonomy -- no user intervention, queue is processed completely
2. Self-healing -- 3 attempts per problem, then skip + document
3. Resume-capable -- `.claude/orchestrator.json` is the runtime database
4. **Delegation** -- task execution via `/execute-task` (no duplicated code)
5. **Lean main context** -- Main = queue manager, `/execute-task` = worker
6. **Branch isolation** -- each task on its own branch (managed by /execute-task)
7. **Sequential execution** -- always 1 task at a time, no parallelism

---

## Architecture

```
/orchestrator (Main = Queue Manager)
  |
  |-- Phase 0: Init / Resume
  |-- Phase 1: Queue Build (Sub-Agent)
  |
  |-- Phase 2: Task Execution (Loop over queue)
  |     |
  |     |-- 2.1: Take task from queue
  |     |-- 2.2: Run /execute-task as Sub-Agent  <-- DELEGATION
  |     |       (Branch, Worktree, Impl, Test, Learn, Merge)
  |     |-- 2.3: Process result + validation + retries
  |     |-- 2.4: Deps check (unblocked tasks)
  |     +-- 2.5: Post-merge validation (full test suite on base branch)
  |
  +-- Phase 3: Completion (consistency check, report)
```

---

## Context Rules

| Rule                      | Details                                                                              |
| ------------------------- | ------------------------------------------------------------------------------------ |
| Main reads                | ONLY `.claude/orchestrator.json` + `.build/followup_queue.json` (after pipeline run) |
| Sub-agent prompts         | Max 10 lines with all required paths                                                 |
| Sub-agent results         | Max 5 lines: STATUS + MERGED + TESTS + LINT + SUMMARY                                |
| orchestrator.json updates | Main: queue/active/history. /execute-task: phase/status/branch/merged                |
| .build/                   | Runtime artifacts (Follow-Up Queue). Gitignored. Lives in repo root                  |

---

## State Schema

`.claude/orchestrator.json`

```
tasks.{ID}.status: "todo"|"in-progress"|"done"|"blocked"|"skipped"
tasks.{ID}.planId: "{PLAN-NNN}" or null
tasks.{ID}.branch: null | "task/TASK-NNN_name"
tasks.{ID}.merged: false | true
orchestrator.status: "idle"|"running"|"paused"|"completed"|"error"
orchestrator.phase: null|"init"|"executing"|"completed"
orchestrator.base_branch: "main"
queue: [IDs]
active: [IDs]
errors: [{id, phase, error, category, attempts: [{attempt, phase, error, timestamp}]}]
```

---

## Phase 0: Init / Resume

Read `.claude/orchestrator.json`.

- **running**: RESUME. Active tasks without "done" back to queue. Done tasks out of queue.
- **paused**: Check blockers. Resolved -> continue.
- **idle/completed**: Fresh start -> Phase 1.
- **error**: Recovery. "done" -> history. Rest -> queue.

**Queue Hygiene (on EVERY start/resume):**
Filter queue: only tasks with status "todo" or "in-progress" remain.
Remove tasks with "done", "blocked", "skipped".

State: orchestrator.status -> "running", base_branch -> "main".

---

## Phase 1: Queue Build (Sub-Agent)

Sub-Agent (Explore, model: Claude Haiku 4.5):

> Read `{ABS_PATH}/.claude/orchestrator.json`.
> Topological sort: tasks without open deps first.
> Filter: {ARGUMENTS if provided}. Include tasks with status "todo" or "in-progress".
> ("in-progress" = retry or crash recovery with existing branch)
> Sequential order: respect dependencies, one task at a time.
> Result (ONLY 1 line):
> QUEUE: [TASK-001, TASK-002, ...]

Main: write queue + orchestrator.phase "executing" to orchestrator.json.

---

## Phase 2: Task Execution (Loop)

### 2.1 Prepare Task (Main)

Take task ID from queue, move to `active`, status -> "in-progress".

### 2.2 Delegate to /execute-task (Sub-Agent)

> Read and follow `{ABS_PATH}/.claude/commands/execute-task.md`.
> Task-ID: {ID}
> Working directory: {ABS_PATH}
> `cd {ABS_PATH}` as first bash command.
> Execute the COMPLETE /execute-task workflow (pre-flight through Phase 7).
>
> Result (ONLY 5 lines):
> STATUS: done|blocked|failed
> MERGED: true|false
> TESTS: {new} new, {total} total, {passed} passed, {failed} failed
> LINT: OK|FAILED ({N} errors)
> SUMMARY: [1 sentence]

### 2.3 Process Result (Main)

**Validation:**

1. STATUS != "done" -> blocked/failed handling
2. TESTS: `{failed} > 0` -> rejected
3. LINT: "FAILED" -> rejected

**On success:** active -> history with `{id, result: "done", timestamp}`
**On failure:** -> Phase 2.3b (fix analysis)

### 2.3b Fix Analysis (Sub-Agent)

Triggered when:

- `/execute-task` returns STATUS blocked/failed

**Prerequisite:** `attempts.length < 2` (otherwise -> skip + document)

#### Step 1: Document attempt in errors[]

```json
{
  "id": "TASK-003",
  "phase": "post-merge-validation",
  "error": "3 test failures",
  "category": null,
  "attempts": [{ "attempt": 1, "phase": "implement", "error": "go vet failed", "timestamp": "..." }]
}
```

#### Step 2: Start analysis agent

Sub-Agent (general-purpose, model: Claude Sonnet 4.6):

> Analyze the failure of task {ID} and update the plan.
> `cd {ABS_PATH}`
>
> ## Context
>
> - Task file: `{TASK_FILE}`
> - Failed phase: {PHASE}
> - Error details: {ERROR}
> - Previous attempts: {ATTEMPTS_JSON}
>
> ## Step 1: Error Analysis
>
> 1. Read error details and identify root cause
> 2. If branch exists: check `git log --oneline {BRANCH_NAME}` and `git diff {BASE}...{BRANCH_NAME}`
>
> ## Step 2: Classify Error
>
> | Category     | Criteria                                   |
> | ------------ | ------------------------------------------ |
> | `fixable`    | Clear root cause, plan adjustment suffices |
> | `structural` | Design error, architecture mismatch        |
>
> Loop detection: If attempt N has same error as N-1 -> `structural`
>
> ## Step 3: Update Task File (only for fixable)
>
> Add/update `## Pitfalls` section. Do NOT change acceptance criteria.
>
> ## Result (ONLY 4 lines):
>
> CATEGORY: fixable|structural
> ROOT_CAUSE: [1 sentence]
> PITFALLS_ADDED: {count}
> PLAN_UPDATED: true|false

#### Step 3: Process Result (Main)

| CATEGORY     | Action                                                         |
| ------------ | -------------------------------------------------------------- |
| `fixable`    | task status -> "in-progress", back in queue front, keep branch |
| `structural` | task status -> "skipped", document reason                      |

### 2.4 Deps Check

Move unblocked tasks (all deps "done") into queue.

### 2.5 Post-Merge Validation

After EVERY successful merge:

**Inline (Main -- no sub-agent). Fail-fast order:**

```bash
cd {ABS_PATH}
go vet ./... 2>&1
CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/... 2>&1
CGO_ENABLED=1 go test -race -count=1 ./... 2>&1
```

- **OK** -> next task
- **FAILED** -> Revert + fix analysis:
  1. Identify last merge commit: `git log -1 --format="%H %s" --merges`
  2. `git revert -m 1 {MERGE_COMMIT_HASH} --no-edit`
  3. orchestrator.json: task merged -> false
  4. -> Phase 2.3b (fix analysis with phase "post-merge-validation")

---

## Phase 3: Completion

Sub-Agent (Explore, model: Claude Haiku 4.5): final consistency check for orchestrator.json.

Main: orchestrator.status -> "completed".

### 3a: Task Report

```
=== ORCHESTRATOR REPORT ===
DONE: {n} tasks
SKIPPED: {n} tasks (list IDs + reason)
BLOCKED: {n} tasks (list IDs + blocker)
UNMERGED BRANCHES: {list or "none"}
```

### 3b: Follow-Up Summary

Read `.build/followup_queue.json`. If items exist, present grouped by category:

```
=== FOLLOW-UP SUMMARY ===

VERIFY ({n} items):
  - [high] {description} (source: {task_id}, agent: {source_agent})

REFAC ({n} items):
  - {description} (source: {task_id})

IDEA ({n} items):
  - {description} (source: {task_id})
```

---

## Error Handling

| Level    | Description                                                           |
| -------- | --------------------------------------------------------------------- |
| 1        | /execute-task fixes inline (build/test fix loop)                      |
| 1b       | Main checks TESTS/LINT fields                                         |
| 2        | **Fix analysis agent** (Phase 2.3b): classify, update task + pitfalls |
| 2b       | On `fixable`: re-enqueue at queue position 0, max 3 total attempts    |
| 2c       | On `structural`: immediate skip                                       |
| 3        | After 3 attempts: skip + document                                     |
| 4        | Post-merge validation: FAILED -> revert -> fix analysis               |
| Critical | >50% blocked -> pause, output report                                  |

---

## Context Budget

- Sub-agent prompts: max 10 lines
- Sub-agent results: max 5 lines
- Main NEVER reads code files
- Main reads ONLY `.claude/orchestrator.json` + `.build/followup_queue.json`
- Always sequential: 1 task at a time

---

## RULES

- Main = queue manager. /execute-task = task worker. Strict separation.
- **NO `git checkout` in the main agent** -- Main ALWAYS stays on the base branch
- /execute-task handles: branch, worktree, impl, test, learn, merge, docs
- Results: structured, max 5 lines, no prose
- On merge conflict: /execute-task sets merged:false, recommend `/resolve {ID}`
- Fully autonomous -- no human-in-the-loop, process queue completely
- On >50% blocked: output report and pause

## Learnings

- Sequential execution is safer than parallel. No shared-file conflicts,
  simpler debugging, clearer merge history.
- > 50% blocked tasks indicate a systematic problem. Pause and output report.
- Post-merge validation with auto-revert prevents broken base branch.
  On FAILED: revert + fix analysis + re-enqueue (max 3 attempts), then skip.
- Queue hygiene on every start: filter out done/skipped tasks.
  Prevents re-execution and unapproved task execution.
- Fix analysis agent with pitfalls section: analyze errors, classify
  (fixable/structural), update task, re-enqueue. Max 3 attempts.
- Structural errors should be skipped immediately: design errors need manual replanning.
- Inline instead of sub-agent for simple phases: post-merge validation doesn't
  need a sub-agent. Saves context overhead + agent spawning.
- Transient build failures vs code failures: If attempt N fails at validate
  but attempt N+1 with identical code passes immediately, it was infrastructure
  (cache, timeout). No pitfalls analysis needed, direct re-enqueue.
