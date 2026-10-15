# Using nexusOrchestrator across all your projects

One daemon serves every project and every agent. You set it up once, introduce each project once,
and from then on agents ask the orchestrator for the project's context instead of you explaining it
again in every session.

## 1. Start it once

```sh
make start      # builds, then runs the daemon with ONE shared database: ~/.nexus/nexus.db
make status     # in another terminal: is it up, and which LLM providers does it see?
```

- The full UI (Mission Control, agents, providers, plans, history, brain, settings) is at
  <http://localhost:63987/app/>; the root redirects there. `make start` builds it when Node >= 22 is
  active (`nvm use`); without Node it serves only the light dashboard at <http://localhost:63987/ui>
  (queue and providers, nothing about agents or history). `make web` rebuilds the embedded UI by hand.
  MCP endpoint: <http://localhost:63988/mcp>.
- The same UI is the desktop app (`make build-gui`); the desktop app starts its own daemon, so do not run
  it next to `make start`.
- Every way of starting Nexus (`make start`, a bare `nexus-daemon`, the desktop app) now uses the
  same database, `~/.nexus/nexus.db`, unless `NEXUS_DB_PATH` says otherwise. (An older setup with a
  `nexus.db` in the current directory keeps using it until `~/.nexus/nexus.db` exists.)
- Prefer the desktop app or the VS Code extension? They run the same daemon; skip this step.

## 2. Connect your agents once

```sh
make connect    # prints the exact commands
```

For Claude Code, register it for every project (user scope):

```sh
claude mcp add --transport http --scope user nexus http://localhost:63988/mcp
```

Claude Desktop uses the JSON snippet `make connect` prints; the VS Code extension registers the
server itself. After this, every agent in every project sees the same tools.

## 3. Introduce each project once

```sh
make init-project PROJECT=/absolute/path/to/your/repo
```

This reads the project's `CLAUDE.md` (or `.claude/CLAUDE.md`) and stores each `## Section` as
knowledge, classified as architecture, file map, convention, learning or glossary. Notes:

- Sections with fewer than 20 characters of body are skipped, so a very short `CLAUDE.md` can
  produce `entryCount: 0`. Check with `nexus-cli brain status --project <path>`.
- Add what the agents keep forgetting to that file, then re-run the command. Further knowledge can
  be added by an agent through the `ingest_knowledge` tool.

`make init-project` also writes `.nexus/policy.json` (default verification command, file-size limit,
tests-first, your rules); commit it and edit it. See [rulebased-workflow.md](rulebased-workflow.md).

## 4. Work

Agents pull context instead of you pasting it:

| Need                                    | MCP tool                                  |
| --------------------------------------- | ----------------------------------------- |
| Start-of-session summary for a project  | `get_onboarding_context`                  |
| Relevant knowledge for the task at hand | `get_focused_context`, `search_knowledge` |
| Project knowledge within a token budget | `get_project_context`                     |
| Where things live                       | `get_file_map`                            |
| Instructions for the agent itself       | `howto`, `howto_brief`                    |

Two ways to run tasks:

- **Agents coordinate through the queue.** An agent calls `register_session`, then `claim_task`,
  reports with `update_task_status` and keeps alive with `heartbeat_task`. Several agents can share
  one queue without taking the same task. `delegate_to_nexus` hands work to the orchestrator.
- **The orchestrator runs the task with a model.** Submit with `submit_task` (or
  `nexus-submit --project . --target path/file.go --task-file task.md --verify "go test ./..."`),
  watch it with `nexus-cli queue list` or the dashboard. Use `create_draft` and `promote_task` to
  park ideas in the backlog first.

Running tasks itself needs at least one LLM provider. LM Studio and Ollama on localhost are found
automatically; for hosted models set `NEXUS_ANTHROPIC_API_KEY`, `NEXUS_OPENAI_API_KEY`,
`NEXUS_GEMINI_API_KEY` or `NEXUS_GITHUBCOPILOT_TOKEN` before `make start`. `make status` lists what the
daemon found. Without a provider the queue and the agent-coordination flow still work.

## Troubleshooting

| Symptom                                        | Cause                                                                                                        |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `no such module: fts5` at start-up             | Binary built without `CGO_CFLAGS=-DSQLITE_ENABLE_FTS5`. The Makefile now sets it; rebuild with `make build`. |
| Queue or knowledge looks empty after a restart | The daemon was started from another folder without `NEXUS_DB_PATH`. Use `make start`.                        |
| `connection refused` from `nexus-cli`          | Daemon not running: `make start`.                                                                            |

To delegate to the models your editor subscriptions provide (Copilot and others), see [vscode-model-bridge.md](vscode-model-bridge.md).
