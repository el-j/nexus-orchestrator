---
id: TASK-128
title: GUI provider refresh and error display
role: frontend
planId: PLAN-018
status: todo
dependencies: [TASK-127]
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
The frontend `ProviderStatus.vue` needs to display unreachable providers with error details and a "Refresh" button, using the new `BaseURL` and `Error` fields from TASK-127.

## Files to Read
- `frontend/src/components/ProviderStatus.vue`
- `frontend/src/composables/useProviders.ts`
- `frontend/src/types/domain.ts`

## Implementation Steps
1. Update `domain.ts` Provider type to include `baseURL` and `error` fields.
2. In `ProviderStatus.vue`, show unreachable providers with a red/orange indicator and the error message.
3. Display the base URL for each provider (helps diagnose wrong host/port).
4. Add a "Refresh" button that triggers an immediate re-poll instead of waiting 5 seconds.
5. Show a tooltip or expandable section with the ping error when a provider is unreachable.

## Acceptance Criteria
- [ ] Unreachable providers show with error detail in the GUI
- [ ] Each provider card/row shows its base URL
- [ ] "Refresh" button triggers immediate provider re-discovery
- [ ] Active providers still show green pulse dot as before

## Anti-patterns to Avoid
- NEVER introduce new npm dependencies without justification
- Keep Vue components simple — no unnecessary abstractions
