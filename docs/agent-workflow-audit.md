# Agent-workflow audit (2026-10-14)

Question: can nexusOrchestrator today be the single place where any agent and any model gets
project knowledge, plans, history and quality control, across projects, without re-aligning
agents every time? Every statement below was checked against the code or by running it; the
evidence is named next to it. Anything not checked is listed under "Not verified".

Audited commit: `develop` at 86be6f5 (after PRs #76–#78), macOS arm64, Go 1.27.1, Node 22.

## Verdict

The orchestrator is a solid **task queue, provider router and verification runner**, and a
working **project knowledge store that external agents can pull from over MCP**. It is **not yet**
a system that makes agents follow your workflow. The rules you listed (plan first, tests first,
small files, document everything, don't forget, don't waste tokens) are mostly enforced on
_this repository's_ contributors by CI, not on the agents the orchestrator runs for _other_
projects. Three gaps decide how far you get:

1. The daemon's own LLM runs never receive the project knowledge (proof below).
2. Nothing records what was done into the knowledge base, so "remember" stops at raw task history.
3. Quality rules reach an agent only if you hand it a `verificationCommand` for that project.

## Update: rule-based workflow (branch `feature/rulebased-orchestration`)

Gaps 1–3 above were addressed, see [rulebased-workflow.md](rulebased-workflow.md): project briefing
and rules are put in front of every task; a per-project `.nexus/policy.json` supplies the default
verification command and enforces file size and tests-first on agentic runs; finished tasks are
remembered (newest 30) and session history is capped to 20 messages; the onboarding leads with the
rules. Still open: a real planning gate, a derived file map, enforcing rules on agents that edit
files outside the orchestrator, and any run against a real model. The scorecard below describes
the state before this change.
## Scorecard against your list

| Requirement              | State                       | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                  |
| ------------------------ | --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Know what the project is | Partly (pull only)          | `get_onboarding_context` returns a budgeted summary (measured: ~410–450 tokens for this repo). Only external agents that call it get it. `internal/core/services` has no reference to the brain outside `brain_*.go`; the brain is wired into the file watcher only (`internal/bootstrap/runtime.go:135`). `prepareChatPrompt` (`prompts.go:13`) sends instruction + named context files + session history, nothing else. |
| Know what is where       | Partly (hand-written)       | `file_map` knowledge is whatever a `## ...file/map/layout...` section of a markdown file says. `brain init` on this repo returned `initialized:false, entryCount:0` because there was no `CLAUDE.md`. Nothing derives a map from the tree. The map built from `.github/copilot-instructions.md` had 9 paths, all existing (checked), but it is static.                                                                    |
| Comprehensive planning   | Convention only             | `command: plan` is an ordinary LLM call. The only gate is `orchestrator_workers.go:38`: an `execute` task is refused unless _some_ task of command `plan` is completed for that project. Plan content, task breakdown and size are not checked. Plan/task files (`.claude/plans`, `.claude/tasks`, 510 task files) follow the conventions in `.claude/commands`.                                                          |
| Small files              | CI for this repo only       | `scripts/check-file-size.sh` (500 lines) runs in CI. Agent output is not size-checked: `write_file` takes any content (`agent_loop.go`).                                                                                                                                                                                                                                                                                  |
| Document everything      | CI for this repo only       | revive `exported`, `scripts/check-tsdoc.mjs`. Not applied to agent output in other projects.                                                                                                                                                                                                                                                                                                                              |
| Tests before code        | Not enforced                | No test-first rule in any prompt (`agentSystemPrompt`, `correctionPrompt`) or gate. Searched `internal/core` for test-first/TDD/size/doc wording: no hits outside tests.                                                                                                                                                                                                                                                  |
| Black-box testing        | Strong for this repo        | `internal/e2e`: 11 real-daemon tests pass (ran them): verification gate, self-healing, agentic multi-file edit, restart persistence, project isolation, every MCP tool, stdio proxy, both CLIs.                                                                                                                                                                                                                           |
| Correct bad agent output | Works if a command is given | Self-healing loop re-prompts with the failure output (`verification.go`); in agentic mode `finish` is refused until verification passes (`agent_loop.go:176-181`). Verified by e2e tests. The command is chosen per task by the submitter; there is no per-project default.                                                                                                                                               |
| Remember what was done   | Partly                      | Tasks, logs and per-project conversation history persist in SQLite and survive restarts (`TestDaemonPersistsAcrossRestart`). Nothing writes lessons into the brain: the only callers of `IngestKnowledge` are the markdown ingesters.                                                                                                                                                                                     |
| Don't waste tokens       | Partly                      | Budgeted onboarding, focused context and FTS search exist; a pre-flight check skips providers whose context window is too small (`execution_engine.go:196-212`). But session history only grows: no trimming or summarising anywhere in `session_service.go`, `session_repo.go` or `prompts.go`; when too large the task fails `TOO_LARGE`.                                                                               |
| Model switching          | Works                       | Ordered fallback chain, role-driven routing (architect/planner to frontier, linter/test to local), failover logged on the task (tests in `execution_engine_test.go`, pass). History is keyed by project, not model, so a switch keeps it.                                                                                                                                                                                 |
| Project switching        | Works                       | Everything is keyed by project path; `TestDaemonIsolatesProjects` passes.                                                                                                                                                                                                                                                                                                                                                 |

## What is proven to work

- Full Go suite, frontend (782 tests), docs (104) and the 11 e2e tests pass; Go merged coverage
  97.9% locally; 41 MCP tools exposed (counted via `tools/list` on the running daemon).
- Self-healing and agentic editing against a fake provider, with path confinement for writes.
- One daemon, one database, many projects, restart-safe.

## Make targets, run one by one

Run in a clean worktree. Before this PR's fixes:

| Target                                                                                                                                                                                                                                        | Result                                                                                                                              |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `help`, `version`, `vet`, `lint`, `test`, `test-unit`, `test-cover`, `build`, `build-frontend`, `build-vscode`, `build-dev`, `build-darwin-*`, `docker-build`, `status`, `connect`, `check-air`, `dev`, `dev-daemon`, `start`, `init-project` | work                                                                                                                                |
| `stop`                                                                                                                                                                                                                                        | listed as a target, **did nothing** ("Nothing to be done")                                                                          |
| `build-linux-*`, `build-windows-amd64`                                                                                                                                                                                                        | **failed to link** (`__ubsan_handle_*` undefined): zig needs `-fno-sanitize=undefined`, which only CI passed                        |
| `build-gui`                                                                                                                                                                                                                                   | **reported success without building**: the local Wails CLI was 2.11.0 vs go.mod 2.16.0, the build crashed and the recipe carried on |
| `nice`                                                                                                                                                                                                                                        | failed in a fresh checkout: docs/root dependencies not installed                                                                    |
| build targets on Node 20                                                                                                                                                                                                                      | failed deep inside Vite; no version check                                                                                           |

Fixed here: `stop` (pid file), new `start-bg` and `dogfood`, zig flags, `build-gui` now fails
loudly and checks the CLI version, `check-node` (`.nvmrc` = 22), `nice` installs what it needs.
Re-run results: `start-bg`/`stop`, the three zig targets and `build-gui` (with Wails 2.16.0)
now succeed; `build-gui` with the old CLI exits 1 with the install command. Not re-run:
`release*` (creates tags), `upgrade-latest`, `docker-push`, `mutation` (long, local-only).

## Dogfooding

The repository already had a push-plan-to-daemon command (`.claude/commands/execute-via-nexus.md`)
and an MCP-first rule in `.github/copilot-instructions.md`, but nothing made a session start on
it: no `CLAUDE.md`, no `.mcp.json`. Added `CLAUDE.md` (what it is, file map, rules, commands),
`.mcp.json` (project-scoped `nexus` server) and `make dogfood` (start daemon, load this project).
Verified: `make dogfood` against a throwaway database gave 5 entries (4 convention, 1 file_map)
and a ~450-token onboarding. This audit itself was **not** run through the orchestrator: the MCP
server was not registered in the session that did the work.

## Recommended next steps, in order of payoff

1. **Inject project knowledge into the daemon's own runs.** Prepend the budgeted onboarding
   (and `get_focused_context` for the target file) in `prepareChatPrompt` and `agentSystemPrompt`.
   Smallest change with the largest effect for non-Claude models.
2. **Per-project workflow policy.** A stored default `verificationCommand`, max file size and
   required checks per project, applied to every task (and to `finish`), so quality rules travel
   with the project instead of with each submission.
3. **Automatic memory.** On task completion write a short `learning`/`decision` entry
   (files touched, outcome, verification) to the brain; cap and summarise session history.
4. **Real planning gate.** Require a plan document with task list for `execute`, not just any
   completed `plan` task; link tasks to plan items.
5. **Derived file map.** Build the map from the tree (package list plus doc comments) so it
   cannot drift from a hand-written section.
6. Keep the rule list in `CLAUDE.md` short; every rule worth keeping should become a check in (2).

## Not verified

- No run against a real hosted or local model: all execution proofs use fake providers
  (e2e harness, unit doubles). Quality of real model output under these prompts is unmeasured.
- Token savings were not measured against a baseline, only the size of the onboarding payload.
- `claude mcp add` / `.mcp.json` pickup in Claude Code was not exercised in a live session.
- Windows and Linux binaries were cross-built, not run.
- The desktop app was built, not launched.
