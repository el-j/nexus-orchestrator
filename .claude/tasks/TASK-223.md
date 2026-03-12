---
id: TASK-223
title: Frontend accessibility audit fixes
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-213, TASK-217]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Multiple accessibility issues across the frontend: missing ARIA labels on interactive elements, color-only status differentiation (inaccessible to colorblind users), missing `role="dialog"` on modals, no keyboard navigation for dropdown menus, and statistics without semantic meaning for screen readers.

## Files to Read
- `frontend/src/components/ProjectSelector.vue` — missing label on select (~line 20)
- `frontend/src/components/ProviderConfigForm.vue` — missing dialog/modal ARIA (~line 65)
- `frontend/src/views/AISessionsView.vue` — color-only status (~lines 55-60)
- `frontend/src/views/DashboardView.vue` — stats without semantic meaning (~lines 6-11)
- `frontend/src/views/HistoryView.vue` — missing ARIA on filter buttons
- `frontend/src/components/TaskSubmitForm.vue` — split button dropdown keyboard handling (~line 219)

## Implementation Steps
1. `ProjectSelector.vue`: add `<label for="project-select">` or `aria-label="Select project"` to the `<select>` element
2. `ProviderConfigForm.vue`: add `role="dialog"`, `aria-modal="true"`, `aria-labelledby` to the modal overlay. Add focus trap so Tab doesn't escape the modal
3. `AISessionsView.vue`: add text indicators alongside color: `Active`, `Idle`, `Disconnected` badges. Don't rely on color alone — use icon + text
4. `DashboardView.vue`: wrap statistics in `<span role="status">` with descriptive text: `{{ activeCount }} active tasks`
5. `HistoryView.vue`: add `aria-label="Filter tasks by status"` to filter button group. Mark active filter with `aria-pressed="true"`
6. `TaskSubmitForm.vue`: add `@click.outside` and Escape key handler to close split button dropdown. Add `role="menu"` + `aria-expanded` to dropdown

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All interactive elements have ARIA labels or associated `<label>` tags
- [ ] Status indicators use text + color (not color alone)
- [ ] Modals have proper `role="dialog"` and focus trap
- [ ] `cd frontend && npx vue-tsc --noEmit` exits 0

## Anti-patterns to Avoid
- NEVER rely on color alone to convey meaning — always pair with text/icon
- NEVER skip ARIA labels on interactive elements
- NEVER allow keyboard focus to escape modal dialogs
