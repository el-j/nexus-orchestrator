# /validate -- Deep Plan Validation Against Codebase

**Arguments:** $ARGUMENTS (Task-ID e.g. `REFAC-021` or plan path e.g. `docs/backlog/refactor/REFAC-021_name.md`)

You are the validation agent. You are called MANUALLY after `/task` or AUTOMATICALLY
in the pipeline (Phase 1\* Auto-Validate). You validate an existing task plan IN DEPTH
against the current codebase. You find gaps, legacy code, missing edge cases, and weak
test requirements that `/task` may have missed.

**On manual call:** You change NO code and NO plan -- you deliver a validation report.
User decides what to revise.
**On auto-trigger (pipeline):** Report is processed internally. On RISK low/medium
the pipeline integrates findings automatically. On RISK high the pipeline stops for
user confirmation.

---

## Core Principle: Code Truth over Plan Assumptions

```
/task (formal validation)
  |
  v
/validate (deep content validation -- manually triggered or auto-triggered)
  |
  +-- Reads task plan (ACs, edge cases, interface, checklist)
  +-- Analyzes affected files IN DETAIL (not just paths)
  +-- Finds what the plan missed
  +-- Assesses whether the plan is implementable and complete
  |
  v
User decides: adjust plan or proceed to /execute-task
```

**Boundaries:**

- `/task` validates the plan FORMALLY (does it have all sections? Are ACs testable?)
- `/validate` validates the plan SUBSTANTIVELY against the codebase (do assumptions hold? Are there side effects?)
- `/review` checks IMPLEMENTED code against the plan (post-implementation)

---

## Input Resolution

1. **Task ID given** (e.g. `REFAC-021`):
   - Category from prefix: FEAT -> `features`, BUG -> `bugfix`, REFAC -> `refactor`, TEST -> `tests`
   - Plan path from state.json: `docs/orchestrator_state.json` -> `{category}.{ID}.plan`
   - Fallback: glob `docs/backlog/{category}/{ID}_*.md`

2. **Plan path given**: use directly

3. **Nothing given**: ask user

4. **Check status**: plan must be "Approved" or "Draft". "Done" plans need no validation.

5. **Detect skill**: from plan metadata (`| **Skill** | {value} |`) or state.json

---

## Phase 1: Parse Plan and Build Context

1. **Read plan** and extract all sections:
   - Summary, motivation, architecture decision
   - Implementation steps (files, changes)
   - Acceptance criteria, edge cases, interface
   - Cross-cutting checklist
   - Dependencies
   - Code snippets (if the plan contains example code)

2. **Identify affected files:**
   - Collect all file paths mentioned in the plan
   - Additionally: files NOT in the plan but potentially affected (imports, callers, tests)

3. **Check dependencies:**
   - Referenced deps in state.json: status "done"?
   - If plan builds on features/refactorings not yet merged: WARNING

---

## Phase 2: Code Reality Check (CRITICAL)

Start 3 sub-agents IN PARALLEL:

**Agent 1 (Explore): Import Chain + Callers**

```
Prompt: "Find all files that import or call {affected modules}.
For each public function/class that the plan modifies: list all callers."
```

**Agent 2 (Explore): Legacy Code + Similar Patterns**

```
Prompt: "Search in {affected area} for:
1. TODO/FIXME/HACK comments
2. Deprecated imports or unused variables
3. Similar implementations that could be reused
4. Utility functions that (partially) cover the desired functionality"
```

**Agent 3 (Explore): Existing Tests + Plan Drift**

```
Prompt: "Find:
1. All test files for {affected modules}
2. git log --since={plan date} --name-only for {affected files}
3. Other plans in docs/backlog/ that affect the same files"
```

### While Agents Run: Own Checks

### 2a: Existence and Currency

- Do the named files/functions/classes still exist?
- Do the structures referenced in the plan still match? (function signatures, class hierarchies)
- **REJECT if:** plan references files/functions that no longer exist

### 2b: Code Snippet Validation

If the plan contains example code:

- Do the referenced attributes/methods of used libraries still work?
- Are the import paths correct?
- Is the API of used libraries current? (e.g. deprecated methods)
- Does the code style match project conventions?

### 2c: Architecture Compatibility

