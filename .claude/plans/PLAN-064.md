# PLAN-064 — Test Coverage: Critical Path Hardening

**Status:** active
**Created:** 2026-04-04
**Author:** copilot

## Summary

Pre-launch audit identified systemic test coverage gaps across the Go backend, frontend composables, and GitHub Action. The core execution engine, primary frontend mutation surfaces, MCP state-machine transitions, and the CLI transport layer all have zero isolation coverage despite running on every user-visible flow. This plan addresses those gaps in priority order: critical zero-coverage production paths first, then medium-risk edge cases.

## Task Map

| Task     | Layer         | Priority | Description                                      |
| -------- | ------------- | -------- | ------------------------------------------------ |
| TASK-493 | Go / Core     | Critical | execution_engine.go unit tests                   |
| TASK-494 | Frontend      | Critical | useTasks composable tests                        |
| TASK-495 | Frontend      | Critical | useGlobalSSE composable tests                    |
| TASK-496 | Frontend      | Critical | useAISessions composable tests                   |
| TASK-497 | Go / MCP      | Critical | claim_task → update_task_status integration test |
| TASK-498 | Go / Core     | Critical | Stale-task watchdog unit test                    |
| TASK-499 | Go / Outbound | Critical | httpapi_client test suite                        |
| TASK-500 | Frontend      | Critical | SettingsView + BacklogView component tests       |
| TASK-501 | Go / Core     | Medium   | Execution engine provider failover test          |
| TASK-502 | Frontend      | Medium   | MissionControlView interaction tests             |
| TASK-503 | GitHub Action | Medium   | Full-flow integration test with mock daemon      |
| TASK-504 | E2E           | Medium   | E2E smoke test hardening                         |

## Waves

### Wave 1 — Zero-coverage Core Backend (TASK-493, TASK-498, TASK-499)

| Task     | Description                                                                                         |
| -------- | --------------------------------------------------------------------------------------------------- |
| TASK-493 | execution_engine.go: provider selection, token estimation, chat vs generate branching, output write |
| TASK-498 | Stale-task watchdog: PROCESSING task re-queued/failed after timeout                                 |
| TASK-499 | httpapi_client table-driven tests via httptest.NewServer                                            |

### Wave 2 — Zero-coverage Frontend Composables (TASK-494, TASK-495, TASK-496)

| Task     | Description                                                          |
| -------- | -------------------------------------------------------------------- |
| TASK-494 | useTasks: initial fetch, SSE updates, cancel, promote, error         |
| TASK-495 | useGlobalSSE: connect, route events, exponential backoff, disconnect |
| TASK-496 | useAISessions: fetch, deregister, heartbeat interval, SSE, error     |

### Wave 3 — Zero-coverage MCP State Machine & Views (TASK-497, TASK-500)

| Task     | Description                                                           |
| -------- | --------------------------------------------------------------------- |
| TASK-497 | MCP claim_task → exclusive ownership → update_task_status integration |
| TASK-500 | SettingsView + BacklogView component mount tests                      |

### Wave 4 — Medium-risk Gaps (TASK-501, TASK-502, TASK-503, TASK-504)

| Task     | Description                                                        |
| -------- | ------------------------------------------------------------------ |
| TASK-501 | Provider failover on primary Chat() error                          |
| TASK-502 | MissionControlView: submit form, cancel, SSE update, promote       |
| TASK-503 | GitHub Action full-flow with mock nexus HTTP API                   |
| TASK-504 | E2E: SSE content verification, concurrent submission, skip markers |

---

## Task Details

---

### TASK-493: Add execution_engine.go unit tests

**Status:** todo
**Plan:** PLAN-064
**Wave:** 1
**Priority:** 1

**Description:**
`internal/core/services/execution_engine.go` contains every concrete decision made during task execution: provider selection by hint, chat-vs-generate branching, token estimation, code extraction, and file output. Despite this, it has zero isolation tests. Every execution path in production runs through it, making this the highest-risk untested file in the codebase.

**Checklist:**

