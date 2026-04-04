# PLAN-063 — VS Code Extension: Feature Completion & Distribution

**Status:** active
**Created:** 2026-04-04
**Author:** copilot

## Summary

Pre-launch audit of the VS Code extension and associated release tooling exposed ten gaps ranging from
commands that ship as empty stubs, a session monitor that permanently blocks the daemon worker queue,
a CI pipeline that never actually publishes to the Marketplace, and a `nexus-submit` CLI that loops
forever on cancelled tasks. This plan drives all ten to done before the first public release.

Critical work (TASK-483 – TASK-488) must ship together; high-priority work (TASK-489 – TASK-492)
can follow in a parallel track but must land before promotion to `latest` on the Marketplace.

## Waves

### Wave 1 — Critical: Broken Commands & Daemon Safety

| Task     | Description                                                           |
| -------- | --------------------------------------------------------------------- |
| TASK-483 | Implement `nexus.showProviders` command — currently an empty stub     |
| TASK-484 | Fix `SessionMonitor.pollAndClaim` — claimed tasks stuck in PROCESSING |
| TASK-488 | Fix `waitForCompletion` in nexus-submit — loops on terminal states    |

### Wave 2 — Critical: Distribution & Correctness

| Task     | Description                                                       |
| -------- | ----------------------------------------------------------------- |
| TASK-485 | Add `vsce publish` + Open VSX publish to CI pipeline              |
| TASK-486 | Replace blind JSON casts in `nexusClient.ts` with Zod validation  |
| TASK-487 | Wire unused `nexus.mcpPort` / `nexus.enableMCPPortSweep` settings |

### Wave 3 — High: Test Coverage

| Task     | Description                                                   |
| -------- | ------------------------------------------------------------- |
| TASK-489 | Add unit tests for `workspaceOrchView` and `workspaceScanner` |
| TASK-490 | Add unit tests for `sessionMonitor`                           |

### Wave 4 — High: Release Tooling

| Task     | Description                                                                 |
| -------- | --------------------------------------------------------------------------- |
| TASK-491 | Fix `github-action` `resolveAgents` — rate-limit error handling             |
| TASK-492 | Fix `release.sh` — rebuild `github-action/dist` and extension before commit |

---

## Task Detail

---

### TASK-483 — Implement `nexus.showProviders` command

**Status:** todo
**Priority:** critical
**Wave:** 1

**Problem:**
`vscode-extension/src/commands/index.ts` exports `showProvidersCommand()` as an empty function body.
`vscode-extension/src/extension.ts` registers the handler and immediately shows a toast that reads
`'showing providers coming in TASK-135'`. This command appears in the Command Palette and is
advertised in the README — shipping it in this state will embarrass the project on day one.

**Checklist:**

- [ ] In `nexusClient.ts`, add a `getProviders()` method that `GET /api/providers` and returns the
      validated provider list (unblock depends on TASK-486 Zod schemas, but can use raw cast
      initially and migrate when TASK-486 lands)
- [ ] In `commands/index.ts`, implement `showProvidersCommand(client: NexusClient)`:
      call `getProviders()`, map results to `vscode.QuickPickItem` entries with `label = name`,
      `description = kind`, and `detail = status` (active/inactive/error)
- [ ] Show the QuickPick with `vscode.window.showQuickPick()` — set `placeHolder` to
      `'Select a provider to view details'` and `matchOnDetail = true`
- [ ] On item selection, show an info message with the provider name and base URL; leave room for
      a future navigate-to-settings action via a `detail` button
- [ ] Update `extension.ts` to pass `nexusClient` instance into `showProvidersCommand` and remove
      the placeholder toast
- [ ] Handle the case where no providers are configured: show a warning message
      `'No providers configured. Add a provider in Settings.'` with a button that opens
      `nexus.openSettings`
- [ ] Add `nexus.showProviders` to the README command table with a one-line description

**Files to change:**

