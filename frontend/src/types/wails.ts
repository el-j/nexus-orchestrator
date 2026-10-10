import type {
  Task,
  TaskInput,
  ProviderInfo,
  ProviderConfig,
  AISession,
  TaskStatus,
  RuntimeConfig,
  RuntimeConfigUpdate,
  BrainStatus,
  ContextResponse,
  ContextSection,
  ProjectKnowledge,
} from './domain';
import type { DiscoveredProvider } from './domain';

// Wails Go bindings are injected at runtime via window.go
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          SubmitTask(task: TaskInput): Promise<string>;
          GetTask(id: string): Promise<Task>;
          GetQueue(): Promise<Task[]>;
          GetAllTasks(): Promise<Task[]>;
          GetProviders(): Promise<ProviderInfo[]>;
          CancelTask(id: string): Promise<void>;
          AddProviderConfig(cfg: Partial<ProviderConfig>): Promise<ProviderConfig>;
          ListProviderConfigs(): Promise<ProviderConfig[]>;
          UpdateProviderConfig(cfg: ProviderConfig): Promise<ProviderConfig>;
          RemoveProviderConfig(id: string): Promise<void>;
          GetDiscoveredProviders(): Promise<DiscoveredProvider[]>;
          TriggerScan(): Promise<void>;
          CreateDraft(task: Partial<Task>): Promise<string>;
          GetBacklog(projectPath: string): Promise<Task[]>;
          PromoteTask(id: string): Promise<void>;
          UpdateTask(id: string, updates: Partial<Task>): Promise<Task>;
          ListAISessions(): Promise<AISession[]>;
          RegisterAISession(session: AISession): Promise<AISession>;
          DeregisterAISession(id: string): Promise<void>;
          PurgeDisconnectedSessions(): Promise<number>;
          GetServerAddr(): Promise<string>;
          HeartbeatAISession(id: string): Promise<void>;
          ClaimTask(taskID: string, sessionID: string): Promise<Task>;
          UpdateTaskStatus(
            taskID: string,
            sessionID: string,
            status: TaskStatus,
            logs: string,
          ): Promise<Task>;
          GetRuntimeConfig(): Promise<RuntimeConfig>;
          UpdateRuntimeConfig(update: RuntimeConfigUpdate): Promise<RuntimeConfig>;
          IngestKnowledge(projectPath: string, filePath: string): Promise<number>;
          GetBrainStatus(projectPath: string): Promise<BrainStatus>;
          GetProjectContext(projectPath: string, maxTokens: number): Promise<ContextResponse>;
          GetFocusedContext(
            projectPath: string,
            question: string,
            maxTokens: number,
          ): Promise<ContextResponse>;
          SearchKnowledge(
            projectPath: string,
            query: string,
            limit: number,
          ): Promise<ContextSection[]>;
          InitProject(projectPath: string, claudeMDPath: string): Promise<BrainStatus>;
          ListKnowledge(projectPath: string, kind: string): Promise<ProjectKnowledge[]>;
          DeleteKnowledge(id: string): Promise<void>;
          GetFileMap(projectPath: string, focusArea: string): Promise<string[]>;
        };
      };
    };
  }
}

// Safe wrappers that fall back gracefully when not in Wails (browser dev mode)
const isWails = (): boolean => !!window.go?.main?.App;

/**
 * Calls the daemon's REST API (browser dev mode) and returns the parsed JSON body.
 * A non-2xx response throws an Error carrying the server's `{"error": "..."}`
 * message (or `HTTP <status>`), so callers can never mistake a 429/422/500 for
 * success. An empty body (204) yields undefined.
 */
