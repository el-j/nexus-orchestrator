# /execute-task -- Fully Automated Task Execution

Fully automated execution of a single task with branch isolation, plan validation,
AC-gate, black-box testing, and learning.

**Task-ID:** $ARGUMENTS

---

## Phase Tracking

Update the phase in `.claude/orchestrator.json` at EVERY phase transition.

```bash
jq --arg id "{ID}" '.tasks[$id].phase = "{phase}"' .claude/orchestrator.json > .claude/orchestrator.json.tmp \
  && mv .claude/orchestrator.json.tmp .claude/orchestrator.json
```

Phases: `preflight`, `branch`, `plan_validate`, `implement`, `ac_gate`, `review`, `test`, `validate`, `merge`, `cleanup`, `docs`, `learn`

---

## Phase 0: Pre-Flight (Main)

**Phase update:** `preflight`

1. **Validate ID:** Format `^TASK-\d+$`
2. **Task file:** Glob `.claude/tasks/{ID}.md`. Store absolute path as `TASK_FILE`.
3. **Check status:** "todo" or "in-progress" (retry/crash recovery) allowed. Others: stop.
4. **Check deps:** `.claude/orchestrator.json` -> referenced deps must be "done".
5. **Base branch:** `git rev-parse --abbrev-ref HEAD` -> `BASE_BRANCH`
6. **MCP check:** Try connecting to nexus MCP at `http://127.0.0.1:63988/mcp`.
   - If available: call `howto_brief`, register session via `register_session`.
   - If unavailable: proceed with local execution only.
7. **Baseline build:**
   ```bash
   go vet ./... 2>&1
   CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/... 2>&1
   ```
   If this fails, report errors and stop — do not execute task on broken baseline.
8. **Role detection:** Read `role:` field from task file. Store as `TASK_ROLE`.

### Role → Agent Routing Table

```
backend / api / cli / mcp / devops  --> engineering-senior-developer or engineering-go-specialist
architecture                         --> engineering-software-architect
qa                                   --> testing-qa-engineer
verify                               --> engineering-code-reviewer
planning                             --> project-manager-senior
```

Agent files at: `.claude/agents/{agent-name}.md`
Fallback: `.github/agents/{agent-name}.agent.md`

---

## Phase 1: Branch + Worktree (Main)

**Phase update:** `branch`

### 1a: Branch

```bash
BRANCH_NAME="task/{ID}_{short_name}"
```

**If branch already exists (retry/crash recovery):**

```bash
mkdir -p .claude/worktrees
if ! git worktree list | grep -q ".claude/worktrees/{ID}"; then
    git worktree add .claude/worktrees/{ID} {BRANCH_NAME}
fi
cd .claude/worktrees/{ID}
# IMPORTANT: No rebase after revert! Use merge to preserve commits.
git merge {BASE_BRANCH} --no-edit
```

Set `IS_RETRY = true`. Inventory existing changes:

```bash
EXISTING_DIFF=$(git diff --stat {BASE_BRANCH}...HEAD)
EXISTING_LOG=$(git log --oneline {BASE_BRANCH}..HEAD)
```

**If new:**

```bash
git branch {BRANCH_NAME} {BASE_BRANCH}
mkdir -p .claude/worktrees
git worktree add .claude/worktrees/{ID} {BRANCH_NAME}
```

Set `IS_RETRY = false`.
State: branch -> `{BRANCH_NAME}`, status -> "in-progress"

### 1b: Worktree Working Directory

All subsequent phases work in: `.claude/worktrees/{ID}/`

---

## Phase 1b: Plan Validation (Sub-Agent)

**Phase update:** `plan_validate`

**On retry (IS_RETRY == true): LIGHTWEIGHT validation.**

Sub-Agent (model: Claude Haiku 4.5):

> Check pitfalls section is present and coherent.
> Verify pitfall solutions are consistent with implementation steps.
> Result (3 lines):
> PLAN_STATUS: valid|revised
> PITFALLS_COHERENT: true|false
> REVISION_SUMMARY: [1 sentence]

