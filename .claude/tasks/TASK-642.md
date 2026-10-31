---
id: TASK-642
planId: PLAN-077
title: 'e2e for the CLI-driver path and the browser views'
role: testing
status: todo
createdAt: 2026-10-31
---

# TASK-642 — e2e for the CLI-driver path and the browser views

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

A daemon e2e that runs a task through fake claude, agy and copilot executables placed on PATH
(discovery, promote, run, quota failure becomes WAITING). Playwright specs for Live Activity,
Providers promote flow, History; replace the "no script error" smoke test by assertions. An opt-in
real test for the editor bridge (Extension Host).
