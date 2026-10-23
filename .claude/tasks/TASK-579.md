---
id: TASK-579
planId: PLAN-073
title: 'Desktop app next to a running daemon'
role: coder
status: done
createdAt: 2026-10-15
---

# TASK-579 — Desktop app next to a running daemon

Started while another daemon (make start, the VS Code extension) holds :63987/:63988, the desktop app
stays open with a silent log instead of using that daemon or saying why its own servers did not start.
Attach to a healthy daemon on the configured address (or show the port conflict in the window), and add
a smoke test for it.
