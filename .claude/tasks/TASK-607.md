---
id: TASK-607
planId: PLAN-074
title: 'httpapi_client.IngestKnowledge: implement or remove from the port'
role: coder
wave: 7
status: done
createdAt: 2026-10-16
---

# TASK-607 — httpapi_client.IngestKnowledge: implement or remove from the port

The only functional stub found in the audit.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Done: POST /api/brain/knowledge and BrainClient.IngestKnowledge, with handler, client and sweep tests.
