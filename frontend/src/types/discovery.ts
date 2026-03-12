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
  lastSeen: string // ISO timestamp
}

export interface LogEntry {
  timestamp: string // ISO
  level: 'info' | 'warn' | 'error' | 'debug'
  source: string // e.g. "httpapi", "orchestrator", "lmstudio"
  message: string
}
