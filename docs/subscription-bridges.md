# Using your subscriptions together: the bridge design

Goal: one orchestrator that can hand a task to **Claude Pro/Max**, **GitHub Copilot** and **Google
Antigravity** with the logins you already have, so no separate API subscriptions are needed, and that
switches between them when a quota runs out. This page is what was investigated on 2026-10-16, what is
verified and what is not, and the plan. It extends [delegating-subscriptions.md](delegating-subscriptions.md).

Legend: **verified** = run on this machine; **documented** = in the vendor's own docs or changelog, not run
here; **unverified** = assumed, must be tested before it is built on.

## The one rule that makes this possible

A subscription has no API. What every vendor does offer is its **own official client**, and each client can
run without a person at the keyboard. Nexus therefore never touches the login (no token copying, no replaying
OAuth, no private protocols). It starts the vendor's program, the same one you would start, and reads its
answer. That is also the only form the vendors' terms allow.

| Vendor      | Official headless client                                 | Status                                                                                                                                       |
| ----------- | -------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Claude      | `claude -p` (`--output-format json` / `stream-json`)     | **verified** (v2.1.301 here): one call returned the reply, token usage, cost and a rate-limit event                                          |
| Antigravity | `agy` (Antigravity CLI), prompt on stdin                 | **verified** (1.4.12): `echo hi \| agy --model <id>` answers on stdout; a used-up model exits 3 with `AGY_ERROR` JSON and "Resets in 43h54m" |
| Copilot     | `copilot -s` (Copilot CLI) and `vscode.lm` in the editor | `vscode.lm` bridge exists in Nexus; `copilot` **verified** (1.0.100): prompt on stdin with `-s`, one call returned the reply                 |

## What each route can and cannot do

### 1. Claude: `claude -p` (works today, can be much better)

Nexus already runs it (`llm_cli`, kind `cli`). It treats the CLI as a text pipe: one invocation per request,
the whole conversation re-sent as one transcript, the reply read from stdout. What the real output gives that
Nexus throws away (**verified**, `--output-format json`, one tiny call):

- `rate_limit_event`: `status` (`allowed_warning`), `rateLimitType` (`seven_day`), `utilization` (**0.93**),
  `resetsAt` (epoch seconds), overage state. This is the live quota reading that PLAN-074 A and B need, for
  free, with no API key. On this account the weekly limit was at 93 % and resets 2026-10-17 06:00 CEST.
- `usage` (input, output, cache read/created tokens) and `total_cost_usd` per call, per model.
- `terminal_reason` and `api_error_status`, so a limit is a typed error and not a string in stderr.
- `stream-json` with `--include-partial-messages` gives real token streaming (today the reply arrives as one
  chunk).

Costs and limits to design for:

- A trivial call still read ~15 000 cached input tokens, the client's own system prompt and tools. Use
  `--tools ""`, `--no-session-persistence`, `--disable-slash-commands`, a minimal `--setting-sources`, and an
  empty working directory (Nexus already uses a temp dir). `--bare` is **not** an option: it never reads the
  subscription login.
- `ANTHROPIC_API_KEY` in the environment silently switches billing to the API. The adapter must remove it
  from the child's environment (and say so in the provider status).
- The CLI is an agent, not a model endpoint. Nexus keeps its own tool loop, so tools stay off and the CLI is
  used as a completion engine.

**Terms.** Anthropic's documentation says subscription OAuth tokens are for Claude Code and claude.ai only
and that other products must use API keys (reported by several outlets in February 2026; check Anthropic's
current usage policy before relying on this). Running the real `claude` binary yourself, on your own machine,
for your own work is the use that policy describes. Reading its token, serving other people through it, or
running unattended bulk jobs is not. Nexus must stay on the right side: local, single user, official binary,
never the token.

### 2. Antigravity: `agy -p`, not the IDE

- **Verified (read from the installed app):** Antigravity IDE 2.5.5 bundles no extension that contributes
  language models (`languageModelChatProviders`) and its product config has no model API for extensions. So
  the `vscode.lm` bridge cannot reach Antigravity's models, even with the Nexus extension installed. The
  extension can reach the editor, not the agent inside it.
- **Verified (agy 1.4.12, `brew install --cask antigravity-cli`):** the Antigravity CLI runs the same agent
  engine with the Google sign-in. Headless use is the equivalent of `claude -p`:
  - The prompt goes on stdin (`echo "say pong" | agy --model gemini-3.8-flash-low` prints `pong`, exit 0).
    `-p` / `--print` / `--prompt` want the prompt as their value (`-p=text`) and swallow the next flag, so
    Nexus does not use them. `agy models` lists the ids, one per line, id first.
  - The allowance is counted per model: with the default model used up the same account still answered on
    `gemini-3.8-flash-low`. Nexus therefore passes the configured model as `--model`.
  - A used-up model exits 3 with nothing on stdout and on stderr `error: <text>` and
    `AGY_ERROR: {"status":"RESOURCE_EXHAUSTED","error_code":429,"short_error":"... Resets in 43h54m17s."}`.
    The driver types it (quota, rate limit, auth, overloaded) and reads the reset time, so routing cools the
    provider until then.
  - Still unknown: the shape of `--output-format json` / `stream-json` (a call that succeeds on a model
    with allowance was only recorded as text), and whether the same quota is drawn as by the IDE.
