import type { Task, TaskInput, ProviderInfo, ProviderConfig, AISession } from './domain'
import type { DiscoveredProvider } from './discovery'

// Wails Go bindings are injected at runtime via window.go
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          SubmitTask(task: TaskInput): Promise<string>
          GetTask(id: string): Promise<Task>
          GetQueue(): Promise<Task[]>
          GetAllTasks(): Promise<Task[]>
          GetProviders(): Promise<ProviderInfo[]>
          CancelTask(id: string): Promise<void>
          AddProviderConfig(cfg: Partial<ProviderConfig>): Promise<ProviderConfig>
          ListProviderConfigs(): Promise<ProviderConfig[]>
          UpdateProviderConfig(cfg: ProviderConfig): Promise<ProviderConfig>
          RemoveProviderConfig(id: string): Promise<void>
          GetDiscoveredProviders(): Promise<DiscoveredProvider[]>
          TriggerScan(): Promise<void>
          CreateDraft(task: Partial<Task>): Promise<string>
          GetBacklog(projectPath: string): Promise<Task[]>
          PromoteTask(id: string): Promise<void>
          UpdateTask(id: string, updates: Partial<Task>): Promise<Task>
          ListAISessions(): Promise<AISession[]>
          RegisterAISession(session: AISession): Promise<AISession>
          DeregisterAISession(id: string): Promise<void>
          PurgeDisconnectedSessions(): Promise<number>
          GetServerAddr(): Promise<string>
          HeartbeatAISession(id: string): Promise<Error>
        }
      }
    }
  }
}

// Safe wrappers that fall back gracefully when not in Wails (browser dev mode)
const isWails = (): boolean => !!(window.go?.main?.App)

export async function submitTask(task: TaskInput): Promise<string> {
  if (isWails()) return window.go!.main!.App!.SubmitTask(task)
  // Dev fallback: mock response
  const r = await fetch('/api/tasks', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(task) })
  const data = await r.json() as { id: string }
  return data.id
}

export async function getTask(id: string): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.GetTask(id)
  const r = await fetch(`/api/tasks/${id}`)
  return r.json() as Promise<Task>
}

export async function getQueue(): Promise<Task[]> {
  if (isWails()) return window.go!.main!.App!.GetQueue()
  const r = await fetch('/api/tasks')
  return r.json() as Promise<Task[]>
}

export async function getAllTasks(): Promise<Task[]> {
  if (isWails()) return (await window.go!.main!.App!.GetAllTasks()) ?? []
  const r = await fetch('/api/tasks/all')
  return (await r.json()) as Task[]
}

export async function getProviders(): Promise<ProviderInfo[]> {
  if (isWails()) return window.go!.main!.App!.GetProviders()
  const r = await fetch('/api/providers')
  return r.json() as Promise<ProviderInfo[]>
}

export async function cancelTask(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.CancelTask(id)
  await fetch(`/api/tasks/${id}`, { method: 'DELETE' })
}

export async function addProviderConfig(cfg: Partial<ProviderConfig>): Promise<ProviderConfig> {
  if (isWails()) return window.go!.main!.App!.AddProviderConfig(cfg)
  const r = await fetch('/api/providers/config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(cfg),
  })
  return r.json() as Promise<ProviderConfig>
}

export async function listProviderConfigs(): Promise<ProviderConfig[]> {
  if (isWails()) return window.go!.main!.App!.ListProviderConfigs()
  const r = await fetch('/api/providers/config')
  return r.json() as Promise<ProviderConfig[]>
}

