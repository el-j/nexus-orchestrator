/**
 * nexusClient.ts — HTTP client for the nexusOrchestrator daemon REST API.
 *
 * All types mirror the Go domain types and ports.ProviderInfo exactly.
 * Uses native fetch (Node 18+ / VS Code's built-in runtime).
 */

import { z } from 'zod';
import {
  NexusValidationError,
  AIActivitySchema,
  DiscoveredAgentSchema,
  BrainStatusSchema,
  ContextResponseSchema,
  ContextSectionSchema,
  ProjectKnowledgeSchema,
  DelegateResponseSchema,
  IngestKnowledgeResponseSchema,
  SearchKnowledgeResponseSchema,
  FileMapResponseSchema,
} from './schemas';

export { NexusValidationError };

// ---- Domain types (mirror internal/core/domain/task.go) ----

export type TaskStatus =
  | 'QUEUED'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER';

export type CommandType = 'plan' | 'execute' | 'auto' | '';

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

export interface DelegateResponse {
  instruction: string;
  sessionId: string;
}

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

export interface ContextQuery {
  projectPath: string;
  question?: string;
  maxTokens?: number;
}

export interface ContextSection {
  title?: string;
  topic: string;
  kind: string;
  content: string;
  tokens: number;
  source: string;
}

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

export interface ContextResponse {
  projectPath: string;
  sections: ContextSection[];
  totalTokens: number;
  truncated: boolean;
}

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

interface CreateTaskResponse {
  task_id: string;
  status: string;
}

// ---- Health response ----

interface HealthResponse {
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

// ---- Validation error ----

export class NexusClientError extends Error {
  constructor(
    message: string,
    readonly cause?: unknown,
  ) {
    super(message);
    this.name = 'NexusClientError';
  }
}

// ---- Zod schemas for runtime validation of critical API responses ----

const TaskStatusSchema = z.enum([
  'QUEUED',
  'PROCESSING',
  'COMPLETED',
  'FAILED',
  'CANCELLED',
  'TOO_LARGE',
  'NO_PROVIDER',
]);

const TaskSchema = z
  .object({
    id: z.string(),
    projectPath: z.string(),
    targetFile: z.string(),
    instruction: z.string(),
    contextFiles: z
      .array(z.string())
      .nullish()
      .transform((v) => v ?? []),
    modelId: z.string().optional(),
    providerHint: z.string().optional(),
    command: z.enum(['plan', 'execute', 'auto', '']).optional(),
    status: TaskStatusSchema,
    createdAt: z.string(),
    updatedAt: z.string(),
    logs: z.string().optional(),
    aiSessionId: z.string().optional(),
    claimedBy: z.string().optional(),
    claimedAt: z.string().optional(),
    retryCount: z.number().optional(),
  })
  .passthrough();

const AISessionSchema = z
  .object({
    id: z.string(),
    agentName: z.string(),
    source: z.string(),
    status: z.string(),
    lastActivity: z.string(),
    projectPath: z.string().optional(),
    delegatedToNexus: z.boolean().optional(),
    delegationTimestamp: z.string().optional(),
    agentCapabilities: z.array(z.string()).optional(),
    detectionMethod: z.string().optional(),
  })
  .passthrough();

const ProviderSchema = z
  .object({
    name: z.string(),
    active: z.boolean(),
    kind: z.string().optional(),
    activeModel: z.string().optional(),
    models: z.array(z.string()).optional(),
    baseURL: z.string().optional(),
    details: z.string().optional(),
    error: z.string().optional(),
    contextLimit: z.number().optional(),
  })
  .passthrough();

const ProviderConfigSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    kind: z.string(),
    baseURL: z.string().optional(),
    model: z.string().optional(),
    enabled: z.boolean(),
  })
  .passthrough();

// ---- Client ----

export class NexusClient {
  constructor(
    private baseUrl: string,
    private mcpPort: number = 63988,
  ) {}

