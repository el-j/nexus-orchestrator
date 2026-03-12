export type TaskStatus =
  | 'DRAFT'
  | 'BACKLOG'
  | 'QUEUED'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER'

export type CommandType = 'plan' | 'execute' | 'auto'

export interface Task {
  ID: string
  ProjectPath: string
  TargetFile: string
  Instruction: string
  ContextFiles: string[]
  ModelID: string
  ProviderHint: string
  Command: CommandType
  Status: TaskStatus
  CreatedAt: string
  UpdatedAt: string
  Logs: string
  ProviderName?: string
  Priority?: number
  Tags?: string[]
}

export interface TaskInput {
  ProjectPath: string
  TargetFile: string
  Instruction: string
  ContextFiles?: string[]
  ModelID?: string
  ProviderHint?: string
  Command?: CommandType
}

export interface ProviderInfo {
  name: string
  active: boolean
  activeModel: string
  models: string[]
  baseURL?: string
  error?: string
}

export interface ProviderConfig {
  id: string
  name: string
  kind: 'lmstudio' | 'ollama' | 'openai' | 'anthropic' | 'openaicompat'
  baseURL: string
  apiKey: string
  defaultModel: string
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export type LogLevel = 'info' | 'warn' | 'error' | 'debug'

export interface LogEntry {
  timestamp: string
  level: LogLevel
  source: string
  message: string
}

export interface DiscoveredProvider {
  id: string
  name: string
  kind: string
  method: 'port' | 'cli' | 'process'
  status: 'reachable' | 'installed' | 'running'
  baseURL?: string
  cliPath?: string
  processName?: string
  models?: string[]
  lastSeen: string
}

export type AISessionSource = 'mcp' | 'vscode' | 'http'
export type AISessionStatus = 'active' | 'idle' | 'disconnected'

export interface AISession {
  id: string
  source: AISessionSource
  externalId?: string
  agentName: string
  projectPath?: string
  status: AISessionStatus
  lastActivity: string
  routedTaskIds?: string[]
  createdAt: string
  updatedAt: string
}