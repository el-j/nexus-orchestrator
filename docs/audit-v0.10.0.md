# Audit of v0.10.0

Measured on `develop` right after the v0.10.0 release (2026-10-15). Every number below was produced by
running the command named next to it; what could not be measured says so. The definition of done is
the one in the request: 100 % functional, no design issues, 100 % test coverage, no gaps, stubs or
placeholders, 100 % documented, 100 % in-code documentation, 100 % end-to-end tested, 100 % mutation
tested, no warnings, no type issues.

## Verdict

Nexus is a solid, honest **local task queue with a verification gate**: it queues work, routes it to a
provider chain, writes and verifies the result, streams the reply and remembers per project. It is
**not yet at the definition of done**, and three of the ten criteria cannot be claimed at all today:

| Criterion                   | State                                                                                                                                                     | Evidence                                                         |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| Test coverage 100 %         | **Go 97.6 %** (247 of 10 242 statements open); frontend, docs, extension, GitHub Action **100 %** (all four metrics)                                      | `scripts/go-coverage.sh`, `vitest --coverage`, `jest --coverage` |
| Mutation tested 100 %       | **Not measured as a whole.** The gate covers `core/domain` and `core/services` only (70 % efficacy); no adapter, no Vue code has a recorded result        | `scripts/mutation.sh`, `docs/mutation-testing.md`                |
| End-to-end tested 100 %     | **No.** 20 daemon scenarios and 15 browser scenarios; see the list of untested features below                                                             | `internal/e2e`, `frontend/e2e`                                   |
| In-code documentation       | Exported Go and TS items: **0 missing** (revive `exported`, `check-tsdoc`). Wrong or stale comments cannot be checked by a tool                           | `golangci-lint run` (0 issues), `npm run check:docs`             |
| Code warnings / type issues | golangci-lint 0, `go vet` 0, `vue-tsc` and `tsc` 0. The Go test link step prints `ld: warning: ignoring duplicate libraries: '-lm'` on macOS (cgo sqlite) | `go test`                                                        |
| Stubs and placeholders      | No `TODO`, `FIXME`, `unimplemented` or `panic("not implemented")` in shipped code. One honest limitation (below)                                          | `git grep`                                                       |
| Boss files                  | **None in shipped code**: the largest hand-written file is 500 lines (`planfiles.go`), the gate is 500. Tests are exempt and reach 1 927 lines            | `scripts/check-file-size.sh`                                     |
| Documentation 100 % correct | **No.** Two historical audits at the repo root are superseded; the task backlog is unreliable (below)                                                     | see "Stale and wrong documentation"                              |
| 100 % functional            | **No.** Quota handling does not exist (below); several real providers were only tested against fakes                                                      | see "Gaps"                                                       |

## What Nexus does today

- **Queue and engine** (`internal/core/services`): one task at a time, persisted in SQLite, interrupted
  tasks re-queued on start. A task has a role; frontier roles prefer cloud models, routine roles local
  ones, the rest follow as fallbacks. Plain tasks write one file; `agentic` tasks run a bounded tool loop
  (read, write, edit, verify, finish). A `verificationCommand` decides `COMPLETED` (with a self-healing
  retry).
- **Providers**: LM Studio, Ollama, OpenAI-compatible (OpenAI, Copilot token, Azure, vLLM), Anthropic,
  Gemini, and the `claude` CLI (Claude Pro) through `llm_cli`. Replies stream.
- **Project memory** (the brain): SQLite FTS5 knowledge per project, a token-budgeted briefing sized to the
  first model of the chain (`limit/8`), policy rules from `.nexus/policy.json`, task outcomes remembered.
- **Surfaces**: REST + SSE, 42 MCP tools, `nexus-cli`, the desktop app, the Vue UI (5 sidebar items) and
  the VS Code extension, tied together by the surface matrix test.