- `vscode-extension/src/commands/index.ts`
- `vscode-extension/src/extension.ts`
- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/README.md`

**Acceptance criteria:**

- Running `nexus.showProviders` from the Command Palette shows a populated QuickPick when the
  daemon is running with at least one provider
- Empty state shows a warning + settings button
- No toast mentioning `TASK-135` appears anywhere in the codebase

---

### TASK-484 — Fix `SessionMonitor.pollAndClaim` — tasks stuck in PROCESSING forever

**Status:** todo
**Priority:** critical
**Wave:** 1

**Problem:**
`vscode-extension/src/sessionMonitor.ts` `pollAndClaim()` polls for QUEUED tasks and calls
`claimTask()`. After claiming, the task transitions to PROCESSING — but the extension contains no
LLM execution path. The task remains PROCESSING indefinitely, blocking the daemon's worker queue
from picking up subsequent tasks.

**Decision:** Implement **Option A** (pre-launch safe fix): remove `pollAndClaim` from the
extension entirely. The extension is an orchestration UI, not an executor. Option B (claim only
tasks with `agentHint === 'vscode'` and execute via Copilot Chat API) is a future milestone
tracked in PLAN-future/TASK-vscode-executor.

**Checklist:**

- [ ] Delete the `pollAndClaim()` method from `SessionMonitor` class entirely
- [ ] Remove the `pollAndClaim` call from `startPolling()` or wherever it is invoked in the poll
      loop — the poll loop should only run `heartbeat()` and `detectAndRegister()`
- [ ] Remove the `claimTask()` import/call from `sessionMonitor.ts`; if `nexusClient.claimTask()`
      is used nowhere else, mark it with a `// reserved for future executor` comment but do not
      delete the client method (it may be used by tests or future code)
- [ ] Add a comment block above `SessionMonitor` explaining the scope boundary:
      `// SessionMonitor handles registration, heartbeat, and session metadata only.`
      `// Task execution is not performed by the extension (see PLAN-future TASK-vscode-executor).`
- [ ] Verify no tasks are left orphaned in PROCESSING state in existing integration tests or e2e
      fixtures — if any, add a startup cleanup call to `detectAndRegister` that un-claims stale
      PROCESSING tasks owned by the current session ID
- [ ] Update the CHANGELOG entry to note this behavioral change

**Files to change:**

- `vscode-extension/src/sessionMonitor.ts`
- `vscode-extension/CHANGELOG.md` (if present) or `CHANGELOG.md`

**Acceptance criteria:**

- The extension activates without claiming any tasks
- The daemon's worker queue processes tasks normally after extension is installed
- `pollAndClaim` does not appear anywhere in `sessionMonitor.ts`
- Existing `sessionMonitor` tests pass (add new tests in TASK-490)

---

### TASK-485 — Implement Marketplace publishing in CI pipeline

**Status:** todo
**Priority:** critical
**Wave:** 2

**Problem:**
`.github/workflows/publish.yml` only uploads the compiled `.vsix` as a GitHub Release artifact —
no `vsce publish` step exists. Users searching the VS Code Marketplace or Open VSX cannot find the
extension. The extension is functionally invisible.

**Checklist:**

- [ ] In `publish.yml`, add a `vsce publish` step after the existing artifact upload step,
      gated on `secrets.VSCE_PAT` being present; use
      `npx @vscode/vsce publish --pat ${{ secrets.VSCE_PAT }}` from the `vscode-extension/`
      working directory
- [ ] Add an `ovsx publish` step for Open VSX Registry using
      `npx ovsx publish *.vsix --pat ${{ secrets.OVSX_PAT }}`; gate this step similarly on the
      secret being present so the workflow degrades gracefully if only one secret is set
- [ ] Ensure both publish steps run only on tag pushes matching `v*` (not every commit)
- [ ] Add both `VSCE_PAT` and `OVSX_PAT` to the repository secret names documented in
      `vscode-extension/README.md` under a new "Release" section
- [ ] In `scripts/release.sh`, add a step that bumps `vscode-extension/package.json` `version`
      field to match the root `package.json` version being cut (use `jq` or `npm version` with
      `--no-git-tag-version`); place this step before the final `git commit`
- [ ] Verify the `publish.yml` trigger also fires on the `release` event (`types: [published]`)
      as a belt-and-suspenders alongside `push: tags`

