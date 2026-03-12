# PLAN-031: CI Hard-Failure Fix + gofmt Enforcement

**Status:** Completed  
**Completed:** 2026-03-12T17:15:00.000Z

## Tasks

| ID | Title | Role | Status |
|----|-------|------|--------|
| TASK-226 | Fix duplicate package decl in llm_openaicompat test | qa | done |
| TASK-227 | Apply go fmt to all Go source files | devops | done |
| TASK-228 | Harden CI: gofmt check + vet gate for test | devops | done |
| TASK-229 | Verify full CI green locally | verify | done |

## Summary

Emergency fix sprint that resolved a hard CI failure affecting the `test`, `vet`, and `lint` jobs.

**Root cause:** `internal/adapters/outbound/llm_openaicompat/adapter_test.go` had two consecutive `package llm_openaicompat` declarations (lines 1–2) — a Go syntax error caused by content being appended to a file that already had a package declaration.

**Key changes:**
- `internal/adapters/outbound/llm_openaicompat/adapter_test.go`: Removed duplicate `package` declaration; file now starts with exactly one `package llm_openaicompat` line
- `internal/core/services/orchestrator_options_test.go`: Applied `gofmt -w` to fix formatting (was the only unformatted file in the codebase)
- `.github/workflows/ci.yml`: Added explicit `gofmt -l` diff check step to the `lint` job; added `needs: [vet]` to the `test` job so syntax errors caught by vet prevent test compilation from running

**Verification:** Evidence Collector confirmed all 5 CI simulation steps pass locally — `go vet`, `gofmt -l`, CLI+daemon builds, and `CGO_ENABLED=1 go test -race -count=1 ./...` all exit 0 with no data races.

**Why build passed but test/vet failed:** `go build` only compiles `cmd/` entry points which don't import `_test.go` files. Test files are only compiled by `go test` and `go vet`, explaining why `build` was green while `test`+`vet` failed.