- [ ] Create `internal/core/services/execution_engine_test.go`
- [ ] Test `selectProviderForTask`: matching hint selects correct provider; non-matching hint falls back; no hint uses first available provider
- [ ] Test `buildChatContext`: empty session history produces one system message; populated session history is prepended in order; project path appears in system message
- [ ] Test `executeGeneration`: mock `LLMClient.Chat()` returns response → result captured; mock `Chat()` returns error → fallback to `GenerateCode()` attempted; both fail → error returned
- [ ] Test `estimateTokens`: known ASCII string returns expected estimate; empty string returns 0; non-ASCII characters handled without panic
- [ ] Test `extractCode`: input with fenced ``` block returns only inner content; input without code block returned unchanged; multiple code blocks returns first block only
- [ ] Test `writeTaskOutput`: absolute path outside project root is rejected (traversal guard); valid relative path writes file; parent directories created if missing
- [ ] Use mock structs matching the style in `orchestrator_test.go` (implement `ports.LLMClient`, `ports.FileWriter` interfaces inline)
- [ ] All tests pass under `CGO_ENABLED=1 go test -race ./internal/core/services/...`

**Files:**

- `internal/core/services/execution_engine_test.go` (create)
- `internal/core/services/execution_engine.go` (reference only)

**Acceptance Criteria:**

- `execution_engine_test.go` exists with minimum 8 test cases
- Each exported/unexported function named above has ≥1 happy-path and ≥1 error-path test
- `go test -race ./internal/core/services/...` exits 0 with no data-race warnings
- No production code is modified to accommodate tests (use interface mocks only)

---

### TASK-494: Add useTasks composable tests

**Status:** todo
**Plan:** PLAN-064
**Wave:** 2
**Priority:** 2

**Description:**
`frontend/src/composables/useTasks.ts` is the primary task mutation surface for the entire GUI. It provides the `tasks` ref, `submitTask`, `cancelTask`, and `promoteTask` mutations consumed by MissionControlView, BacklogView, and HistoryView. It has zero tests. Functional regressions here silently break the entire task lifecycle in the UI.

**Checklist:**

- [ ] Create `frontend/src/composables/__tests__/useTasks.test.ts`
- [ ] Test initial fetch: `fetchTasks()` called on mount; `tasks.value` populated from mocked API response; `isLoading` set false after resolution
- [ ] Test SSE event: when a `task-updated` SSE event is received, the matching task in `tasks.value` is updated in place without a full re-fetch
- [ ] Test `cancelTask`: correct HTTP DELETE (or Wails call) is triggered with the task ID; task removed from `tasks.value` on success; `error.value` set on 500 response
- [ ] Test `promoteTask`: correct API call made with task ID; task status updated to `QUEUED` in `tasks.value`; error path sets `error.value`
- [ ] Test error path for initial fetch: API returns 500 → `error.value` is set with message; `tasks.value` remains empty array
- [ ] Mock Wails runtime (`window.go.*`) using `vi.mock` or `vi.fn()` stubs, matching style in `useActivities.test.ts`
- [ ] All tests pass with `pnpm vitest run`

**Files:**

- `frontend/src/composables/__tests__/useTasks.test.ts` (create)
- `frontend/src/composables/useTasks.ts` (reference)
- `frontend/src/composables/__tests__/useActivities.test.ts` (reference for mock style)

**Acceptance Criteria:**

- Minimum 5 test cases covering the scenarios above
- No modification to `useTasks.ts` required (test against public API surface)
- `pnpm vitest run` exits 0 with all new tests green

---

### TASK-495: Add useGlobalSSE composable tests

**Status:** todo
**Plan:** PLAN-064
**Wave:** 2
**Priority:** 2

**Description:**
`frontend/src/composables/useGlobalSSE.ts` is the real-time event bus for the entire application. Task updates, activity feeds, and session changes all arrive through it. It has zero tests. A regression here would silently freeze all live-update views without any error surfacing.

**Checklist:**

- [ ] Create `frontend/src/composables/__tests__/useGlobalSSE.test.ts`
- [ ] Test connect on mount: `EventSource` constructor called with correct URL; `onmessage` handler registered
- [ ] Test event routing: message event with `type: "task"` dispatched to all `task` subscribers; message with `type: "activity"` dispatched to `activity` subscribers only; unrecognised type is silently ignored
- [ ] Test subscriber cleanup: handler added via `onEvent("task", fn)` then removed; after removal, `fn` is not called for subsequent `task` events
- [ ] Test disconnect on unmount: `EventSource.close()` called when component using composable is unmounted; no further events processed after close
- [ ] Test reconnect with exponential backoff: `EventSource.onerror` triggered; reconnect attempted after initial delay; second failure doubles delay; backoff capped at max configured value
- [ ] Mock `window.EventSource` with a `vi.fn()` factory returning a controllable stub
- [ ] All tests pass with `pnpm vitest run`

**Files:**

- `frontend/src/composables/__tests__/useGlobalSSE.test.ts` (create)
- `frontend/src/composables/useGlobalSSE.ts` (reference)

**Acceptance Criteria:**

- Minimum 6 test cases covering the scenarios above
- `EventSource` is fully mocked — no real network connections in tests
- `pnpm vitest run` exits 0 with all new tests green

---

### TASK-496: Add useAISessions composable tests

**Status:** todo
**Plan:** PLAN-064
**Wave:** 2
**Priority:** 2

**Description:**
`frontend/src/composables/useAISessions.ts` manages the AI session lifecycle surface in the GUI — listing sessions, deregistering them, and sending heartbeats. It has zero tests. Failures here would silently prevent the AgentsView from reflecting live session state.

**Checklist:**

- [ ] Create `frontend/src/composables/__tests__/useAISessions.test.ts`
- [ ] Test fetch on mount: `sessions.value` populated from mocked API response; `isLoading` transitions false→true→false
- [ ] Test `deregisterSession`: correct Wails/API call made with session ID; session removed from `sessions.value` on success
- [ ] Test heartbeat interval: `vi.useFakeTimers()`; advance clock by heartbeat interval; verify heartbeat call issued once per interval; verify no duplicate calls
- [ ] Test SSE update: receiving a `session-updated` event with a known session ID updates the matching entry in `sessions.value`
- [ ] Test error state on fetch failure: API throws → `error.value` is non-null; `sessions.value` remains `[]`
- [ ] All tests pass with `pnpm vitest run`

**Files:**

- `frontend/src/composables/__tests__/useAISessions.test.ts` (create)
- `frontend/src/composables/useAISessions.ts` (reference)

**Acceptance Criteria:**

- Minimum 5 test cases covering the scenarios above
- Fake timers used for heartbeat interval test (no real timers/sleeps)
- `pnpm vitest run` exits 0 with all new tests green

---

### TASK-497: Add MCP claim_task → update_task_status integration test

**Status:** todo
**Plan:** PLAN-064
**Wave:** 3
**Priority:** 1

**Description:**
`internal/adapters/inbound/mcp/tools_test.go` uses a mock orchestrator (`toolHarnessOrch`) that returns zero-value `domain.Task` structs. This means no existing test can verify exclusive ownership semantics (only the claiming session may update a task) or that the QUEUED → PROCESSING → COMPLETED state machine transition is actually persisted. A real SQLite-backed integration test is required.

**Checklist:**

- [ ] Create or extend `internal/adapters/inbound/mcp/integration_test.go`
- [ ] Wire up a real SQLite repo + OrchestratorService (matching the pattern in `internal/e2e/` or existing `integration_test.go`)
- [ ] Test happy path: create task in QUEUED state; call `claim_task` via MCP tool handler; verify task status is PROCESSING in DB; call `update_task_status` COMPLETED with correct session_id; verify task status is COMPLETED in DB and `logs` field saved
- [ ] Test exclusive ownership: after first `claim_task`, second `claim_task` from different session returns error or `task not QUEUED`; task status remains PROCESSING (not double-claimed)
- [ ] Test ownership enforcement on update: call `update_task_status` with wrong session_id; verify it is rejected; task status unchanged
- [ ] Use `t.TempDir()` for ephemeral SQLite DB path; no shared DB between test runs
- [ ] Test runs under `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/...`

**Files:**

- `internal/adapters/inbound/mcp/integration_test.go` (create or extend)
- `internal/adapters/inbound/mcp/tools.go` (reference)
- `internal/core/services/orchestrator.go` (reference)
- `internal/adapters/outbound/repo_sqlite/` (dependency)

**Acceptance Criteria:**

- 3 integration test cases (happy path, double-claim, wrong-session update)
- All assertions query the real SQLite DB, not a mock return value
- `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/...` exits 0

---

### TASK-498: Add stale-task watchdog test

**Status:** todo
**Plan:** PLAN-064
**Wave:** 1
**Priority:** 1

**Description:**
The PROCESSING task watchdog in `OrchestratorService` calls `GetStaleProcessing` and re-queues or fails tasks that have been stuck too long. The `memRepo` stub used in existing orchestrator tests always returns nil for `GetStaleProcessing`, meaning this entire recovery path has never been exercised. A single missed watchdog cycle could leave tasks wedged in PROCESSING forever in production.

**Checklist:**

- [ ] Create `internal/core/services/orchestrator_hardening_test.go` (or add to existing hardening file if present)
- [ ] Implement a test `memRepo` method for `GetStaleProcessing` that returns a pre-seeded PROCESSING task with `UpdatedAt` set to `time.Now().Add(-staleness_threshold - 1s)`
- [ ] If staleness threshold is a hardcoded constant, expose it as a package-level `var StaleThreshold` or accept it as an option so tests can inject a short timeout (e.g. 100ms)
- [ ] Test: create one PROCESSING task older than threshold; start orchestrator with short tick interval; wait 2 tick cycles; assert task status is no longer PROCESSING (either QUEUED retry or FAILED depending on implementation)
- [ ] Test: PROCESSING task younger than threshold is NOT re-queued during same window
- [ ] Confirm no data race under `-race` flag
- [ ] If threshold exposure requires a small production code change, make it an unexported `var` with `//nolint:gochecknoglobals` comment to keep impact minimal

