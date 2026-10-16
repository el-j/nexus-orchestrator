# Rule-based workflow: `.nexus/policy.json`

A project states its working rules once, in the repository. From then on every agent and every
model that works on the project through the orchestrator gets the same rules, including a fresh
session that remembers nothing.

```json
{
  "version": 1,
  "verify": "go vet ./... && go test ./...",
  "maxFileLines": 500,
  "testsFirst": true,
  "remember": true,
  "rules": ["Plan first, then one small step at a time.", "Document every exported item."]
}
```

`make init-project PROJECT=/path` (or `brain init`, or the `init_project` MCP tool) creates this file
for a stack-appropriate default when it is missing. It never overwrites an existing file. Commit it.

## What each field does

| Field            | Effect                                                                                                                                                                                                                                                                                                                                  | Enforced?                         |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------- |
| `verify`         | Default verification command for tasks that do not set their own. Self-healing re-prompts on failure; an agentic task cannot `finish` until it passes.                                                                                                                                                                                  | Yes, every task                   |
| `maxFileLines`   | A source-file write that would exceed the limit is refused and the model is told to split it.                                                                                                                                                                                                                                           | Yes, agentic tasks                |
| `testsFirst`     | Source-file writes and edits are refused until the run has written a test file.                                                                                                                                                                                                                                                         | Yes, agentic tasks                |
| `remember`       | After each completed or failed task a short entry (outcome, instruction, target, verification command) is stored in the project knowledge, newest 30 kept.                                                                                                                                                                              | Yes                               |
| `rules`          | Free text, shown to the model verbatim in every task prompt and in the onboarding.                                                                                                                                                                                                                                                      | Advisory (the model must follow)  |
| `quotaThreshold` | Share (0 to 1, default 0.9) of a provider's plan allowance at which routing tries the other providers of the chain first. It uses the plan's own reading, where the provider's client reports it (today `claude` in stream-json mode); a provider is never removed, only tried later, and a task that names its provider is left alone. | Yes, every task                   |
| `planFiles`      | Where the project keeps its plan and task markdown files (`plans`, `tasks` folders, `discover`); see [plan-files.md](plan-files.md).                                                                                                                                                                                                    | Used by onboarding and the editor |

Test and source files are told apart by name and directory (`*_test.go`, `*.test.ts`, `*.spec.*`,
`test_*.py`, `tests/`, `__tests__/`, `e2e/` are tests; documentation, configuration, fixtures and
`testdata/` are neither and are never refused). Limits keep the file cheap to send: at most 30 rules
of 300 characters, 64 KB.

A policy that exists but cannot be used (invalid JSON, an unknown field such as a misspelt
`maxFileLine`, a bad value) is never skipped silently: the task fails before any model call with the
reason, and the onboarding shows the problem.

## What a model receives

For every task the engine puts in front of the instruction, in this order:

1. A briefing of the project (stack, active plan, task counts, architecture, conventions), at most
   800 tokens and at most one eighth of the model's context window. Models with a context window
   under 4000 tokens get no briefing, only the rules.
2. The rules block built from the policy (stated once).
3. `## Task` and the instruction.

Session history is capped to the newest 20 messages, so a long-running project does not grow the prompt.

A fresh session that is not run by the orchestrator (an editor agent) gets the same rules by calling
`get_onboarding_context`; they come first so a tight token budget cuts the tail, not the rules.

## What this does not do

- It does not control an agent that edits files itself (for example Claude Code in your editor):
  those agents only receive the rules as advice through the onboarding. To have them enforced, hand
  the work to the orchestrator (`submit_task`, `agentic: true`).
- `maxFileLines` and `testsFirst` apply to agentic tasks. A single-shot task writes one target file;
  it gets the default `verify`, the rules text and the self-healing loop.
- "Document everything" and "black-box tests" are rules the model is told, not checks. Put what can
  be checked into `verify` (a linter, a doc-comment check, the e2e suite).