  /** Expose the configured MCP port. */
  getMcpPort(): number {
    return this.mcpPort;
  }

  /**
   * Probe each port with HEAD /health and return the first responding port.
   * Deduplicates the list; user-supplied port is probed first.
   */
  async tryConnect(ports: number[]): Promise<number> {
    const seen = new Set<number>();
    const url = new URL(this.baseUrl);
    for (const port of ports) {
      if (seen.has(port)) continue;
      seen.add(port);
      try {
        const probeUrl = `${url.protocol}//${url.hostname}:${port}/health`;
        const resp = await fetch(probeUrl, {
          method: 'HEAD',
          signal: AbortSignal.timeout(1500),
        });
        if (resp.ok) return port;
      } catch {
        // port not responding — try next
      }
    }
    throw new Error('nexus: no responding daemon found on any candidate port');
  }

  /**
   * Submit a new task. The daemon returns only {task_id, status}, so we
   * immediately fetch the full task to return a complete Task object.
   */
  async submitTask(payload: SubmitTaskRequest): Promise<Task> {
    const resp = await this.post<CreateTaskResponse>('/api/tasks', payload);
    return this.getTask(resp.task_id);
  }

  /** Return all tasks currently in the queue. */
  async getTasks(): Promise<Task[]> {
    return this.get('/api/tasks', z.array(TaskSchema));
  }

  /** Return ALL tasks regardless of status (completed, failed, queued, etc.). */
  async getAllTasks(): Promise<Task[]> {
    return this.get('/api/tasks/all', z.array(TaskSchema));
  }

  /** Return a single task by ID. Throws if not found (404). */
  async getTask(id: string): Promise<Task> {
    return this.get(`/api/tasks/${encodeURIComponent(id)}`, TaskSchema);
  }

  /**
   * Cancel a task. Returns void on success (204 No Content).
   * Throws if the task is not found (404).
   */
  async cancelTask(id: string): Promise<void> {
    const url = `${this.baseUrl}/api/tasks/${encodeURIComponent(id)}`;
    const resp = await fetch(url, { method: 'DELETE' });
    if (!resp.ok) {
      const body = await resp.text().catch(() => '');
      throw new Error(
        `nexus: cancel task ${id}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ''}`,
      );
    }
  }

  /** Return all registered LLM providers and their liveness status. */
  async getProviders(): Promise<Provider[]> {
    return this.get('/api/providers', z.array(ProviderSchema));
  }

  /** Register a new AI session with the daemon. */
  async registerSession(req: RegisterSessionRequest): Promise<AISession> {
    return this.post('/api/ai-sessions', req, AISessionSchema);
  }

  /** Deregister an AI session by ID. */
  async deregisterSession(id: string): Promise<void> {
    const url = `${this.baseUrl}/api/ai-sessions/${encodeURIComponent(id)}`;
    const resp = await fetch(url, { method: 'DELETE' });
    if (!resp.ok) {
      const body = await resp.text().catch(() => '');
      throw new Error(
        `nexus: deregister session ${id}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ''}`,
      );
    }
  }

  /** Send a heartbeat for an existing session to refresh its last-activity timestamp. */
  async heartbeatSession(id: string): Promise<void> {
    const url = `${this.baseUrl}/api/ai-sessions/${encodeURIComponent(id)}/heartbeat`;
    const resp = await fetch(url, { method: 'POST' });
    if (!resp.ok && resp.status !== 404) {
      // 404 means the session was cleaned up — caller should re-register.
      const body = await resp.text().catch(() => '');
      throw new Error(
        `nexus: heartbeat session ${id}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ''}`,
      );
    }
  }

  /** Return all registered AI sessions. */
  async getAISessions(): Promise<AISession[]> {
    return this.get('/api/ai-sessions', z.array(AISessionSchema));
  }

