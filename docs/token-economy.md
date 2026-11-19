# Spending fewer tokens

Subscriptions and API plans have limits, and the expensive models run into them first. Nexus has four
levers, each switched on in the project's `.nexus/policy.json` (see [rulebased-workflow.md](rulebased-workflow.md))
and each measurable with `nexus-cli usage` (tokens per provider, model and role).

| Lever               | What it does                                                                                                                           | Setting                      |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- |
| Lean Claude calls   | `claude` is started without your settings, hooks, plugins, skills and MCP servers, so every call does not carry them                   | always on (stream-json mode) |
| Model ladder        | Cheap models first; the expensive one is asked only when a cheaper one failed verification or is limited                               | `ladder`                     |
| Context budget      | A task's context files are capped in tokens: the largest are sent as outlines (declarations only), then shortened, each marked as such | `contextBudget`              |
| History budget      | Old session messages are dropped until the history fits the budget                                                                     | `historyBudget`              |
| Quota-aware routing | A provider whose plan is nearly used up is tried after the others                                                                      | `quotaThreshold`             |

A starting point for a project that uses a local model, Claude Sonnet and Claude Opus:

```json
{
  "ladder": {
    "default": ["Ollama/*", "Claude (subscription)/sonnet", "Claude (subscription)/opus"]
  },
  "contextBudget": 8000,
  "historyBudget": 4000,
  "quotaThreshold": 0.85
}
```

## What was measured, and what was not

- **Measured** (claude 2.1.301, one-word prompt, this machine, 2026-10-16): the fixed part of a `claude -p`
  call read about 15 500 input tokens with a typical setup and about 3 700 with the lean flags (`--setting-sources ""`
  and `--strict-mcp-config`), 76 % fewer, at about half the cost per call. The number depends on how much
  your own setup loads; on a bare installation the difference is smaller. A short `--system-prompt` did not
  help: the weight was the user's setup, not the client's own prompt. `go test -tags realclaude ./internal/adapters/outbound/llm_cli/`
  repeats the check and fails if a call reads more than 8 000 tokens.
- **Measured** (tests): the context budget keeps the prompt at about the budget (an 60-method Go file went
  from about 2 500 to about 500 tokens as an outline); the history budget drops the oldest messages first.
- **Not measured**: how much the ladder saves in a real project. That depends on how often the cheap model
  passes verification. Use `nexus-cli usage --since 7d` before and after; the numbers per model and role are
  in the ledger. Outlines can cost quality when the model needs a body it does not see: raise the budget, or
  list the file in an agentic task, where the model reads files itself.
- **Not done**: summarising old history with a cheap model, and Anthropic prompt caching for the rules and
  briefing (Nexus puts them in the user message today; caching them needs a restructure that mostly helps API
  keys, not subscriptions, which cache on their own).
