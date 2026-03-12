---
id: TASK-181
title: Frontend — enhanced TaskSubmitForm with provider/model/priority/split-button
role: frontend
planId: PLAN-024
status: todo
dependencies: [TASK-179]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
The task submission form needs provider selector, model selector, priority, tags, and a split-button for submit mode (Queue / Draft / Backlog). This enables per-task provider delegation and idea capture directly from the form.

## Files to Read
- `frontend/src/components/TaskSubmitForm.vue` — existing form
- `frontend/src/composables/useProviders.ts` — provider list with models
- `frontend/src/types/domain.ts` — TaskInput type from TASK-179

## Implementation Steps

1. Add **Provider selector** (dropdown):
   - Options: "Auto (best available)" + each provider name from `useProviders().providers`
   - When "Auto" selected: `providerName` is empty string
   - When a specific provider is chosen: set `providerName` on the task

2. Add **Model selector** (dropdown):
   - Disabled when provider is "Auto"
   - When provider is selected: populate from that provider's `.models[]` array
   - First option: "Default (active model)" which sends empty `modelId`

3. Add **Priority selector** (dropdown):
   - Options: 1-High (red), 2-Medium (amber, default), 3-Low (slate)
   - Default: 2

4. Add **Tags input** (chips/comma-separated):
   - Free-text entry, press Enter or comma to add tag
   - Tags displayed as small removable pills
   - Stored as `string[]`

5. Replace single "Submit Task" button with **split-button**:
   - Primary action: **"Submit to Queue"** → calls `submitTask({...task, status: undefined})` (QUEUED)
   - Dropdown: **"Save as Draft"** → calls `createDraft(task)` (DRAFT)
   - Dropdown: **"Add to Backlog"** → calls `createDraft({...task, status: 'BACKLOG'})` (BACKLOG is set by changing task status after creation — or `CreateDraft` can accept StatusBacklog as well)

6. Form resets after submission. Toast confirms with status-specific message.

## Acceptance Criteria
- [ ] Provider dropdown shows all active providers + "Auto" option
- [ ] Model dropdown populates from selected provider's model list
- [ ] Priority defaults to 2 (medium)
- [ ] Tags input allows adding/removing tags as pills
- [ ] Split button offers 3 submit modes (Queue, Draft, Backlog)
- [ ] All new fields are sent in the task submission
- [ ] Form resets and shows toast on success

## Anti-patterns to Avoid
- NEVER hardcode provider names — always fetch from `useProviders()`
- NEVER allow model selection without a provider being selected first
- NEVER send arbitrary status values — only QUEUED, DRAFT, BACKLOG
