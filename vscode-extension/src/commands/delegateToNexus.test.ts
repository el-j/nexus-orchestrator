import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { NexusClient, AISession } from '../nexusClient'

// ── fs mock ────────────────────────────────────────────────────────────────────
vi.mock('fs/promises', () => ({
  readFile: vi.fn().mockRejectedValue(new Error('ENOENT')),
}))

// ── vscode mock ────────────────────────────────────────────────────────────────
const writeFileMock = vi.fn().mockResolvedValue(undefined)
const createTerminalMock = vi.fn()
const executeCommandMock = vi.fn().mockResolvedValue(undefined)
const showInformationMessageMock = vi.fn().mockResolvedValue(undefined)
const showErrorMessageMock = vi.fn().mockResolvedValue(undefined)
const showTextDocumentMock = vi.fn().mockResolvedValue(undefined)

const terminalInstance = {
  show: vi.fn(),
  sendText: vi.fn(),
}

class Uri {
  static file(p: string) { return new Uri(p) }
  constructor(public fsPath: string) {}
}

class TreeItem {
  label: string
  constructor(label: string) { this.label = label }
}

vi.mock('vscode', () => ({
  Uri,
  TreeItem,
  TreeItemCollapsibleState: { None: 0 },
  ThemeIcon: class { constructor(public id: string) {} },
  workspace: {
    workspaceFolders: [{ uri: { fsPath: '/workspace' } }],
    fs: {
      writeFile: writeFileMock,
    },
  },
  window: {
    createTerminal: createTerminalMock.mockReturnValue(terminalInstance),
    showInformationMessage: showInformationMessageMock,
    showErrorMessage: showErrorMessageMock,
    showTextDocument: showTextDocumentMock,
    showQuickPick: vi.fn(),
  },
  commands: {
    executeCommand: executeCommandMock,
  },
  env: {
    clipboard: { writeText: vi.fn() },
  },
}))

vi.mock('../activityLog', () => ({
  logNexusActivity: vi.fn(),
  rememberQueuedTask: vi.fn(),
  showNexusActivityLog: vi.fn(),
}))

function buildClient(overrides: Partial<Record<string, unknown>> = {}): NexusClient {
  return {
    delegateSession: vi.fn().mockResolvedValue({ instruction: 'Test instruction', sessionId: 'sess-1' }),
    submitTask: vi.fn().mockResolvedValue({ id: 'task-1', status: 'QUEUED' }),
    listAISessions: vi.fn().mockResolvedValue([]),
    ...overrides,
  } as unknown as NexusClient
}

function session(overrides: Partial<AISession> = {}): AISession {
  return {
    id: 'sess-1',
    agentName: 'Cline',
    source: 'vscode',
    status: 'active',
    lastActivity: '2026-03-14T00:00:00Z',
    projectPath: '/workspace',
    ...overrides,
  }
}

describe('delegateToNexusCommand', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    createTerminalMock.mockReturnValue(terminalInstance)
    executeCommandMock.mockResolvedValue(undefined)
    showInformationMessageMock.mockResolvedValue(undefined)
  })

  // ── Test 1: CLI path writes file and opens terminal ───────────────────────────
  it('writes .nexus-delegate.md and creates a terminal for a vscode session', async () => {
    const { delegateToNexusCommand } = await import('./delegateToNexus')
    const client = buildClient()

    await delegateToNexusCommand(client, session({ source: 'vscode', agentName: 'Cline' }))

    expect(writeFileMock).toHaveBeenCalledWith(
      expect.anything(),           // Uri.file(...)
      expect.any(Buffer),
    )
    expect(createTerminalMock).toHaveBeenCalledWith({ name: 'Nexus Delegate' })
    expect(terminalInstance.show).toHaveBeenCalled()
  })

  // ── Test 2: Copilot path opens chat panel ─────────────────────────────────────
  it('calls workbench.action.chat.open for a GitHub Copilot session', async () => {
    const { delegateToNexusCommand } = await import('./delegateToNexus')
    const client = buildClient()

    await delegateToNexusCommand(client, session({ agentName: 'GitHub Copilot', source: 'vscode' }))

    expect(executeCommandMock).toHaveBeenCalledWith(
      'workbench.action.chat.open',
      expect.objectContaining({ query: 'Test instruction' }),
    )
    // Terminal must NOT be created for the copilot path
    expect(createTerminalMock).not.toHaveBeenCalled()
  })
})
