# Design: System-Wide Provider Discovery & Background Service Mode

**Author:** UXArchitect  
**Date:** 2026-03-11  
**Status:** Draft  
**Scope:** Domain model, port interfaces, outbound adapter, inbound adapter, build config

---

## Table of Contents

1. [Overview](#1-overview)
2. [Feature 1 — System-Wide Provider Discovery](#2-feature-1--system-wide-provider-discovery)
   - 2.1 New Domain Types
   - 2.2 New Port Interface
   - 2.3 Outbound Adapter Design
   - 2.4 Discovery Flow (Scanner → DiscoveryService → HTTP API → GUI)
   - 2.5 Orchestrator Port Changes
3. [Feature 2 — Background Service Mode (Hide to Tray)](#3-feature-2--background-service-mode)
   - 3.1 Tray Adapter Design
   - 3.2 Wails Config Changes
   - 3.3 Build & Packaging Changes
   - 3.4 Daemon Service Mode
4. [Task Breakdown](#4-task-breakdown)

---

## 1. Overview

Two capabilities are missing from nexusOrchestrator:

| Gap | Impact |
|-----|--------|
| Only hardcoded providers are known at boot | Claude CLI, Copilot, Antigravity, LocalAI, vLLM — all invisible |
| Close window = quit app | Users lose running tasks; no background operation |

This design adds:
- A **system scanner** outbound adapter that probes ports, CLIs, and processes periodically
- A **tray** inbound adapter that keeps the app alive when the window is closed
- Wails lifecycle hooks for hide-to-tray behavior

---

## 2. Feature 1 — System-Wide Provider Discovery

### 2.1 New Domain Types

**File:** `internal/core/domain/provider.go` — extend existing

```go
// New ProviderKind values for discovered-but-not-yet-configured providers.
const (
    ProviderKindLocalAI  ProviderKind = "localai"   // LocalAI on :8080
    ProviderKindVLLM     ProviderKind = "vllm"       // vLLM on :8000
    ProviderKindTextGen  ProviderKind = "textgen"    // text-generation-webui :5000
    ProviderKindCLI      ProviderKind = "cli"        // bare CLI binary (claude, aichat, llm)
    ProviderKindDesktopApp ProviderKind = "desktopapp" // GUI app (ChatGPT.app, Claude.app)
)
```

**New file:** `internal/core/domain/discovered_provider.go`

```go
package domain

import "time"

// DiscoveryMethod describes how a provider was found.
type DiscoveryMethod string

const (
    DiscoveredViaPortScan DiscoveryMethod = "port_scan"   // HTTP probe on well-known port
    DiscoveredViaCLI      DiscoveryMethod = "cli_lookup"   // command -v / which
    DiscoveredViaProcess  DiscoveryMethod = "process_scan" // running process list
)

// DiscoveredProvider is a provider found by the system scanner that has NOT
// been explicitly configured by the user. It is ephemeral — it exists only
// while the scanner detects it on the system.
type DiscoveredProvider struct {
    // ID is a deterministic key: "<method>:<identifier>" e.g. "port_scan:127.0.0.1:1234"
    ID              string          `json:"id"`
    Name            string          `json:"name"`            // human-friendly, e.g. "LM Studio (auto)"
    Kind            ProviderKind    `json:"kind"`
    Method          DiscoveryMethod `json:"method"`
    // Endpoint is the base URL when the provider exposes an HTTP API.
    // Empty for CLI-only discoveries.
    Endpoint        string          `json:"endpoint,omitempty"`
    // CLIPath is the absolute path to the binary when discovered via CLI lookup.
    CLIPath         string          `json:"cliPath,omitempty"`
    // ProcessName is the OS process name when discovered via process scan.
    ProcessName     string          `json:"processName,omitempty"`
    // HasAPI is true when the discovered provider exposes an HTTP-compatible API
    // that can be wrapped with an LLMClient adapter.
    HasAPI          bool            `json:"hasAPI"`
    // Models lists model IDs retrieved from the API (empty if HasAPI is false).
    Models          []string        `json:"models,omitempty"`
    // Compatible is true when the provider speaks a protocol we can wrap
    // (OpenAI /v1, Ollama /api, Anthropic /v1/messages).
    Compatible      bool            `json:"compatible"`
    // CompatibleKind is the ProviderKind to use if auto-registering this
    // provider. Only meaningful when Compatible is true.
    CompatibleKind  ProviderKind    `json:"compatibleKind,omitempty"`
    DiscoveredAt    time.Time       `json:"discoveredAt"`
    LastSeenAt      time.Time       `json:"lastSeenAt"`
}
```

### 2.2 New Port Interface

**File:** `internal/core/ports/ports.go` — add after existing outbound ports

```go
// --- System Discovery (Outbound Port) ---

// ScanTarget defines a single thing to probe during system discovery.
type ScanTarget struct {
    // Kind is the expected provider type if found.
    Kind   domain.ProviderKind
    // Name is the human-friendly label.
    Name   string
    // Probe is one of: "http:<host:port>/<path>", "cli:<command>", "process:<name>"
    Probe  string
}

// SystemScanner is the outbound port for discovering AI providers on the host.
// Implementations probe network ports, CLI binaries, and running processes.
type SystemScanner interface {
    // Scan performs a full discovery sweep and returns all found providers.
    // Must be safe for concurrent use. Must respect ctx cancellation.
    Scan(ctx context.Context) ([]domain.DiscoveredProvider, error)
}
```

**No change** to `LLMClient` — discovered providers that are compatible get auto-registered as real `LLMClient` instances via the existing `providerFactory`.

### 2.3 Outbound Adapter Design

**New package:** `internal/adapters/outbound/sys_scanner/`

```
internal/adapters/outbound/sys_scanner/
├── scanner.go          # Scanner struct, Scan() orchestration
├── port_probe.go       # HTTP port probes (connect + health check)
├── cli_probe.go        # exec.LookPath + version detection
├── process_probe.go    # OS process list scanning
├── targets.go          # default ScanTarget registry
└── scanner_test.go
```

#### `scanner.go`

```go
package sys_scanner

import (
    "context"
    "nexus-orchestrator/internal/core/domain"
    "nexus-orchestrator/internal/core/ports"
    "sync"
    "time"
)

// Scanner implements ports.SystemScanner.
type Scanner struct {
    targets []ports.ScanTarget
    timeout time.Duration  // per-probe timeout, default 2s
}

func New(targets []ports.ScanTarget, probeTimeout time.Duration) *Scanner {
    if probeTimeout == 0 {
        probeTimeout = 2 * time.Second
    }
    return &Scanner{targets: targets, timeout: probeTimeout}
}

func (s *Scanner) Scan(ctx context.Context) ([]domain.DiscoveredProvider, error) {
    var (
        mu      sync.Mutex
        results []domain.DiscoveredProvider
        wg      sync.WaitGroup
    )
    // Fan out: one goroutine per target, bounded by a semaphore.
    sem := make(chan struct{}, 8) // max 8 concurrent probes
    for _, t := range s.targets {
        wg.Add(1)
        go func(target ports.ScanTarget) {
            defer wg.Done()
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
            case <-ctx.Done():
                return
            }
            dp, found := s.probe(ctx, target)
            if found {
                mu.Lock()
                results = append(results, dp)
                mu.Unlock()
            }
        }(t)
    }
    wg.Wait()
    return results, ctx.Err()
}
```

#### `targets.go` — Default scan targets

```go
package sys_scanner

import (
    "nexus-orchestrator/internal/core/domain"
    "nexus-orchestrator/internal/core/ports"
)

// DefaultTargets returns the built-in scan targets.
func DefaultTargets() []ports.ScanTarget {
    return []ports.ScanTarget{
        // --- HTTP port probes ---
        {Kind: domain.ProviderKindLMStudio,     Name: "LM Studio",           Probe: "http:127.0.0.1:1234/v1/models"},
        {Kind: domain.ProviderKindOllama,        Name: "Ollama",              Probe: "http:127.0.0.1:11434/api/tags"},
        {Kind: domain.ProviderKindLocalAI,       Name: "LocalAI",             Probe: "http:127.0.0.1:8080/v1/models"},
        {Kind: domain.ProviderKindVLLM,          Name: "vLLM",                Probe: "http:127.0.0.1:8000/v1/models"},
        {Kind: domain.ProviderKindTextGen,       Name: "text-generation-webui",Probe: "http:127.0.0.1:5000/v1/models"},
        // Antigravity (hypothetical API port — needs confirmation)
        {Kind: domain.ProviderKindOpenAICompat,  Name: "Antigravity",         Probe: "http:127.0.0.1:8642/v1/models"},

        // --- CLI binary probes ---
        {Kind: domain.ProviderKindCLI,           Name: "Claude CLI",          Probe: "cli:claude"},
        {Kind: domain.ProviderKindCLI,           Name: "Ollama CLI",          Probe: "cli:ollama"},
        {Kind: domain.ProviderKindCLI,           Name: "LM Studio CLI",       Probe: "cli:lms"},
        {Kind: domain.ProviderKindCLI,           Name: "aichat",              Probe: "cli:aichat"},
        {Kind: domain.ProviderKindCLI,           Name: "llm (Python)",        Probe: "cli:llm"},

        // --- Process probes (GUI apps) ---
        {Kind: domain.ProviderKindDesktopApp,    Name: "Claude Desktop",      Probe: "process:Claude"},
        {Kind: domain.ProviderKindDesktopApp,    Name: "ChatGPT",             Probe: "process:ChatGPT"},
        {Kind: domain.ProviderKindDesktopApp,    Name: "Antigravity",         Probe: "process:Antigravity"},
    }
}
```

#### `port_probe.go` — HTTP probing

```go
// probe parses "http:<host:port>/<path>" from target.Probe
// 1. TCP connect with s.timeout
// 2. GET /<path>, expect 200 + JSON body
// 3. Parse model list from response (OpenAI /v1/models or Ollama /api/tags)
// 4. Return DiscoveredProvider with HasAPI=true, Compatible=true, models filled
```

Probe logic by endpoint:

| Endpoint pattern | Parse strategy | CompatibleKind |
|---|---|---|
| `/v1/models` | OpenAI list-models response → `data[].id` | `openaicompat` (or the specific Kind like `lmstudio`) |
| `/api/tags` | Ollama tags response → `models[].name` | `ollama` |

#### `cli_probe.go` — CLI detection

```go
// probe parses "cli:<command>" from target.Probe
// 1. exec.LookPath(command) — returns absolute path or error
// 2. If found, try "<command> --version" with 3s timeout to confirm it's functional
// 3. Return DiscoveredProvider with HasAPI=false, CLIPath set
// 4. Special case: if command is "ollama", also check if ollama serve is running
//    by probing http:127.0.0.1:11434 → set HasAPI=true
```

#### `process_probe.go` — Running process detection

```go
// probe parses "process:<name>" from target.Probe
// macOS: pgrep -f <name> or ps aux | grep <name>
// Windows: tasklist /FI "IMAGENAME eq <name>.exe"
// Linux: pgrep -f <name>
// Returns DiscoveredProvider with ProcessName set
// Does NOT try to find API ports from process (that's a future enhancement)
```

### 2.4 Discovery Flow

```
                                  ┌─────────────────┐
                                  │  sys_scanner     │
                                  │  (outbound       │
                                  │   adapter)       │
                                  └────────┬─────────┘
                                           │ []DiscoveredProvider
                                           ▼
┌───────────────────────────────────────────────────────────────┐
│                   DiscoveryService (core/services)             │
│                                                               │
│  existing: []ports.LLMClient   ← hardcoded + user-configured │
│  NEW:      []domain.DiscoveredProvider ← from SystemScanner   │
│  NEW:      autoRegister()      ← if Compatible, create client │
│                                                               │
│  NEW:      StartBackgroundScan(ctx, scanner, interval)        │
│            └─ ticker loop: scan → merge → auto-register       │
│            └─ interval: 30s (configurable via env var)        │
│            └─ skip scan if previous still running (tryLock)   │
└───────────────┬─────────────────────────────┬─────────────────┘
                │                             │
        ListProviders()               ListDiscoveredProviders()
        (existing behavior)           (new method)
                │                             │
                ▼                             ▼
┌───────────────────────────────────────────────────────────────┐
│              OrchestratorService (inbound port)                │
│                                                               │
│  GetProviders()  → existing LLMClient liveness (unchanged)    │
│  NEW: GetDiscoveredProviders() → []DiscoveredProvider          │
│  NEW: PromoteProvider(discoveredID string) → ProviderConfig    │
│       └─ takes a DiscoveredProvider, creates ProviderConfig,   │
│          registers the adapter, persists to SQLite             │
└───────────────┬───────────────────────────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│  HTTP API (inbound adapter)                                    │
│                                                               │
│  GET  /api/providers            → existing (unchanged)         │
│  GET  /api/providers/discovered → []DiscoveredProvider (NEW)   │
│  POST /api/providers/promote    → {discoveredId} → promote    │
└───────────────────────────────────────────────────────────────┘
                │
                ▼
           GUI / Frontend
           (new "Discovered" panel in provider settings)
```

#### DiscoveryService Changes

**File:** `internal/core/services/discovery.go` — add fields and methods

```go
type DiscoveryService struct {
    mu               sync.RWMutex
    availableClients []ports.LLMClient

    // NEW: system-discovered providers
    scanMu           sync.RWMutex
    discovered       []domain.DiscoveredProvider
    scanning         int32  // atomic flag: 1 = scan in progress
}

// StartBackgroundScan launches a periodic scanner goroutine.
// Cancelling ctx stops the loop. Safe to call once.
func (s *DiscoveryService) StartBackgroundScan(
    ctx context.Context,
    scanner ports.SystemScanner,
    interval time.Duration,
    autoRegister func(domain.DiscoveredProvider),
) { ... }

// ListDiscoveredProviders returns the latest scan results.
func (s *DiscoveryService) ListDiscoveredProviders() []domain.DiscoveredProvider { ... }

// SetDiscovered replaces the discovered list (called by scan loop).
func (s *DiscoveryService) SetDiscovered(providers []domain.DiscoveredProvider) { ... }
```

#### Auto-Registration Logic

When a `DiscoveredProvider` has `Compatible == true` and `HasAPI == true`:

1. Check if an `LLMClient` with a matching `BaseURL()` already exists → skip
2. Check if a persisted `ProviderConfig` with the same endpoint exists → skip
3. Do NOT auto-register by default — only surface in `GET /api/providers/discovered`
4. User explicitly promotes via `POST /api/providers/promote` (or clicks in GUI)
5. Optional: `NEXUS_AUTO_REGISTER_DISCOVERED=true` env var to auto-register compatible providers

This avoids surprise registrations that could route tasks to unintended backends.

### 2.5 Orchestrator Port Changes

**File:** `internal/core/ports/ports.go` — add to `Orchestrator` interface

```go
type Orchestrator interface {
    // ... existing methods ...

    // GetDiscoveredProviders returns providers found by the system scanner
    // that are not yet registered as active LLM backends.
    GetDiscoveredProviders() ([]domain.DiscoveredProvider, error)

    // PromoteProvider takes a discovered provider ID and registers it as a
    // configured, active LLM backend. Returns the created ProviderConfig.
    PromoteProvider(ctx context.Context, discoveredID string) (domain.ProviderConfig, error)
}
```

---

## 3. Feature 2 — Background Service Mode

### 3.1 Tray Adapter Design

**File:** `internal/adapters/inbound/tray/tray.go` — replace placeholder

Dependency: `github.com/getlantern/systray` (v1.2.2+) — pure Go, supports macOS/Windows/Linux.

```go
package tray

import (
    _ "embed"
    "nexus-orchestrator/internal/core/ports"
    "github.com/getlantern/systray"
)

//go:embed icon.png
var iconData []byte

// Callbacks are the GUI actions the tray adapter can trigger.
// Implemented by the Wails app layer (not by core services).
type Callbacks struct {
    ShowWindow func()
    Quit       func()
}

// Run starts the system tray. It blocks until Quit is selected.
// Must be called from the main goroutine on macOS (systray requirement).
func Run(orch ports.Orchestrator, cb Callbacks) {
    systray.Run(func() { onReady(orch, cb) }, func() {})
}

// RunDetached starts the tray in a background goroutine (Linux/Windows).
func RunDetached(orch ports.Orchestrator, cb Callbacks) {
    go systray.Run(func() { onReady(orch, cb) }, func() {})
}

func onReady(orch ports.Orchestrator, cb Callbacks) {
    systray.SetIcon(iconData)
    systray.SetTitle("nexusOrchestrator")
    systray.SetTooltip("nexusOrchestrator — AI Task Orchestrator")

    mShow := systray.AddMenuItem("Show Window", "Bring the main window to front")
    systray.AddSeparator()

    // Dynamic: show active provider count
    mStatus := systray.AddMenuItem("Providers: checking...", "")
    mStatus.Disable() // informational only

    systray.AddSeparator()
    mQuit := systray.AddMenuItem("Quit nexusOrchestrator", "Shut down completely")

    // Event loop
    go func() {
        for {
            select {
            case <-mShow.ClickedCh:
                cb.ShowWindow()
            case <-mQuit.ClickedCh:
                cb.Quit()
                systray.Quit()
                return
            }
        }
    }()

    // Periodic status update
    go updateStatus(orch, mStatus)
}

func updateStatus(orch ports.Orchestrator, item *systray.MenuItem) {
    // Every 10s, call orch.GetProviders(), count active, update menu item title
}
```

**File structure:**

```
internal/adapters/inbound/tray/
├── tray.go          # systray setup, menu, event loop
├── icon.png         # 22×22 / 44×44 (retina) PNG icon for menu bar
├── tray_darwin.go   # macOS-specific: ensure main thread (if needed)
├── tray_windows.go  # Windows-specific: nothing special (systray handles it)
└── tray_linux.go    # Linux: appindicator fallback notes
```

### 3.2 Wails Config Changes

**File:** `main.go` — modify `wails.Run` options

```go
import (
    wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// In main():
app := NewApp(orchestratorSvc)

if err := wails.Run(&options.App{
    Title:  "nexusOrchestrator",
    Width:  1024,
    Height: 768,
    AssetServer: &assetserver.Options{
        Assets: assets,
    },

    // NEW: Hide window instead of quitting on close
    HideWindowOnClose: true,

    OnStartup: app.startup,

    // NEW: intercept close to hide instead of quit
    OnBeforeClose: func(ctx context.Context) (prevent bool) {
        // Hide to tray instead of quitting
        wailsruntime.WindowHide(ctx)
        return true // prevent default close
    },

    OnShutdown: func(_ context.Context) { cancelHTTP() },
    Bind: []interface{}{
        app,
    },
}); err != nil {
    // ...
}
```

**File:** `app.go` — add window control methods for tray callbacks

```go
// ShowWindow brings the Wails window to front (called by tray adapter).
func (a *App) ShowWindow() {
    wailsruntime.WindowShow(a.ctx)
    wailsruntime.WindowSetAlwaysOnTop(a.ctx, true)
    wailsruntime.WindowSetAlwaysOnTop(a.ctx, false) // hack to bring to front
}

// QuitApp performs graceful shutdown.
func (a *App) QuitApp() {
    wailsruntime.Quit(a.ctx)
}
```

### 3.3 Build & Packaging Changes

#### macOS — `build/darwin/Info.plist`

Change `LSUIElement` from `false` to `true` so the app doesn't show in the Dock when hidden:

```xml
<key>LSUIElement</key>
<true/>
```

> **Note:** With `LSUIElement: true`, the app is an "agent" — no Dock icon, no app menu bar.
> The system tray icon becomes the primary interaction point when the window is hidden.
> The app should toggle back to Dock mode when the window is shown. This requires runtime
> `NSApp.setActivationPolicy(.regular)` via cgo or Wails runtime calls.

**Recommended approach:** Keep `LSUIElement: false` (show in Dock normally). Only hide the Dock icon at runtime when the user closes the window. Wails v2 doesn't expose `NSApp.setActivationPolicy` natively, so this is a future enhancement. For v1, the app stays in the Dock but the window hides.

#### Windows — Console suppression

**File:** `Makefile` — add `-H windowsgui` for the GUI binary

```makefile
build-gui:
	wails build -platform darwin/arm64
	# For manual Windows GUI build:
	# CGO_ENABLED=1 GOOS=windows go build -ldflags "-s -w -H windowsgui" -o nexusOrchestrator.exe .
```

Wails automatically handles `-H windowsgui` when building with `wails build`. No manual change needed for the standard flow. The console suppression flag only matters for `go build .` on Windows without Wails.

#### Windows — Manifest (optional)

No changes needed to `build/windows/wails.exe.manifest`. DPI awareness and long path support are already configured.

### 3.4 Daemon Service Mode

**File:** `cmd/nexus-daemon/main.go` — add PID file and systemd readiness

```go
// NEW: write PID file for service managers
if pidPath := os.Getenv("NEXUS_PID_FILE"); pidPath != "" {
    if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
        log.Printf("daemon: write pid file: %v", err)
    }
    defer os.Remove(pidPath)
}
```

**New file:** `scripts/nexus-daemon.service` (systemd unit)

```ini
[Unit]
Description=nexusOrchestrator Daemon
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nexus-daemon
Environment=NEXUS_DB_PATH=/var/lib/nexus/nexus.db
Environment=NEXUS_LISTEN_ADDR=127.0.0.1:9999
Environment=NEXUS_MCP_ADDR=127.0.0.1:9998
Environment=NEXUS_PID_FILE=/run/nexus-daemon.pid
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## 4. Task Breakdown

### Wave 1 — Domain & Ports (no adapter code, no build changes)

| # | Task | File(s) | Depends | Complexity |
|---|------|---------|---------|------------|
| 1 | Add new `ProviderKind` constants | `internal/core/domain/provider.go` | — | S |
| 2 | Create `DiscoveredProvider` domain type | `internal/core/domain/discovered_provider.go` | #1 | S |
| 3 | Add `SystemScanner` port interface | `internal/core/ports/ports.go` | #2 | S |
| 4 | Add `GetDiscoveredProviders()` + `PromoteProvider()` to `Orchestrator` port | `internal/core/ports/ports.go` | #2, #3 | S |
| 5 | Add `discovered` field + `ListDiscoveredProviders()` + `SetDiscovered()` to `DiscoveryService` | `internal/core/services/discovery.go` | #2 | M |
| 6 | Add `StartBackgroundScan()` to `DiscoveryService` | `internal/core/services/discovery.go` | #3, #5 | M |
| 7 | Implement `GetDiscoveredProviders()` + `PromoteProvider()` on `OrchestratorService` | `internal/core/services/orchestrator.go` | #4, #5 | M |

### Wave 2 — System Scanner Adapter

| # | Task | File(s) | Depends | Complexity |
|---|------|---------|---------|------------|
| 8 | Create `sys_scanner` package + `Scanner` struct + `Scan()` | `internal/adapters/outbound/sys_scanner/scanner.go` | #3 | M |
| 9 | Implement HTTP port probing | `internal/adapters/outbound/sys_scanner/port_probe.go` | #8 | M |
| 10 | Implement CLI binary detection | `internal/adapters/outbound/sys_scanner/cli_probe.go` | #8 | M |
| 11 | Implement process scanning | `internal/adapters/outbound/sys_scanner/process_probe.go` | #8 | M |
| 12 | Default targets registry | `internal/adapters/outbound/sys_scanner/targets.go` | #8 | S |
| 13 | Unit tests for scanner (mock probes) | `internal/adapters/outbound/sys_scanner/scanner_test.go` | #8–#12 | M |

### Wave 3 — HTTP API + Wiring

| # | Task | File(s) | Depends | Complexity |
|---|------|---------|---------|------------|
| 14 | Add `GET /api/providers/discovered` endpoint | `internal/adapters/inbound/httpapi/server.go` | #4, #7 | S |
| 15 | Add `POST /api/providers/promote` endpoint | `internal/adapters/inbound/httpapi/server.go` | #4, #7 | S |
| 16 | Wire `SystemScanner` in `main.go` (desktop) | `main.go` | #6, #8, #12 | S |
| 17 | Wire `SystemScanner` in `cmd/nexus-daemon/main.go` | `cmd/nexus-daemon/main.go` | #6, #8, #12 | S |
| 18 | Add `NEXUS_SCAN_INTERVAL` env var (default 30s) | `main.go`, `cmd/nexus-daemon/main.go` | #16, #17 | S |

### Wave 4 — Tray Adapter + Hide-to-Tray

| # | Task | File(s) | Depends | Complexity |
|---|------|---------|---------|------------|
| 19 | Add `github.com/getlantern/systray` to `go.mod` | `go.mod` | — | S |
| 20 | Implement tray adapter (`Run`, `onReady`, menu) | `internal/adapters/inbound/tray/tray.go` | #19 | M |
| 21 | Add tray icon PNG asset | `internal/adapters/inbound/tray/icon.png` | — | S |
| 22 | Add `ShowWindow()` + `QuitApp()` to `app.go` | `app.go` | — | S |
| 23 | Set `HideWindowOnClose: true` + `OnBeforeClose` in `main.go` | `main.go` | #22 | S |
| 24 | Wire tray adapter startup in `main.go` | `main.go` | #20, #22, #23 | M |
| 25 | Platform build file: `tray_darwin.go` (cgo main-thread constraint if needed) | `internal/adapters/inbound/tray/tray_darwin.go` | #20 | S |

### Wave 5 — Build & Packaging

| # | Task | File(s) | Depends | Complexity |
|---|------|---------|---------|------------|
| 26 | Keep `LSUIElement: false` in Info.plist (defer dynamic toggle) | `build/darwin/Info.plist` | — | S |
| 27 | Add PID file support to daemon | `cmd/nexus-daemon/main.go` | — | S |
| 28 | Add systemd unit file | `scripts/nexus-daemon.service` | #27 | S |
| 29 | Document `-H windowsgui` for manual Windows builds | `Makefile` | — | S |

### Wave 6 — Frontend + Integration Tests

| # | Task | File(s) | Depends | Complexity |
|---|------|---------|---------|------------|
| 30 | Add `DiscoveredProvider` TypeScript type | `frontend/src/types/domain.ts` | #2 | S |
| 31 | Add `useDiscoveredProviders()` composable | `frontend/src/composables/useDiscoveredProviders.ts` | #14, #30 | M |
| 32 | Add "Discovered Providers" panel to provider settings view | `frontend/src/components/DiscoveredProviders.vue` | #31 | M |
| 33 | "Promote" button in discovered provider card | `frontend/src/components/DiscoveredProviders.vue` | #15, #32 | S |
| 34 | Integration test: scanner → discovery service → HTTP API round-trip | `internal/core/services/discovery_test.go` | #7, #13 | M |
| 35 | E2E smoke test: start daemon, hit `/api/providers/discovered` | `scripts/e2e-smoke.sh` | #14, #17 | S |

---

### Dependency Graph (critical path)

```
Wave 1:  #1 → #2 → #3 → #4 → #5 → #6 → #7
                              ↓
Wave 2:                      #8 → #9,#10,#11 (parallel) → #12 → #13
                              ↓
Wave 3:                      #14,#15 (parallel) → #16,#17 → #18
                              ↕
Wave 4:  #19 → #20 → #21,#22 (parallel) → #23 → #24 → #25
                              ↓
Wave 5:  #26,#27,#28,#29 (all independent)
                              ↓
Wave 6:  #30 → #31 → #32 → #33, #34, #35
```

### Env Var Summary (new)

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXUS_SCAN_INTERVAL` | `30s` | Time between system discovery sweeps |
| `NEXUS_AUTO_REGISTER_DISCOVERED` | `false` | Auto-register compatible discovered providers |
| `NEXUS_PID_FILE` | (empty) | Write PID file at this path (daemon only) |

### Key Architecture Decisions

1. **Scanner is an outbound port** — `SystemScanner` interface in `ports/`, implementation in `adapters/outbound/sys_scanner/`. This lets tests inject a mock scanner.

2. **Discovered != Registered** — Scan results are ephemeral `DiscoveredProvider` structs held in memory by `DiscoveryService`. They only become `ProviderConfig` records (persisted in SQLite) when the user explicitly promotes them. This prevents surprise behavior.

3. **No new tray port needed** — The tray is an inbound adapter that calls `ports.Orchestrator`. It also needs Wails window control callbacks, which are injected as a function struct (`Callbacks`), not a port — because window management is GUI infrastructure, not business logic.

4. **`getlantern/systray` over Wails built-in** — Wails v2 has no native tray API. `getlantern/systray` is the most mature pure-Go option (supports macOS, Windows, Linux). It does require main-thread init on macOS, but Wails already runs on the main thread, so coordination is needed (start systray before or after `wails.Run`).

5. **systray + Wails main-thread conflict** — Both `systray.Run()` and `wails.Run()` want the main thread on macOS. Solution: start systray in a goroutine (`RunDetached`) and let Wails own the main thread. On macOS, `getlantern/systray` v1.2.2+ supports this via `systray.Register()` + `systray.Run()` split, or use the `fyne.io/systray` fork which is goroutine-safe. **This is the highest-risk integration point and should be prototyped in Wave 4 before committing.**

6. **LSUIElement stays false for v1** — Toggling Dock visibility at runtime requires Objective-C interop (`NSApp.setActivationPolicy`). Deferring this to a follow-up. The window hides but the Dock icon remains.
