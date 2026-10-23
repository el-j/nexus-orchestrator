---
id: TASK-580
planId: PLAN-073
title: 'Ship the full UI inside the released daemon binaries'
role: coder
status: done
createdAt: 2026-10-15
---

# TASK-580 — Full UI in the released daemon

`make start` embeds the frontend (`make web`); the CLI archives of a release are built without Node, so
their daemon only has the light dashboard at /ui. Build the frontend in the Build CLI jobs of
publish.yml and copy it to internal/adapters/inbound/webui/dist before `go build`.