**Files to change:**

- `.github/workflows/publish.yml`
- `scripts/release.sh`
- `vscode-extension/README.md`

**Acceptance criteria:**

- A release tag push triggers `vsce publish` and `ovsx publish` steps in CI
- Workflow succeeds with secrets set; degrades to a warning (not a failure) when secrets are absent
- `vscode-extension/package.json` version matches root version after running `release.sh`

---

### TASK-486 — Fix `nexusClient.ts` — remove blind JSON casts

**Status:** todo
**Priority:** critical
**Wave:** 2

**Problem:**
`vscode-extension/src/nexusClient.ts` generic `get<T>()`, `post<T>()`, and `put<T>()` methods use
`resp.json() as Promise<T>` — a bare TypeScript cast with no runtime shape validation. A backend
schema change (e.g. renaming `providerName` to `name`) silently propagates as `undefined` fields,
producing subtle UI bugs that show up as blank cards rather than errors.

**Checklist:**

- [ ] Add `zod` as a dependency in `vscode-extension/package.json` (or use hand-rolled guard
      functions if the team prefers zero new deps; confirm and document the choice)
- [ ] Define Zod schemas (or guard functions) for the four critical response types:
      `Task`, `AISession`, `Provider`, and `ProviderConfig` — derive these from the Go domain
      structs in `internal/core/domain/task.go` and `session.go`
- [ ] Add a private `parseResponse<T>(schema: ZodSchema<T>, data: unknown): T` helper method
      to `NexusClient` that calls `schema.parse(data)` and wraps `ZodError` in a descriptive
      `NexusClientError` with the endpoint path included in the message
- [ ] Replace the raw cast in `get<T>()`, `post<T>()`, `put<T>()` with the `parseResponse` call
      for the four critical types; less-critical internal types may retain the raw cast with a
      `// TODO: add schema` comment
- [ ] For array responses (e.g. `GET /api/sessions`, `GET /api/providers`), use
      `z.array(Schema).parse(data)` to validate each element
- [ ] Update `nexusClient` exports to surface the `NexusClientError` type so callers can
      distinguish parse errors from network errors

**Files to change:**

- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/package.json` (add `zod` dep if chosen)
- `vscode-extension/src/types.ts` (or equivalent types file) — export Zod-inferred types

**Acceptance criteria:**

- `getProviders()`, `getTasks()`, `getSessions()`, `getProviderConfigs()` throw `NexusClientError`
  with a meaningful message when the backend returns an unexpected shape
- No bare `as T` cast remains on any response path for the four critical types
- `npm run compile` and existing tests pass with no new type errors

---

### TASK-487 — Wire `nexus.mcpPort` / `nexus.enableMCPPortSweep` settings

**Status:** todo
**Priority:** critical
**Wave:** 2

**Problem:**
`vscode-extension/package.json` `contributes.configuration` declares `nexus.mcpPort` (default
`63988`) and `nexus.enableMCPPortSweep` (default `false`). Neither setting is read anywhere in
the extension source — they are phantom settings with zero effect. Users who change these will see
no behavior change and will file confusing bug reports.

**Checklist:**

- [ ] In `extension.ts`, read `nexus.mcpPort` via
      `vscode.workspace.getConfiguration('nexus').get<number>('mcpPort', 63988)` when constructing
      the `NexusClient` (or `MCPClient`) base URL; thread the port through the constructor
- [ ] In `nexusClient.ts`, change the hardcoded `63988` MCP port constant to accept the port as a
      constructor parameter with `63988` as the default
- [ ] Implement port sweep logic in `nexusClient.ts` behind a `tryConnect(ports: number[]): Promise<number>` method: probe each port in the list with a short `HEAD /health` or `GET /.well-known/nexus.json` request and return the first port that responds 200; throw if none respond
- [ ] In `extension.ts`, when `nexus.enableMCPPortSweep` is `true`, call `tryConnect` with the
      curated sweep list `[63988, 63987, 63989, 63990, 63986]` plus the user-configured
      `nexus.mcpPort` (deduplicated, user port tried first); use the resolved port for all
      subsequent client calls
- [ ] Listen for `vscode.workspace.onDidChangeConfiguration` and reconnect the client when either
      setting changes
- [ ] Add a manual sweep trigger: `nexus.reconnect` command that calls `tryConnect` and shows a
      status bar message with the discovered port

**Files to change:**

- `vscode-extension/src/extension.ts`
- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/src/commands/index.ts` (add `reconnect` command)
- `vscode-extension/package.json` (register `nexus.reconnect` command)