- **Connecting agents**: `nexus connect`, the extension's MCP registration, `nexus-mcp-stdio`.

## Gaps (what is missing or does not do what it appears to)

1. **Quota and rate limits are not handled as a state.** A 429 becomes the string `"<provider>: rate
limited (429)"`. The engine then tries the next provider of the chain, which works, but: `Retry-After`
   and the reset time are ignored; the provider is tried again on the very next task; and
   `ports.ProviderInfo.RateLimited` is **dead**: `discovery.go` only ever assigns `false`, nothing sets it
   to `true`. When every provider is limited the task ends `FAILED` instead of waiting. There is no
   "wait until 14:00 and continue" and no switch to a smaller model of the same vendor.
2. **No token or cost accounting.** The adapters do not read the `usage` block of the responses (only the
   Claude activity reader does, for display). The engine estimates tokens with `len/4`. So Nexus cannot say
   what a task cost, which role burns the quota, or when a limit is near.
3. **Whole files as context.** `ContextFiles` are pasted in full. The briefing is bounded, but it is the
   same project brief for every task; the FTS search that already exists is not used to pick what this task
   needs. Session history is capped by message count (`MaxHistoryMessages`), not by tokens.
4. **No model ladder.** Routing knows "frontier" and "local" only by provider name matching
   (`isFrontierProvider`, `isLocalProvider`); there is no ordered list such as Opus → Sonnet → Haiku →
   local, and no rule that says which task may use which rung.
5. **`httpapi_client` cannot upsert knowledge**: `IngestKnowledge` returns "direct knowledge upsert not
   supported via HTTP client". It is the only functional stub; the CLI path works around it with
   `IngestFromFile`.
6. **Real providers are tested against fakes.** Anthropic, Gemini, LM Studio streaming and the Claude CLI
   adapter run in tests against local fake servers; only the `claude` CLI was run once for real.
7. **macOS app is ad-hoc signed, not notarized** (TASK-574); the released CLI archives' daemon serves only
   the light `/ui` (TASK-580); desktop app and a running daemon fight for port 63987 (TASK-579).

## Missing tests

- Go: 247 uncovered statements. The biggest: `repo_sqlite/repo_repair.go` (15), `httpapi/server.go` (13),
  `fs_watcher/watcher.go` (12), `bootstrap/runtime.go` (11), `repo_sqlite/repo.go` (8), `sys_scanner/plan_scanner.go` (8),
  `cli/connect.go` (7). The merged figure excludes `internal/testutil` and `main.go`.
- The repository gate is `MIN_COVERAGE=96`, not 100.
- Platform skips (`t.Skip`) exist in 12 Go test files; they are legitimate (Windows, root, macOS) but mean
  the 100 % figure is per platform.

## End-to-end features without a test

Daemon scenarios exist for: task + verification, self-healing, agentic edits, restart, shutdown, port clash,
project isolation, every MCP tool, stdio proxy, CLI backlog, `nexus-submit`, streaming, `wait_for_task`,
session rules, `init`, policy, tests-first rule. **Not covered end to end:**

- provider failover and rate limits (nothing tests a chain with a limited first provider through the daemon)
- `nexus connect` against a real config file and `claude mcp add`
- plan-file folders (`policy.planFiles`), discovered plans, activities, AI-session delegation
- the `claude` CLI provider and every cloud provider (only fakes)
- the desktop app (smoke only in CI) and the VS Code extension against a real daemon (the host tests use fakes)
- Playwright against the real daemon covers one flow file (`frontend/e2e/real/flows.spec.ts`)

## Unlinked or half-linked features

- The surface matrix lists **38 deliberate absences** (13 written out, 25 through the shared `noCLI`, `noExt`, `noWails` reasons), for
  example `wait for a task` has no CLI command and `update a task` has no extension command. They are
  honest, but each is a feature a user cannot reach from that surface.
