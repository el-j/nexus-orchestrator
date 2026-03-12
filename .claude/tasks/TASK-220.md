---
id: TASK-220
title: Security hardening — CSP, permissions, XSS
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-212]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Three security issues identified: (1) CSP header in `dashboard.go` uses `'unsafe-inline'` for scripts, defeating the purpose of Content Security Policy. (2) `fs_writer` creates files with `0o644` (world-readable) and directories with `0o755` — generated code could contain secrets. (3) `TaskQueueProvider.ts` in VS Code extension renders raw `task.logs` in tooltips without escaping or truncation, which is an XSS vector with large logs.

## Files to Read
- `internal/adapters/inbound/httpapi/dashboard.go` — line ~11 (CSP header)
- `internal/adapters/outbound/fs_writer/writer.go` — lines ~46-49 (file permissions)
- `vscode-extension/src/taskQueueProvider.ts` — lines ~61-62 (raw logs in tooltip)
- `frontend/src/components/ProviderConfigForm.vue` — lines ~30-35 (missing form validation)

## Implementation Steps
1. In `dashboard.go`: replace `script-src 'unsafe-inline'` with proper CSP. Either:
   - Move inline scripts to external files and use `script-src 'self'`
   - Or use nonce-based CSP: generate nonce per request, inject into script tags
2. In `fs_writer/writer.go`: change directory permissions from `0o755` to `0o750` and file permissions from `0o644` to `0o640`. Add a comment explaining the security rationale
3. In `taskQueueProvider.ts`: truncate `task.logs` to 500 chars in tooltip. Escape special markdown/HTML characters. Add `...` suffix when truncated
4. In `ProviderConfigForm.vue`: add URL format validation for base URL field, required field validation before submit, and `minlength` attributes on text inputs
5. Add `nosniff` and `X-Frame-Options` headers if not already present in HTTP middleware

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] No `'unsafe-inline'` in CSP headers
- [ ] Generated files are not world-readable
- [ ] Task logs in VS Code tooltips are truncated and escaped
- [ ] Provider config form validates inputs before submission

## Anti-patterns to Avoid
- NEVER use `'unsafe-inline'` in Content Security Policy
- NEVER render untrusted content without escaping
- NEVER create world-writable files
- NEVER skip input validation on forms that submit to APIs
