/** Task lifecycle status. */
export type TaskStatus =
  | 'DRAFT'
  | 'BACKLOG'
  | 'QUEUED'
  | 'WAITING'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER';

/** Task command: plan, execute or auto-routed. */
export type CommandType = 'plan' | 'execute' | 'auto';

/** A task as returned by the daemon. */
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
  /** When a WAITING task is queued again (ISO 8601): every provider is cooling down until then. */
  waitUntil?: string;
  logs: string;
  providerName?: string;
  /** The provider Nexus ran the task on; absent when an agent set the status or it has not run. */
  ranBy?: string;
  durationMs?: number;
  priority?: number;
  tags?: string[];
  role?: string;
  agentic?: boolean;
  maxAgentTurns?: number;
}

/** Fields accepted when submitting or drafting a task. */
export interface TaskInput {
  projectPath: string;
  targetFile: string;
  instruction: string;
  contextFiles?: string[];
  modelId?: string;
  providerHint?: string;
  command?: CommandType;
  priority?: number;
  tags?: string[];
  agentic?: boolean;
  maxAgentTurns?: number;
}

/** Liveness summary of one LLM backend. */
export interface ProviderInfo {
  name: string;
  active: boolean;
  activeModel: string;
  models: string[];
  baseURL?: string;
  error?: string;
  contextLimit?: number;
  timeoutSec?: number;
  lastChecked?: string;
  consecutiveFails?: number;
  /** True while the provider cools down after a rate limit, an exhausted quota or an overload. */
  rateLimited?: boolean;
  /** When a cooling-down provider is tried again (ISO 8601). */
  rateLimitedUntil?: string;
  /** Why the provider cools down, for example "anthropic: rate limited (429)". */
  cooldownReason?: string;
  /** The plan's own reading of its allowance, where the provider's client reports it. */
  quota?: {
    window: string;
    status: string;
    utilization: number;
    resetsAt?: string;
  };
}

/** Tokens and cost the models consumed, added up for one provider, model and role. */
export interface UsageSummary {
  provider: string;
  model?: string;
  role?: string;
  calls: number;
  inputTokens: number;
  outputTokens: number;
  cachedTokens?: number;
  costUSD?: number;
}

/** The usage ledger answer of `GET /api/usage`. */
export interface UsageReport {
  /** Start of the window (ISO 8601). */
  since: string;
  totals: UsageSummary;
  usage: UsageSummary[];
}

/** One candidate provider of a task, in the order routing would try it. */
export interface RoutingEntry {
  position: number;
  provider: string;
  model?: string;
  rung: number;
  coolingUntil?: string;
  cooldownReason?: string;
  utilization?: number;
}

/** What routing would do for a role in a project right now (`GET /api/routing`). */
export interface RoutingPreview {
  role?: string;
  ladder?: string[];
  quotaThreshold: number;
  order: RoutingEntry[];
}

/** A saved provider configuration. */
export interface ProviderConfig {
  id: string;
  name: string;
  kind: 'lmstudio' | 'ollama' | 'openai' | 'anthropic' | 'gemini' | 'openaicompat' | 'cli';
  baseUrl: string;
  apiKey: string;
  model: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

/** Severity of a log entry. */
export type LogLevel = 'info' | 'warn' | 'error' | 'debug';

/** One daemon log line. */
export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  source: string;
  message: string;
}

/** A provider found by the system scanner, not yet saved. */
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
  /** Models currently loaded in memory (Ollama /api/ps). Non-empty means actively in use. */
  activeModels?: string[];
  /** True when the provider is actively generating a response right now. */
  generating?: boolean;
  lastSeen: string;
}

/** How an AI session came to exist: registered over HTTP or discovered by VS Code. */
export type AISessionSource = 'mcp' | 'vscode' | 'http';
/** How recently an AI session was heard from. */
export type AISessionStatus = 'active' | 'idle' | 'disconnected';

/** An AI agent session known to the daemon. */
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

/** An agent found on the machine by the system scanner. */
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

/** Which convention a discovered plan file follows. */
export type PlanFileKind =
  'nexus' | 'claude-task' | 'markdown' | 'cursor' | 'mcp-config' | 'crewai' | 'claude';

/** A plan file found on disk. */
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

/** Kind of an agent activity entry. */
export type ActivityType = 'message' | 'tool_use' | 'thinking' | 'file_edit' | 'generation';

/** One entry of an agent's activity stream. */
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

/** Daemon settings that can change without a restart. */
export interface RuntimeConfig {
  queueCap: number;
  apiTokenEnabled: boolean;
  mcpTokenEnabled: boolean;
  apiToken?: string;
  mcpToken?: string;
}

/** Partial update of the runtime config. */
export interface RuntimeConfigUpdate {
  queueCap?: number;
  apiToken?: string;
  mcpToken?: string;
  rotateApiToken?: boolean;
  rotateMcpToken?: boolean;
}

/** Size and health of the Context Brain. */
export interface BrainStatus {
  projectPath: string;
  initialized: boolean;
  entryCount: number;
  kindCounts: Record<string, number>;
  totalTokens: number;
  lastUpdated?: string;
}

/** Parameters of a project context request. */
export interface ContextQuery {
  projectPath: string;
  question?: string;
  maxTokens?: number;
}

/** One ranked section of project knowledge. */
export interface ContextSection {
  title?: string;
  topic?: string;
  kind: string;
  content: string;
  tokens: number;
  source?: string;
  tokenCount?: number;
  relevanceScore?: number;
}

/** Token-budgeted project context. */
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
export type KnowledgeResult = ProjectKnowledge;

/** A stored knowledge entry. */
export interface ProjectKnowledge {
  id: string;
  projectPath: string;
  kind: string;
  topic: string;
  content: string;
  source?: string;
  tokenCount: number;
  relevanceScore: number;
  createdAt: string;
  updatedAt: string;
}
