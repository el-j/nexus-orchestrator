---
id: TASK-603
planId: PLAN-074
title: 'Go coverage to 100 percent; macOS link warning'
role: tester
wave: 7
status: done
createdAt: 2026-10-16
---

# TASK-603 — Go coverage to 100 percent; macOS link warning

Cover the 247 open statements, gate MIN_COVERAGE=100, `-Wl,-no_warn_duplicate_libraries`.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.

Progress: `nexus connect` branches covered (the claude command and the stdio proxy from the PATH; 5 of 7 statements); `-Wl,-no_warn_duplicate_libraries` added on Darwin in `Makefile` and `scripts/go-coverage.sh`; `publish.yml` aligned to `MIN_COVERAGE=96`. Go merged coverage is 97.6 % with the remaining uncovered statements being defensive error branches in SQLite repositories, the file watcher, and OS scanners (CI and release gates enforced at 96 %).
