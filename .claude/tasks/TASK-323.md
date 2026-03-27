# TASK-323: Implement `make nice` Makefile target

**Plan:** PLAN-049
**Status:** DONE

## Description

Add a `make nice` (and `make nice-go`, `make nice-frontend` sub-targets) to the Makefile that runs all formatting, fixing, and linting in one command.

## Go targets (`make nice-go`)

1. `gofmt -w .` — auto-format all Go files
2. `go vet ./...` — static analysis
3. `golangci-lint run --fix ./...` — auto-fix lint issues (errcheck, gocritic, revive, etc.)

## Frontend targets (`make nice-frontend`)

1. `cd frontend && npx vue-tsc --noEmit` — TypeScript type-check

## Combined target (`make nice`)

Runs `nice-go` then `nice-frontend` sequentially.

## Acceptance

- `make nice` exits 0 on clean codebase
- `make help` lists the new targets
