---
id: TASK-603
planId: PLAN-074
title: 'Go coverage to 100 percent; macOS link warning'
role: tester
wave: 7
status: todo
createdAt: 2026-10-16
---

# TASK-603 — Go coverage to 100 percent; macOS link warning

Cover the 247 open statements, gate MIN_COVERAGE=100, `-Wl,-no_warn_duplicate_libraries`.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
