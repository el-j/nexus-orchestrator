/**
 * nexusClient.ts — HTTP client for the nexusOrchestrator daemon REST API.
 *
 * All types mirror the Go domain types and ports.ProviderInfo exactly.
 * Uses native fetch (Node 18+ / VS Code's built-in runtime).
 */

// ---- Domain types (mirror internal/core/domain/task.go) ----

export type TaskStatus =
  | "QUEUED"
  | "PROCESSING"
  | "COMPLETED"
  | "FAILED"
  | "CANCELLED"
  | "TOO_LARGE"
  | "NO_PROVIDER";

export type CommandType = "plan" | "execute" | "auto" | "";

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
  activeModel?: string;
  models?: string[];
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
}

export interface RegisterSessionRequest {
  agentName: string;
  source: "vscode" | "mcp" | "http";
  projectPath?: string;
  externalId?: string;
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

// ---- Client ----

export class NexusClient {
  constructor(private readonly baseUrl: string) {}

  /**
   * Submit a new task. The daemon returns only {task_id, status}, so we
   * immediately fetch the full task to return a complete Task object.
   */
  async submitTask(payload: SubmitTaskRequest): Promise<Task> {
    const resp = await this.post<CreateTaskResponse>("/api/tasks", payload);
    return this.getTask(resp.task_id);
  }

  /** Return all tasks currently in the queue. */
  async getTasks(): Promise<Task[]> {
    return this.get<Task[]>("/api/tasks");
  }

  /** Return a single task by ID. Throws if not found (404). */
  async getTask(id: string): Promise<Task> {
    return this.get<Task>(`/api/tasks/${encodeURIComponent(id)}`);
  }

  /**
   * Cancel a task. Returns void on success (204 No Content).
   * Throws if the task is not found (404).
   */
  async cancelTask(id: string): Promise<void> {
    const url = `${this.baseUrl}/api/tasks/${encodeURIComponent(id)}`;
    const resp = await fetch(url, { method: "DELETE" });
    if (!resp.ok) {
      const body = await resp.text().catch(() => "");
      throw new Error(
        `nexus: cancel task ${id}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ""}`
      );
    }
  }

  /** Return all registered LLM providers and their liveness status. */
  async getProviders(): Promise<Provider[]> {
    return this.get<Provider[]>("/api/providers");
  }

  /** Register a new AI session with the daemon. */
  async registerSession(req: RegisterSessionRequest): Promise<AISession> {
    return this.post<AISession>("/api/ai-sessions", req);
  }

  /** Deregister an AI session by ID. */
  async deregisterSession(id: string): Promise<void> {
    const url = `${this.baseUrl}/api/ai-sessions/${encodeURIComponent(id)}`;
    const resp = await fetch(url, { method: "DELETE" });
    if (!resp.ok) {
      const body = await resp.text().catch(() => "");
      throw new Error(
        `nexus: deregister session ${id}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ""}`
      );
    }
  }

  /** Send a heartbeat for an existing session to refresh its last-activity timestamp. */
  async heartbeatSession(id: string): Promise<void> {
    const url = `${this.baseUrl}/api/ai-sessions/${encodeURIComponent(id)}/heartbeat`;
    const resp = await fetch(url, { method: "POST" });
    if (!resp.ok && resp.status !== 404) {
      // 404 means the session was cleaned up — caller should re-register.
      const body = await resp.text().catch(() => "");
      throw new Error(
        `nexus: heartbeat session ${id}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ""}`
      );
    }
  }

  /** Return all registered AI sessions. */
  async getAISessions(): Promise<AISession[]> {
    return this.get<AISession[]>("/api/ai-sessions");
  }

  /** Claim a queued task for the given session. */
  async claimTask(taskId: string, sessionId: string): Promise<Task> {
    return this.post<Task>(`/api/tasks/${encodeURIComponent(taskId)}/claim`, { sessionId });
  }

  /** Update a task's status (COMPLETED or FAILED). */
  async updateTaskStatus(taskId: string, sessionId: string, status: "COMPLETED" | "FAILED", logs?: string): Promise<Task> {
    return this.put<Task>(`/api/tasks/${encodeURIComponent(taskId)}/status`, { sessionId, status, logs });
  }

  /** Get all tasks bound to a specific AI session. */
  async getSessionTasks(sessionId: string): Promise<Task[]> {
    return this.get<Task[]>(`/api/ai-sessions/${encodeURIComponent(sessionId)}/tasks`);
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
      return body.status === "ok";
    } catch {
      return false;
    }
  }

  // ---- Private helpers ----

  private async get<T>(path: string): Promise<T> {
    const resp = await fetch(`${this.baseUrl}${path}`);
    if (!resp.ok) {
      const body = await resp.text().catch(() => "");
      throw new Error(
        `nexus: GET ${path}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ""}`
      );
    }
    return resp.json() as Promise<T>;
  }

  private async post<T>(path: string, payload: unknown): Promise<T> {
    const resp = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!resp.ok) {
      const body = await resp.text().catch(() => "");
      throw new Error(
        `nexus: POST ${path}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ""}`
      );
    }
    return resp.json() as Promise<T>;
  }

  private async put<T>(path: string, payload: unknown): Promise<T> {
    const resp = await fetch(`${this.baseUrl}${path}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!resp.ok) {
      const body = await resp.text().catch(() => "");
      throw new Error(
        `nexus: PUT ${path}: HTTP ${resp.status}${body ? ` — ${body.trim()}` : ""}`
      );
    }
    return resp.json() as Promise<T>;
  }
}
