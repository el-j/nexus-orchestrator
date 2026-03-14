import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { NexusClient, Task } from '../nexusClient'

const showInputBox = vi.fn()
const showQuickPick = vi.fn()
const showInformationMessage = vi.fn()
const showErrorMessage = vi.fn()
const withProgress = vi.fn()
const executeCommand = vi.fn()
const getConfiguration = vi.fn()

vi.mock('vscode', () => ({
  ProgressLocation: { Notification: 15 },
  window: {
    activeTextEditor: {
      document: {
        uri: { fsPath: '/workspace/src/file.ts' },
      },
      selection: {},
    },
    showInputBox,
    showQuickPick,
    showInformationMessage,
    showErrorMessage,
    withProgress,
  },
  workspace: {
    workspaceFolders: [{ uri: { fsPath: '/workspace' } }],
    textDocuments: [
      {
        isDirty: false,
        isUntitled: false,
        uri: { scheme: 'file', fsPath: '/workspace/src/file.ts' },
      },
    ],
    getConfiguration,
  },
  commands: {
    executeCommand,
  },
}))

const logNexusActivity = vi.fn()
const rememberQueuedTask = vi.fn()
const showNexusActivityLog = vi.fn()

vi.mock('../activityLog', () => ({
  logNexusActivity,
  rememberQueuedTask,
  showNexusActivityLog,
}))

describe('sendCurrentContextCommand', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    vi.useFakeTimers()

    getConfiguration.mockReturnValue({
      get: vi.fn().mockReturnValue(''),
    })

    showInputBox.mockResolvedValue('Fix the current file')
    showQuickPick
      .mockImplementationOnce(async (items: Array<{ contextFiles: string[] }>) => [items[0]])
      .mockImplementationOnce(async (items: Array<{ label: string }>) => items[0])
    showInformationMessage
      .mockResolvedValueOnce('Queue Task')
      .mockResolvedValueOnce(undefined)
    withProgress.mockImplementation(async (_options, task) =>
      task({ report: vi.fn() }, { onCancellationRequested: vi.fn() })
    )
  })

  it('queues the current context through Nexus with reviewed context files', async () => {
    const { sendCurrentContextCommand } = await import('./submitTask')
    const task: Task = {
      id: 'task-12345678',
      projectPath: '/workspace',
      targetFile: 'src/file.ts',
      instruction: 'Fix the current file',
      contextFiles: ['src/file.ts'],
      status: 'COMPLETED',
      createdAt: '2026-03-13T01:00:00.000Z',
      updatedAt: '2026-03-13T01:00:10.000Z',
    }
    const client = {
      getProviders: vi.fn().mockResolvedValue([]),
      submitTask: vi.fn().mockResolvedValue(task),
      getTask: vi.fn().mockResolvedValue(task),
      cancelTask: vi.fn().mockResolvedValue(undefined),
    } as unknown as NexusClient

    const promise = sendCurrentContextCommand(client, 'http://127.0.0.1:63987')
    await vi.runAllTimersAsync()
    await promise

    expect(client.submitTask).toHaveBeenCalledWith(
      expect.objectContaining({
        instruction: 'Fix the current file',
        projectPath: '/workspace',
        targetFile: 'src/file.ts',
        contextFiles: ['src/file.ts'],
      })
    )
    expect(rememberQueuedTask).toHaveBeenCalledWith('task-12345678')
    expect(logNexusActivity).toHaveBeenCalled()
  })
})