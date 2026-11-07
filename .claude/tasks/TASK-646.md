---
id: TASK-646
planId: PLAN-077
title: 'Mark the historical audits superseded'
role: docs
status: done
createdAt: 2026-10-31
---

# TASK-646 — Mark the historical audits superseded

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

docs/agent-workflow-audit.md, audit-v0.10.0.md and audit-v0.11.0.md state numbers and verdicts that
are no longer true (41/42/40 MCP tools, "not yet"). Add a status header to each and link this audit;
docscheck test that every audit doc has one.

Done: the seven older audit and assessment documents start with a `Status: superseded` line that links the audit of 2026-10-31, and that audit says `Status: current`. `TestEveryAuditSaysWhetherItIsCurrent` (docscheck) fails when an audit document has no status in its first ten lines or says superseded without linking its successor.
