# Delegating to your AI subscriptions

Nexus can hand tasks to the subscriptions you already pay for, without API keys. There are two
routes.

## Claude Pro / Max: through the `claude` command-line tool

If `claude` is installed and logged in on this machine (`claude` once, then `/login`), Nexus can run
it for each task: the prompt goes to `claude -p` on stdin and the printed reply is the result.

- **From discovery**: the scan finds `claude` and `copilot` on the PATH (status `installed`). Promote
  them in the Discovery panel, or `promote_provider` over MCP, or
  `POST /api/providers/discovered/{id}/promote`.
- **In the UI**: open **Providers → Add Provider**, select **CLI Tool (Claude / Copilot subscription)**,
  and enter the command line (`claude -p --output-format stream-json` or `copilot -p -s --no-ask-user`).
- **By hand**: add a provider config of kind `cli` whose `baseUrl` is the command line:

  ```sh
  curl -X POST localhost:63987/api/providers/config -H 'Content-Type: application/json' -d '{
    "kind": "cli", "name": "Claude (subscription)", "baseUrl": "claude -p --output-format stream-json",
    "model": "sonnet", "enabled": true }'
  ```

### Usage and quota: `--output-format stream-json`

Promoting the discovered Claude CLI creates the command line
`claude -p --output-format stream-json`. In that mode Nexus reads more than the text:

- **Live text**: the reply streams as the model writes it, instead of arriving as one chunk.
- **Usage**: input, output and cached tokens and the list-price cost of every call.
- **The plan's own quota reading** (`rate_limit_event`): which window (`five_hour`, `seven_day`), how much
  is used (`utilization`, 0 to 1) and when it resets. No API key is needed for this; it is what the
  client itself reports.
- **Typed limit errors**: a refused call (limit reached) becomes a rate-limit or quota error with the reset
  time, so the engine skips the provider until then instead of failing every task.

The call is made lean: no tools, no saved session, no slash commands, no settings and no MCP servers
(`--tools ""`, `--no-session-persistence`, `--disable-slash-commands`, `--setting-sources ""`,
`--strict-mcp-config`), because Nexus runs its own tool loop and uses Claude as a completion engine. Your own
hooks, plugins, memory files and skills would otherwise be loaded into every call: measured with claude
2.1.301, a one-word call read about **15 500 input tokens** with a typical setup and about **3 700** with
these flags (76 % fewer; cost per call about half). Your login is not a setting, so the subscription still
applies. Flags you already put in the command are not repeated. A command line
without `--output-format stream-json` keeps the old behaviour (plain text on stdout).

The error shapes for a refused call are modelled on the fields the client prints for a successful one;
they have not been captured from a real refusal yet.

## GitHub Copilot: through `copilot -p` or the VS Code bridge

- **Headless CLI (`copilot`)**: promoting a discovered `copilot` binary (or adding a `cli` provider for
  `copilot`) runs `copilot -p -s --no-ask-user` (plus `--model <model>` when configured). Quota,
  rate-limit and authentication errors printed by the CLI are classified as typed provider errors so
  routing cools down or switches providers automatically.
- **VS Code model bridge (`vscode.lm`)**: the VS Code extension can share whatever models your editor offers
  (Copilot and others) with Nexus. See [vscode-model-bridge.md](vscode-model-bridge.md). Each model shows up as
  a provider named `vscode-<vendor>-<model>`, with the context length the editor reports, and quota/rate-limit
  or auth errors returned mid-stream carry their HTTP status (`429`, `403`) into the routing engine.
- **Antigravity IDE note**: Antigravity IDE 2.5.5 bundles no `languageModelChatProviders` extension and does
  not expose `vscode.lm` models to extensions; use MCP pull delegation (`claim_task` / `update_task_status`)
  for Antigravity sessions (see [subscription-bridges.md](subscription-bridges.md)).

### Things to know

- Only known tools may run: `claude`, `copilot`, `aichat`, `llm`, `gemini`, `codex`. Add more executable names
  with `NEXUS_CLI_PROVIDERS=mytool,other` (the config API must not become a way to run any program).
- The tool runs in an empty temporary directory, so an agentic CLI cannot touch a project; Nexus
  still writes the result and runs your verification command itself.
- `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` are removed from the environment of `claude` in
  stream-json mode: with an API key set, `claude` bills the API instead of your subscription.
- Subscription limits apply; a call the client refuses fails the task and the fallback chain moves on.
- Use this for your own work on your own machine, with the real vendor program. Nexus never reads or
  copies the login; see [subscription-bridges.md](subscription-bridges.md) for the terms.