  /** Claim a queued task for the given session. */
  async claimTask(taskId: string, sessionId: string): Promise<Task> {
    return this.post(`/api/tasks/${encodeURIComponent(taskId)}/claim`, { sessionId }, TaskSchema);
  }

  /** Update a task's status (COMPLETED or FAILED). */
  async updateTaskStatus(
    taskId: string,
    sessionId: string,
    status: 'COMPLETED' | 'FAILED',
    logs?: string,
  ): Promise<Task> {
    return this.put(
      `/api/tasks/${encodeURIComponent(taskId)}/status`,
      {
        sessionId,
        status,
        logs,
      },
      TaskSchema,
    );
  }

  /** Get all tasks bound to a specific AI session. */
  async getSessionTasks(sessionId: string): Promise<Task[]> {
    return this.get(`/api/ai-sessions/${encodeURIComponent(sessionId)}/tasks`, z.array(TaskSchema));
  }

  /** Return all registered AI sessions (alias with explicit name). */
  async listAISessions(): Promise<AISession[]> {
    return this.get('/api/ai-sessions', z.array(AISessionSchema));
  }

  /** Return all discovered AI agents on this machine. */
  async getDiscoveredAgents(): Promise<DiscoveredAgent[]> {
    return this.get('/api/ai-sessions/discovered', z.array(DiscoveredAgentSchema));
  }

  /** Return recent AI activities, optionally filtered by agent, project, or type. */
  async getActivities(options?: {
    agentName?: string;
    projectPath?: string;
    type?: string;
    limit?: number;
  }): Promise<AIActivity[]> {
    const params = new URLSearchParams();
    if (options?.agentName) params.set('agent', options.agentName);
    if (options?.projectPath) params.set('project', options.projectPath);
    if (options?.type) params.set('type', options.type);
    if (options?.limit) params.set('limit', String(options.limit));
    const path = `/api/activities${params.size ? '?' + params.toString() : ''}`;
    const resp = await fetch(`${this.baseUrl}${path}`, { signal: AbortSignal.timeout(3000) });
    if (!resp.ok) return [];
    const data: unknown = await resp.json();
    return this.parseResponse(z.array(AIActivitySchema), data);
  }

  /** Delegate a session to nexusOrchestrator for task handling. */
  async delegateSession(sessionId: string): Promise<DelegateResponse> {
    return this.post(
      `/api/ai-sessions/${encodeURIComponent(sessionId)}/delegate`,
      {},
      DelegateResponseSchema,
    );
  }

  /**
   * Ping the daemon's health endpoint.
   * Returns true when the daemon is reachable and reports status "ok".
   */
  async health(): Promise<boolean> {
    try {
      const resp = await fetch(`${this.baseUrl}/api/health`);
      if (!resp.ok) {
        return false;
      }
      const body = (await resp.json()) as HealthResponse;
      return body.status === 'ok';
    } catch {
      return false;
    }
  }

  // ---- Brain APIs ----

  /**
   * Ingest a markdown file into the project's brain context.
   */
  async ingestKnowledge(projectPath: string, filePath: string): Promise<number> {
    const data = await this.post(
      '/api/brain/ingest',
      { projectPath, filePath },
      IngestKnowledgeResponseSchema,
    );
    return data.ingestedSections;
  }

  /**
   * Get the indexing status and token size of the project knowledge brain.
   */
  async getBrainStatus(projectPath: string): Promise<BrainStatus> {
    return this.get(
      `/api/brain/status?projectPath=${encodeURIComponent(projectPath)}`,
      BrainStatusSchema,
    );
  }

  /**
   * Aggregate the top-level macro context for LLM system prompts.
   */
  async getProjectContext(projectPath: string, maxTokens?: number): Promise<ContextResponse> {
    return this.post('/api/brain/context', { projectPath, maxTokens }, ContextResponseSchema);
  }

