# Release Readiness Audit — 2026-03-13

## Scope

This audit inspected runtime orchestration, HTTP and MCP contracts, Wails and frontend bindings, VS Code extension integration, CI and publish gates, task metadata, and explicit TODO and placeholder paths.

The audit was cross-checked with parallel subagent passes using architecture, evidence, and integration-review personas. The live orchestrator was reachable during the audit and this audit session was registered with it.

## Ship Status

Current assessment: not ship-ready without another hardening pass.

The repository has a substantially better release gate than before, but it still contains verified runtime correctness gaps, dead-end UX paths, stale release claims, and weakly enforced coverage boundaries.

## Verified Critical Findings

### 1. Queue truth is split between SQLite and in-memory state

- Files:
  - `internal/core/services/orchestrator.go`
  - `internal/adapters/outbound/repo_sqlite/repo.go`
  - `README.md`
- Problem:
  - Startup recovery only rehydrates `PROCESSING` tasks.
  - `QUEUED` tasks stored in SQLite are not reconstructed into the in-memory queue on restart.
  - `CancelTask` only removes tasks from the in-memory queue, not from persisted pending state generally.
- Why this matters:
  - A crash or inconsistent transition can leave persisted `QUEUED` work that no worker will ever process.
  - This breaks the crash-recovery claim and can strand tasks permanently.
- Required remediation:
  - Make persisted pending tasks authoritative and reconstruct from all executable pending rows, or remove the separate in-memory queue entirely.

### 2. Task admission rules are bypassable

- Files:
  - `internal/core/services/orchestrator.go`
  - `internal/adapters/inbound/httpapi/server.go`
  - `scripts/e2e-smoke.sh`
- Problem:
  - `SubmitTask` enforces queue cap and execute-after-plan.
  - `PromoteTask` and `UpdateTask` can move work into `QUEUED` without going through the same admission rules.
  - `UpdateTask` can set `QUEUED` without enqueueing work for processing.
- Why this matters:
  - The public lifecycle allows states that look valid in storage but are not actually executable.
  - Queue cap and plan prerequisite can be bypassed.
- Required remediation:
  - Centralize all transitions to `QUEUED` through one atomic path that validates policy and appends work to the runnable queue.

### 3. Artifact publish pipeline is materially weaker than PR CI

- Files:
  - `.github/workflows/ci.yml`
  - `.github/workflows/publish.yml`
- Problem:
  - PR CI runs vet, lint, Go tests, frontend smoke coverage, extension smoke coverage, and daemon E2E.
  - Publish only reruns Go race tests before building release artifacts.
- Why this matters:
  - The exact commit producing release artifacts is not guarded by the same checks that validated a pull request.
  - Direct pushes to `main` or manual publishes can skip critical release checks.
- Required remediation:
  - Make publish depend on a reusable full-gate workflow or add equivalent release prerequisites before any artifact build.

### 4. Provider promotion is not durably enabled

- Files:
  - `internal/core/services/orchestrator.go`
  - `main.go`
  - `cmd/nexus-daemon/main.go`
- Problem:
  - `PromoteProvider` creates a `ProviderConfig` without setting `Enabled: true`.
  - Startup only reloads enabled provider configs.
- Why this matters:
  - A provider can appear promoted during the current process and disappear after restart.
- Required remediation:
  - Persist promoted providers as enabled and add a restart-level regression test.

### 5. Session history is advanced independently of durable output persistence

- Files:
  - `internal/core/services/orchestrator.go`
  - `internal/adapters/outbound/repo_sqlite/session_repo.go`
- Problem:
  - Chat history append happens in two separate calls.
  - File write failure can leave session history advanced even when task completion fails.
- Why this matters:
  - Later tasks may consume conversation history that does not correspond to any durable writeback.
- Required remediation:
  - Persist task result and session changes transactionally, or append session history only after successful output persistence.

## Verified High Findings

### 6. Wails and HTTP provider-config contracts disagree on secret handling

- Files:
  - `app.go`
  - `internal/adapters/inbound/httpapi/server.go`
  - `frontend/src/views/SettingsView.vue`
  - `frontend/src/components/ProviderConfigForm.vue`
