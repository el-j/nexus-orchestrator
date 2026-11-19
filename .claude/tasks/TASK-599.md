---
id: TASK-599
planId: PLAN-074
title: 'Antigravity driver (agy -p)'
role: coder
wave: 6
status: done
createdAt: 2026-10-16
dependsOn: [TASK-583]
---

# TASK-599 — Antigravity driver (agy -p)

Driver for `agy` (Antigravity CLI), recorded from the installed agy 1.4.12: prompt on stdin, `--model` from the configuration, typed failures from the `AGY_ERROR` line with the reset time read from "Resets in …". Allow-list entry, discovery, promote (path only), the Providers view default command and docs done. The JSON output formats stay unused.

Done when: the failing test came first, `make test` and `make nice` pass, docs and surface matrix updated where an interface changed.