  /**
   * Query bounded context sections specific to a reasoning question.
   */
  async getFocusedContext(
    projectPath: string,
    question: string,
    maxTokens?: number,
  ): Promise<ContextResponse> {
    return this.post(
      '/api/brain/focused-context',
      { projectPath, question, maxTokens },
      ContextResponseSchema,
    );
  }

  /**
   * Search project intelligence via BM25 matching.
   */
  async searchKnowledge(projectPath: string, query: string, limit = 5): Promise<ContextSection[]> {
    const params = new URLSearchParams({ projectPath, query, limit: String(limit) });
    const data = await this.get(`/api/brain/search?${params}`, SearchKnowledgeResponseSchema);
    return data.results ?? [];
  }

  /**
   * Initialize the project brain, optionally seeding from a CLAUDE.md file.
   */
  async initProject(projectPath: string, claudeMDPath?: string): Promise<BrainStatus> {
    return this.post(
      '/api/brain/init',
      { projectPath, claudeMDPath: claudeMDPath ?? '' },
      BrainStatusSchema,
    );
  }

  /**
   * List all knowledge entries for a project, optionally filtered by kind.
   */
  async listKnowledge(projectPath: string, kind?: string): Promise<ProjectKnowledge[]> {
    const params = new URLSearchParams({ projectPath });
    if (kind) params.set('kind', kind);
    const data = await this.get(
      `/api/brain/knowledge?${params}`,
      z.array(ProjectKnowledgeSchema).nullable(),
    );
    return data ?? [];
  }

  /**
   * Delete a knowledge entry by ID.
   */
  async deleteKnowledge(id: string): Promise<void> {
    await this.delete(`/api/brain/knowledge/${encodeURIComponent(id)}`);
  }

  /**
   * Get the ordered list of file paths from the project file-map knowledge.
   */
  async getFileMap(projectPath: string, focusArea?: string): Promise<string[]> {
    const params = new URLSearchParams({ projectPath });
    if (focusArea) params.set('focusArea', focusArea);
    const data = await this.get(`/api/brain/file-map?${params}`, FileMapResponseSchema);
    return data.filePaths ?? [];
  }

  // ---- Private helpers ----

  private parseResponse<T>(schema: z.ZodSchema<T>, data: unknown): T {
    const result = schema.safeParse(data);
    if (!result.success) {
      throw new NexusClientError(
        `nexus: response validation failed: ${result.error.message}`,
        result.error,
      );
    }
    return result.data;
  }

  private async get<T>(path: string, schema?: z.ZodSchema<T>): Promise<T> {
    const resp = await fetch(`${this.baseUrl}${path}`);
    if (!resp.ok) {
      const body = await resp.text().catch(() => '');
      throw new Error(`nexus: GET ${path}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ''}`);
    }
    const data: unknown = await resp.json();
    return schema ? this.parseResponse(schema, data) : (data as T);
  }

  private async post<T>(path: string, payload: unknown, schema?: z.ZodSchema<T>): Promise<T> {
    const resp = await fetch(`${this.baseUrl}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!resp.ok) {
      const body = await resp.text().catch(() => '');
      throw new Error(`nexus: POST ${path}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ''}`);
    }
    const data: unknown = await resp.json();
    return schema ? this.parseResponse(schema, data) : (data as T);
  }

  private async delete(path: string): Promise<void> {
    const resp = await fetch(`${this.baseUrl}${path}`, { method: 'DELETE' });
    if (!resp.ok) {
      const body = await resp.text().catch(() => '');
      throw new Error(
        `nexus: DELETE ${path}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ''}`,
      );
    }
  }

  private async put<T>(path: string, payload: unknown, schema?: z.ZodSchema<T>): Promise<T> {
    const resp = await fetch(`${this.baseUrl}${path}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!resp.ok) {
      const body = await resp.text().catch(() => '');
      throw new Error(`nexus: PUT ${path}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ''}`);
    }
    const data: unknown = await resp.json();
    return schema ? this.parseResponse(schema, data) : (data as T);
  }
}
