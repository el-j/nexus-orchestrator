---
id: TASK-565
planId: PLAN-072
title: 'UI & Docs Polish — Discovery Pre-fill, Screenshot Placeholder, README v0.10.0'
role: polish
status: done
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:43:00Z
---

# TASK-565 — UI & Docs Polish

## Context

The audit identified three documentation and UI completeness gaps:

1. `DiscoveryView.vue:44`: Contains `// TODO: navigate to providers view and open add form pre-filled`.
2. `docs/getting-started.html:338`: Contains `<p>Dashboard screenshot placeholder</p>`.
3. `README.md`: Version badge is `v0.9.1` (should be `v0.10.0`), and MCP feature list mentions `14 tools` (actual: 40 tools).

## Work Required

1. `frontend/src/views/DiscoveryView.vue`:
   - Implement `handleAddProvider(agent)` to navigate to `/providers` with query params prefilling name, baseURL, and kind.
2. `docs/getting-started.html`:
   - Replace screenshot placeholder with clean SVG architectural/dashboard card.
3. `README.md`:
   - Update version badge to `v0.10.0`.
   - Update MCP tools count to `40 tools`.