- Problem:
  - HTTP responses mask `apiKey`.
  - Wails returns raw `ProviderConfig` values directly.
- Why this matters:
  - Desktop mode exposes plaintext secrets into the renderer while browser mode does not.
  - The same feature has two incompatible contracts.
- Required remediation:
  - Introduce a shared safe DTO for all inbound adapters and handle secret updates through an explicit write-only flow.

### 7. `UpdateTask` accepts fields that the repository silently drops

- Files:
  - `internal/core/services/orchestrator.go`
  - `internal/adapters/outbound/repo_sqlite/repo.go`
- Problem:
  - Service merges `ModelID` and `ProviderHint` updates.
  - `Repository.Update` does not persist those fields.
- Why this matters:
  - Routing updates appear accepted and are then lost on reload.
- Required remediation:
  - Persist every mutable field the service allows and add round-trip tests.

### 8. Backlog UX is incomplete and misleading

- Files:
  - `frontend/src/components/TaskSubmitForm.vue`
  - `internal/core/services/orchestrator.go`
- Problem:
  - UI exposes “Add to Backlog”.
  - `CreateDraft` always creates `DRAFT`, not `BACKLOG`.
- Why this matters:
  - Users are told they created a backlog item, but the public API creates drafts only.
- Required remediation:
  - Either support explicit backlog creation or rename the UI and workflow to reflect actual behavior.

### 9. Desktop close-to-tray flow is broken on macOS

- Files:
  - `main.go`
  - `internal/adapters/inbound/tray/tray.go`
- Problem:
  - Desktop close is intercepted and the user is told the app hides to tray.
  - Tray adapter is explicitly a no-op.
- Why this matters:
  - The app promises a tray behavior it does not implement.
- Required remediation:
  - Either implement a real tray path or stop advertising hide-to-tray and remove the close interception.

### 10. The core service still owns goroutine lifecycle

- Files:
  - `internal/core/services/orchestrator.go`
  - `.github/copilot-instructions.md`
- Problem:
  - `NewOrchestrator` starts worker and cleanup goroutines directly.
- Why this matters:
  - This violates the project’s own architectural rule that core services should not own goroutine lifecycle.
- Required remediation:
  - Move worker ownership into an inbound runner/adapter layer and keep service construction side-effect free.

### 11. Frontend browser fallback still parses submit response incorrectly

- Files:
  - `frontend/src/types/wails.ts`
  - `internal/adapters/inbound/httpapi/server.go`
- Problem:
  - Browser fallback expects `{ id }`.
  - HTTP API returns `{ task_id, status }`.
- Why this matters:
  - Browser-mode submission can succeed while frontend code gets an undefined task ID.
- Required remediation:
  - Align fallback parsing with the HTTP contract and add a focused non-Wails frontend test.

### 12. Discovery UI is only partially wired

- Files:
  - `frontend/src/views/ProvidersView.vue`
  - `frontend/src/views/DiscoveryView.vue`
  - `frontend/src/composables/useDiscovery.ts`
  - `internal/adapters/inbound/httpapi/hub.go`
- Problem:
  - Promote actions are console-only TODOs.
  - Frontend listens for `provider_discovered` SSE, but backend never emits it.
- Why this matters:
  - Discovery appears interactive but promotion and live-update behavior are incomplete.
- Required remediation:
  - Wire promote actions to the real backend and emit a real discovery event or remove the dead listener.

### 13. README and getting-started are stale against runtime

- Files:
  - `README.md`
  - `docs/getting-started.md`
  - `internal/adapters/inbound/mcp/server.go`
  - `internal/adapters/inbound/httpapi/server.go`
- Problem:
  - README still advertises 14 MCP tools and system tray behavior.
  - Getting-started shows the wrong `POST /api/tasks` response shape.
  - Example provider kind uses `openai-compat` while domain/runtime use `openaicompat`.
- Why this matters:
  - External users can follow the docs and still break against the live product.
- Required remediation:
  - Regenerate release-facing docs and examples from the live contract.

## Verified Medium Findings

