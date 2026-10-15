---
id: TASK-568
planId: PLAN-073
title: 'Default database in ~/.nexus and project-only workspace watch'
role: coder
status: done
createdAt: 2026-10-15
---

# TASK-568 — Default database in ~/.nexus and project-only workspace watch

Finder launches with cwd "/": the relative nexus.db failed and the watcher tried to watch the disk. Default ~/.nexus/nexus.db (legacy ./nexus.db honoured), isProjectDir, MaxWatchedDirs cap.
