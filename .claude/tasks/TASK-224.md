---
id: TASK-224
title: Add golangci-lint + eslint to CI pipeline
role: devops
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
The CI pipeline runs `go vet` but no comprehensive linter. Common Go issues (unused variables, shadow, ineffassign, gosec security checks) are not caught. The TypeScript code has `vue-tsc` typecheck but no ESLint or Biome — style inconsistencies and potential bugs go uncaught. Adding linters to CI prevents regression and enforces code quality.

## Files to Read
- `.github/workflows/ci.yml` — current CI pipeline steps
- `.github/workflows/action-ci.yml` — GitHub Action CI pipeline
- `go.mod` — Go version
- `frontend/package.json` — frontend dependencies
- `github-action/package.json` — GitHub Action dependencies
- `Makefile` — existing targets

## Implementation Steps
1. Add `.golangci.yml` at repo root with these linters enabled: `govet`, `gosec`, `ineffassign`, `unused`, `errcheck`, `staticcheck`, `gocritic`, `revive`. Disable only rules that conflict with project conventions
2. Add golangci-lint step to `.github/workflows/ci.yml` between `vet` and `test` steps. Use `golangci/golangci-lint-action@v7`
3. Add `lint` target to `Makefile`: `golangci-lint run ./...`
4. Add `.eslintrc.cjs` or `eslint.config.mjs` for frontend with Vue 3 + TypeScript rules. Use `@vue/eslint-config-typescript`
5. Add ESLint to `frontend/package.json` devDependencies
6. Add lint step to `action-ci.yml` for GitHub Action TypeScript
7. Fix any lint violations that are immediately flagged (keep fixes minimal — only what the new linter catches)

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `golangci-lint run ./...` exits 0 (local and CI)
- [ ] ESLint runs in CI for frontend code
- [ ] `.golangci.yml` config file is present and documented
- [ ] New Makefile target `lint` works

## Anti-patterns to Avoid
- NEVER enable ALL linters — curate a reasonable set that catches real bugs
- NEVER auto-fix lint violations in bulk without review
- NEVER block CI on style-only rules (warnings only for style, errors for bugs/security)
