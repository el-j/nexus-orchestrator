---
id: TASK-643
planId: PLAN-077
title: 'Mutation testing to the whole core and tighter gates'
role: testing
status: todo
createdAt: 2026-10-31
---

# TASK-643 — Mutation testing to the whole core and tighter gates

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

Extend scripts/mutation.sh defaults from 4 to every internal package, record the efficacy per
package in docs/mutation-testing.md, kill the lived mutants (first known: core/domain policy.go:100
CONDITIONALS_BOUNDARY), add Stryker for the docs site, raise TS break thresholds toward 90 in steps.
Local only; never in a workflow.