**Acceptance criteria:**

- Changing `nexus.mcpPort` in VS Code Settings causes the extension to reconnect to the new port
- With `nexus.enableMCPPortSweep: true`, the extension finds the daemon when it is running on any
  port in the sweep list without manual configuration
- No hardcoded `63988` literals remain in client construction paths

---

### TASK-488 — Fix `waitForCompletion` in `nexus-submit` — stuck on terminal states

**Status:** todo
**Priority:** critical
**Wave:** 1

**Problem:**
`cmd/nexus-submit/main.go` `waitForCompletion()` polls task status and only calls `return` when
status is `COMPLETED` or `FAILED`. The daemon also emits `CANCELLED`, `TOO_LARGE`, and
`NO_PROVIDER` as terminal states. A task in any of these states causes `nexus-submit` to silently
loop until its timeout expires, making CI pipelines appear hung.

**Checklist:**

- [ ] Audit `internal/core/domain/task.go` (or equivalent) for the complete set of terminal
      `TaskStatus` values; confirm the full list is
      `COMPLETED`, `FAILED`, `CANCELLED`, `TOO_LARGE`, `NO_PROVIDER`
- [ ] In `waitForCompletion()`, replace the two-value `switch`/`if` with a set-based terminal
      check: define `var terminalStates = map[string]string{...}` mapping each status to a
      human-readable exit message
- [ ] Log a descriptive message for each terminal state before returning, e.g.:
      `CANCELLED: task was cancelled before execution`,
      `TOO_LARGE: prompt exceeded provider context window`,
      `NO_PROVIDER: no compatible provider available for this task`
- [ ] Return a non-zero exit code for all error terminal states (`FAILED`, `CANCELLED`,
      `TOO_LARGE`, `NO_PROVIDER`) and exit code 0 only for `COMPLETED`
- [ ] Add a test in `cmd/nexus-submit/` (or `internal/` if testable) covering each terminal state
- [ ] Update the `nexus-submit` usage docs / `--help` output to list all terminal states

**Files to change:**

- `cmd/nexus-submit/main.go`
- `internal/core/domain/task.go` (read only, to confirm status values)

**Acceptance criteria:**

- `nexus-submit` exits immediately (within one poll interval) when a task reaches any terminal state
- Exit code is non-zero for all failure-class terminal states
- Log output identifies which terminal state caused the exit
- No timeout-loop behavior observable when task is `CANCELLED`

---

### TASK-489 — Add `workspaceOrchView` and `workspaceScanner` unit tests

**Status:** todo
**Priority:** high
**Wave:** 3

**Problem:**
`vscode-extension/src/workspaceOrchView.ts` `getChildren()` tree-building logic is completely
untested. `vscode-extension/src/workspaceScanner.ts` `parseOrchestratorFile()` is untested —
malformed or missing fields in `orchestrator.json` cause silent failures that surface as empty
tree views with no user feedback.

**Checklist:**

- [ ] Create `vscode-extension/src/test/workspaceOrchView.test.ts`; mock
      `vscode.workspace.workspaceFolders` and `workspaceScanner.scan()` using vitest/jest mocks
- [ ] Test case: empty workspace (no folders) → `getChildren()` returns `[]`
- [ ] Test case: folder with valid `orchestrator.json` → correct tree items with task counts
- [ ] Test case: folder with malformed `orchestrator.json` (missing `counters`, bad JSON) →
      `parseOrchestratorFile()` returns a safe default object and does not throw
- [ ] Test case: multi-folder workspace — both folders appear as root tree items; task counts
      are disambiguated per folder
- [ ] Create `vscode-extension/src/test/workspaceScanner.test.ts`; use `fs` mocks or temp files
      to simulate various `orchestrator.json` shapes
