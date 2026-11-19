---
id: TASK-642
planId: PLAN-077
title: 'e2e for the CLI-driver path and the browser views'
role: testing
status: done
createdAt: 2026-10-31
---

# TASK-642 — e2e for the CLI-driver path and the browser views

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

A daemon e2e that runs a task through fake claude, agy and copilot executables placed on PATH
(discovery, promote, run, quota failure becomes WAITING). Playwright specs for Live Activity,
Providers promote flow, History; replace the "no script error" smoke test by assertions. An opt-in
real test for the editor bridge (Extension Host).

Done: `internal/e2e/cli_drivers_test.go` puts fake `claude`, `agy` and `copilot` programs on the daemon's PATH, scans, promotes all three, runs a task through each (the files they answer are written), checks the flags each was started with (claude's lean stream-json flags, copilot's `-s --no-ask-user`) and that the tokens claude reports reach the usage ledger; a second test makes claude report a rejected seven-day limit and expects the task to be WAITING with the limit in its log. `frontend/e2e/real/views.spec.ts` (real daemon, no mocks) promotes the discovered Antigravity client in the Providers view and runs a task through it, checks History lists a finished task, Live Activity's heading and empty state, and that every view shows its own heading; it replaces the old "no script error" smoke test. The Providers spec found a real bug: saving a promoted provider posted empty `createdAt`/`updatedAt` strings, which the daemon cannot parse ("invalid JSON: parsing time"); fixed with a regression test.

Not done: the opt-in real test for the editor bridge (needs a VS Code Extension Host with Copilot signed in, which no CI runner has); the manual check stays with TASK-578.