export async function updateProviderConfig(cfg: ProviderConfig): Promise<ProviderConfig> {
  if (isWails()) return window.go!.main!.App!.UpdateProviderConfig(cfg)
  const r = await fetch(`/api/providers/config/${cfg.id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(cfg),
  })
  return r.json() as Promise<ProviderConfig>
}

export async function removeProviderConfig(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.RemoveProviderConfig(id)
  await fetch(`/api/providers/config/${id}`, { method: 'DELETE' })
}

export async function getDiscoveredProviders(): Promise<DiscoveredProvider[]> {
  if (isWails()) return window.go!.main!.App!.GetDiscoveredProviders()
  const r = await fetch('/api/providers/discovered')
  return r.json() as Promise<DiscoveredProvider[]>
}

export async function triggerScan(): Promise<void> {
  if (isWails()) return window.go!.main!.App!.TriggerScan()
  await fetch('/api/providers/discovered/scan', { method: 'POST' })
}

export async function createDraft(task: Partial<Task>): Promise<string> {
  if (isWails()) return window.go!.main!.App!.CreateDraft(task)
  const r = await fetch('/api/tasks/draft', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(task),
  })
  const data = await r.json() as { id: string }
  return data.id
}

export async function getBacklog(projectPath: string): Promise<Task[]> {
  if (isWails()) return (await window.go!.main!.App!.GetBacklog(projectPath)) ?? []
  const query = projectPath ? `?project=${encodeURIComponent(projectPath)}` : ''
  const r = await fetch(`/api/tasks/backlog${query}`)
  return (await r.json()) as Task[]
}

export async function promoteTask(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.PromoteTask(id)
  await fetch(`/api/tasks/${id}/promote`, { method: 'POST' })
}

export async function updateTask(id: string, updates: Partial<Task>): Promise<Task> {
  if (isWails()) return window.go!.main!.App!.UpdateTask(id, updates)
  const r = await fetch(`/api/tasks/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  return r.json() as Promise<Task>
}

export async function listAISessions(): Promise<AISession[]> {
  if (isWails()) return window.go!.main!.App!.ListAISessions()
  const r = await fetch('/api/ai-sessions')
  if (!r.ok) throw new Error(`HTTP ${r.status}`)
  return r.json() as Promise<AISession[]>
}

export async function registerAISession(session: Omit<AISession, 'id' | 'createdAt' | 'updatedAt'>): Promise<AISession> {
  if (isWails()) return window.go!.main!.App!.RegisterAISession(session as AISession)
  const r = await fetch('/api/ai-sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(session),
  })
  if (!r.ok) throw new Error(`HTTP ${r.status}`)
  return r.json() as Promise<AISession>
}

/** Returns the base HTTP URL of the embedded API server (e.g. http://127.0.0.1:9999). */
export async function getServerAddr(): Promise<string> {
  if (isWails()) {
    try {
      return await window.go!.main!.App!.GetServerAddr()
    } catch {
      // Older binary or binding not yet available — fall back to the default embedded address.
      return 'http://127.0.0.1:9999'
    }
  }
  // Browser dev mode: respect VITE_SERVER_URL env or fall back to the default address.
  return (import.meta as { env?: { VITE_SERVER_URL?: string } }).env?.VITE_SERVER_URL ?? 'http://127.0.0.1:9999'
}

export async function heartbeatAISession(id: string): Promise<void> {
  if (isWails()) {
    try {
      await window.go!.main!.App!.HeartbeatAISession(id)
    } catch (e) {
      console.warn('heartbeatAISession: failed:', e)
    }
    return
  }
  await fetch(`/api/ai-sessions/${id}/heartbeat`, { method: 'POST' })
}

export async function deregisterAISession(id: string): Promise<void> {
  if (isWails()) return window.go!.main!.App!.DeregisterAISession(id)
  const r = await fetch(`/api/ai-sessions/${id}`, { method: 'DELETE' })
  if (!r.ok) throw new Error(`HTTP ${r.status}`)
}

export async function purgeDisconnectedSessions(): Promise<number> {
  if (isWails()) return window.go!.main!.App!.PurgeDisconnectedSessions()
  const r = await fetch('/api/ai-sessions', { method: 'DELETE' })
  if (!r.ok) throw new Error(`HTTP ${r.status}`)
  const data = await r.json() as { deleted: number }
  return data.deleted ?? 0
}
