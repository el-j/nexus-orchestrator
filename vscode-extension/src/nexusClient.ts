/**
 * nexusClient.ts — HTTP client for the nexusOrchestrator daemon REST API.
 *
 * All types mirror the Go domain types and ports.ProviderInfo exactly.
 * Uses native fetch (Node 18+ / VS Code's built-in runtime).
 */

import { z } from 'zod';
import type {
  AISession,
  Provider,
  ProviderConfig,
  Task,
  TaskStatus,
  CommandType,
  SubmitTaskRequest,
  DiscoveredAgent,
  AIActivity,
  DelegateResponse,
  RegisterSessionRequest,
  BrainStatus,
  ContextSection,
  ProjectKnowledge,
  ContextResponse,
  KnowledgeResult,
  CreateTaskResponse,
  HealthResponse,
} from './nexusTypes';
import {
  TaskStatusSchema,
  TaskSchema,
  AISessionSchema,
  ProviderSchema,
  ProviderConfigSchema,
} from './nexusSchemas';

export * from './nexusTypes';

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

// ---- Client ----

/** Default per-request timeout; a hung daemon must not hang the editor. */
const REQUEST_TIMEOUT_MS = 15_000;

/** HTTP client for the daemon REST API (bearer token, 15 s timeout, typed errors). */
export class NexusClient {
  /**
   * @param baseUrl REST API base URL, e.g. http://127.0.0.1:63987
   * @param mcpPort port of the companion MCP server
   * @param token   bearer token when the daemon runs with NEXUS_API_TOKEN set
   */
  constructor(
    private baseUrl: string,
    private mcpPort: number = 63988,
    private token: string = '',
  ) {}

  /** Expose the configured MCP port. */
  getMcpPort(): number {
    return this.mcpPort;
  }

  /** The REST API base URL this client talks to. */
  getBaseUrl(): string {
    return this.baseUrl;
  }

  /**
   * Re-point this client in place. Components capture the client once at
   * activation, so a settings change must update the shared instance rather
   * than create a new one that only some of them would see.
   */
  configure(baseUrl: string, mcpPort: number, token = ''): void {
    this.baseUrl = baseUrl;
    this.mcpPort = mcpPort;
    this.token = token;
  }

