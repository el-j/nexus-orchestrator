# Use your editor's models (Copilot and others) from the orchestrator

The daemon runs outside the editor, so it cannot call the chat models your subscriptions give
you. The VS Code extension closes that gap: it serves a small OpenAI-compatible endpoint on
`127.0.0.1` in front of VS Code's language-model API (`vscode.lm`) and registers every model the
editor offers as a provider in the daemon. Tasks, roles and the fallback chain can then use them
like any other provider, and the project rules from `.nexus/policy.json` apply as usual.

## Turn it on

1. Install the extension and start the daemon (`make dogfood`, or `make start`).
2. In the editor settings set `nexus.lmBridge.enabled` to `true`. It is **off by default**:
   requests are billed to the signed-in subscription, so sharing is your decision.
3. Optional: `nexus.lmBridge.vendors`, for example `["copilot"]`, to share only those vendors.
   Empty shares every vendor the editor offers.
4. VS Code may ask for your consent the first time a model is used.

Each model appears as its own provider named `vscode-<vendor>-<model>` (for example
`vscode-copilot-gpt-4o`). Pick it with `providerHint` on a task, in the task form, or by role.

## How it behaves

- The endpoint listens on a random loopback port and needs a bearer token created when it starts;
  nothing else on the machine can use it without the token.
- The daemon keeps these providers in memory only. The extension checks every 30 seconds and
  registers them again after a daemon restart or when the editor's model list changes.
- Switching the setting off, or closing the window, removes them from the daemon.
- A cancelled or abandoned task cancels the model request, so no quota is spent on it.
- Failures map to what the fallback chain understands: no consent is 403, a missing model 404,
  quota, rate limit or blocked content 429, anything else 502.
- With several editor windows open the last one to start owns the `vscode-` providers.

## Limits (checked, not assumed)

- The daemon assumes a 128K context window for these providers, whatever the model really has.
- The bridge streams when asked (`"stream": true`, OpenAI-style server-sent events); the daemon does not ask yet. System messages are sent as user messages, because the
  editor's chat API has no system role.
- **Copilot**: works through `vscode.lm`. **Antigravity** and **Claude Pro**: see below.
- Which models you see depends on the editor and your plan. Open `Nexus: Show Providers` or
  `GET /api/providers` to see what was registered.

## Antigravity and Claude Pro

- Antigravity IDE is a VS Code fork (version 1.107.0 here), so the extension installs there, and
  the bridge shares whatever models Antigravity's `vscode.lm` offers. Whether it offers any was
  not verified; if the provider list stays empty, it offers none.
- A Claude Pro login has no model API. The `cli` provider kind only labels a tool the scanner
  found; the daemon has no adapter that runs it, so a Claude Pro login cannot be delegated to
  today. Claude models are available through the bridge only if the editor lists them.
