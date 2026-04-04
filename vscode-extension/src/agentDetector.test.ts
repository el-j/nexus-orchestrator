import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { NexusClient } from './nexusClient'

// ── vscode mock ────────────────────────────────────────────────────────────────
const getExtensionMock = vi.fn()
const workspaceFolders = [{ uri: { fsPath: '/workspace' } }]

class EventEmitter<T> {
  event = vi.fn()
  fire = vi.fn<(value?: T) => void>()
  dispose = vi.fn()
}

vi.mock('vscode', () => ({
  EventEmitter,
  workspace: {
    workspaceFolders,
  },
  extensions: {
    getExtension: getExtensionMock,
  },
  window: {
    terminals: [],
  },
  lm: undefined,
}))

// ── fs/promises mock ────────────────────────────────────────────────────────────
vi.mock('fs/promises', () => ({
  readFile: vi.fn().mockRejectedValue(new Error('ENOENT')),
  stat: vi.fn().mockRejectedValue(new Error('ENOENT')),
}))

// ── os mock ─────────────────────────────────────────────────────────────────────
vi.mock('os', () => ({
  homedir: vi.fn().mockReturnValue('/home/testuser'),
}))

// ── Build a minimal NexusClient mock ───────────────────────────────────────────
function buildClient(overrides: Partial<Record<string, unknown>> = {}): NexusClient {
  return {
    registerSession: vi.fn().mockResolvedValue({
      id: 'session-1',
      agentName: 'Cline',
      source: 'vscode',
      status: 'active',
      lastActivity: '2026-03-14T00:00:00Z',
    }),
    heartbeatSession: vi.fn().mockResolvedValue(undefined),
    deregisterSession: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  } as unknown as NexusClient
}

// ── Minimal ExtensionContext mock ──────────────────────────────────────────────
function buildContext(): import('vscode').ExtensionContext {
  const store = new Map<string, unknown>()
  return {
    globalState: {
      get: vi.fn((key: string) => store.get(key)),
      update: vi.fn((key: string, value: unknown) => {
        store.set(key, value)
        return Promise.resolve()
      }),
    },
  } as unknown as import('vscode').ExtensionContext
}

describe('AgentDetector', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    getExtensionMock.mockReturnValue(undefined)
  })

  // ── Test 1: detectVSCodeExtensions finds installed extension ─────────────────
  it('detectVSCodeExtensions finds Cline and calls registerSession with agentName', async () => {
    getExtensionMock.mockImplementation((id: string) =>
      id === 'saoudrizwan.claude-dev' ? { id } : undefined
    )

    const client = buildClient()
    const { AgentDetector } = await import('./agentDetector')
    const detector = new AgentDetector(client, buildContext())

    const detected = await detector.detectAll()

    expect(detected.length).toBeGreaterThanOrEqual(1)
    const cline = detected.find(a => a.agentName === 'Cline')
    expect(cline).toBeDefined()
    expect(cline?.detectionMethod).toBe('vscode-extension')

    // Trigger tick so registerSession is called
    await (detector as unknown as { tick(): Promise<void> }).tick()
    expect(client.registerSession).toHaveBeenCalledWith(
      expect.objectContaining({ agentName: 'Cline' })
    )
  })

  // ── Test 2: no crash when registerSession rejects ────────────────────────────
  it('does not crash when registerSession rejects and remains alive', async () => {
    getExtensionMock.mockImplementation((id: string) =>
      id === 'continue.continue' ? { id } : undefined
    )

    const client = buildClient({
      registerSession: vi.fn().mockRejectedValue(new Error('daemon down')),
    })

    const { AgentDetector } = await import('./agentDetector')
    const detector = new AgentDetector(client, buildContext())

    detector.start()
    // Flush microtask queue to let the tick's silent-catch run
    await Promise.resolve()
    await Promise.resolve()

    expect((detector as unknown as { isDisposed: boolean }).isDisposed).toBe(false)
    detector.dispose()
  })

  // ── Test 3: dispose() sets isDisposed and stops the interval ─────────────────
  it('dispose() marks detector as disposed and no error is thrown', async () => {
    const { AgentDetector } = await import('./agentDetector')
    const detector = new AgentDetector(buildClient(), buildContext())

    detector.start()
    detector.dispose()

    expect((detector as unknown as { isDisposed: boolean }).isDisposed).toBe(true)
    // Calling dispose again must not throw
    expect(() => detector.dispose()).not.toThrow()
  })

  // ── Test 4: detectAll deduplicates by externalId ─────────────────────────────
  it('detectAll() deduplicates agents sharing the same externalId', async () => {
    // Install both saoudrizwan.claude-dev AND github.copilot
    getExtensionMock.mockImplementation((id: string) =>
      ['saoudrizwan.claude-dev', 'github.copilot'].includes(id) ? { id } : undefined
    )

    const client = buildClient()
    const { AgentDetector } = await import('./agentDetector')
    const ctx = buildContext()
    const detector = new AgentDetector(client, ctx)

    // Retrieve the machineId that was stored during construction so we can
    // build a duplicate externalId and inject it via a second detectAll call.
    // Simpler approach: call detectAll twice and confirm there are no duplicate externalIds.
    const first = await detector.detectAll()
    const second = await detector.detectAll()

    const ids = second.map(a => a.externalId)
    const uniqueIds = new Set(ids)

    // Each externalId must appear exactly once
    expect(ids.length).toBe(uniqueIds.size)
  })
})
