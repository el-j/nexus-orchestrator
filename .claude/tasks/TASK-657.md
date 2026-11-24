---
id: TASK-657
planId: PLAN-080
title: 'e2e for the standalone binaries and the pipeline CLI'
role: testing
status: done
createdAt: 2026-11-24
---

# TASK-657 — e2e for the standalone binaries and the pipeline CLI

nexus-brain (ingest, search, serve), nexus-router (proxy, models), nexus-runner (verify) and `nexus orchestrate` run as separate processes against fake providers in internal/e2e; the e2e route guard also covers the gateway routes.

Done: `internal/e2e/standalone_test.go` builds `nexus-brain`, `nexus-router` and `nexus-runner` with the others and runs them as processes: brain (ingest, search, status, list, stdio MCP `tools/list`), runner (verify pass and fail exit codes, inspect violation), router (prompt and `serve` answering `/v1/chat/completions` through a fake provider), and `nexus-cli orchestrate` against the daemon. The gateway routes were already covered by `gateway_test.go`. The last test found two bugs, now fixed with unit tests: `orchestrate` gave up waiting after 3 s (any real model takes longer) and reported a half-done task, and it exited 0 when the task failed.
