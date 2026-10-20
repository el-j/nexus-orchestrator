---
id: PLAN-073
title: 'Real-world readiness: release pipeline, desktop app, subscription delegation'
status: completed
createdAt: 2026-10-15
tasks:
  [
    TASK-566,
    TASK-567,
    TASK-568,
    TASK-569,
    TASK-570,
    TASK-571,
    TASK-572,
    TASK-573,
    TASK-574,
    TASK-575,
    TASK-576,
    TASK-577,
    TASK-578,
    TASK-579,
  ]
---

# PLAN-073 — Real-world readiness

Found by downloading the released beta and starting it like a user: the pipeline had not built a
release since #62, the desktop UI got 403 on every call, the app crashed when started from Finder,
and Claude Pro could not be delegated to. TASK-566..571 fix that; TASK-572..578 are what is left
(everything that needs a real service, an Apple certificate or a screen).