async function request<T = void>(method: string, path: string, body?: unknown): Promise<T> {
  const r = await fetch(path, {
    method,
    ...(body === undefined
      ? {}
      : { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
  });
  if (!r.ok) {
    let detail = `HTTP ${r.status}`;
    try {
      const data = (await r.json()) as { error?: unknown };
      if (typeof data.error === 'string' && data.error) detail = data.error;
    } catch {
      /* body was not JSON: keep the status line */
    }
    throw new Error(detail);
  }
  const text = await r.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

const q = encodeURIComponent;

export async function submitTask(task: TaskInput): Promise<string> {
  if (isWails()) return window.go!.main!.App!.SubmitTask(task);
  // The REST API answers {"task_id": "...", "status": "QUEUED"}.
  return (await request<{ task_id: string }>('POST', '/api/tasks', task)).task_id;
}

export async function getTask(id: string): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.GetTask(id);
  return request<Task>('GET', `/api/tasks/${q(id)}`);
}

export async function getQueue(): Promise<Task[]> {
  if (isWails()) return window.go!.main!.App!.GetQueue();
  return request<Task[]>('GET', '/api/tasks');
}

export async function getAllTasks(): Promise<Task[]> {
  if (isWails()) return (await window.go!.main!.App!.GetAllTasks()) ?? [];
  return request<Task[]>('GET', '/api/tasks/all');
}

export async function getProviders(): Promise<ProviderInfo[]> {
  if (isWails()) return window.go!.main!.App!.GetProviders();
  return request<ProviderInfo[]>('GET', '/api/providers');
}

export async function cancelTask(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.CancelTask(id);
  await request('DELETE', `/api/tasks/${q(id)}`);
}

export async function addProviderConfig(cfg: Partial<ProviderConfig>): Promise<ProviderConfig> {
  if (isWails()) return window.go!.main!.App!.AddProviderConfig(cfg);
  return request<ProviderConfig>('POST', '/api/providers/config', cfg);
}

export async function listProviderConfigs(): Promise<ProviderConfig[]> {
  if (isWails()) return window.go!.main!.App!.ListProviderConfigs();
  return request<ProviderConfig[]>('GET', '/api/providers/config');
}

export async function updateProviderConfig(cfg: ProviderConfig): Promise<ProviderConfig> {
  if (isWails()) return window.go!.main!.App!.UpdateProviderConfig(cfg);
  return request<ProviderConfig>('PUT', `/api/providers/config/${q(cfg.id)}`, cfg);
}

export async function removeProviderConfig(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.RemoveProviderConfig(id);
  await request('DELETE', `/api/providers/config/${q(id)}`);
}

export async function getDiscoveredProviders(): Promise<DiscoveredProvider[]> {
  if (isWails()) return window.go!.main!.App!.GetDiscoveredProviders();
  return request<DiscoveredProvider[]>('GET', '/api/providers/discovered');
}

export async function triggerScan(): Promise<void> {
  if (isWails()) return window.go!.main!.App!.TriggerScan();
  await request('POST', '/api/providers/discovered/scan');
}

export async function createDraft(task: Partial<Task>): Promise<string> {
  if (isWails()) return window.go!.main!.App!.CreateDraft(task);
  return (await request<{ id: string }>('POST', '/api/tasks/draft', task)).id;
}

export async function getBacklog(projectPath: string): Promise<Task[]> {
  if (isWails()) return (await window.go!.main!.App!.GetBacklog(projectPath)) ?? [];
  const query = projectPath ? `?project=${q(projectPath)}` : '';
  return (await request<Task[] | null>('GET', `/api/tasks/backlog${query}`)) ?? [];
}

export async function promoteTask(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.PromoteTask(id);
  await request('POST', `/api/tasks/${q(id)}/promote`);
}

export async function updateTask(id: string, updates: Partial<Task>): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.UpdateTask(id, updates);
  return request<Task>('PUT', `/api/tasks/${q(id)}`, updates);
}

export async function listAISessions(): Promise<AISession[]> {
  if (isWails()) return window.go!.main!.App!.ListAISessions();
  return request<AISession[]>('GET', '/api/ai-sessions');
}

export async function registerAISession(
  session: Omit<AISession, 'id' | 'createdAt' | 'updatedAt'>,
): Promise<AISession> {
  if (isWails()) return window.go!.main!.App!.RegisterAISession(session as AISession);
  return request<AISession>('POST', '/api/ai-sessions', session);
}

/** Returns the base HTTP URL of the embedded API server (e.g. http://127.0.0.1:63987). */
export async function getServerAddr(): Promise<string> {
  if (isWails()) {
    try {
      return await window.go!.main!.App!.GetServerAddr();
    } catch {
      // Older binary or binding not yet available — fall back to the default embedded address.
      return 'http://127.0.0.1:63987';
    }
  }
  // Browser dev mode: respect VITE_SERVER_URL env or fall back to the default address.
  return import.meta.env.VITE_SERVER_URL ?? 'http://127.0.0.1:63987';
}

/** Keeps a session alive. Failures are expected while the daemon is down, so they only warn. */
export async function heartbeatAISession(id: string): Promise<void> {
  try {
    if (isWails()) await window.go!.main!.App!.HeartbeatAISession(id);
    else await request('POST', `/api/ai-sessions/${q(id)}/heartbeat`);
  } catch (e) {
    console.warn('heartbeatAISession: failed:', e);
  }
}

export async function deregisterAISession(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.DeregisterAISession(id);
  await request('DELETE', `/api/ai-sessions/${q(id)}`);
}

export async function purgeDisconnectedSessions(): Promise<number> {
  if (isWails()) return window.go!.main!.App!.PurgeDisconnectedSessions();
  const data = await request<{ deleted?: number }>('DELETE', '/api/ai-sessions');
  return data?.deleted ?? 0;
}

export async function claimTask(taskID: string, sessionID: string): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.ClaimTask(taskID, sessionID);
  return request<Task>('POST', `/api/tasks/${q(taskID)}/claim`, { sessionId: sessionID });
}