**On first attempt (IS_RETRY == false): FULL validation.**

Sub-Agent (general-purpose, model: Claude Opus 4.6):

> Read and follow `{ABS_PATH}/.claude/commands/execute-task-plan-validate.md`.
> Task plan: {TASK_FILE}
> Base branch: {BASE_BRANCH}
> Working directory: {WORKTREE_PATH}
> Task-ID: {ID}

Return the 9-line result defined in `execute-task-plan-validate.md`.

### Result Handling

| PLAN_STATUS | RISK    | Action                                       |
| ----------- | ------- | -------------------------------------------- |
| valid       | low     | Continue to Phase 2                          |
| revised     | medium  | Continue, store REVISION_SUMMARY for Phase 2 |
| blocked     | blocker | status -> "blocked". Stop.                   |

---

## Phase 2: Implementation

**Phase update:** `implement`

### 2a: Quick Plan Check (Inline)

Read task file. Check:

- Acceptance criteria present and concrete?
- If retry: pitfalls section present? **Apply pitfalls!**

### 2b: Implementation Sub-Agent

**On first attempt (IS_RETRY == false):**

Sub-Agent (model: Claude Sonnet 4.6) using `TASK_ROLE`-matched agent:

> You are implementing task {ID}.
> `cd {WORKTREE_PATH}`
>
> ## Task
>
> {TASK_FILE content -- context, steps, acceptance criteria}
>
> ## nexusOrchestrator Architecture Rules
>
> - Module: `nexus-orchestrator`, Go 1.26, `CGO_ENABLED=1` for sqlite3
> - Hexagonal: core never imports adapters
> - Error wrapping: `fmt.Errorf("package: operation: %w", err)`
> - Concurrency: `sync.Mutex` for shared state; no goroutines in `internal/core/services/`
> - `domain.ErrNotFound` sentinel for missing entities
> - HTTP API: chi router on `:63987`; MCP: JSON-RPC 2.0 on `:63988`
> - Tests: `CGO_ENABLED=1 go test -race -count=1 ./...`
> - If MCP is reachable: call `howto_brief`, `register_session`, `heartbeat_task`
>
> ## Rules
>
> - Implement ALL acceptance criteria
> - Handle ALL edge cases from the task
> - Commit after implementation: `feat({ID}): {summary}`
>
> Result (ONLY 5 lines):
> STATUS: done|blocked|failed
> FILES: [created/changed files]
> TESTS: n/a (written by /test)
> LINT: OK|FAILED ({N} errors)
> SUMMARY: [1 sentence]

**On retry (IS_RETRY == true):**

Sub-Agent (model: Claude Sonnet 4.6):

> **RETRY** of task {ID}. Read PITFALLS section first.
> Existing work on branch:
>
> ```
> {EXISTING_LOG}
> {EXISTING_DIFF}
> ```
>
> Fix targeted -- do NOT rewrite everything.
> Commit: `fix({ID}): address pitfalls from attempt {N}`

---

## Phase 2.5: AC-Checklist Gate (Main -- inline)

**Position:** After implementation, before Review.

Verifies every acceptance criterion has corresponding implementation.

**Flow:**

1. Extract all acceptance criteria from task file (lines with `- [ ]` or `- [x]`)
2. Read `git diff {BASE_BRANCH}...HEAD` in worktree
3. For EACH AC individually: is there corresponding code in the diff?

**Output format:**

```
AC-CHECKLIST:
AC-1: PASS -- Implemented in internal/core/services/orchestrator.go:42
AC-2: FAIL -- Missing: go vet must pass but vet errors remain
RESULT: 2/3 ACs covered -- BLOCKED
```

**Rules:**

- Each AC checked INDIVIDUALLY
- On FAIL: back to Phase 2 implementation
- No proceeding to Phase 2b while any AC is FAIL

**Follow-Up Queue:** Out-of-scope findings during AC check go to `.build/followup_queue.json`.

---

## Phase 2b: Rebase

**On retry (IS_RETRY == true): SKIP.** (Base already integrated in Phase 1 via merge)

**On first attempt:**