### 14. VS Code extension still contains a placeholder for a supposedly completed task

- Files:
  - `vscode-extension/src/extension.ts`
  - `vscode-extension/src/commands/index.ts`
  - `.claude/orchestrator.json`
- Problem:
  - `nexus.showProviders` still shows “coming in TASK-135”.
  - `showProvidersCommand` is still a placeholder.
  - `.claude` metadata marks `TASK-135` done.
- Why this matters:
  - Shipping code and project metadata disagree about what is complete.
- Required remediation:
  - Either implement the command or re-open the task and remove the stale completion claim.

### 15. `.claude` orchestration metadata is still inconsistent

- Files:
  - `.claude/orchestrator.json`
- Problem:
  - `activePlanId` still points at completed `PLAN-035`.
  - Completed `PLAN-002` still contains `todo` tasks like `TASK-014`, `TASK-021`, and `TASK-022`.
- Why this matters:
  - The internal planning system cannot be trusted as a source of current truth.
- Required remediation:
  - Run a metadata reconciliation pass and enforce invariant checks in tooling.

### 16. VS Code extension TypeScript config is not aligned with editor/runtime globals

- Files:
  - `vscode-extension/tsconfig.json`
  - `vscode-extension/src/statusBar.ts`
- Problem:
  - Workspace diagnostics report missing `NodeJS`, `setInterval`, `clearInterval`, and `console` globals.
- Why this matters:
  - Build may pass through esbuild, but the editor type surface is inconsistent and can hide real mistakes.
- Required remediation:
  - Add correct libs and types for the extension environment and keep editor diagnostics clean.

### 17. Frontend enum drift still exists in provider kind values

- Files:
  - `frontend/src/components/ProviderConfigForm.vue`
  - `frontend/src/types/domain.ts`
  - `internal/core/domain/provider.go`
- Problem:
  - Frontend offers `openai` while backend factory supports `openaicompat`.
- Why this matters:
  - The form allows a value the backend does not actually recognize.
- Required remediation:
  - Use the backend enum exactly or introduce an explicit alias translation layer.

### 18. Settings page documents a queue-cap env var that entrypoints do not read

- Files:
  - `frontend/src/views/SettingsView.vue`
  - `main.go`
  - `cmd/nexus-daemon/main.go`
- Problem:
  - UI documents `NEXUS_QUEUE_CAP`.
  - Neither desktop nor daemon entrypoint reads it.
- Why this matters:
  - Operators are told a deployment control exists when it does not.
- Required remediation:
  - Implement env-based queue-cap wiring or remove the claim.

### 19. Unused duplicate Wails binding surface still exists

- Files:
  - `app.go`
  - `internal/adapters/inbound/wailsbind/bind.go`
- Problem:
  - There are two Wails binding surfaces, one active and one stale.
- Why this matters:
  - Contributors can edit the wrong adapter and ship stale behavior.
- Required remediation:
  - Collapse to a single binding surface.

## Stubs, TODOs, and Placeholder Code

- `frontend/src/views/ProvidersView.vue`
  - Promote flow is a TODO and only logs to console.
- `frontend/src/views/DiscoveryView.vue`
  - Promote flow is a TODO and only logs to console.
- `internal/adapters/inbound/tray/tray.go`
  - Explicit no-op adapter with pending main-thread integration.
- `internal/adapters/inbound/tray/icon.go`
  - Placeholder icon bytes.
- `vscode-extension/src/extension.ts`
  - `nexus.showProviders` placeholder.
- `vscode-extension/src/commands/index.ts`
  - `showProvidersCommand` placeholder.

## Test and Coverage Gaps

### Release-gate mismatch

- PR CI is stronger than publish.
- Publish does not re-run frontend smoke coverage, extension smoke coverage, daemon E2E, vet, or lint.

### Frontend coverage is still narrow and threshold-free

- Files:
  - `frontend/vitest.config.ts`
  - `frontend/src/views/BacklogView.spec.ts`
  - `frontend/src/views/HistoryView.spec.ts`
- Current state:
  - Only two views are in the coverage include list.
  - No coverage thresholds are enforced.