export async function updateTaskStatus(
  taskID: string,
  sessionID: string,
  status: TaskStatus,
  logs: string,
): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.UpdateTaskStatus(taskID, sessionID, status, logs);
  return request<Task>('PUT', `/api/tasks/${q(taskID)}/status`, {
    sessionId: sessionID,
    status,
    logs,
  });
}

export async function getRuntimeConfig(): Promise<RuntimeConfig> {
  if (isWails()) return window.go!.main!.App!.GetRuntimeConfig();
  return request<RuntimeConfig>('GET', '/api/config');
}

export async function updateRuntimeConfig(update: RuntimeConfigUpdate): Promise<RuntimeConfig> {
  if (isWails()) return window.go!.main!.App!.UpdateRuntimeConfig(update);
  return request<RuntimeConfig>('PUT', '/api/config', update);
}

export async function ingestKnowledge(projectPath: string, filePath: string): Promise<number> {
  if (isWails()) return window.go!.main!.App!.IngestKnowledge(projectPath, filePath);
  const data = await request<{ ingestedSections: number }>('POST', '/api/brain/ingest', {
    projectPath,
    filePath,
  });
  return data.ingestedSections;
}

export async function getBrainStatus(projectPath: string): Promise<BrainStatus> {
  if (isWails()) return window.go!.main!.App!.GetBrainStatus(projectPath);
  return request<BrainStatus>('GET', `/api/brain/status?projectPath=${q(projectPath)}`);
}

export async function getProjectContext(
  projectPath: string,
  maxTokens: number,
): Promise<ContextResponse> {
  if (isWails()) return window.go!.main!.App!.GetProjectContext(projectPath, maxTokens);
  return request<ContextResponse>('POST', '/api/brain/context', { projectPath, maxTokens });
}

export async function getFocusedContext(
  projectPath: string,
  question: string,
  maxTokens: number,
): Promise<ContextResponse> {
  if (isWails()) return window.go!.main!.App!.GetFocusedContext(projectPath, question, maxTokens);
  return request<ContextResponse>('POST', '/api/brain/focused-context', {
    projectPath,
    question,
    maxTokens,
  });
}

export async function searchKnowledge(
  projectPath: string,
  query: string,
  limit: number,
): Promise<ContextSection[]> {
  if (isWails()) return window.go!.main!.App!.SearchKnowledge(projectPath, query, limit);
  // The REST API wraps the hits: {"results": [...]}.
  const data = await request<{ results?: ContextSection[] | null }>(
    'GET',
    `/api/brain/search?projectPath=${q(projectPath)}&q=${q(query)}&limit=${limit}`,
  );
  return data.results ?? [];
}

export async function initProject(projectPath: string, claudeMDPath = ''): Promise<BrainStatus> {
  if (isWails()) return window.go!.main!.App!.InitProject(projectPath, claudeMDPath);
  return request<BrainStatus>('POST', '/api/brain/init', { projectPath, claudeMDPath });
}

export async function listKnowledge(projectPath: string, kind = ''): Promise<ProjectKnowledge[]> {
  if (isWails()) return window.go!.main!.App!.ListKnowledge(projectPath, kind);
  const query = `projectPath=${q(projectPath)}${kind ? `&kind=${q(kind)}` : ''}`;
  return (await request<ProjectKnowledge[] | null>('GET', `/api/brain/knowledge?${query}`)) ?? [];
}

export async function deleteKnowledge(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.DeleteKnowledge(id);
  await request('DELETE', `/api/brain/knowledge/${q(id)}`);
}

export async function getFileMap(projectPath: string, focusArea = ''): Promise<string[]> {
  if (isWails()) return window.go!.main!.App!.GetFileMap(projectPath, focusArea);
  const query = `projectPath=${q(projectPath)}${focusArea ? `&focusArea=${q(focusArea)}` : ''}`;
  const data = await request<{ filePaths?: string[] | null }>(
    'GET',
    `/api/brain/file-map?${query}`,
  );
  return data.filePaths ?? [];
}
