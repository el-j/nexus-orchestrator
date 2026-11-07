---
id: TASK-645
planId: PLAN-077
title: 'Bound the history and add benchmarks'
role: performance
status: done
createdAt: 2026-10-31
---

# TASK-645 — Bound the history and add benchmarks

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

TASK-631 (limit/paging, logs out of list rows) plus Go benchmarks for the router (FindForModel with
50 providers), the SQLite task list and brain search, and a CI-free script that fails on a
regression against a recorded baseline.

Done: `GET /api/tasks/all` is paged (`limit` default 1000, capped at 5000, `offset`, total in `X-Total-Count`; a bad value is a 400), so History shows the newest 1 000 tasks and a caller can page back. Benchmarks for the router (`FindForModel` with 50 providers), the SQLite task list (2 000 tasks) and the knowledge search (1 000 entries) live next to their packages; `scripts/bench.sh` (`make bench`) fails when one is more than 3x slower than `scripts/bench-baseline.txt`, `--record` rewrites it. Local only.

Not done: taking `logs` out of the list rows and paging in the database. The page is cut after the repository read, so the daemon still reads every row; the response is bounded, which was the failure the audit measured. The benchmark gives the number to improve on (8.4 ms for 2 000 tasks).
