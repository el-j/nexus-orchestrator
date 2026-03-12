---
id: TASK-228
title: Harden CI with gofmt check and vet gate for test job
role: devops
planId: PLAN-031
status: done
dependencies: [TASK-226, TASK-227]
createdAt: 2026-03-12T17:00:00.000Z
---

## Context
The CI pipeline currently has `vet`, `lint`, and `test` jobs running in parallel with no dependency graph between them. This means a syntax error in a test file can slip through `vet` and still trigger the `test` job independently — both fail, but neither gates the other. Additionally, there is no explicit `gofmt` check in the pipeline, so format drift only surfaces indirectly via golangci-lint.

This task adds:
1. An explicit `gofmt` diff check to the `lint` job so format violations are caught with a clear, actionable error message.
2. `needs: [vet]` on the `test` job so test compilation never runs on a package with known vet errors.

## Files to Read
- `.github/workflows/ci.yml`

## Implementation Steps
1. Open `.github/workflows/ci.yml`.
2. In the `lint` job, add a new step **before** `golangci-lint-action`:
   ```yaml
   - name: Check gofmt
     run: |
       unformatted=$(gofmt -l ./...)
       if [ -n "$unformatted" ]; then
         echo "The following files are not gofmt-formatted:"
         echo "$unformatted"
         exit 1
       fi
   ```
3. Add `needs: [vet]` to the `test` job so it only runs after `vet` passes:
   ```yaml
   test:
     name: Test
     needs: [vet]
     runs-on: ubuntu-latest
   ```
4. Verify YAML is valid by visually checking indentation (no YAML linter available in CI dry-run).

## Acceptance Criteria
- The `lint` job contains a `Check gofmt` step that runs `gofmt -l ./...` and exits 1 if any file is unformatted.
- The `test` job has `needs: [vet]` so it only runs after vet passes.
- `.github/workflows/ci.yml` is valid YAML.
- `go vet ./...` and `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` still exit 0.