- `RateLimited` is exposed in the provider API and shown nowhere because it is never true (gap 1).
- The brain's FTS search (`get_focused_context`) is available to agents over MCP but not used by the
  engine when it builds a prompt (gap 3).

## Stale and wrong documentation

- `AUDIT_REPORT.md` and `PROJECT_ASSESSMENT_AND_WORKFLOW_BLUEPRINT.md` at the repository root are marked
  historical; they state tool counts and gaps that no longer hold (Gemini is supported now). Move them to
  `docs/archive/` or delete them.
- `docs/agent-workflow-audit.md` says 41 MCP tools; the daemon exposes 42 (README, site and
  `mcpTools.ts` already say 42).
- `.claude/tasks` holds 525 task files and **about 230 still say `todo`**; most belong to plans that are
  finished, have no plan file, or have no `planId`, and 10 plan files still say `active`. The Workspace
  Agents view and the onboarding therefore show an unreliable backlog. It needs a one-off closing pass.
- Comments that became wrong cannot be found by a tool; only a reading pass can. None was found in the
  files read for this audit.

## Mutation testing, honestly

The gate (`make mutation`) runs gremlins on `core/domain` and `core/services` and StrykerJS on the three
TypeScript packages, with an efficacy floor of 70 %, not 100 %. A previous StrykerJS report for
`github-action` exists locally (`github-action/reports/mutation`, ignored by git). No result is recorded in
CI or in the docs, so "100 % mutation tested" is currently unverifiable. A run for this audit on
`core/domain` (gremlins, `MUTATION_MIN_EFFICACY=0`) reported 2 killed, 2 lived, 12 not covered and 19 timed
out, i.e. 50 % efficacy and 25 % mutator coverage. That figure is not trustworthy as it stands: gremlins
measures coverage with the package's own tests only, so code that other packages' tests cover shows as "not
covered", and loops with a changed bound show as timeouts. The run on `core/services` was stopped after
several minutes without a result. Making the Go numbers meaningful (a coverage profile across packages, a
longer timeout coefficient) is part of PLAN-074 E3. Equivalent mutants make 100 %
unreachable in practice; a realistic definition is "every surviving mutant is either killed or listed with
a reason".

## Recommendations

Ordered by the benefit for the user, each one a small change (see PLAN-074):

1. **Quota-aware routing** (gap 1): classify errors (rate limit, quota exhausted, overloaded, auth, network);
   keep a per-provider/model _cooldown until_ time from `Retry-After` or the vendor's reset header; skip
   cooled-down candidates; when all candidates are cooled down, hold the task in a new `WAITING` state with
   the resume time shown in the UI, CLI and extension, and resume by itself.
2. **Model ladder** (gap 4): an ordered, configurable list per role in the policy
   (`"ladder": ["anthropic/opus", "anthropic/sonnet", "anthropic/haiku", "lmstudio/*"]`) and a rule for
   when to step down (limit, task size, role) and when to step back up.
3. **Usage ledger** (gap 2): read `usage` from every adapter, store tokens in/out/cached per task, provider,
   model and role; show it in Mission Control; expose `get_usage` over MCP and REST; add a budget per
   provider and day that feeds the routing rule above.
4. **Spend fewer tokens** (gap 3): put cheap work on cheap models by role; send only the context a task
   needs (FTS-ranked brain entries instead of whole files, file outlines before whole files); cap history by
   tokens and summarize old turns; use Anthropic prompt caching for the stable part of the prompt (rules,
   briefing); verify with the local test command before any frontier retry so an expensive model is not
   asked twice for the same failure.
5. **Close the measured gaps**: Go to 100 % (247 statements), raise the gate to 100, add the missing end-to-end
   scenarios, record mutation results for the adapters.
6. **Clean the repository**: archive the two stale audits, correct the tool count, close the stale tasks.

Nothing in this audit was changed in code. The proposal that follows from it is in
`.claude/plans/PLAN-074.md`.
