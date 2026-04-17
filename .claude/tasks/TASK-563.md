---
id: TASK-563
title: Add Zod schema validation to all VS Code nexusClient API response paths
role: vscode
planId: PLAN-071
status: todo
dependencies: [TASK-556]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Every API method in `nexusClient.ts` that calls Brain, Activity, or Agents endpoints casts raw `res.json()` directly to TypeScript types with no runtime validation. If the backend changes shape, the extension silently processes garbage data. Zod is already available in the extension (`zod` is in the project); all new and existing response paths should be validated at the boundary. The recently-modified `nexusClient.ts` has zero test coverage, making undetected regressions likely.

## Files to Read

- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/package.json` (check zod dependency)
- `vscode-extension/src/types.ts` (or wherever domain types are defined)

## Implementation Steps

1. Confirm `zod` is in `vscode-extension/package.json` dependencies; if not, add it
2. Create `vscode-extension/src/schemas.ts` with Zod schemas for all API response types currently cast unsafely:
   - `AIActivitySchema`, `AISessionSchema`, `DiscoveredAgentSchema`, `BrainStatusSchema`, `ContextResponseSchema`, `ContextSectionSchema`, `ProjectKnowledgeSchema`, `DelegateResponseSchema`
3. In `nexusClient.ts` — refactor the internal `get<T>()` / `post<T>()` helpers to accept an optional `z.ZodType<T>` schema parameter. When provided, call `schema.parse(await res.json())` instead of `res.json() as T`
4. Pass the appropriate schema from `schemas.ts` to each of the affected call sites:
   - `getDiscoveredAgents()` → `z.array(DiscoveredAgentSchema)`
   - `getActivities()` → `z.array(AIActivitySchema)`
   - `getBrainStatus()` → `BrainStatusSchema`
   - `getProjectContext()` → `ContextResponseSchema`
   - `getFocusedContext()` → `ContextResponseSchema`
   - `searchKnowledge()` → `z.object({ results: z.array(ContextSectionSchema) })`
   - `initProject()` → `BrainStatusSchema`
   - `listKnowledge()` → `z.array(ProjectKnowledgeSchema).nullable()`
   - `getFileMap()` → `z.object({ filePaths: z.array(z.string()) })`
   - `delegateSession()` → `DelegateResponseSchema`
5. Wrap `schema.parse()` calls in try-catch; on `ZodError` throw a typed `NexusValidationError` with the path and message so callers can distinguish validation failures from network failures

## Acceptance Criteria

- [ ] `npm run compile` exits 0 in `vscode-extension/`
- [ ] `npm test` exits 0 in `vscode-extension/`
- [ ] `schemas.ts` exists with Zod schemas for all 8+ response types
- [ ] Zero `(data as T)` raw casts remain for the listed methods
- [ ] `ZodError` is caught and re-thrown as a typed error, not swallowed

## Anti-patterns to Avoid

- NEVER use `z.any()` as a schema — define the actual shape
- NEVER let Zod validation failures silently return `undefined` — always throw or surface the error
