---
id: TASK-226
title: Fix duplicate package declaration in llm_openaicompat test
role: qa
planId: PLAN-031
status: done
dependencies: []
createdAt: 2026-03-12T17:00:00.000Z
---

## Context
`adapter_test.go` in `llm_openaicompat` has two consecutive `package llm_openaicompat` declarations on lines 1–2. This is a Go syntax error that causes the entire package to fail during `go vet`, `go test`, and `golangci-lint`, breaking all CI jobs that compile this package.

Root cause: the file was likely created by appending generated content to an existing file that already had a `package` declaration, resulting in the duplicate.

## Files to Read
- `internal/adapters/outbound/llm_openaicompat/adapter_test.go`

## Implementation Steps
1. Open `adapter_test.go`.
2. Remove the first (redundant) `package llm_openaicompat` line so the file starts with exactly one package declaration.
3. Ensure there is no blank line artifact left at the top of the file.
4. Run `go vet ./internal/adapters/outbound/llm_openaicompat/...` to confirm the package compiles.
5. Run `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/llm_openaicompat/...` to confirm all tests pass.

## Acceptance Criteria
- `adapter_test.go` has exactly one `package llm_openaicompat` declaration at the top.
- `go vet ./...` exits 0.
- `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/llm_openaicompat/...` exits 0.
