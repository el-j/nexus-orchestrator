---
id: TASK-604
planId: PLAN-074
title: 'e2e for connect, plan folders, delegation, discovered plans'
role: tester
wave: 7
status: done
createdAt: 2026-10-16
---

# TASK-604 — e2e for connect, plan folders, delegation, discovered plans

Black-box tests in internal/e2e and frontend/e2e/real.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Done: internal/e2e/connect_test.go and agents_flow_test.go (connect, plan folders into the onboarding, agent register/delegate/claim/report). Still without an end-to-end test: discovered plans and activities over REST, the real-daemon extension flow, the desktop app, and every real cloud provider.