```bash
cd {WORKTREE_PATH}
git fetch origin {BASE_BRANCH}
git rebase origin/{BASE_BRANCH}
```

On conflict: STATUS: blocked, recommend `/resolve {ID}`.

---

## Phase 2c: Code Review

**Phase update:** `review`

Sub-Agent (model: Claude Sonnet 4.6):

> Read and follow `{ABS_PATH}/.claude/commands/review.md`.
> Task plan: {TASK_FILE}
> Working directory: {WORKTREE_PATH}
>
> After review, write a compact implementation summary to
> `{ABS_PATH}/.build/impl-summary-{ID}.md` (max 5 bullet points: what was added/changed).
>
> Result (ONLY 5 lines):
> STATUS: pass|warn|fail
> FINDINGS: {n} total ({critical} critical, {warnings} warnings)
> CRITICAL: [list of critical findings]
> CONVENTIONS: {OK|{n} violations}
> SUMMARY: [1 sentence]

### Review Fix Loop (max 2 iterations)

If STATUS = "fail":

1. Fix critical findings (sub-agent with findings list)
2. Re-run /review
3. After 2 failed iterations: STATUS: blocked

---

## Phase 3: Black-Box Testing

**Phase update:** `test`

**On retry: check if tests already exist.**

```bash
git diff --name-only {BASE_BRANCH}...HEAD | grep "_test.go" | head -5
```

- Tests exist: run only, do NOT rewrite.
- No tests: full /test workflow.

Sub-Agent (model: Claude Sonnet 4.6):

> Read and follow `{ABS_PATH}/.claude/commands/test.md`.
> Task-ID: {ID}
> Task plan: {TASK_FILE}
> Working directory: {WORKTREE_PATH}
> Implementation context: read `{ABS_PATH}/.build/impl-summary-{ID}.md` if present.
> DO NOT read implementation code directly.
>
> Test framework: `CGO_ENABLED=1 go test -race -count=1 ./...`
>
> Result (ONLY 5 lines):
> STATUS: pass|fail
> TESTS_WRITTEN: {n}
> TESTS_PASSED: {n}/{m}
> EDGE_CASES: {tested}/{planned}
> SUMMARY: [1 sentence]

### Test Fix Loop (max 3 iterations)

If STATUS = "fail":

Sub-Agent:

> Read and follow `{ABS_PATH}/.claude/commands/testfix.md`.
> TASK_ID: {ID}
> TASK_PLAN: {TASK_FILE}
> WORKTREE: {WORKTREE_PATH}
> FAILED_TESTS: {failure output}
> ITERATION: {1|2|3}

After 3 failed iterations: STATUS: blocked.

---

## Phase 4: Pre-Merge Validation

**Phase update:** `validate`

**Inline (Main -- no sub-agent). Fail-fast order:**

```bash
cd {WORKTREE_PATH}
go vet ./... 2>&1
CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... ./cmd/nexus-submit/... 2>&1
CGO_ENABLED=1 go test -race -count=1 ./... 2>&1
```

### 4b: Follow-Up Queue VERIFY/high Gate

```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
if [ -f "$REPO_ROOT/.build/followup_queue.json" ]; then
  VERIFY_HIGH=$(jq '[.items[] | select(.category == "VERIFY" and .priority == "high")] | length' \
    "$REPO_ROOT/.build/followup_queue.json")
  if [ "$VERIFY_HIGH" -gt 0 ]; then
    echo "BLOCKED: $VERIFY_HIGH VERIFY/high items in Follow-Up Queue"
  fi
fi
```

All checks OK = continue to Phase 5, otherwise FAILED.

---

## Phase 5: Merge

**Phase update:** `merge`

### 5a: Merge to Base Branch

```bash
cd {ABS_PATH}   # Back to main repo
git merge --no-ff {BRANCH_NAME} -m "merge({ID}): {title}"
```

State: merged -> true

If MCP available: `update_task_status` to done.

### 5b: Worktree Cleanup

**Phase update:** `cleanup`

```bash
git worktree remove .claude/worktrees/{ID}
```

