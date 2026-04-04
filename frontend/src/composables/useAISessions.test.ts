import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAISessions } from './useAISessions';

// ── Hoisted mock variables ────────────────────────────────────────────────────

const {
  mockListAISessions,
  mockDeregisterAISession,
  mockPurgeDisconnectedSessions,
  mockOn,
  mockOff,
} = vi.hoisted(() => ({
  mockListAISessions: vi.fn(),
  mockDeregisterAISession: vi.fn(),
  mockPurgeDisconnectedSessions: vi.fn(),
  mockOn: vi.fn(),
  mockOff: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  listAISessions: mockListAISessions,
  deregisterAISession: mockDeregisterAISession,
  purgeDisconnectedSessions: mockPurgeDisconnectedSessions,
}));

vi.mock('./useGlobalSSE', () => ({
  useGlobalSSE: () => ({ on: mockOn, off: mockOff, connected: { value: true } }),
}));

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeSession(overrides: Record<string, unknown> = {}) {
  return {
    id: 's-1',
    agentName: 'TestAgent',
    status: 'active',
    source: 'mcp',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    lastActivity: new Date().toISOString(),
    ...overrides,
  };
}

function mountUseAISessions() {
  let state: ReturnType<typeof useAISessions>;
  const Harness = defineComponent({
    setup() {
      state = useAISessions();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('useAISessions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListAISessions.mockResolvedValue([]);
    mockDeregisterAISession.mockResolvedValue(undefined);
    mockPurgeDisconnectedSessions.mockResolvedValue(2);
  });

  it('populates sessions on mount; loading transitions to false', async () => {
    mockListAISessions.mockResolvedValue([
      makeSession({ id: 's-1' }),
      makeSession({ id: 's-2', agentName: 'Another' }),
    ]);

    const { wrapper, get } = mountUseAISessions();
    await flushPromises();

    expect(get().sessions.value).toHaveLength(2);
    expect(get().sessions.value[0].id).toBe('s-1');
    expect(get().loading.value).toBe(false);
    expect(get().error.value).toBeNull();

    wrapper.unmount();
  });

  it('deregister() calls deregisterAISession and refreshes the session list', async () => {
    mockListAISessions
      .mockResolvedValueOnce([makeSession({ id: 's-3' })])
      .mockResolvedValueOnce([]); // empty after deregister

    const { wrapper, get } = mountUseAISessions();
    await flushPromises();
    expect(get().sessions.value).toHaveLength(1);

    await get().deregister('s-3');
    await flushPromises();

    expect(mockDeregisterAISession).toHaveBeenCalledWith('s-3');
    expect(get().sessions.value).toHaveLength(0);

    wrapper.unmount();
  });

  it('polls listAISessions repeatedly on the 15s heartbeat interval', async () => {
    vi.useFakeTimers();
    mockListAISessions.mockResolvedValue([makeSession()]);

    const { wrapper } = mountUseAISessions();
    await flushPromises();
    await flushPromises();

    const callsBefore = mockListAISessions.mock.calls.length;

    // Advance by 15 seconds — triggers one interval tick
    await vi.advanceTimersByTimeAsync(15_000);
    await flushPromises();

    expect(mockListAISessions.mock.calls.length).toBeGreaterThan(callsBefore);

    wrapper.unmount();
    vi.useRealTimers();
  });

  it('SSE ai_session_changed event triggers a refresh', async () => {
    mockListAISessions
      .mockResolvedValueOnce([makeSession({ id: 's-sse', status: 'idle' })])
      .mockResolvedValueOnce([makeSession({ id: 's-sse', status: 'active' })]);

    const { wrapper, get } = mountUseAISessions();
    await flushPromises();
    expect(get().sessions.value[0].status).toBe('idle');

    // Capture the ai_session_changed SSE handler
    const sseCall = mockOn.mock.calls.find(([type]) => type === 'ai_session_changed');
    expect(sseCall).toBeDefined();
    const sseHandler = sseCall![1] as () => void;

    sseHandler();
    await flushPromises();

    expect(mockListAISessions).toHaveBeenCalledTimes(2);
    expect(get().sessions.value[0].status).toBe('active');

    wrapper.unmount();
  });

  it('sets error.value when listAISessions throws on initial fetch', async () => {
    mockListAISessions.mockRejectedValue(new Error('HTTP 500'));

    const { wrapper, get } = mountUseAISessions();
    await flushPromises();

    expect(get().error.value).toBe('HTTP 500');
    expect(get().sessions.value).toHaveLength(0);
    expect(get().loading.value).toBe(false);

    wrapper.unmount();
  });

  it('cleans up SSE handler and polling interval on unmount', async () => {
    const { wrapper } = mountUseAISessions();
    await flushPromises();

    wrapper.unmount();

    expect(mockOff).toHaveBeenCalledWith('ai_session_changed', expect.any(Function));
  });
});
