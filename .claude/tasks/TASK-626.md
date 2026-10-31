---
id: TASK-626
planId: PLAN-076
title: 'Log only changes, failures and slow requests; rotate daemon.log'
role: backend
status: done
createdAt: 2026-10-31
---

# TASK-626 — Quiet access log and log rotation

`middleware.Logger` wrote a line for every request, so session heartbeats and UI polls were 80 % of a
20 MB log. `accessLog` (`internal/adapters/inbound/httpapi/access_log.go`) logs changes, status >= 400
and requests slower than 1 s. `make start-bg` moves `daemon.log` to `daemon.log.1` above 10 MB when it
starts a new daemon. Tests: `access_log_test.go`.
