import type { Task, TaskInput, ProviderInfo, ProviderConfig } from './domain'

// Wails Go bindings are injected at runtime via window.go
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          SubmitTask(task: TaskInput): Promise<string>
          GetTask(id: string): Promise<Task>
          GetQueue(): Promise<Task[]>
          GetProviders(): Promise<ProviderInfo[]>
          CancelTask(id: string): Promise<void>
          AddProviderConfig(cfg: Partial<ProviderConfig>): Promise<ProviderConfig>
          ListProviderConfigs(): Promise<ProviderConfig[]>
          UpdateProviderConfig(cfg: ProviderConfig): Promise<ProviderConfig>
          RemoveProviderConfig(id: string): Promise<void>
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
