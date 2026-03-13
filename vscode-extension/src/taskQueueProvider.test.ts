import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { NexusClient, Task } from './nexusClient'

class ThemeIcon {
  constructor(public readonly id: string) {}
}

class MarkdownString {
  value: string
  constructor(value: string) {
    this.value = value
  }
}

class EventEmitter<T> {
  event = vi.fn()
  fire = vi.fn<(value?: T) => void>()
}

class TreeItem {
  label: string
  description?: string
  tooltip?: unknown
  iconPath?: unknown
  contextValue?: string

  constructor(label: string) {
    this.label = label
  }
}

vi.mock('vscode', () => ({
  ThemeIcon,
  MarkdownString,
  EventEmitter,
  TreeItem,
  TreeItemCollapsibleState: { None: 0 },
  Disposable: class {
    constructor(private readonly disposeFn: () => void) {}
    dispose() { this.disposeFn() }
  },
}))

vi.mock('./activityLog', () => ({
  getKnownTaskSource: vi.fn((taskId: string) => (taskId === 'task-1' ? 'vscode queue' : undefined)),
}))

describe('TaskQueueProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders queue items with local provenance when known', async () => {
    const { TaskQueueProvider } = await import('./taskQueueProvider')
    const client = {
      getTasks: vi.fn<() => Promise<Task[]>>().mockResolvedValue([
        {
          id: 'task-1',
          projectPath: '/workspace',
          targetFile: 'file.ts',
          instruction: 'Queued via extension',
          contextFiles: [],
          providerHint: 'MockProvider',
          modelId: 'mock-model',
          status: 'QUEUED',
          createdAt: '2026-03-13T01:00:00.000Z',
          updatedAt: '2026-03-13T01:00:01.000Z',
        },
      ]),
    } as unknown as NexusClient

    const provider = new TaskQueueProvider(client)
    const children = await provider.getChildren()

    expect(children).toHaveLength(1)
    expect(children[0].description).toContain('vscode queue')
    expect((children[0].tooltip as { value: string }).value).toContain('Source')
  })
})