**Files:**

- `internal/core/services/orchestrator_hardening_test.go` (create)
- `internal/core/services/orchestrator.go` (may require minor threshold exposure)
- `internal/core/ports/ports.go` (reference for `GetStaleProcessing` signature)

**Acceptance Criteria:**

- 2 test cases: stale task re-queued/failed; fresh task unaffected
- No mock `GetStaleProcessing` returns nil; tests use non-trivial fixture data
- `CGO_ENABLED=1 go test -race ./internal/core/services/...` exits 0

---

### TASK-499: Add httpapi_client test suite (Go)

**Status:** todo
**Plan:** PLAN-064
**Wave:** 1
**Priority:** 1

**Description:**
`internal/adapters/outbound/httpapi_client/` is the entire network transport layer for the CLI binary. Every `nexus-cli` command eventually calls through this package. It has zero tests. Silent breakage here would silently fail all CLI users without any test failure catching it during development.

**Checklist:**

- [ ] Create `internal/adapters/outbound/httpapi_client/client_test.go`
- [ ] Use `httptest.NewServer` with a `http.ServeMux` to simulate nexus HTTP API responses for each endpoint
- [ ] Table-driven tests for `SubmitTask`: 201 Created returns populated task; 400 returns error with message; 500 returns error
- [ ] Table-driven tests for `GetTask`: 200 returns task; 404 returns typed "not found" error; malformed JSON body returns parse error
- [ ] Tests for `CancelTask`: 204 returns nil; 404 returns error
- [ ] Tests for provider config CRUD: `CreateProviderConfig`, `ListProviderConfigs`, `UpdateProviderConfig`, `DeleteProviderConfig` each with 2xx success and error paths
- [ ] Tests for AI session lifecycle: `RegisterSession` (201), `HeartbeatSession` (200), `DeregisterSession` (204)
- [ ] Test that the injected HTTP client is used (not `http.DefaultClient`): configure test server to close connection immediately; use a custom transport that records calls; verify requests reach the test server, not a real host
- [ ] Test `UpdateRuntimeConfig` if the method exists