**Cleanup on failure/blocked:**

1. `git worktree remove --force .claude/worktrees/{ID}`
2. Branch is NOT deleted (kept for debugging/retry)
3. orchestrator.json: status -> "blocked"/"failed", phase -> null

---

## Phase 6: Documentation Update

**Prerequisite:** Only if `merged: true`.

**Phase update:** `docs`

Inline:

1. orchestrator.json: status -> "done", phase -> null, `completedAt = now`
2. Task file: set `status: done` in frontmatter
3. Git commit: `docs({ID}): mark task done`

### Follow-Up Queue Presentation

```bash
QUEUE_FILE=".build/followup_queue.json"
```

If queue has items, present grouped by category:

```
===================================================
  FOLLOW-UP QUEUE: {ID} -- {n} Items
===================================================
| # | Category | Priority | Title | Source |
|---|----------|----------|-------|--------|
```

Queue archived in orchestrator.json under task entry after Phase 6.

---

## Phase 7: Learning

**Phase update:** `learn`

Sub-Agent (model: Claude Haiku 4.5):

> Read and follow `{ABS_PATH}/.claude/commands/learn.md`.
> Task-ID: {ID}
> Task plan: {TASK_FILE}
> Review result: {REVIEW_RESULT}
> Test result: {TEST_RESULT}
> Retry info: {IS_RETRY}, Attempt: {N}
> Pitfalls: {pitfalls section from task file, if present}

---

## Follow-Up Queue Format

**Write from any agent:**

```bash
QUEUE_FILE=".build/followup_queue.json"
mkdir -p .build
# Read if exists, else initialize with {"source_task":"{ID}","items":[]}
# Append item, write back
```

**Item format:**

```json
{
  "id": "FQ-001",
  "category": "VERIFY|REFAC|IDEA",
  "source_agent": "review|test|testfix|ac_gate|validate",
  "title": "{short title}",
  "description": "{description}",
  "affected_files": ["path/to/file.go"],
  "priority": "high|medium|low"
}
```

**Rules:**

1. Append-only during pipeline -- no agent deletes others' entries
2. VERIFY with priority:high blocks pre-merge (Phase 4b)
3. Max 10 entries per pipeline run
4. Cleared on task retry

---

## Result Format

```
STATUS: done|blocked|failed
MERGED: true|false
TESTS: {new} new, {total} total, {passed} passed, {failed} failed
LINT: OK|FAILED ({N} errors)
SUMMARY: [1 sentence]
```

---

## Context Rules

- Main reads ONLY: task file (status), orchestrator.json (deps), `.build/followup_queue.json`
- Main NEVER reads: Go source files, project structure
- Sub-agent results: max 5 lines, no prose
- **NO `git checkout` in the main agent** -- Main stays on BASE_BRANCH
- `.build/` lives in repo root (NOT in worktree)

---

## Core Architecture Rules (enforce in every task)

- `internal/core/domain/` — pure domain types, no framework imports
- `internal/core/ports/` — Go interfaces only, no concrete implementations
- `internal/core/services/` — business logic, depends only on ports
- `internal/adapters/inbound/` — driven by CLI, HTTP, MCP, Wails
- `internal/adapters/outbound/` — LM Studio, Ollama, SQLite, filesystem
- Dependency direction: inbound → services → ports ← outbound

## Learnings

- Worktree isolation prevents conflicts between tasks and keeps base branch clean.
- Separating implementation and testing (different agents) catches more bugs because
  the test agent has no implementation bias.
- Review before test catches structural issues early, saving test iterations.
- Pitfalls section in retry plans prevents repeating the same mistakes.
- AC-Checklist Gate (Phase 2.5) catches forgotten ACs before Review/Test.
  Without it, ACs are discovered in test or post-merge, wasting iterations.
- Git rebase after revert is destructive: if base branch contains a revert of
  branch commits, `git rebase` silently drops the original commits. Use merge instead.
- Atomic orchestrator.json updates prevent corruption. Use temp-file-then-rename pattern.
- MCP availability check at preflight prevents dead code paths through the whole execution.