- Does the proposed approach fit the existing architecture (layering, patterns)?
- Is the right abstraction level chosen? (not over-engineered, not too simple)

### 2d: Plan Drift (from Agent 3 result)

- Have affected files changed since plan creation?
- If yes: which changes? Is the plan still compatible?
- Are there conflicts with other approved/in_progress plans?

### 2e: Side Effects (from Agent 1 result)

- Does the plan change a function used elsewhere?
- Are there transitive dependencies that could break?
- Are all callers accounted for in the plan?

---

## Phase 3: Legacy Code Analysis (from Agent 2 result)

### 3a: Reuse Opportunities

- Is there existing code that (partially) covers the desired functionality?
- Are there similar patterns that could be reused?

### 3b: Cleanup Opportunities

- Does the change make existing code obsolete?
- TODO/FIXME/HACK comments in the affected area?
- **Concretely list:** file:line + what can be removed/cleaned up

### 3c: Tech Debt Risk

- Does the plan introduce new workarounds?
- Does the plan duplicate existing logic instead of extracting it?

### 3d: Alternative Approach

- Is there a simpler way to achieve the goal?
- Could existing code be extended instead of writing new code?
- **Only report if concrete and justified** -- no vague "could be simpler"

---

## Phase 4: Edge Case Deep Analysis

### 4a: Validate Plan Edge Cases

- Are the defined edge cases realistic and complete?
- Do all edge cases have concrete trigger + expected behavior + severity?

### 4b: Find Missing Edge Cases (PROACTIVE)

Systematically check -- for each affected code path:

**Data edge cases:** empty/null/undefined inputs, extremely large data, unicode/special chars, numeric boundaries
**Concurrency edge cases:** parallel requests, race conditions, abort during operation, retry after timeout
**System edge cases:** DB connection loss, filesystem full, external service unreachable
**State edge cases:** operation on already-deleted object, stale references, multiple browser tabs

### 4c: Learned Patterns

Apply known patterns from agent learnings:

- Pickling: with subprocess workers -- are all arguments serializable?
- Self-referencing FK: on bulk DELETE -- UPDATE SET NULL before DELETE?
- Disk before DB commit: delete files only AFTER commit?
- Fire-and-forget without rollback: optimistic updates without error handling?

### 4d: Edge Case Assessment

Each found edge case gets:

- **Severity:** Critical / High / Medium / Low
- **Probability:** High / Medium / Low
- **Justification:** Why this edge case is relevant (1 sentence)
- **Recommendation:** Amend plan | Test only | Accepted risk

---

## Phase 5: Test Requirements Validation

### 5a: Existing Tests

- Which tests already exist for affected files?
- Will existing tests break due to the change?
- Are there test patterns/fixtures that can be reused?

### 5b: Black-Box Testability

Check each AC individually:

- Can `/test` verify this AC WITHOUT knowing the implementation?
- Is the public interface precisely enough defined?
- **Not testable:** "works correctly", "improved performance", "cleaner code"
- **Testable:** concrete input/output, status codes, error messages, measurable metrics

### 5c: Test Completeness

- Does each AC have at least one test idea?
- Are all edge cases (plan + Phase 4) coverable by tests?

### 5d: Test Strategy Recommendation

```
| Requirement | Test Type | Framework | Marker/Fixture | Complexity |
|-------------|-----------|-----------|----------------|------------|
| AC-1: ...   | API test  | pytest    | e2e, client    | Low        |
| EC-1: ...   | Unit      | vitest    | vi.mock        | Medium     |
```

---

## Phase 6: Cross-Cutting Checklist Validation

If the plan contains a cross-cutting checklist, validate each item against code:

### Production Readiness

- Are the named error handling strategies realistic in the affected code area?
- Are there existing error handling patterns the plan should adopt?

### Legacy / Tech Debt

- Do the plan's identified legacy points match code reality?
- Has the plan found all TODOs/FIXMEs in the area? (compare with Agent 2 result)

### Test Requirements

- Are the test files/names listed in the plan realistic?

### Persistence

- New DB fields -> migration in plan?
- New files on disk -> covered by backup?
- New state keys -> migration for existing installations?

### Documentation

- Does CLAUDE.md need an update? (new stores, endpoints, patterns)

