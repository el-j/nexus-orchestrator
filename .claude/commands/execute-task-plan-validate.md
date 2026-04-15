# Phase 1b: Plan Validation Sub-Agent (Full Check)

This file is loaded by `/execute-task` for the **first-attempt full plan validation** only.
On retry (`IS_RETRY == true`) a lightweight pitfall check is run inline — this file is not needed.

---

Most critical quality gate in the workflow. Checks whether the task plan
still matches the current codebase BEFORE implementation starts.

Sub-Agent (general-purpose):

> Working directory: {WORKTREE_PATH}
> Read task plan: `{TASK_FILE}`
> Read CLAUDE.md for project conventions.
>
> ## Step 1: Build Plan Inventory
>
> Extract ALL concrete references from the task plan:
>
> - File paths, function/method names + expected signatures
> - Interface/type definitions, import paths
> - Store fields, API endpoints, DB models
> - External dependencies
>
> ## Step 2: Code Reality Check (8 Checks)
>
> ### Check 1: Files & Paths
>
> For EACH referenced file: does it exist? Renamed/moved?
>
> ### Check 2: Interfaces & Signatures
>
> For EACH referenced function/interface: does the signature match?
>
> ### Check 3: Recently Merged Changes
>
> `git log --oneline --since="$(git log -1 --format=%ci {TASK_FILE})" {BASE_BRANCH}`
> Overlap with plan files?
>
> ### Check 4: Acceptance Criteria Testability
>
> For EACH AC: concrete enough for an automated test?
>
> ### Check 5: Scope & Side Effects
>
> Circular imports? Unexpected test breakage?
>
> ### Check 6: Production Readiness
>
> Error handling, loading/empty states, validation, concurrent access,
> resource cleanup, logging, backwards compatibility?
>
> ### Check 7: Edge Cases
>
> Min. 3 edge cases documented? Data, concurrency, system, state edge cases?
>
> ### Check 8: Test Specification for Black-Box Agent
>
> Public interface, expected behavior, edge case behavior,
> testable preconditions, scope boundaries?
>
> ## Step 3: Classify Findings
>
> INFO | MINOR | MAJOR | BLOCKER
>
> ## Step 4: Update Plan (for MINOR/MAJOR)
>
> Update affected sections, add Plan-Revision section with findings table.
> Git commit: `docs({ID}): revise plan -- {N} findings ({severity_summary})`
>
> ## Step 5: Risk Assessment
>
> low (0 findings or INFO only) | medium (MINOR, no MAJOR) |
> high (MAJOR, ACs changed) | blocker (BLOCKER)
>
> ## Result (9 lines):
>
> PLAN_STATUS: valid|revised|blocked
> FINDINGS: {total} ({info} INFO, {minor} MINOR, {major} MAJOR, {blocker} BLOCKER)
> FILES_CHECKED: {count}
> CRITERIA_VALID: {valid}/{total} acceptance criteria
> EDGE_CASES: {count} defined (min. 3 required)
> PROD_READY: {open} open / {total} checked
> TEST_SPEC: complete|added|incomplete
> REVISION_SUMMARY: [1 sentence]
> RISK: low|medium|high|blocker

### Result Handling (caller)

| PLAN_STATUS | RISK    | Action                                               |
| ----------- | ------- | ---------------------------------------------------- |
| valid       | low     | Continue to Phase 1\*                                |
| revised     | medium  | Continue, store REVISION_SUMMARY for Phase 2         |
| revised     | high    | Continue automatically, findings documented          |
| blocked     | blocker | state.json status -> "blocked". Task back to `/task` |
