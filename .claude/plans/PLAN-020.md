---
id: PLAN-020
title: Fix GUI provider display + wire Providers view + enable dogfooding
status: active
createdAt: 2026-03-11T19:00:00.000Z
---

## Goal

Providers panel in the Wails GUI shows completely blank cards despite the backend
correctly returning LM Studio (active) and Ollama (unreachable). Fix the display so
providers render with correct names, activity status, and model info, then wire the
sidebar "Providers" nav item to a proper dedicated view. After the GUI is rebuilt,
validate end-to-end dogfooding via the VS Code extension.

## Root Causes Identified

1. **`domain.ts` field name mismatch** — `ProviderInfo` interface uses `Name`, `Active`,
   `ActiveModel`, `Models` (PascalCase) but the Go backend serializes with explicit JSON
   tags as `name`, `active`, `activeModel`, `models` (camelCase). Every provider card
   renders with blank text and wrong (all-inactive) styling.

2. **AppSidebar not wired** — `AppSidebar.vue` stores `activeView` as a local `ref`
   that is never emitted or shared. `App.vue` hardcodes `<DashboardView />` and ignores
   the sidebar selection entirely. Clicking "Providers" does nothing.

3. **No ProvidersView component** — There is no dedicated page rendered when the user
   navigates to "Providers" — only a compact `ProviderStatus` header strip inside the
   DashboardView.

## Tasks

| ID       | Role     | Title                                              | Wave |
|----------|----------|----------------------------------------------------|------|
| TASK-143 | frontend | Fix ProviderInfo TypeScript type + template refs   | 1    |
| TASK-144 | frontend | Wire AppSidebar routing + create ProvidersView     | 1    |
| TASK-145 | devops   | Rebuild Wails GUI + dogfood smoke test             | 2    |

TASK-143 and TASK-144 are fully independent — run in parallel.
TASK-145 depends on both.

## Success Criteria

- Provider cards in the GUI show "LM Studio" (green, active, model name) and "Ollama" (red, unreachable)
- Clicking "Providers" in the sidebar renders a dedicated full-page providers view
- `make build-gui` succeeds
- VS Code extension successfully submits a task and shows it queued/processing in the GUI