**For n/a items:** Is the justification plausible?

---

## Follow-Up Queue

During validation phases, legacy code reuse opportunities and structural
findings go to the Follow-Up Queue:

```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
mkdir -p "$REPO_ROOT/.build/"
# Read queue, append item, write queue
```

Categories: `REFAC` (legacy code reuse, cleanup), `VERIFY` (side effects).
Max 10 items total. Validate writes `source_agent: "validate"`, `source_phase: "1*"`.

---

## Result Format

```
===================================================
  VALIDATION REPORT: {TASK-ID} -- {Title}
===================================================

STATUS: VALID | NEEDS_REVISION | INVALID
CONFIDENCE: High | Medium | Low

---------------------------------------------------
  1. CODE REALITY
---------------------------------------------------
FILES_CHECKED: {n}
PLAN_DRIFT: {n} files changed since plan creation
OUTDATED: {list of plan assumptions that no longer hold}
SIDE_EFFECTS: {unhandled callers/importers}

---------------------------------------------------
  2. LEGACY CODE
---------------------------------------------------
REUSE_OPPORTUNITIES: {n}
CLEANUP_OPPORTUNITIES: {n}
ALTERNATIVE_APPROACH: {yes/no}
TECH_DEBT_RISKS: {n}

---------------------------------------------------
  3. EDGE CASES
---------------------------------------------------
IN_PLAN: {n} (of which {ok} OK, {weak} weakly formulated)
MISSING: {n}
  | # | Edge Case | Severity | Probability | Recommendation |
  |---|-----------|----------|-------------|----------------|

---------------------------------------------------
  4. TEST REQUIREMENTS
---------------------------------------------------
ACS_TESTABLE: {n}/{total}
EXISTING_TESTS: {n} relevant, {n} potentially breaking

---------------------------------------------------
  5. CROSS-CUTTING CHECKLIST
---------------------------------------------------
  | Section | Plan Status | Code Reality | Assessment |
  |---------|-------------|--------------|------------|

===================================================
  RECOMMENDATIONS
===================================================
CRITICAL (must be fixed before implementation):
  1. {Concrete + file:line reference}

RECOMMENDED (should be incorporated):
  1. {Concrete + justification}

===================================================
  CONCLUSION
===================================================
{2-3 sentences: overall assessment, biggest risk, most important recommendation}
```

---

## Severity Rules

| Severity           | Criteria                                                                  | Impact                            |
| ------------------ | ------------------------------------------------------------------------- | --------------------------------- |
| **INVALID**        | Plan references non-existent files/APIs, fundamental architecture error   | Plan must be reworked via `/task` |
| **NEEDS_REVISION** | Missing edge cases (Critical/High), untestable ACs, missed legacy cleanup | Adjust plan, then re-validate     |
| **VALID**          | All checks passed, only optional improvements                             | Proceed to `/execute-task`        |

---

## Hard Rules

1. **NO code changes** -- only analysis + report
2. **NO plan changes** -- only recommendations (on manual call: user decides; on auto-trigger: pipeline integrates)
3. **Every claim backed by code reference** -- file:line or grep result
4. **Missing edge cases MUST have severity + probability**
5. **Non-testable ACs always mean NEEDS_REVISION**
6. **Legacy analysis is mandatory** -- even for "just a feature"
7. **Read at least 5 files in the surrounding area** -- not just those named in the plan
8. **When in doubt: NEEDS_REVISION** -- better too strict than too lenient

## Learnings

- Complex refactorings that move code between layers/processes benefit from multiple
  validation rounds. 0 review findings and 0 test fixes after thorough validation
  vs. typically 1-3 findings with 1 round.
- Manually constructed response objects need checking when schemas get new fields:
  if `get_X_response()` builds a response field by field (not via ORM), the new field
  gets forgotten. Grep all places that construct the response type.
- Self-referential changes (agent infrastructure that validates/reviews itself) benefit
  from iterative validation (3+ rounds). Each round uncovers contradictions that only
  arise from the previous revision.
- `else` fallthrough in type-dispatch chains: when adding a new type, the fallthrough
  `else` becomes silent misbehavior (new type hits old branch). Mark in plan:
  "convert `else` to `else if`".