- [ ] Test `parseOrchestratorFile()` with: missing file, empty file, valid file, file with extra
      unknown fields (should be ignored), file where `counters.nextTaskId` is a string not a number

**Files to change:**

- `vscode-extension/src/test/workspaceOrchView.test.ts` (new)
- `vscode-extension/src/test/workspaceScanner.test.ts` (new)
- `vscode-extension/src/workspaceScanner.ts` (harden `parseOrchestratorFile` if needed)

**Acceptance criteria:**

- `npm test` in `vscode-extension/` passes all new tests
- `parseOrchestratorFile()` never throws on any file content — always returns a typed default
- Coverage for `workspaceOrchView.getChildren` and `workspaceScanner.parseOrchestratorFile`
  reaches ≥ 80 %

---

### TASK-490 — Add `sessionMonitor` unit tests

**Status:** todo
**Priority:** high
**Wave:** 3

**Problem:**
`vscode-extension/src/sessionMonitor.ts` is the most stateful and complex file in the extension;
it manages registration retries, heartbeat intervals, and deregistration on deactivation. It has
zero tests, making the TASK-484 refactor risky and future changes fragile.

**Checklist:**

- [ ] Create `vscode-extension/src/test/sessionMonitor.test.ts`
- [ ] Mock `nexusClient` with a vitest/jest spy; control which calls succeed or fail
- [ ] Test case: `detectAndRegister()` fails on first attempt then succeeds on second →
      session ID is stored and heartbeat starts
- [ ] Test case: `detectAndRegister()` fails 3 consecutive times → no session ID stored,
      no heartbeat timer running
- [ ] Test case: `start()` followed by `stop()` → heartbeat interval is cleared and
      `deregisterSession()` is called exactly once
- [ ] Test case: heartbeat call fails → a warning is logged, interval continues (does not crash)
- [ ] After TASK-484: test that `startPolling()` does NOT call `claimTask()` under any conditions
- [ ] Test timer management: multiple calls to `start()` without `stop()` do not leak intervals

**Files to change:**

- `vscode-extension/src/test/sessionMonitor.test.ts` (new)
- `vscode-extension/src/sessionMonitor.ts` (minor refactor to injectable clock/timer if needed
  for test isolation)

**Acceptance criteria:**

- All new tests pass under `npm test`
- Timer leak test confirms no duplicate intervals on repeated `start()` calls
- Post-TASK-484: `claimTask` spy is never called during normal monitor lifecycle

---

### TASK-491 — Fix `github-action` `resolveAgents` — rate-limit error handling

**Status:** todo
**Priority:** high
**Wave:** 4

**Problem:**
`github-action/src/agents.ts` `fetchCategoryIndex()` silently returns `[]` on a GitHub API 403 or
429 (rate-limit), or when the API returns an HTML error page instead of JSON. Downstream callers
report `Agent not found` which is a misleading error that causes confused support tickets.

**Checklist:**

- [ ] In `fetchCategoryIndex()`, check `response.status` after `fetch()`: if `403` or `429`,
      throw a `RateLimitError` (custom class extending `Error`) with message
      `'GitHub API rate limited (HTTP ${status}). Retry after ${retryAfter}s.'` — read
      `Retry-After` header when present
- [ ] If the response `Content-Type` is not `application/json` or the body is not a JSON array,
      throw a `ParseError` with the first 200 characters of the raw response body included so
      callers can diagnose HTML error pages
- [ ] Wrap `fetchCategoryIndex()` call site in a retry loop with exponential backoff:
      base delay 2 s, multiplier 2×, max 3 attempts, max delay 30 s; only retry on `RateLimitError`
- [ ] Export `RateLimitError` and `ParseError` from `agents.ts` so tests can assert on them
- [ ] Add tests in `github-action/__tests__/agents.test.ts`:
      — fetch returns 429 → `RateLimitError` thrown after 3 retries
      — fetch returns 200 with HTML → `ParseError` thrown immediately (no retry)
      — fetch returns 200 with valid JSON array → returns parsed agents
