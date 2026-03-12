---
id: TASK-146
title: Fix ProviderConfig.type → kind TypeScript type mismatch
role: api
planId: PLAN-021
status: todo
dependencies: []
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
The TypeScript `ProviderConfig` interface in `frontend/src/types/domain.ts` has a field named `type`
but the Go backend serialises `ProviderConfig.Kind` as JSON key `"kind"` (via `json:"kind"` tag in
`internal/core/domain/provider.go`). Every time a configured provider is loaded from the backend,
the `type` field stays `undefined` in the frontend, breaking the form type-dropdown and type-dependent
URL presets.

## Files to Read
- `frontend/src/types/domain.ts`  ← the file to modify

## Current file content of frontend/src/types/domain.ts
```typescript
export type TaskStatus =
  | 'QUEUED'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER'

export type CommandType = 'plan' | 'execute' | 'auto'

export interface Task {
  ID: string
  ProjectPath: string
  TargetFile: string
  Instruction: string
  ContextFiles: string[]
  ModelID: string
  ProviderHint: string
  Command: CommandType
  Status: TaskStatus
  CreatedAt: string
  UpdatedAt: string
  Logs: string
}

export interface TaskInput {
  ProjectPath: string
  TargetFile: string
  Instruction: string
  ContextFiles?: string[]
  ModelID?: string
  ProviderHint?: string
  Command?: CommandType
}

export interface ProviderInfo {
  name: string
  active: boolean
  activeModel: string
  models: string[]
  baseURL?: string
  error?: string
}

export interface ProviderConfig {
  id: string
  name: string
  type: 'lmstudio' | 'ollama' | 'openai' | 'anthropic' | 'openaicompat'
  baseURL: string
  apiKey: string
  defaultModel: string
  enabled: boolean
  createdAt: string
  updatedAt: string
}
```

## Implementation Steps
1. In the `ProviderConfig` interface, rename the field `type` to `kind`.
   - The union type `'lmstudio' | 'ollama' | 'openai' | 'anthropic' | 'openaicompat'` stays identical.
2. Output the complete updated file — do NOT omit any other existing type/interface definitions.

## Acceptance Criteria
- [ ] `ProviderConfig` interface has `kind: 'lmstudio' | 'ollama' | 'openai' | 'anthropic' | 'openaicompat'` (not `type`)
- [ ] No other interfaces or type exports are changed
- [ ] Output is the complete file (all lines, nothing truncated)

## Anti-patterns to Avoid
- Do NOT rename `CommandType` or `TaskStatus` — those are correct already
- Do NOT add or remove any other fields
- Do NOT add comments or documentation beyond what already exists
