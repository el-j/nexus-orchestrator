# TASK-278 — sys_scanner: Agent-specific probes

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Go · outbound (`internal/adapters/outbound/sys_scanner/`)  
**Depends on:** TASK-276  

## Objective

Implement `ports.AgentScanner` on the existing `Scanner` struct by adding a `ScanAgents(ctx)` method in a new file. Five probe types, all concurrent.

## New file: `internal/adapters/outbound/sys_scanner/agent_scanner.go`

Add `var _ ports.AgentScanner = (*Scanner)(nil)` compile-time check.

### Probe 1 — `probeClaudeConfig(ctx)`
- `os.ReadFile(filepath.Join(os.Getenv("HOME"), ".claude", "settings.json"))` → if non-empty valid JSON emit `DiscoveredAgent{ID: "claude-cli", Kind: AgentKindClaudeCLI, Name: "Claude CLI", DetectionMethod: "fs-config", ConfigPath: ...}`
- macOS: `os.Stat(filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Claude"))` → if exists emit `{ID: "claude-desktop", Kind: AgentKindClaudeDesktop, ...}`
- Linux: `os.Stat(filepath.Join(os.Getenv("HOME"), ".config", "claude"))` → same
- Never return error for missing files; use `os.IsNotExist` guard.

### Probe 2 — `probeVSCodeExtensions(ctx)`
```
~/.vscode/extensions/  (Linux/macOS)
%USERPROFILE%\.vscode\extensions\  (Windows)
```
Read directory; match name prefix against:
```go
var vscodeExtMap = map[string]struct{ kind AgentKind; name string }{
    "saoudrizwan.claude-dev": {AgentKindCline, "Cline"},
    "continue.continue":      {AgentKindContinue, "Continue"},
    "codeium.codeium":        {AgentKindCodeGPT, "Codeium"},  // Codeium
    "codegpt.codegpt":        {AgentKindCodeGPT, "CodeGPT"},
    "anysphere.cursor":       {AgentKindCursor, "Cursor AI"},
    "github.copilot":         {AgentKindCopilot, "GitHub Copilot"},
}
```
For each matched prefix, emit a `DiscoveredAgent{DetectionMethod: "vscode-extension"}`.

### Probe 3 — `probeMCPPorts(ctx)`
Sweep ports: `[]int{3000, 3001, 3100, 5100, 6006, 7007, 8008, 9009}`. For each:
1. TCP dial with 500 ms timeout.
2. If open, POST JSON-RPC `initialize` request to `http://127.0.0.1:<port>/mcp`.  
   Request: `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"nexus-probe","version":"1"}}}`
3. Parse response; if `result.serverInfo.name` present, emit `DiscoveredAgent{Kind: AgentKindGeneric, DetectionMethod: "port-mcp", MCPEndpoint: "http://127.0.0.1:<port>/mcp", Name: serverInfo.name}`.
4. Use per-request context with 500 ms deadline.

### Probe 4 — `probeProcessFlags(ctx)`
Skip on `runtime.GOOS == "windows"`. Run:
- `pgrep -lf -- --mcp`
- `pgrep -lf -- --mcp-server`
Parse each output line; split on first space to get PID + process name. Emit one `DiscoveredAgent{Kind: AgentKindGeneric, DetectionMethod: "process-flag", IsRunning: true, ProcessName: ...}` per unique process name.

### Probe 5 — `probeAgentProcesses(ctx)`
Re-use existing `runPgrep` helper (extracted from `probeProcess`). Sweep patterns:
```go
[]struct{ pattern string; kind AgentKind; name string }{
    {"Claude", AgentKindClaudeDesktop, "Claude Desktop"},
    {"Antigravity", AgentKindAntigravity, "Antigravity"},
}
```
Emit `DiscoveredAgent{DetectionMethod: "process", IsRunning: true}`.

### Deduplication
After all probes complete, merge results by `ID` (kind string). If same ID seen via multiple methods, keep the one with `IsRunning: true` if any, else keep first; merge `MCPEndpoint` and `ConfigPath` from whichever has them.

## Acceptance Criteria

- `go build ./internal/adapters/outbound/sys_scanner/...` succeeds
- `var _ ports.AgentScanner = (*Scanner)(nil)` compiles
- Tests in TASK-289 pass
