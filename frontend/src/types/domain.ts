export type TaskStatus =
  | 'DRAFT'
  | 'BACKLOG'
  | 'QUEUED'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER';

export type CommandType = 'plan' | 'execute' | 'auto';

export interface Task {
  id: string;
  projectPath: string;
  targetFile: string;
  instruction: string;
  contextFiles: string[];
  modelId: string;
  providerHint: string;
  command: CommandType;
  status: TaskStatus;
  createdAt: string;
  updatedAt: string;
  retryCount?: number;
  logs: string;
  providerName?: string;
  priority?: number;
  tags?: string[];
}

export interface TaskInput {
  projectPath: string;
  targetFile: string;
  instruction: string;
  contextFiles?: string[];
  modelId?: string;
  providerHint?: string;
  command?: CommandType;
}

export interface ProviderInfo {
  name: string;
  active: boolean;
  activeModel: string;
  models: string[];
  baseURL?: string;
  error?: string;
}

export interface ProviderConfig {
  id: string;
  name: string;
  kind: 'lmstudio' | 'ollama' | 'openai' | 'anthropic' | 'openaicompat';
  baseUrl: string;
  apiKey: string;
  model: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export type LogLevel = 'info' | 'warn' | 'error' | 'debug';

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  source: string;
  message: string;
}

export interface DiscoveredProvider {
  id: string;
  name: string;
  kind: string;
  method: 'port' | 'cli' | 'process';
  status: 'reachable' | 'installed' | 'running';
  baseUrl?: string;
  cliPath?: string;
  processName?: string;
  models?: string[];
  lastSeen: string;
}

export type AISessionSource = 'mcp' | 'vscode' | 'http';
export type AISessionStatus = 'active' | 'idle' | 'disconnected';

export interface AISession {
  id: string;
  source: AISessionSource;
  externalId?: string;
  agentName: string;
  projectPath?: string;
  status: AISessionStatus;
  lastActivity: string;
  routedTaskIds?: string[];
  delegatedToNexus?: boolean;
  agentCapabilities?: string[];
  createdAt: string;
  updatedAt: string;
  modelId?: string;
}

export interface DiscoveredAgent {
  id: string;
  kind: string;
  name: string;
  detectionMethod: string;
  isRunning: boolean;
  lastSeen: string;
  mcpEndpoint?: string;
  configPath?: string;
  modelId?: string;
  workingDir?: string;
  parentAgentId?: string;
  subAgentIds?: string[];
}

export type PlanFileKind =
  | 'nexus'
  | 'claude-task'
  | 'markdown'
  | 'cursor'
  | 'mcp-config'
  | 'crewai'
  | 'claude';

export interface DiscoveredPlanFile {
  id: string;
  path: string;
  kind: PlanFileKind;
  format: string;
  projectPath: string;
  summary?: string;
  lastModified: string;
  isActive: boolean;
}

export type ActivityType = 'message' | 'tool_use' | 'thinking' | 'file_edit' | 'generation';

export interface AIActivity {
  id: string;
  sessionId?: string;
  agentName: string;
  activityType: ActivityType;
  summary: string;
  projectPath?: string;
  model?: string;
  tokensIn?: number;
  tokensOut?: number;
  timestamp: string;
  metadata?: Record<string, string>;
}

export interface RuntimeConfig {
  queueCap: number;
  apiTokenEnabled: boolean;
  mcpTokenEnabled: boolean;
  apiToken?: string;
  mcpToken?: string;
}

export interface RuntimeConfigUpdate {
  queueCap?: number;
  apiToken?: string;
  mcpToken?: string;
  rotateApiToken?: boolean;
  rotateMcpToken?: boolean;
}
