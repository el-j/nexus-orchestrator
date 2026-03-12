---
id: TASK-170
title: Entry point wiring for scanner, tray, and log capture
role: devops
planId: PLAN-023
status: todo
dependencies: [TASK-161, TASK-162, TASK-163, TASK-164, TASK-165]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
All individual components (scanner adapter, tray adapter, log capture, service methods, Wails config) are built. Now they must be wired together in both entry points: the desktop GUI (`main.go`) and the headless daemon (`cmd/nexus-daemon/main.go`).

## Files to Read
- `main.go` — desktop entry point
- `cmd/nexus-daemon/main.go` — daemon entry point
- `internal/adapters/outbound/sys_scanner/scanner.go` — scanner constructor
- `internal/adapters/inbound/tray/tray.go` — tray adapter constructor
- `internal/adapters/inbound/httpapi/log_hub.go` — log hub constructor
- `internal/core/services/orchestrator.go` — `WithSystemScanner` setter

## Implementation Steps

1. **In `main.go` (desktop GUI)**:
   ```go
   // After constructing orchestratorSvc:
   
   // a) Log capture
   logHub := httpapi.NewLogHub()
   log.SetOutput(logHub)
   // Pass logHub to HTTP server for SSE broadcasting
   
   // b) System scanner
   scanner := sys_scanner.New()
   orchestratorSvc.WithSystemScanner(scanner)
   
   // c) Run initial scan at startup
   go func() {
       if _, err := orchestratorSvc.TriggerScan(context.Background()); err != nil {
           log.Printf("startup: initial scan: %v", err)
       }
   }()
   
   // d) Tray adapter (integrated in the OnStartup hook — see TASK-165 for Wails config)
   ```

2. **In `cmd/nexus-daemon/main.go` (headless daemon)**:
   ```go
   // Same scanner + log capture wiring as main.go
   scanner := sys_scanner.New()
   orchestratorSvc.WithSystemScanner(scanner)
   
   logHub := httpapi.NewLogHub()
   log.SetOutput(logHub)
   
   // No tray adapter for daemon mode
   // Initial scan
   go func() {
       if _, err := orchestratorSvc.TriggerScan(context.Background()); err != nil {
           log.Printf("startup: initial scan: %v", err)
       }
   }()
   ```

3. Add the `sys_scanner` import to both entry points:
   ```go
   "nexus-orchestrator/internal/adapters/outbound/sys_scanner"
   ```

4. Ensure `httpapi.StartServer` accepts the `logHub` so it can wire SSE log events. This may require updating the `StartServer` function signature or using a shared event bus.

5. Add a periodic re-scan timer in the HTTP server or daemon:
   - Re-scan every 30 seconds (configurable via `NEXUS_SCAN_INTERVAL`)
   - Run in a goroutine with context cancellation
   - Log scan results: `log.Printf("discovery: found %d providers", len(results))`

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Desktop app constructs scanner, tray, and log hub at startup
- [ ] Daemon constructs scanner and log hub at startup (no tray)
- [ ] Initial scan runs automatically on startup (non-blocking)
- [ ] Periodic re-scan runs every 30s
- [ ] `log.Printf` output flows to both stderr and SSE

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER block startup waiting for the initial scan to complete
- NEVER run periodic scans without context cancellation (must stop on shutdown)
