---
id: TASK-608
planId: PLAN-074
title: 'Lean Claude calls: no settings, no MCP servers'
role: coder
wave: 5
status: done
createdAt: 2026-10-16
---

# TASK-608 — Lean Claude calls

Found while measuring TASK-596: the fixed prompt of a `claude -p` call is the user's own setup (hooks, plugins, skills, memory files, MCP servers), not the client's system prompt (a short `--system-prompt` did not help). `--setting-sources ""` and `--strict-mcp-config` cut a one-word call from about 15 500 to about 3 700 input tokens (-76 %), cost about half.

Done when: the driver adds both flags, tests assert them, the opt-in real test checks the size, docs state the measurement.