**Files:**

- `internal/adapters/outbound/httpapi_client/client_test.go` (create)
- `internal/adapters/outbound/httpapi_client/client.go` (reference)

**Acceptance Criteria:**

- Minimum 15 table-driven test cases across all methods
- All requests verified against `httptest.Server` — no real network traffic
- Injected client transport verifiably used (transport-level interception test)
- `go test ./internal/adapters/outbound/httpapi_client/...` exits 0 (no CGO required)

---

### TASK-500: Add SettingsView and BacklogView component tests

**Status:** todo
**Plan:** PLAN-064
**Wave:** 3
**Priority:** 2

**Description:**
`SettingsView.vue` controls token rotation, provider config, and queue caps — mutations with immediate live impact on the running daemon. `BacklogView.vue` is the primary backlog management surface. Both have zero tests. UI regressions in these views would be invisible until manual QA.

**Checklist:**

- [ ] Create `frontend/src/views/__tests__/SettingsView.spec.ts`
- [ ] Test settings form submit: fill fields, click Save; verify correct Wails/API call made with form values; success toast shown
- [ ] Test token rotate: click Rotate Token button; verify API call triggered; new token displayed in field (or copied to clipboard per implementation)
- [ ] Test validation: submit with empty required field; verify error message shown and API not called
- [ ] Create `frontend/src/views/__tests__/BacklogView.spec.ts`
- [ ] Test backlog load: on mount, `fetchBacklog()` called; backlog items rendered in list
- [ ] Test promote from backlog: click Promote button on a backlog item; verify `promoteTask()` called with correct ID; item moves out of backlog list (status changes)
- [ ] Test error state: API failure on fetch → error banner shown
- [ ] Use `mount` (not `shallowMount`) with a router stub and a `createTestingPinia` store as needed
- [ ] All tests pass with `pnpm vitest run`

