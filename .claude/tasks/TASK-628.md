---
id: TASK-628
planId: PLAN-076
title: 'Fix the race in TestWaitForTask_StreamsProgressWhenAskedAndEndsWithTheResult'
role: testing
status: done
createdAt: 2026-10-31
---

# TASK-628 — Flaky wait_for_task progress test

`go test -race -count=3 ./...` failed once: "want 3 progress notifications and the result, got 1". The
test fed events as soon as the handler had subscribed and set the task to COMPLETED right after; when
the handler's first read came later it saw a finished task and, correctly, returned the bare result.
The product behaves as designed; the test now waits for the handler's first read (`firstRead`) before
sending anything. 30 runs with `-race` pass.
