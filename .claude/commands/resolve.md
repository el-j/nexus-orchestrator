# /resolve -- Detect Merge Conflicts and Create Resolution Tasks

Finds branches with merge conflicts, analyzes the cause, and creates a new task
describing the conflict. The actual resolution is handled by the normal pipeline
(`/execute-task`).

**Optional parameter:** $ARGUMENTS (Task-ID, e.g. `TASK-001`). Without parameter: all unmerged branches.

---

## Phase 1: Find Unmerged Branches

If task ID given:

- Read branch from `git branch --list` matching the task ID pattern
- Check state in `.claude/orchestrator.json` for `tasks.<ID>.status`

If no task ID:

- List all branches: `git branch --list "feat/*" "fix/*" "task/*" "copilot/*"`
- Cross-reference with `.claude/orchestrator.json` to find which task each belongs to
- Display as table (ID, title, branch)
- If none found: "No open branches." -> done

---

## Phase 2: Conflict Analysis (per branch)

1. **Dry-run merge:** `git merge --no-commit --no-ff {BRANCH_NAME}`
   - No conflict: `git merge --abort`. Report: "Branch can be cleanly merged!"
     - Merge directly: `git merge --no-ff {BRANCH_NAME} -m "merge({ID}): {title}"`
     - Update `.claude/orchestrator.json`: task `status -> "done"`
     - No new task needed -> next branch or done
   - Conflict: continue with step 2

2. **Collect conflicts:** `git diff --name-only --diff-filter=U`
3. **Collect context:**
   - `git log --oneline {BASE}..{BRANCH_NAME}` -- branch commits
   - `git log --oneline {BRANCH_NAME}..{BASE}` -- base commits since fork
4. **Abort merge:** `git merge --abort`

---

## Phase 2b: Duplicate Check

Before creating a new task:

- Search `.claude/orchestrator.json` `tasks` for existing conflict-resolution task for this branch
- If found AND status "todo"/"in-progress": NO new task, just report
- If found AND status "done" but conflict still present: proceed with new task

---

## Phase 3: Create Resolution Task

1. Run `/new-task` with description:
   `"Resolve merge conflict for branch {BRANCH_NAME} -- conflicts in: {CONFLICT_FILES}"`
2. The new task file will be created at `.claude/tasks/TASK-<N>.md`
3. Commit: `docs(TASK-{NNN}): add conflict resolution task for {ORIGINAL_ID}`

---

## Phase 4: Summary

```
=== /resolve Result ===

Cleanly merged:    {N} branches
Tasks created:     {N} tasks

New tasks:
  TASK-{NNN}: Resolve merge conflict {BRANCH} ({N} files)

Next step: /execute-task TASK-{NNN}
```

---

## Rules

- Main = dispatcher: detects + documents conflicts, does NOT resolve them itself
- Cleanly mergeable branches are merged directly
- Real conflicts -> new task -> pipeline resolves them
- **NO `git checkout` in main agent** -- main stays on base branch
- State path: `.claude/orchestrator.json`

## Learnings

- Git rebase after revert is destructive: if the base branch contains a revert of
  the branch commits, `git rebase` silently drops the original commits. Instead of
  rebase, changes must be re-implemented or cherry-picked.