**Files:**

- `frontend/src/views/__tests__/SettingsView.spec.ts` (create)
- `frontend/src/views/__tests__/BacklogView.spec.ts` (create)
- `frontend/src/views/SettingsView.vue` (reference)
- `frontend/src/views/BacklogView.vue` (reference)

**Acceptance Criteria:**

- Minimum 3 test cases for SettingsView, 3 for BacklogView
- `mount` used (not shallow) so child component rendering is validated
- `pnpm vitest run` exits 0 with all new tests green

---

### TASK-501: Add execution engine provider failover test

**Status:** todo
**Plan:** PLAN-064
**Wave:** 4
**Priority:** 3

**Description:**
No test exercises the scenario where the first available LLM provider fails during `Chat()` or `GenerateCode()` and a second provider must be tried. This failover path is critical for reliability in multi-provider production deployments but is entirely untested.

**Checklist:**

- [ ] Add test to `internal/core/services/execution_engine_test.go` (from TASK-493) or create `execution_engine_failover_test.go`
- [ ] Register two mock `LLMClient` providers: first returns `errors.New("timeout")` on every call; second returns a valid response
- [ ] Submit a task with no provider hint so both are eligible
- [ ] Assert task completes successfully using the second provider's response
- [ ] Assert the first provider's `Chat()` was called exactly once (not retried)
- [ ] Assert the second provider's `Chat()` was called exactly once
- [ ] Add a second sub-test: both providers fail → task status set to FAILED with aggregated error message

**Files:**

- `internal/core/services/execution_engine_test.go` (extend) or `execution_engine_failover_test.go` (create)

**Acceptance Criteria:**

- 2 test cases: single-provider-fail → fallback succeeds; all-providers-fail → FAILED status
- Mock call counters verified (first provider called once, not repeatedly)
- `CGO_ENABLED=1 go test -race ./internal/core/services/...` exits 0

---

### TASK-502: Add MissionControlView interaction tests

**Status:** todo
**Plan:** PLAN-064
**Wave:** 4
**Priority:** 3

**Description:**
`MissionControlView.spec.ts` currently contains only 2 tests verifying render and header text. The submit form, cancel action, status filter, SSE event integration, and promote action are completely untested. This view is the primary operational control surface of the application.

**Checklist:**

- [ ] Extend `frontend/src/views/__tests__/MissionControlView.spec.ts`
- [ ] Test submit form: fill in project path + instruction fields; click Submit; verify `submitTask()` (Wails call or composable method) invoked with correct arguments; new task appears in list
- [ ] Test cancel click: render view with one task; click Cancel button for that task; verify `cancelTask(taskId)` called; task removed from displayed list
- [ ] Test status filter: render with tasks in QUEUED, PROCESSING, COMPLETED states; select filter "PROCESSING"; verify only PROCESSING task visible
- [ ] Test SSE event updates task list: inject a `task-updated` event via the mocked global SSE bus; verify the task card reflects updated status without page reload
- [ ] Test promote action: render a DRAFT task; click Promote; verify `promoteTask(taskId)` called; task status updated in view

**Files:**

- `frontend/src/views/__tests__/MissionControlView.spec.ts` (extend)
- `frontend/src/views/MissionControlView.vue` (reference)

**Acceptance Criteria:**

- At least 5 new test cases added (submit, cancel, filter, SSE update, promote)
- Total test count in `MissionControlView.spec.ts` ≥ 7
- `pnpm vitest run` exits 0 with all tests green

