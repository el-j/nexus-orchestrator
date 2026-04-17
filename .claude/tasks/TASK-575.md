---
id: TASK-575
title: CI quality gates — coverage gate, E2E hard gate, and missing Makefile targets
role: devops
planId: PLAN-071
status: todo
dependencies: [TASK-554]
createdAt: 2026-04-15T00:00:00Z
---

## Context

The CI `test` job runs with no `-coverprofile` and enforces zero coverage gate — regressions in coverage are invisible. The Playwright E2E job has `continue-on-error: true` meaning failures are silently swallowed. The Makefile is missing `test-e2e`, `build-action`, and a discoverable `lint-fix` alias. These gaps reduce confidence in the CI pipeline.

## Files to Read

- `.github/workflows/ci.yml`
- `.github/workflows/e2e.yml`
- `Makefile`
- `github-action/package.json`

## Implementation Steps

1. In `.github/workflows/ci.yml` `test` job:
   - Add `-coverprofile=coverage.out` to the `go test` command
   - Add a step after tests: `go tool cover -func=coverage.out | tail -1` to print total coverage
   - Add `actions/upload-artifact` to publish `coverage.out` as a CI artifact
   - Add a minimum coverage gate (e.g., `awk '/^total/{if ($3+0 < 50) exit 1}'`) — set threshold at current measured level to prevent regression
2. In `.github/workflows/e2e.yml`: remove `continue-on-error: true` from the Playwright step — failures must block the workflow
3. In `Makefile`:
   - Add `test-e2e` target: `cd frontend && npx playwright test` (matching the CI e2e invocation)
   - Add `build-action` target: `cd github-action && npm ci && npm run build` (builds `dist/index.js`)
   - Ensure `lint-fix` is listed under the `## Test & quality` section in the help output (add alias pointing to `nice-go`)
   - Add `test-cover` to help output with description
4. In `github-action/package.json`: confirm `eslint` is a devDependency and add an `.eslintrc` or `eslint.config.js` so the existing `--if-present` workaround can be removed from `action-ci.yml`

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] CI `test` job produces `coverage.out` artifact
- [ ] CI `test` job fails if total Go coverage drops below the set threshold
- [ ] E2E workflow has no `continue-on-error: true` on the Playwright step
- [ ] `make test-e2e`, `make build-action`, `make lint-fix` all exist and are documented in `make help`

## Anti-patterns to Avoid

- NEVER set coverage threshold higher than current measured coverage (it will immediately fail CI)
- NEVER remove working CI steps while adding new ones
