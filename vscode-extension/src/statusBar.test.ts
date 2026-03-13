import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AISession, NexusClient, Provider, Task } from './nexusClient'

const statusBarItem = {
  text: '',
  tooltip: '',
  command: undefined as string | undefined,
  show: vi.fn(),
  dispose: vi.fn(),
}

vi.mock('vscode', () => ({
  StatusBarAlignment: { Left: 1 },
  window: {
    createStatusBarItem: vi.fn(() => statusBarItem),
  },
}))

vi.mock('./activityLog', () => ({
  getActivitySnapshot: vi.fn(() => ({ lastQueuedTaskId: 'task-12345678' })),
}))

describe('NexusStatusBar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    statusBarItem.text = ''
    statusBarItem.tooltip = ''
  })

  it('renders route-aware queue and session counts', async () => {
    const { NexusStatusBar } = await import('./statusBar')

    const client = {
      health: vi.fn().mockResolvedValue(true),
      getProviders: vi.fn<() => Promise<Provider[]>>().mockResolvedValue([{ name: 'Mock', active: true }]),
      getTasks: vi.fn<() => Promise<Task[]>>().mockResolvedValue([
        {
          id: 'task-12345678',
          projectPath: '/workspace',
          targetFile: 'file.ts',
          instruction: 'Queued task',
          contextFiles: [],
          status: 'QUEUED',
          createdAt: '2026-03-13T01:00:00.000Z',
          updatedAt: '2026-03-13T01:00:01.000Z',
        },
      ]),
      getAISessions: vi.fn<() => Promise<AISession[]>>().mockResolvedValue([
        { id: 'v1', agentName: 'GitHub Copilot', source: 'vscode', status: 'active', lastActivity: '2026-03-13T01:00:00.000Z' },
        { id: 'm1', agentName: 'Claude', source: 'mcp', status: 'active', lastActivity: '2026-03-13T01:00:00.000Z' },
      ]),
    } as unknown as NexusClient

    const bar = new NexusStatusBar(client)
    await bar.update()

    expect(statusBarItem.text).toContain('Q1')
    expect(statusBarItem.text).toContain('M1')
    expect(statusBarItem.text).toContain('V1')
    expect(String(statusBarItem.tooltip)).toContain('MCP sessions: 1')
    expect(String(statusBarItem.tooltip)).toContain('Copilot direct sessions: 1')
  })
})