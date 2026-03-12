---
id: TASK-222
title: Add GitHub Action tests — daemon, installer, index
role: qa
planId: PLAN-030
status: done
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
The GitHub Action (`github-action/`) only has 2 test files (`agents.test.ts`, `submit.test.ts`). Three critical modules are completely untested: `daemon.ts` (daemon download/start/stop/health check), `installer.ts` (binary download/checksum verification), and `index.ts` (main action entry point with input parsing and orchestration). The existing mocks in `__mocks__/` are incomplete — they return values but don't assert on call parameters.

## Files to Read
- `github-action/src/daemon.ts` — downloadDaemon, startDaemon, waitForHealth, stopDaemon
- `github-action/src/installer.ts` — download, verify, install binary
- `github-action/src/index.ts` — main action: input parsing, daemon lifecycle, task submission
- `github-action/src/types.ts` — type definitions
- `github-action/__mocks__/@actions/` — existing mock stubs
- `github-action/__tests__/agents.test.ts` — example test patterns
- `github-action/__tests__/submit.test.ts` — example test patterns
- `github-action/package.json` — test configuration

## Implementation Steps
1. Create `github-action/__tests__/daemon.test.ts`:
   - Mock `@actions/exec` for process spawning
   - Mock `@actions/http-client` for health check polling
   - Test `downloadDaemon()` — successful download, download failure, platform detection
   - Test `startDaemon()` — process starts, port configuration
   - Test `waitForHealth()` — success after retries, timeout failure
   - Test `stopDaemon()` — graceful shutdown
2. Create `github-action/__tests__/installer.test.ts`:
   - Mock `@actions/tool-cache` for download/extract
   - Test binary download for different platforms (linux, darwin, windows)
   - Test checksum verification (if implemented)
   - Test install path configuration
3. Create `github-action/__tests__/index.test.ts`:
   - Mock all @actions dependencies
   - Test input parsing (required inputs missing → error)
   - Test happy path: daemon starts → task submitted → result returned
   - Test failure paths: daemon won't start, task fails, timeout
4. Enhance `__mocks__/` to capture call arguments with `jest.fn()` for proper assertion

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] 3 new test files created in `github-action/__tests__/`
- [ ] `cd github-action && npm test` passes with all new tests
- [ ] daemon.ts, installer.ts, index.ts each have >70% line coverage
- [ ] Mocks properly assert on call parameters, not just return values

## Anti-patterns to Avoid
- NEVER make real HTTP calls or spawn real processes in tests
- NEVER test implementation details — test behavior through the public API
- NEVER use `any` type in test code — properly type mock return values
