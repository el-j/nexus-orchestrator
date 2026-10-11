/**
 * nexusTypes.ts — request/response types mirroring the Go domain types and
 * ports.ProviderInfo. Re-exported from nexusClient.ts.
 */

// ---- Domain types (mirror internal/core/domain/task.go) ----

export type TaskStatus =
  | 'QUEUED'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER'
  | 'DRAFT'
  | 'BACKLOG';

/** Task command: plan, execute or auto-routed. */
export type CommandType = 'plan' | 'execute' | 'auto' | '';

/** A task as returned by the daemon REST API. */
export interface Task {
  id: string;
  projectPath: string;
  targetFile: string;
  instruction: string;
  contextFiles: string[];
  modelId?: string;
  providerHint?: string;
  command?: CommandType;
  status: TaskStatus;
  createdAt: string;
  updatedAt: string;
  logs?: string;
  aiSessionId?: string;
}

// ---- Provider type (mirrors ports.ProviderInfo) ----

export interface Provider {
  name: string;
  active: boolean;
  kind?: string;
  activeModel?: string;
  models?: string[];
  baseURL?: string;
  details?: string;
  error?: string;
  contextLimit?: number;
}

// ---- Request types ----

export interface SubmitTaskRequest {
  instruction: string;
  projectPath: string;
  targetFile?: string;
  providerHint?: string;
  modelId?: string;
  command?: CommandType;
  contextFiles?: string[];
}

// ---- AI Session types ----

export interface AISession {
  id: string;
  agentName: string;
  source: string;
  status: string;
  lastActivity: string;
  projectPath?: string;
  delegatedToNexus?: boolean;
  delegationTimestamp?: string;
  agentCapabilities?: string[];
  detectionMethod?: string;
}

/** An agent found on the machine by the daemon's system scanner. */
export interface DiscoveredAgent {
  id: string;
  kind: string;
  name: string;
  detectionMethod: string;
  processName?: string;
  cliPath?: string;
  configPath?: string;
  mcpEndpoint?: string;
  isRunning: boolean;
  lastSeen: string;
}

// ---- AI Activity types ----

export interface AIActivity {
  id: string;
  sessionId?: string;
  agentName: string;
  activityType: 'message' | 'tool_use' | 'thinking' | 'file_edit' | 'generation';
  summary: string;
  projectPath?: string;
  model?: string;
  tokensIn?: number;
  tokensOut?: number;
  timestamp: string;
  metadata?: Record<string, string>;
}

/** The daemon's answer to a delegation: instruction text and session id. */
export interface DelegateResponse {
  instruction: string;
  sessionId: string;
}

/** Body for registering an AI session with the daemon. */
export interface RegisterSessionRequest {
  agentName: string;
  source: 'vscode' | 'mcp' | 'http';
  projectPath?: string;
  externalId?: string;
  modelId?: string;
}

// ---- Brain Context types ----

export interface BrainStatus {
  projectPath: string;
  initialized: boolean;
  entryCount: number;
  kindCounts: Record<string, number>;
  totalTokens: number;
  lastUpdated?: string;
}

/** Parameters for a project context request. */
export interface ContextQuery {
  projectPath: string;
  question?: string;
  maxTokens?: number;
}

/** One ranked section of project knowledge in a context response. */
export interface ContextSection {
  title?: string;
  topic: string;
  kind: string;
  content: string;
  tokens: number;
  source: string;
}

/** A stored knowledge entry for a project. */
export interface ProjectKnowledge {
  id: string;
  projectPath: string;
  kind: string;
  topic: string;
  content: string;
  tokenCount: number;
  relevanceScore: number;
  createdAt: string;
  updatedAt: string;
}

/** Token-budgeted project context returned by the Context Brain. */
export interface ContextResponse {
  projectPath: string;
  sections: ContextSection[];
  /** File paths the context suggests opening next. */
  suggestedFiles?: string[];
  /** Tokens used by `sections`. */
  tokensUsed: number;
  /** The budget the sections were assembled within. */
  tokenBudget: number;
  truncated: boolean;
}

/** One hit of a knowledge search. */
export interface KnowledgeResult {
  id: string;
  projectPath: string;
  kind: string;
  topic: string;
  content: string;
  source: string;
  tokens: number;
  relevance: number;
  createdAt: string;
  updatedAt: string;
}

// ---- Internal response shape from POST /api/tasks ----

export interface CreateTaskResponse {
  task_id: string;
  status: string;
}

// ---- Health response ----

export interface HealthResponse {
  status: string;
  service: string;
}

// ---- Provider config (mirrors daemon ProviderConfig) ----

export interface ProviderConfig {
  id: string;
  name: string;
  kind: string;
  baseURL?: string;
  model?: string;
  enabled: boolean;
}
