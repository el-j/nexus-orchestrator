---
id: TASK-657
planId: PLAN-080
title: 'e2e for the standalone binaries and the pipeline CLI'
role: testing
status: todo
createdAt: 2026-11-24
---

# TASK-657 — e2e for the standalone binaries and the pipeline CLI

nexus-brain (ingest, search, serve), nexus-router (proxy, models), nexus-runner (verify) and `nexus orchestrate` run as separate processes against fake providers in internal/e2e; the e2e route guard also covers the gateway routes.
