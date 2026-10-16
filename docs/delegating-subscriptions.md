# Delegating to your AI subscriptions

Nexus can hand tasks to the subscriptions you already pay for, without API keys. There are two
routes.

## Claude Pro / Max: through the `claude` command-line tool

If `claude` is installed and logged in on this machine (`claude` once, then `/login`), Nexus can run
it for each task: the prompt goes to `claude -p` on stdin and the printed reply is the result.

- **From discovery**: the scan finds `claude` on the PATH (provider "Claude CLI", status
  installed). Promote it in the Discovery view, or `promote_provider` over MCP, or
  `POST /api/providers/discovered/{id}/promote`.
- **By hand**: add a provider config of kind `cli` whose `baseUrl` is the command line:

  ```sh
  curl -X POST localhost:63987/api/providers/config -H 'Content-Type: application/json' -d '{
    "kind": "cli", "name": "Claude (subscription)", "baseUrl": "claude -p --model sonnet",
    "model": "sonnet", "enabled": true }'
  ```

### Usage and quota: `--output-format stream-json`

Promoting the discovered Claude CLI now creates the command line
`claude -p --output-format stream-json`. In that mode Nexus reads more than the text:

- **Live text**: the reply streams as the model writes it, instead of arriving as one chunk.
- **Usage**: input, output and cached tokens and the list-price cost of every call.
- **The plan's own quota reading** (`rate_limit_event`): which window (`five_hour`, `seven_day`), how much
  is used (`utilization`, 0 to 1) and when it resets. No API key is needed for this; it is what the
  client itself reports.
- **Typed limit errors**: a refused call (limit reached) becomes a rate-limit or quota error with the reset
  time, so the engine skips the provider until then instead of failing every task.

The call is made lean: no tools, no saved session, no slash commands (`--tools ""`,
`--no-session-persistence`, `--disable-slash-commands`), because Nexus runs its own tool loop and uses
Claude as a completion engine. Flags you already put in the command are not repeated. A command line
without `--output-format stream-json` keeps the old behaviour (plain text on stdout).

The error shapes for a refused call are modelled on the fields the client prints for a successful one;
they have not been captured from a real refusal yet.

### Things to know

- Only known tools may run: `claude`, `aichat`, `llm`, `gemini`, `codex`. Add more executable names
  with `NEXUS_CLI_PROVIDERS=mytool,other` (the config API must not become a way to run any program).
- The tool runs in an empty temporary directory, so an agentic CLI cannot touch a project; Nexus
  still writes the result and runs your verification command itself.
- `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` are removed from the environment of `claude` in
  stream-json mode: with an API key set, `claude` bills the API instead of your subscription.
- Even a one-word call reads about 15 000 cached input tokens (the client's own system prompt). Subscription
  limits apply; a call the client refuses fails the task and the fallback chain moves on.
- Use this for your own work on your own machine, with the real `claude` program. Nexus never reads or
  copies the login; see [subscription-bridges.md](subscription-bridges.md) for the terms.

## GitHub Copilot, Antigravity and other editor models: through the VS Code bridge

The VS Code extension can share whatever models your editor offers (`vscode.lm`: Copilot and
others) with Nexus. See [vscode-model-bridge.md](vscode-model-bridge.md). Each model shows up as a
provider named `vscode-<vendor>-<model>`, with the context length the editor reports. Whether
Antigravity exposes its models to `vscode.lm` depends on the Antigravity build: open the extension's
"Share editor models" command and see which providers appear; if none from Antigravity do, it does
not expose them to extensions.
