# What Nexus does, and how work reaches it

Nexus is **not** a window onto what your AI agents are doing. It does four things, and each has one
way in. If an agent does not use that way in, Nexus does not know about the work, whatever the
screens say.

| What Nexus does                                                                                                                                                                                                                         | How work gets there                                                                                                                                        | What you see                                                                                                                                                                |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Runs a task itself** on one of its providers (a local model, an API, your Claude subscription through the `claude` CLI, the editor's Copilot models through the VS Code bridge), writes the result and runs your verification command | `submit_task` (MCP), `POST /api/tasks`, `nexus-submit`, the UI form                                                                                        | The task moves QUEUED → PROCESSING → COMPLETED; its log says `completed via <provider>`, the API field `ranBy` names the provider, live text streams while the model writes |
| **Lets an agent report its own work** as a task                                                                                                                                                                                         | the agent registers (`register_session`), claims a task (`claim_task`), reports (`update_task_status`)                                                     | The task carries the agent's session; Nexus did not run it, the UI says "Status set by an agent"                                                                            |
| **Watches** what some agents do                                                                                                                                                                                                         | nothing to do: it reads Claude Code's transcripts, Continue and network activity                                                                           | Activity feed and Projects (only for agents it can read)                                                                                                                    |
| **Remembers a project** (the brain): architecture, conventions, file map, what was tried                                                                                                                                                | `init_project`, `ingest_knowledge`, `search_knowledge`, `get_onboarding_context`; with `"remember": true` in the policy, finished tasks are remembered too | Project Brain, and the briefing every new session gets                                                                                                                      |

## What is _not_ a way in

- Editing `.claude/orchestrator.json`, the plan and task markdown files, or the SQLite database by
  hand. Plan files are read (see [plan-files.md](plan-files.md)) and shown, but nothing is _executed_
  or _remembered_ because a file changed. A task an agent sets to "in progress" in the database is
  only a label: no worker picks it up, no provider is involved, no brain entry is written.
- Having an agent installed. The VS Code extension lists the agents it sees in the editor; that
  is presence, not work.

## Connect the agents

An agent calls Nexus through MCP. Check which agents on this machine can:

```sh
make connect                       # or: nexus-cli connect      (changes nothing)
nexus-cli connect --write          # adds the nexus server to each agent's MCP config
nexus-cli connect --write --agent antigravity
```

It knows Antigravity (`~/.gemini/.../mcp_config.json`, through the `nexus-mcp-stdio` proxy), Cursor
(`~/.cursor/mcp.json`) and Claude Code (`claude mcp add`). Other servers in a config stay, the
previous file is kept as `<file>.nexus-backup`, and a config that is not valid JSON is never touched.
Restart the agent afterwards. VS Code with the extension needs nothing (the extension registers the
server).

Then tell the agent to use it. `nexus init` (the `init_project` tool) writes the rule into the project's
`.nexus/policy.json`, which every session receives in its briefing: _report progress through the Nexus
tools; never edit the Nexus database or task states by hand._ For an existing policy add that rule
yourself.

## A working session, end to end

1. `get_onboarding_context` for the project: stack, rules, active plan, plan files, what was learned.
2. `register_session`, then `claim_task` for the task, or `submit_task` to have Nexus run it.
3. Work. `update_task_status` with a short result when done.
4. `ingest_knowledge` (kind `learning`) for anything the next session should know.
