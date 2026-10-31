---
id: TASK-630
planId: PLAN-076
title: 'Cache the agy model list'
role: backend
status: done
createdAt: 2026-10-31
---

# TASK-630 — `agy models` every 30 seconds

`agy models` asks Google's servers and takes about 2 s (measured). The daemon's health refresh called it
every 30 s per agy provider. The list is now reused for 10 minutes; a failed lookup is not remembered.
Tests in `agy_driver_test.go`.