  /**
   * Probe each REST port with GET /api/health and return the first responding port.
   * Deduplicates the list; the first candidate is probed first.
   */
  async tryConnect(ports: number[]): Promise<number> {
    const seen = new Set<number>();
    const url = new URL(this.baseUrl);
    for (const port of ports) {
      if (seen.has(port)) continue;
      seen.add(port);
      try {
        const probeUrl = `${url.protocol}//${url.hostname}:${port}/api/health`;
        const resp = await fetch(probeUrl, { signal: AbortSignal.timeout(1500) });
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
    const resp = await this.send(url, { method: 'DELETE' });
    if (!resp.ok) throw await this.failure(`cancel task ${id}`, resp);
  }

  /** Return all registered LLM providers and their liveness status. */
  async getProviders(): Promise<Provider[]> {
    return this.get('/api/providers', z.array(ProviderSchema));
  }

  /** Register an OpenAI-compatible provider with the running daemon (in memory, not persisted). */
  async registerProvider(cfg: {
    name: string;
    kind: string;
    baseUrl: string;
    apiKey: string;
    model: string;
  }): Promise<void> {
    await this.post('/api/providers', { ...cfg, enabled: true });
  }

  /** Remove a registered provider by name. A provider that is already gone is not an error. */
  async removeProvider(name: string): Promise<void> {
    try {
      await this.delete(`/api/providers/${encodeURIComponent(name)}`);
    } catch (err) {
      if (!(err instanceof Error) || !err.message.includes('HTTP 404')) throw err;
    }
  }

  /** Opens the daemon's server-sent event stream. The caller reads the body and ends it with `signal`. */
  async openEvents(signal: AbortSignal): Promise<Response> {
    const resp = await this.send(`${this.baseUrl}/api/events`, { signal });
    if (!resp.ok) throw await this.failure('GET /api/events', resp);
    return resp;
  }

  /** Register a new AI session with the daemon. */
  async registerSession(req: RegisterSessionRequest): Promise<AISession> {
    return this.post('/api/ai-sessions', req, AISessionSchema);
  }

  /** Deregister an AI session by ID. */
  async deregisterSession(id: string): Promise<void> {
    const url = `${this.baseUrl}/api/ai-sessions/${encodeURIComponent(id)}`;
    const resp = await this.send(url, { method: 'DELETE' });
    if (!resp.ok) throw await this.failure(`deregister session ${id}`, resp);
  }

  /**
   * Send a heartbeat for an existing session to refresh its last-activity timestamp.
   * Resolves true when the session is alive, false when the daemon no longer knows it
   * (404: it was purged or the daemon restarted), in which case the caller must
   * register again. Rejects on any other failure.
   */
  async heartbeatSession(id: string): Promise<boolean> {
    const url = `${this.baseUrl}/api/ai-sessions/${encodeURIComponent(id)}/heartbeat`;
    const resp = await this.send(url, { method: 'POST' });
    if (resp.status === 404) return false;
    if (!resp.ok) throw await this.failure(`heartbeat session ${id}`, resp);
    return true;
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
    return this.get<DiscoveredAgent[]>('/api/ai-sessions/discovered');
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
    const url = `${this.baseUrl}/api/activities${params.size ? '?' + params.toString() : ''}`;
    const res = await this.send(url, { signal: AbortSignal.timeout(3000) });
    if (!res.ok) return [];
    return res.json() as Promise<AIActivity[]>;
  }

  /** Delegate a session to nexusOrchestrator for task handling. */
  async delegateSession(sessionId: string): Promise<DelegateResponse> {
    return this.post<DelegateResponse>(
      `/api/ai-sessions/${encodeURIComponent(sessionId)}/delegate`,
      {},
    );
  }

  /**
   * Ping the daemon's health endpoint.
   * Returns true when the daemon is reachable and reports status "ok".
   */
  async health(): Promise<boolean> {
    try {
      const resp = await this.send(`${this.baseUrl}/api/health`);
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
    const data = await this.post<{ ingestedSections: number }>('/api/brain/ingest', {
      projectPath,
      filePath,
    });
    return data.ingestedSections;
  }

  /**
   * Get the indexing status and token size of the project knowledge brain.
   */
  async getBrainStatus(projectPath: string): Promise<BrainStatus> {
    return this.get<BrainStatus>(
      `/api/brain/status?projectPath=${encodeURIComponent(projectPath)}`,
    );
  }

  /**
   * Aggregate the top-level macro context for LLM system prompts.
   */
  async getProjectContext(projectPath: string, maxTokens?: number): Promise<ContextResponse> {
    return this.post<ContextResponse>('/api/brain/context', { projectPath, maxTokens });
  }

  /**
   * Query bounded context sections specific to a reasoning question.
   */
  async getFocusedContext(
    projectPath: string,
    question: string,
    maxTokens?: number,
  ): Promise<ContextResponse> {
    return this.post<ContextResponse>('/api/brain/focused-context', {
      projectPath,
      question,
      maxTokens,
    });
  }

  /**
   * Search project intelligence via BM25 matching.
   */
  async searchKnowledge(projectPath: string, query: string, limit = 5): Promise<ContextSection[]> {
    const params = new URLSearchParams({ projectPath, q: query, limit: String(limit) });
    const data = await this.get<{ results: ContextSection[] }>(`/api/brain/search?${params}`);
    return data.results ?? [];
  }

  /**
   * Initialize the project brain, optionally seeding from a CLAUDE.md file.
   */
  async initProject(projectPath: string, claudeMDPath?: string): Promise<BrainStatus> {
    return this.post('/api/brain/init', { projectPath, claudeMDPath: claudeMDPath ?? '' });
  }

  /**
   * List all knowledge entries for a project, optionally filtered by kind.
   */
  async listKnowledge(projectPath: string, kind?: string): Promise<ProjectKnowledge[]> {
    const params = new URLSearchParams({ projectPath });
    if (kind) params.set('kind', kind);
    const data = await this.get<ProjectKnowledge[] | null>(`/api/brain/knowledge?${params}`);
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
    const data = await this.get<{ filePaths: string[] }>(`/api/brain/file-map?${params}`);
    return data.filePaths ?? [];
  }

  // ---- Private helpers ----

  /** fetch() with the bearer token and a timeout, so no call can hang forever. */
  private async send(
    url: string,
    init: RequestInit = {},
    timeoutMs: number = REQUEST_TIMEOUT_MS,
  ): Promise<Response> {
    const headers: Record<string, string> = {
      ...(this.token ? { Authorization: `Bearer ${this.token}` } : {}),
      ...(init.headers as Record<string, string> | undefined),
    };
    try {
      return await fetch(url, {
        ...init,
        headers,
        signal: init.signal ?? AbortSignal.timeout(timeoutMs),
      });
    } catch (err) {
      if (err instanceof Error && (err.name === 'TimeoutError' || err.name === 'AbortError')) {
        throw new NexusClientError(
          `nexus: no response from ${url} within ${timeoutMs / 1000}s`,
          err,
        );
      }
      throw err;
    }
  }

  /** Builds the error for a non-2xx response, using the daemon's {"error": "..."} message. */
  private async failure(what: string, resp: Response): Promise<Error> {
    const raw = (await resp.text().catch(() => '')).trim();
    let detail = raw;
    try {
      const parsed = JSON.parse(raw) as { error?: unknown };
      if (typeof parsed.error === 'string' && parsed.error) detail = parsed.error;
    } catch {
      /* not JSON: keep the raw text */
    }
    return new Error(`nexus: ${what}: HTTP ${resp.status}${detail ? ` — ${detail}` : ''}`);
  }

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
    const resp = await this.send(`${this.baseUrl}${path}`);
    if (!resp.ok) throw await this.failure(`GET ${path}`, resp);
    const data: unknown = await resp.json();
    return schema ? this.parseResponse(schema, data) : (data as T);
  }

  private async post<T>(path: string, payload: unknown, schema?: z.ZodSchema<T>): Promise<T> {
    const resp = await this.send(`${this.baseUrl}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!resp.ok) throw await this.failure(`POST ${path}`, resp);
    const data: unknown = await resp.json();
    return schema ? this.parseResponse(schema, data) : (data as T);
  }

  private async delete(path: string): Promise<void> {
    const resp = await this.send(`${this.baseUrl}${path}`, { method: 'DELETE' });
    if (!resp.ok) throw await this.failure(`DELETE ${path}`, resp);
  }

  private async put<T>(path: string, payload: unknown, schema: z.ZodSchema<T>): Promise<T> {
    const resp = await this.send(`${this.baseUrl}${path}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!resp.ok) throw await this.failure(`PUT ${path}`, resp);
    const data: unknown = await resp.json();
    return this.parseResponse(schema, data);
  }
}