### VS Code extension coverage is still narrow and threshold-free

- Files:
  - `vscode-extension/vitest.config.ts`
  - `vscode-extension/src/commands/submitTask.test.ts`
  - `vscode-extension/src/statusBar.test.ts`
  - `vscode-extension/src/taskQueueProvider.test.ts`
- Current state:
  - Only three files are covered.
  - No coverage thresholds are enforced.

### Release-critical surfaces still weakly tested

- Desktop startup and lifecycle:
  - `main.go`
  - `internal/adapters/inbound/tray/tray.go`
- Wails bindings:
  - `app.go`
  - generated bindings and browser fallbacks
- Extension runtime paths:
  - `vscode-extension/src/extension.ts`
  - `vscode-extension/src/nexusClient.ts`
  - `vscode-extension/src/sessionMonitor.ts`
  - `vscode-extension/src/workspaceScanner.ts`
  - `vscode-extension/src/workspaceOrchView.ts`
- Frontend high-value views and composables:
  - `DashboardView.vue`
  - `ProvidersView.vue`
  - `DiscoveryView.vue`
  - `SettingsView.vue`
  - `AISessionsView.vue`
  - `useTasks.ts`
  - `useProviders.ts`
  - `useDiscovery.ts`
  - `useAISessions.ts`

## Comprehensive Remediation Todo Lists

### A. Ship Blockers — must fix before release

1. Unify queue truth so persisted pending work is recoverable and cancellable after restart.
2. Centralize all transitions into `QUEUED` behind one admission path.
3. Make publish use the same gate as PR CI.
4. Persist provider promotion with `Enabled: true` and test restart durability.
5. Make session history persistence consistent with writeback outcomes.
6. Remove or implement the fake tray lifecycle.
7. Fix browser fallback submit parsing.
8. Update docs and README to match live HTTP and MCP contracts.

### B. Product Completeness — should fix in the next pass

1. Implement actual discovery promotion from the frontend.
2. Decide whether backlog creation is distinct from draft creation and make the API truthful.
3. Replace raw Wails provider-config exposure with safe DTOs.
4. Persist all mutable task routing fields in `Repository.Update`.
5. Remove duplicate Wails binding surface.
6. Implement or remove `nexus.showProviders` placeholder behavior.
7. Reconcile `.claude` metadata with real implementation status.

### C. Test Hardening — should be enforced by CI

1. Add coverage thresholds for frontend and extension suites.
2. Expand coverage scope to all production frontend and extension sources.
3. Add desktop startup and tray behavior tests.
4. Replace grep-based shell E2E assertions with structured schema assertions.
5. Add release-workflow checks that fail if publish is weaker than CI.
6. Add contract tests between docs/examples and live HTTP and MCP handlers.

### D. Type Safety and Tooling Hygiene

1. Fix VS Code extension `tsconfig.json` to include the correct runtime libs and types.
2. Remove stale or misleading enums such as frontend `openai` provider kind.
3. Keep editor diagnostics clean for the extension and frontend.
4. Prefer generated or shared contract types over hand-maintained mirrored DTOs.

## Recommended Execution Order

### Wave 1

- Queue authority and admission-path fix.
- Publish workflow parity with CI.
- Provider promotion durability.

### Wave 2

- Secret-safe provider config DTO unification.
- Browser fallback contract alignment.
- Docs and README regeneration.
- Tray behavior truthfulness.

### Wave 3

- Discovery UI wiring.
- Backlog versus draft workflow completion.
- Placeholder command cleanup.
- `.claude` metadata reconciliation.

### Wave 4

- Coverage thresholds.
- Structured E2E harness.
- Desktop and extension runtime test expansion.

## Bottom Line

The repository is substantially improved, but it still over-claims reliability in a few important places:

- crash recovery,
- queued-task admission,
- tray behavior,
- publish gating,
- provider-promotion durability,
- and documentation accuracy.

If the goal is “really ship working code”, the correct next step is not another feature wave. It is a release-readiness hardening wave that closes the blockers above and then reruns the full gate from the exact artifact-producing workflow.