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

### Things to know

- Only known tools may run: `claude`, `aichat`, `llm`, `gemini`, `codex`. Add more executable names
  with `NEXUS_CLI_PROVIDERS=mytool,other` (the config API must not become a way to run any program).
- The tool runs in an empty temporary directory, so an agentic CLI cannot touch a project; Nexus
  still writes the result and runs your verification command itself.
- If `ANTHROPIC_API_KEY` is set in the daemon's environment, `claude` bills the API instead of your
  subscription. Unset it for the daemon if you want the subscription.
- The reply arrives when the tool prints it: `claude -p` prints once at the end, so the live output
  shows it as a single chunk. Subscription limits apply; a rate-limit message from the tool fails the
  task and the fallback chain moves on.

## GitHub Copilot, Antigravity and other editor models: through the VS Code bridge

The VS Code extension can share whatever models your editor offers (`vscode.lm`: Copilot and
others) with Nexus. See [vscode-model-bridge.md](vscode-model-bridge.md). Each model shows up as a
provider named `vscode-<vendor>-<model>`, with the context length the editor reports. Whether
Antigravity exposes its models to `vscode.lm` depends on the Antigravity build: open the extension's
"Share editor models" command and see which providers appear; if none from Antigravity do, it does
not expose them to extensions.
