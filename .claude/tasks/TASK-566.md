---
id: TASK-566
planId: PLAN-073
title: 'Release pipeline: replace the missing e2e-smoke script'
role: coder
status: done
createdAt: 2026-10-15
---

# TASK-566 — Release pipeline: replace the missing e2e-smoke script

The publish workflow ran scripts/e2e-smoke.sh (removed in #62), so no release was built. Run the real-daemon e2e suite; guard with TestWorkflowsOnlyRunScriptsThatExist; stop the macOS zip from containing itself.
