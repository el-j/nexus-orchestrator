import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AISessionCard from './AISessionCard.vue';
import type { AISession } from '../types/domain';

vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://daemon'),
}));

const fetchMock = vi.fn();

const session = (o: Partial<AISession> = {}): AISession =>
  ({
    id: 'sess 1',
    agentName: 'claude',
    source: 'mcp',
    status: 'active',
    projectPath: '/Users/me/work/app',
    lastActivity: new Date().toISOString(),
    ...o,
  }) as AISession;

const tasksResponse = (tasks: unknown) => ({ ok: true, json: async () => tasks });

describe('AISessionCard', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it.each([
    [{ status: 'active', delegatedToNexus: true }, '#4ade80'],
    [{ status: 'active', delegatedToNexus: false }, '#facc15'],
    [{ status: 'idle' }, '#fb923c'],
    [{ status: 'disconnected' }, '#6b7280'],
  ])('colours the card for %j', (over, colour) => {
    const w = mount(AISessionCard, { props: { session: session(over as Partial<AISession>) } });
    // jsdom normalises hex colours to rgb(); compare through the same normalisation.
    const probe = document.createElement('div');
    probe.style.borderLeftColor = colour;
    expect(w.attributes('style')).toContain(probe.style.borderLeftColor);
  });

  it.each([
    ['mcp', '🤖 MCP'],
    ['vscode', '🔵 VS Code'],
    ['http', '🌐 HTTP'],
  ])('labels the %s source', (source, label) => {
    expect(
      mount(AISessionCard, {
        props: { session: session({ source: source as AISession['source'] }) },
      }).text(),
    ).toContain(label);
  });

  it('shows model (without the claude- prefix), path tail, capabilities and relative activity', () => {
    const w = mount(AISessionCard, {
      props: {
        session: session({ modelId: 'claude-opus-4', agentCapabilities: ['code', 'review'] }),
      },
    });
    expect(w.text()).toContain('opus-4');
    expect(w.text()).not.toContain('claude-opus');
    expect(w.text()).toContain('work/app');
    expect(w.text()).toContain('code');
    expect(w.text()).toContain('just now');
  });

  it('omits optional blocks when absent', () => {
    const w = mount(AISessionCard, {
      props: { session: session({ projectPath: '', modelId: undefined, agentCapabilities: [] }) },
    });
    expect(w.find('.font-mono.mb-2').exists()).toBe(false);
    expect(w.text()).not.toContain('Delegated');
  });

  it('offers delegation only until the session is delegated', async () => {
    const w = mount(AISessionCard, { props: { session: session() } });
    await w.find('button').trigger('click');
    expect(w.emitted('delegate')?.[0]).toEqual([expect.objectContaining({ id: 'sess 1' })]);
    const delegated = mount(AISessionCard, {
      props: { session: session({ delegatedToNexus: true }) },
    });
    expect(delegated.find('button').exists()).toBe(false);
    expect(delegated.text()).toContain('Delegated');
  });

  async function openTimeline(w: ReturnType<typeof mount>) {
    const details = w.find('details');
    (details.element as HTMLDetailsElement).open = true;
    await details.trigger('toggle');
    await vi.advanceTimersByTimeAsync(0); // flushPromises() would hang under fake timers
  }

  it("loads THIS session's tasks from the session endpoint (not every task of the project)", async () => {
    fetchMock.mockResolvedValue(
      tasksResponse([
        {
          id: 't1',
          status: 'COMPLETED',
          targetFile: 'a.go',
          instruction: 'do a',
          updatedAt: new Date().toISOString(),
        },
        {
          id: 't2',
          status: 'FAILED',
          targetFile: 'b.go',
          instruction: 'do b',
          updatedAt: new Date().toISOString(),
        },
        { id: 't3', status: 'DRAFT', targetFile: 'c.go', instruction: 'x' },
        { id: 't4', status: 'WEIRD' },
        { id: 't5', status: 'PROCESSING' },
        { id: 't6', status: 'QUEUED' },
        { id: 't7', status: 'CANCELLED' },
        { id: 't8', status: 'BACKLOG' },
      ]),
    );
    const w = mount(AISessionCard, { props: { session: session() } });
    expect(w.find('summary').text()).toContain('Show tasks');
    await openTimeline(w);
    expect(fetchMock).toHaveBeenCalledWith('http://daemon/api/ai-sessions/sess%201/tasks');
    expect(w.find('summary').text()).toContain('Hide tasks');
    expect(w.text()).toContain('a.go');
    expect(w.text()).toContain('b.go');
    expect(w.text()).toContain('COMPLETED');
    expect(w.text()).toContain('FAILED');
  });

  it('shows an empty message, tolerates null and failures, and keeps polling while open', async () => {
    fetchMock.mockResolvedValue(tasksResponse(null));
    const w = mount(AISessionCard, { props: { session: session() } });
    await openTimeline(w);
    expect(w.text()).toContain('No tasks yet');

    fetchMock.mockResolvedValue(
      tasksResponse([{ id: 't1', status: 'QUEUED', targetFile: 'new.go', instruction: 'i' }]),
    );
    await vi.advanceTimersByTimeAsync(5000);
    expect(w.text()).toContain('new.go');

    fetchMock.mockRejectedValue(new Error('offline'));
    await vi.advanceTimersByTimeAsync(5000);
    expect(w.text()).toContain('new.go'); // keeps the last known list

    fetchMock.mockResolvedValue({ ok: false });
    await vi.advanceTimersByTimeAsync(5000);
    expect(w.text()).toContain('new.go');
  });

  it('does not start polling when the panel is closed before the first load finishes', async () => {
    let release!: (v: unknown) => void;
    fetchMock.mockReturnValue(new Promise((r) => (release = r)));
    const w = mount(AISessionCard, { props: { session: session() } });
    const details = w.find('details');
    (details.element as HTMLDetailsElement).open = true;
    await details.trigger('toggle');
    (details.element as HTMLDetailsElement).open = false;
    await details.trigger('toggle');
    release(tasksResponse([]));
    await vi.advanceTimersByTimeAsync(0);
    const calls = fetchMock.mock.calls.length;
    await vi.advanceTimersByTimeAsync(30_000);
    expect(fetchMock.mock.calls.length).toBe(calls);
  });

  it('does not leak a timer if the card is unmounted while the first load is pending', async () => {
    let release!: (v: unknown) => void;
    fetchMock.mockReturnValue(new Promise((r) => (release = r)));
    const w = mount(AISessionCard, { props: { session: session() } });
    const details = w.find('details');
    (details.element as HTMLDetailsElement).open = true;
    await details.trigger('toggle');
    w.unmount();
    release(tasksResponse([]));
    await vi.advanceTimersByTimeAsync(0);
    const calls = fetchMock.mock.calls.length;
    await vi.advanceTimersByTimeAsync(30_000);
    expect(fetchMock.mock.calls.length).toBe(calls);
  });

  it('stops polling when the panel is closed or the card is unmounted', async () => {
    fetchMock.mockResolvedValue(tasksResponse([]));
    const w = mount(AISessionCard, { props: { session: session() } });
    await openTimeline(w);
    const calls = fetchMock.mock.calls.length;

    const details = w.find('details');
    (details.element as HTMLDetailsElement).open = false;
    await details.trigger('toggle');
    await vi.advanceTimersByTimeAsync(20_000);
    expect(fetchMock.mock.calls.length).toBe(calls);

    await openTimeline(w);
    const reopened = fetchMock.mock.calls.length;
    w.unmount();
    await vi.advanceTimersByTimeAsync(20_000);
    expect(fetchMock.mock.calls.length).toBe(reopened);
  });
});