- [ ] Update the action's error output step to include a check for `RateLimitError` and set
      `core.setFailed('GitHub API rate limit exceeded. Try again later.')` instead of a generic
      agent-not-found message

**Files to change:**

- `github-action/src/agents.ts`
- `github-action/__tests__/agents.test.ts`
- `github-action/src/index.ts` or wherever the action error handler lives

**Acceptance criteria:**

- A mocked 429 response causes the action to retry 3 times then fail with
  `'GitHub API rate limit exceeded'`
- A mocked HTML response causes the action to fail immediately with `ParseError` and the
  response preview in the message
- All existing `agents.test.ts` tests continue to pass

---

### TASK-492 — Fix `release.sh` — rebuild `github-action/dist` and extension before commit

**Status:** todo
**Priority:** high
**Wave:** 4

**Problem:**
`scripts/release.sh` bumps version numbers but does NOT rebuild `github-action/dist/index.js` or
the VS Code extension compiled output before committing. After a version bump, the committed `dist`
bundle contains the previous version string, meaning the published GitHub Action runs old code
until a manual re-build.

**Checklist:**

- [ ] After the version sync block in `release.sh`, add:
      `cd github-action && npm ci && npm run build && cd ..`
      to rebuild `dist/index.js` with the new version baked in
- [ ] Add: `cd vscode-extension && npm ci && npm run compile && cd ..`
      to recompile the extension; note this is not the `.vsix` package step — that runs in CI
- [ ] Ensure both build steps run before `git add` and `git commit` so the rebuilt artifacts
      are included in the version-bump commit
- [ ] If either `npm run build` fails, `release.sh` should exit non-zero and print a clear error
      message — add `set -e` at the top of the script if not already present, and wrap the build
      steps with an `|| { echo "Build failed in $PWD"; exit 1; }`
- [ ] Smoke test: run `bash -n scripts/release.sh` to check for syntax errors; document the
      manual dry-run steps in a comment at the top of the script
- [ ] Add a GitHub Actions CI job that runs `scripts/release.sh --dry-run` (or equivalent) on
      PRs that touch `release.sh` to prevent broken release scripts from merging

**Files to change:**

- `scripts/release.sh`
- `.github/workflows/` (add or update a lint-release-script job)

**Acceptance criteria:**

- After running `release.sh`, `git diff --stat HEAD` shows updated `github-action/dist/index.js`
  and `vscode-extension/` compiled output alongside the version bump commits
- `release.sh` exits non-zero and prints a diagnostic message if either build step fails
- `bash -n scripts/release.sh` exits 0 (no syntax errors)

---

## Validation

All items below must be green before the extension is promoted to `latest` on the VS Code Marketplace:

- [ ] `nexus.showProviders` appears in the Command Palette and shows a populated QuickPick with
      at least one provider when the daemon is running
- [ ] No call to `claimTask` occurs during the extension's normal session lifecycle; daemon task
      queue processes freely
- [ ] A release tag push triggers `vsce publish` and `ovsx publish` steps in CI; both steps
      succeed with secrets set and degrade gracefully without them
- [ ] `nexusClient` throws `NexusClientError` with a descriptive message on backend shape mismatch
      for `Task`, `AISession`, `Provider`, `ProviderConfig` response types
- [ ] Changing `nexus.mcpPort` in VS Code Settings causes the active connection to reconnect
- [ ] `nexus-submit` exits non-zero within one poll interval for `CANCELLED`, `TOO_LARGE`, and
      `NO_PROVIDER` task states
- [ ] `npm test` in `vscode-extension/` passes with new `workspaceOrchView`, `workspaceScanner`,
      and `sessionMonitor` test files included
- [ ] A mocked 429 from GitHub API causes `github-action` to retry 3× then fail with
      `'GitHub API rate limit exceeded'`
- [ ] `git diff --stat HEAD` after `release.sh` includes rebuilt `github-action/dist/index.js`
- [ ] `vue-tsc --noEmit` zero errors (frontend unaffected)
- [ ] `CGO_ENABLED=1 go test -race ./...` all pass (backend unaffected)
- [ ] `go vet ./...` clean
