export type DiscoveryMethod = 'port' | 'cli' | 'process'
export type DiscoveryStatus = 'reachable' | 'installed' | 'running'

export interface DiscoveredProvider {
  id: string
  name: string
  kind: string // lmstudio, ollama, localai, vllm, textgen, cli, desktopapp
  method: DiscoveryMethod
  baseUrl?: string
  cliPath?: string
  processName?: string
  status: DiscoveryStatus
  models?: string[]
  /** Models currently loaded in memory (Ollama /api/ps). Non-empty means actively in use. */
  activeModels?: string[]
  /** True when the provider is actively generating a response right now. */
  generating?: boolean
  lastSeen: string // ISO timestamp
}

export interface LogEntry {
  timestamp: string // ISO
  level: 'info' | 'warn' | 'error' | 'debug'
  source: string // e.g. "httpapi", "orchestrator", "lmstudio"
  message: string
}