- **Not recommended:** the IDE's language server speaks a private gRPC API (`LanguageServerService`, seen in
  its logs). Driving it is reverse engineering of an undocumented, changing interface and is outside what
  Google's terms allow for the subscription. Nexus will not do it.
- **Pull route, available now:** the connected IDE agent can pull work itself over MCP (`claim_task`,
  `report_task_status`, the delegation prompt in `delegate sessions`). Nexus cannot push into it; the agent has
  to call. Good for "let the agent that is already open do it", not for automatic routing.

### 3. Copilot: `vscode.lm` and the `copilot` CLI

- `vscode.lm` already works (the model bridge): Nexus lists Copilot's models and sends prompts through the
  editor, with the editor's consent dialog. It only works while an editor window with the extension is open,
  and quota and context length are not reported (Nexus assumes 128K).
- `copilot` works without an editor (verified with 1.0.100): the prompt goes in on stdin, `-s` prints the
  answer only, `--no-ask-user` stops it asking questions, `--model auto` (or a model your plan has) selects the
  model. `-p` is not used: it needs the prompt as its value. Premium-request accounting for non-interactive use
  is not documented in what was found; test one call and watch the usage page.

### 4. What cannot be done

- Nexus cannot read a vendor's balance from outside the vendor's client. Only the Claude CLI reports it per
  call; for the others the usage ledger and observed errors are the only signal (cooldown after the first
  limit error, with the reset time if the client prints one).
- A subscription is for one person. Nexus is for one person's machine. Nothing here shares a login.

## Design

One new idea, **provider transports**, and two reused ones:

| Transport | Provider kinds                                     | Notes                                                                  |
| --------- | -------------------------------------------------- | ---------------------------------------------------------------------- |
| `cli`     | `claude`, `agy`, `copilot` (and `gemini`, `codex`) | headless client per request; one **driver** per tool parses its output |
| `editor`  | `vscode-<vendor>-<model>`                          | exists; only while the editor is open                                  |
| `pull`    | any MCP-connected agent                            | exists; the agent claims tasks                                         |

The change is inside `cli`: replace the single text-pipe adapter with a small **driver** per tool, each
answering the same questions: how to build the command line (model, no tools, no persistence), how to read the
stream (text deltas, final text), what usage the call reported, whether the call hit a limit and until when,
which environment variables to drop. The core only sees the existing `LLMClient` port plus the two things
PLAN-074 adds: typed errors with a reset time (A1) and usage (C1). New vendors then cost one driver, not a
new adapter.

Allow-list: `agy` and `copilot` are known tools next to `claude`, `aichat`, `llm`, `gemini` and `codex` (
the `NEXUS_CLI_PROVIDERS` variable already extends it). Discovery already finds installed tools; it should
report installed, signed in (from a cheap `--version` / status call) and, for Claude, the last seen limit.

## Plan (extends PLAN-074 (the F items are described here; they are added to the plan once PLAN-074 is on develop))

1. **F1 Claude driver**: `--output-format stream-json --include-partial-messages`, parse reply, usage, cost,
   `rate_limit_event`, `terminal_reason`; drop `ANTHROPIC_API_KEY`; minimal-overhead flags. Tests against a
   recorded real output (kept as a fixture) and a fake CLI. Feeds A1, A2, C1.
2. **F2 Quota from the client**: store `utilization` and `resetsAt` per provider; show them in Providers,
   Mission Control and the extension; route away at a configurable threshold (default 0.9) before the limit is
   hit, and wait for `resetsAt` when every candidate is above it (A4).
3. **F3 Antigravity driver** (done, TASK-599): `agy` installed, its real headless output recorded, driver in
   `llm_cli/agy_driver.go` with a typed-failure test from the recorded output and an opt-in real test
   (`-tags realagy`).
4. **F4 Copilot CLI driver**, plus the editor bridge reporting quota errors as typed errors (A1).
5. **F5 Honest status**: one screen that says, per vendor, which route is live, what was last seen (limit,
   reset, last error) and what is not possible, so a missing capability is visible instead of silent.
6. **F6 Terms note in the UI and docs**: local, single user, official binary, no token handling.

Order: F1, F2 first (the only part verified end to end, and it unlocks quota-aware routing for Claude, the
account that is at 93 % today); then F3 (needs the CLI installed), F4, F5, F6.
