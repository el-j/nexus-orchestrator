---
id: TASK-131
title: GUI provider management panel
role: frontend
planId: PLAN-018
status: todo
dependencies: [TASK-130, TASK-128]
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Build a provider management panel in the Wails GUI that lets users add cloud providers (OpenAI, Anthropic, GitHub Copilot, custom OpenAI-compatible), edit base URLs and API keys, and remove providers — all without restarting.

## Files to Read
- `frontend/src/components/ProviderStatus.vue`
- `frontend/src/composables/useProviders.ts`
- `frontend/src/types/domain.ts`
- `frontend/src/types/wails.ts`

## Implementation Steps
1. Create `ProviderConfigForm.vue` — a form/modal to add or edit a provider config (type dropdown, base URL, API key, default model).
2. Add provider type presets: selecting "Ollama" auto-fills `http://127.0.0.1:11434`, selecting "LM Studio" auto-fills `http://127.0.0.1:1234/v1`, etc.
3. Add "Add Provider" button to the Providers tab that opens the form.
4. Add edit/delete actions to each provider card in `ProviderStatus.vue`.
5. Wire to the Wails bindings (or HTTP fallback) for CRUD operations.
6. After add/edit/delete, trigger a provider refresh to update the display.

## Acceptance Criteria
- [ ] User can add a new cloud provider from the GUI without env vars
- [ ] User can edit an existing provider's URL/key
- [ ] User can remove a provider
- [ ] Provider presets auto-fill sensible defaults
- [ ] Changes persist across app restarts

## Anti-patterns to Avoid
- NEVER show full API keys in the UI — mask all but last 4 characters
- Keep components focused — don't over-abstract
