---
id: TASK-609
planId: PLAN-075
title: 'Fix Dockerfile FTS5 build flag & embedded Vue UI, ci.yml daemon build flag, and Makefile nexus-submit targets'
role: devops
wave: 1
status: done
createdAt: 2026-10-19
---

# TASK-609 — Fix Dockerfile FTS5 build flag & embedded Vue UI, ci.yml daemon build flag, and Makefile nexus-submit targets

1. **`Dockerfile`**:
   - Add Stage 0 (`FROM node:22-alpine AS webui`) that runs `npm ci && npm run build` in `frontend/` and copies `/build/frontend/.` into `internal/adapters/inbound/webui/dist/` in Stage 1 (`builder`) before `go build`.
   - Set `CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"` when compiling `nexus-daemon` in `Dockerfile` so the container does not crash at startup with `no such module: fts5`.
2. **`.github/workflows/ci.yml`**:
   - Pass `CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"` in the `build` job when compiling `./cmd/nexus-daemon/...`.
3. **`Makefile`**:
   - Build `nexus-submit` alongside `nexus-cli` and `nexus-daemon` in `make build` and `make build-*` cross-compile targets, using `CGO_ENABLED=0` for `nexus-cli` and `nexus-submit`.
4. **Tests**:
   - Add a test in `internal/docscheck/docscheck_test.go` asserting `Dockerfile` includes `-DSQLITE_ENABLE_FTS5` and `internal/adapters/inbound/webui/dist`.

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
