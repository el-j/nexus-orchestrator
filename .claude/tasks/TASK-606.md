---
id: TASK-606
planId: PLAN-074
title: 'Repository cleanup'
role: polish
wave: 7
status: done
createdAt: 2026-10-16
---

# TASK-606 — Repository cleanup

Archive AUDIT_REPORT.md and PROJECT_ASSESSMENT_AND_WORKFLOW_BLUEPRINT.md, fix the 41 vs 42 tool count, close or archive the ~230 stale todo tasks.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Done: AUDIT_REPORT.md and PROJECT_ASSESSMENT_AND_WORKFLOW_BLUEPRINT.md moved to docs/archive/; 221 stale todo tasks (ids below 566) are `closed` with a reason; the tool count in docs/agent-workflow-audit.md states its date. The 20 todo tasks left belong to PLAN-073 and PLAN-074.
