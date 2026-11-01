---
id: TASK-644
planId: PLAN-077
title: 'Frontend lint and the nilerr design'
role: quality
status: todo
createdAt: 2026-10-31
---

# TASK-644 — Frontend lint and the nilerr design

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

Add an ESLint flat config for frontend and docs (vue + typescript-eslint, no-explicit-any as error).
Replace the six `nilerr` returns in activity_network/reader.go with a typed result so "no activity"
and "unreachable" differ.
