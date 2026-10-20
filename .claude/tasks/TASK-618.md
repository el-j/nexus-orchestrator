---
id: TASK-618
planId: PLAN-075
title: 'Support NEXUS_API_TOKEN bearer auth and WAITING status in nexus-submit'
role: coder
wave: 3
status: done
createdAt: 2026-10-19
---

# TASK-618 — Support NEXUS_API_TOKEN bearer auth and WAITING status in nexus-submit

1. **`cmd/nexus-submit/main.go`**:
   - Read `NEXUS_API_TOKEN` from the environment and set `Authorization: Bearer <token>` on HTTP requests (`submitTask` and `getTask`).
   - Handle `domain.StatusWaiting` (`WAITING`) in `waitForCompletion` by logging `waitUntil` when the task is waiting for a provider cooldown reset instead of silently spinning.

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