---

### TASK-503: Add GitHub Action full-flow integration test with mock daemon

**Status:** todo
**Plan:** PLAN-064
**Wave:** 4
**Priority:** 3

**Description:**
`github-action/__tests__/index.test.ts` never exercises a real nexus API shape. All network calls are either skipped or mocked at too high a level to catch HTTP contract regressions. A proper integration test must stand up a mock HTTP server that speaks the real nexus API response schema and verify the action handles all terminal states.

**Checklist:**

- [ ] Create `github-action/__tests__/integration.test.ts`
- [ ] Spin up a Node `http.createServer` (or use `msw`) serving the nexus API shape: `POST /api/tasks` → 201 with task stub; `GET /api/tasks/:id` → 200 with progressively updated status per poll
- [ ] Test success flow: action submits task, polls until COMPLETED, exits 0, outputs task ID as action output
- [ ] Test timeout flow: mock server never returns COMPLETED within configured timeout; action exits with timeout error and non-zero code
- [ ] Test NO_PROVIDER terminal state: mock server returns task with status `FAILED` and error `"no provider available"`; action exits with descriptive error message
- [ ] Test network error: mock server closes connection on first GET; action retries at least once before failing
- [ ] Verify action output variables are set correctly on success (`task-id`, `result`)
- [ ] Server torn down in `afterAll` to prevent port leaks

**Files:**

- `github-action/__tests__/integration.test.ts` (create)
- `github-action/src/` (reference for action entry point)

**Acceptance Criteria:**

- 4 test cases (success, timeout, NO_PROVIDER, network error)
- Mock HTTP server uses real nexus API JSON shapes (not hand-waved stubs)
- `pnpm test` in `github-action/` exits 0 with all new tests green
- No real daemon required to run the tests

---

### TASK-504: E2E smoke test hardening

**Status:** todo
**Plan:** PLAN-064
**Wave:** 4
**Priority:** 3

**Description:**
`scripts/e2e-smoke.sh` and `internal/e2e/` tests currently verify basic HTTP response codes but do not inspect SSE event content, do not test auth token enforcement end-to-end, do not test concurrent multi-task submission, and have no coverage of provider failover at the system level. These gaps mean a significant class of integration regressions could pass smoke tests undetected.

**Checklist:**

- [ ] Add SSE event content verification to `internal/e2e/` or `scripts/e2e-smoke.sh`: subscribe to SSE stream; submit a task; assert at least one `task` event received containing the submitted task ID; assert event JSON is well-formed
- [ ] Add concurrent submission test: submit 3 tasks in parallel via goroutines (or background curl); assert all 3 appear in `GET /api/tasks` with unique IDs; assert no task ID collisions
- [ ] Add auth token enforcement test: if auth is configured, send a request with invalid/missing token; assert 401 or 403 response; assert no task created
- [ ] Add a placeholder test for real-LLM provider failover with a `t.Skip("requires live LLM providers")` marker and a comment documenting the expected behaviour and how to run manually
- [ ] Ensure all new e2e tests are guarded by a build tag `//go:build e2e` so they do not run in unit test CI by default
- [ ] Update `scripts/e2e-smoke.sh` header comment to document new test coverage

**Files:**

- `internal/e2e/smoke_test.go` (extend) or `internal/e2e/sse_test.go` (create)
- `scripts/e2e-smoke.sh` (extend)

**Acceptance Criteria:**

- SSE event content verified in at least one automated test
- Concurrent 3-task submission test present and non-flaky
- Real-LLM skip marker present with explanatory comment
- `CGO_ENABLED=1 go test -tags e2e ./internal/e2e/...` passes against a running daemon

---

## Validation

All tasks completed when:

- `CGO_ENABLED=1 go test -race ./...` exits 0 with no skipped test warnings on non-e2e packages
- `pnpm vitest run` (in `frontend/`) exits 0 with ≥ 30 additional passing test cases across TASK-494 through TASK-496, TASK-500, TASK-502
- `pnpm test` (in `github-action/`) exits 0 including new integration tests
- `go vet ./...` clean — no production code changes introduce new vet warnings
- `vue-tsc --noEmit` clean — no type errors introduced by test stubs
- Zero new `// TODO` comments without an associated task reference
