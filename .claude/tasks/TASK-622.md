---
id: TASK-622
planId: PLAN-075
title: 'Expose project policy (.nexus/policy.json: ladder, budgets, quotaThreshold, contextBudget, historyBudget) over REST, CLI, Wails, and UI'
role: coder
wave: 4
status: done
createdAt: 2026-10-19
dependsOn: [TASK-611, TASK-621]
---

# TASK-622 — Expose project policy (.nexus/policy.json: ladder, budgets, quotaThreshold, contextBudget, historyBudget) over REST, CLI, Wails, and UI

1. **Backend & Adapters (`httpapi`, `cli`, `app.go`, `httpapi_client`)**:
   - Add `GET /api/policy?project=<path>` and `PUT /api/policy?project=<path>` (validated via `domain.ProjectPolicy.Validate()`) to read and update `.nexus/policy.json`.
   - Add `GetProjectPolicy` / `SaveProjectPolicy` to `app.go` and `httpapi_client.Client`, and extend `nexus-cli routing` / `policy` to show budgets and token limits (`PLAN-074` B3/C4).
2. **Frontend UI (`wails.ts`, `SettingsView.vue` / `ProvidersView.vue`)**:
   - Add `getProjectPolicy` and `saveProjectPolicy` in `wails.ts` and render a Project Policy card allowing users to inspect and edit `quotaThreshold`, `contextBudget`, `historyBudget`, daily `budgets`, and role `ladder` rungs (`PLAN-074` B3/C4).

Done when: the failing test came first, 100% frontend coverage and `make nice` pass, docs and surface matrix updated.
