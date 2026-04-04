---
id: TASK-486
plan: PLAN-063
status: done
wave: 2
priority: 1
---

# TASK-486: Fix `nexusClient.ts` - remove blind JSON casts

## Problem

`vscode-extension/src/nexusClient.ts` generic `get<T>()`, `post<T>()`, and `put<T>()` methods use `resp.json() as Promise<T>` — a bare TypeScript cast with no runtime shape validation. A backend schema change (e.g. renaming `providerName` to `name`) silently propagates as `undefined` fields, producing subtle UI bugs that show up as blank cards rather than errors.

## Checklist

- [ ] Add `zod` as a dependency in `vscode-extension/package.json` (or use hand-rolled guard functions if the team prefers zero new deps; confirm and document the choice)
- [ ] Define Zod schemas (or guard functions) for the four critical response types: `Task`, `AISession`, `Provider`, and `ProviderConfig` — derive these from the Go domain structs in `internal/core/domain/task.go` and `session.go`
- [ ] Add a private `parseResponse<T>(schema: ZodSchema<T>, data: unknown): T` helper method to `NexusClient` that calls `schema.parse(data)` and wraps `ZodError` in a descriptive `NexusClientError` with the endpoint path included in the message
- [ ] Replace the raw cast in `get<T>()`, `post<T>()`, `put<T>()` with the `parseResponse` call for the four critical types; less-critical internal types may retain the raw cast with a `// TODO: add schema` comment
- [ ] For array responses (e.g. `GET /api/sessions`, `GET /api/providers`), use `z.array(Schema).parse(data)` to validate each element
- [ ] Update `nexusClient` exports to surface the `NexusClientError` type so callers can distinguish parse errors from network errors

## Files to change

- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/package.json` (add `zod` dep if chosen)
- `vscode-extension/src/types.ts` (or equivalent types file) — export Zod-inferred types

## Acceptance criteria

- [ ] `getProviders()`, `getTasks()`, `getSessions()`, `getProviderConfigs()` throw `NexusClientError` with a meaningful message when the backend returns an unexpected shape
- [ ] No bare `as T` cast remains on any response path for the four critical types
- [ ] `npm run compile` and existing tests pass with no new type errors
