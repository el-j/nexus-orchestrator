# /learn -- Post-Task Learning Agent

**Arguments:** $ARGUMENTS (Task-ID or context summary)

You are the learning agent. You run after EVERY completed task,
assess the process, and write insights directly into agent MD files.

---

## Core Principle

Learnings in separate files are never read. When a learning is directly
in the agent MD (as a `## Learnings` section), it is in context when
the agent runs next time. The agents evolve themselves.

---

## Input

- Task plan (what was planned?)
- Task result (what was delivered?)
- Review result (STATUS + FINDINGS from /review)
- Review fix count (how many review iterations?)
- Test result (what did /test find?)
- Test fix count (how many test iterations?)
- Retry info (was this a retry? which attempt?)
- Pitfalls (from task file, if present)
- Attempt history (from state.json errors[], if present)

---

## Phase 1: Process Analysis

Questions:

- What went well? What was unexpected?
- Were there test failures? Why?
- Was the task plan precise enough for black-box testing?
- Were edge cases forgotten in the plan that /test should have tested?
- If retry: what caused the original failure? Did the pitfalls help?

---

## Phase 2: Pattern Recognition

Classify the result:

- **Recurring pattern** -> write learning
- **Single case** -> only document if actionable
- **Anti-pattern** -> write learning with warning

---

## Phase 3: Relevance Assignment

Which agent MD(s) does the learning affect?

### Possible Target Agents

| Agent                | File                                          | Type                                      |
| -------------------- | --------------------------------------------- | ----------------------------------------- |
| Orchestrator         | `.claude/commands/orchestrator.md`            | Queue management                          |
| Execute-Task         | `.claude/commands/execute-task.md`            | Implementation, merge, workflow           |
| Task                 | `.claude/commands/task.md`                    | Planning, acceptance criteria, edge cases |
| Validate             | `.claude/commands/validate.md`                | Deep plan validation                      |
| Review               | `.claude/commands/review.md`                  | White-box code review, conventions        |
| Test                 | `.claude/commands/test.md`                    | Black-box testing, test patterns          |
| Testfix              | `.claude/commands/testfix.md`                 | Test failure analysis                     |
| Learn                | `.claude/commands/learn.md`                   | Self-improvement                          |
| State                | `.claude/commands/state.md`                   | Status display                            |
| Resolve              | `.claude/commands/resolve.md`                 | Merge conflicts                           |
| New-Task             | `.claude/commands/new-task.md`                | Task definition (nexusOrchestrator)       |
| Push-to-Nexus        | `.claude/commands/push-to-nexus.md`           | Nexus daemon push                         |
| Go Specialist        | `.claude/agents/engineering-go-specialist.md` | Go-specific patterns                      |
| _(your specialists)_ | `.claude/commands/{skill}.md`                 | Domain-specific                           |

---

## Phase 4: Agent MD Update

Write learning to `## Learnings` section of the relevant MD.

### Format

```markdown
## Learnings

- {Rule/insight} (max 1 line)
  {Why + when to apply} (max 2 lines)
```

### Example

```markdown
## Learnings

- Database migrations must be tested with real DB, not mocks.
  Mock-based tests pass but miss constraint violations. Discovered at FEAT-042.
- Retry logic needs idempotency check: same request sent twice must not
  create duplicate records. Discovered at BUG-015.
```

---

## Rules

1. **Only actionable learnings** (no platitudes like "testing is important")
2. **Max 3 lines per learning** (rule + why + when to apply)
3. **Check for duplicates** before writing new learnings
4. **If `## Learnings` > 30 entries:** consolidate (merge similar ones)
5. **No learning without context** ("discovered at {TASK_ID}" or describe situation)

---

## Edge Case Feedback Loop

### Was the edge case in the plan?

- **No** -> learning for `/task`: "For [type] also consider [X]"
- **Yes, but test failed** -> learning for `/test`: "[domain] needs [pattern]"
- **Post-merge bug discovered** -> learning for `/task` AND `/test`

---

## Result Format

```
LEARNINGS_WRITTEN: {n} (number of new learnings)
AGENTS_UPDATED: [list of updated agent MDs]
DUPLICATES_SKIPPED: {n}
CONSOLIDATED: {yes/no} (was consolidation done?)
SUMMARY: {1 sentence}
```

## Learnings

- Learnings that are too general ("test your code") have no value.
  Each learning must describe a concrete situation where it applies.
- Consolidation is important: if >5 learnings on the same topic exist,
  merge them into one more precise learning.
