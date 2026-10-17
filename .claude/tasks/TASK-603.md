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

Progress: `nexus connect` branches covered (the claude command and the stdio proxy from the PATH; 5 of 7 statements); Go merged coverage is 97.6 % with 264 statements open, mostly defensive error branches in the SQLite repositories, the file watcher and OS scanners. Not done: they need injected driver and OS errors; the CI gate stays at 96.
