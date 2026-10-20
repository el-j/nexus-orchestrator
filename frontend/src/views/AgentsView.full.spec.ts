import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import AgentsView from './AgentsView.vue';
import type { AIActivity, AISession, DiscoveredAgent } from '../types/domain';

const h = vi.hoisted(() => ({
  refresh: vi.fn(),
  deregister: vi.fn(),
  delegate: vi.fn(),
  terminate: vi.fn(),
  purge: vi.fn(),
  project: { value: null as string | null },
  s: {
    sessions: [] as unknown[],
    loading: false,
    error: null as string | null,
    activities: [] as unknown[],
    actLoading: false,
  },
}));
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: () => Promise.resolve('http://d'),
}));
vi.mock('../composables/useProjectState', async () => {
  const { ref } = await import('vue');
  const currentProject = ref<string | null>(null);
  h.project = currentProject as unknown as { value: string | null };
  return { currentProject };
});
vi.mock('../composables/useAISessions', () => ({
  useAISessions: () => ({
    sessions: ref(h.s.sessions),
    loading: ref(h.s.loading),
    error: ref(h.s.error),
    refresh: h.refresh,
    deregister: h.deregister,
    delegate: h.delegate,
    terminate: h.terminate,
    purgeDisconnected: h.purge,
  }),
}));
vi.mock('../composables/useActivities', () => ({
  useActivities: () => ({ activities: ref(h.s.activities), loading: ref(h.s.actLoading) }),
}));

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
let n = 0;
const iso = (agoMs: number) => new Date(NOW - agoMs).toISOString();
const sess = (o: Partial<AISession> = {}): AISession =>
  ({
    id: `s${n++}`,
    agentName: 'Copilot',
    status: 'active',
    source: 'vscode',
    lastActivity: iso(60_000),
    ...o,
  }) as AISession;
const agent = (o: Partial<DiscoveredAgent> = {}): DiscoveredAgent =>
  ({
    id: `a${n++}`,
    name: 'Claude CLI',
    kind: 'claude-cli',
    isRunning: true,
    detectionMethod: 'process',
    lastSeen: iso(60_000),
    ...o,
  }) as DiscoveredAgent;
const act = (o: Partial<AIActivity> = {}): AIActivity =>
  ({
    id: `x${n++}`,
    agentName: 'claude',
    activityType: 'message',
    summary: 'did a thing',
    timestamp: iso(1000),
    ...o,
  }) as AIActivity;

const Drawer = {
  name: 'AgentDetailDrawer',
  props: ['agentName', 'modelVisible'],
  emits: ['update:modelVisible'],
  template: '<div class="drawer" />',
};
const fetchMock = vi.fn();
const mounted: Array<{ unmount: () => void }> = [];
const flush = () => vi.advanceTimersByTimeAsync(0);

async function mountView(o: { agents?: DiscoveredAgent[] | null; agentsOk?: boolean } = {}) {
  fetchMock.mockImplementation(() =>
    Promise.resolve(
      new Response(JSON.stringify(o.agents === undefined ? [] : o.agents), {
        status: o.agentsOk === false ? 500 : 200,
      }),
    ),
  );
  const w = mount(AgentsView, { global: { stubs: { AgentDetailDrawer: Drawer } } });
  mounted.push(w);
  await flush();
  return w;
}
type W = Awaited<ReturnType<typeof mountView>>;
const btn = (w: W, t: string) => w.findAll('button').find((b) => b.text().includes(t))!;
const cards = (w: W) => w.findAll('.border-l-4');
const search = (w: W) => w.find('input[placeholder="Search…"]');

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  n = 0;
  h.refresh.mockReset();
  h.deregister.mockReset();
  h.delegate.mockReset();
  h.terminate.mockReset();
  h.purge.mockReset();
  h.project.value = null;
  h.s = { sessions: [], loading: false, error: null, activities: [], actLoading: false };
  vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => {
  mounted.splice(0).forEach((w) => w.unmount());
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('AgentsView — states', () => {
  it('shows loading, error and empty states', async () => {
    h.s.loading = true;
    expect((await mountView()).find('.animate-spin').exists()).toBe(true);
    h.s.loading = false;
    h.s.error = 'daemon down';
    expect((await mountView()).text()).toContain('daemon down');
    h.s.error = null;
    expect((await mountView()).text()).toContain('No AI agents detected');
  });

  it('counts active sessions and discovered agents', async () => {
    h.s.sessions = [sess(), sess({ status: 'idle' })];
    const w = await mountView({ agents: [agent(), agent()] });
    const t = w.find('header').text().replace(/\s+/g, ' ');
    expect(t).toContain('1 active');
    expect(t).toContain('2 discovered');
  });

  it('tolerates a failing, erroring or null discovered-agents endpoint', async () => {
    expect((await mountView({ agentsOk: false })).text()).toContain('No AI agents detected');
    expect((await mountView({ agents: null })).text()).toContain('No AI agents detected');
    fetchMock.mockRejectedValue(new Error('offline'));
    const w = mount(AgentsView, { global: { stubs: { AgentDetailDrawer: Drawer } } });
    mounted.push(w);
    await flush();
    expect(w.text()).toContain('No AI agents detected');
  });
});

describe('AgentsView — sessions', () => {
  it('renders source, id, project and last activity', async () => {
    h.s.sessions = [
      sess({ source: 'mcp', externalId: 'ext-1', projectPath: '/work/app' }),
      sess({ source: 'vscode' }),
      sess({ source: 'http' }),
    ];
    const t = (await mountView()).text();
    expect(t).toContain('🤖 MCP');
    expect(t).toContain('🔵 VS Code');
    expect(t).toContain('🌐 HTTP');
    expect(t).toContain('ext-1');
    expect(t).toContain('work/app');
    expect(t).toContain('1 min ago');
  });

  it('sorts active-first, most recent, or by name', async () => {
    h.s.sessions = [
      sess({ agentName: 'B', status: 'disconnected', lastActivity: iso(1000) }),
      sess({ agentName: 'A', status: 'idle', lastActivity: iso(5000) }),
      sess({ agentName: 'C', status: 'active', lastActivity: iso(9000) }),
    ];
    const w = await mountView();
    const names = () => cards(w).map((c) => c.find('.font-semibold').text());
    expect(names()).toEqual(['C', 'A', 'B']);
    await w.find('header select').setValue('recent');
    expect(names()).toEqual(['B', 'A', 'C']);
    await w.find('header select').setValue('name');
    expect(names()).toEqual(['A', 'B', 'C']);
  });

  it('filters by status and by text over name, external id and project', async () => {
    h.s.sessions = [
      sess({ agentName: 'Alpha', status: 'active', externalId: 'zzz-1' }),
      sess({ agentName: 'Beta', status: 'idle', projectPath: '/work/needle' }),
      sess({ agentName: 'Gamma', status: 'disconnected' }),
    ];
    const w = await mountView();
    await btn(w, 'Idle').trigger('click');
    expect(cards(w)).toHaveLength(1);
    await btn(w, 'Off').trigger('click');
    expect(cards(w)).toHaveLength(1);
    await btn(w, 'All').trigger('click');
    await search(w).setValue('zzz');
    expect(cards(w)).toHaveLength(1);
    await search(w).setValue('needle');
    expect(cards(w)[0].text()).toContain('Beta');
    await search(w).setValue('nothing-matches');
    expect(w.text()).toContain('No matches found');
  });

  it('disconnecting, delegating or terminating a session does not open its detail drawer, and filters by currentProject', async () => {
    h.s.sessions = [
      sess({ id: 's-x', projectPath: '/work/app' }),
      sess({ id: 's-y', projectPath: '/work/other', delegatedToNexus: true }),
    ];
    const w = await mountView();
    await btn(w, 'Disconnect').trigger('click');
    expect(h.deregister).toHaveBeenCalledWith('s-x');
    await btn(w, 'Delegate').trigger('click');
    expect(h.delegate).toHaveBeenCalledWith('s-x');
    await btn(w, 'Terminate').trigger('click');
    expect(h.terminate).toHaveBeenCalledWith('s-x');
    expect(w.findComponent(Drawer).props('modelVisible')).toBe(false);
    expect(w.find('[data-testid="delegation-note"]').exists()).toBe(false);

    h.project.value = '/work/other';
    await flush();
    expect(cards(w)).toHaveLength(1);
    expect(cards(w)[0].text()).toContain('Delegated');
  });

  it('shows the instruction a delegation returns until it is dismissed', async () => {
    h.s.sessions = [sess({ id: 's-d' })];
    h.delegate.mockResolvedValueOnce({
      instruction: 'Submit your work through nexus',
      sessionId: 's-d',
    });
    const w = await mountView();
    await btn(w, 'Delegate').trigger('click');
    await flush();
    expect(w.find('[data-testid="delegation-note"]').text()).toContain(
      'Submit your work through nexus',
    );
    await btn(w, 'Dismiss').trigger('click');
    expect(w.find('[data-testid="delegation-note"]').exists()).toBe(false);
  });

  it('opens the drawer for the clicked session and closes it on request', async () => {
    h.s.sessions = [sess({ agentName: 'Copilot' })];
    const w = await mountView();
    await cards(w)[0].trigger('click');
    expect(w.findComponent(Drawer).props()).toMatchObject({
      agentName: 'Copilot',
      modelVisible: true,
    });
    w.findComponent(Drawer).vm.$emit('update:modelVisible', false);
    await flush();
    expect(w.findComponent(Drawer).props('modelVisible')).toBe(false);
  });

  it('hides Disconnect for disconnected sessions and offers Clear Disconnected', async () => {
    h.s.sessions = [sess({ status: 'disconnected' })];
    const w = await mountView();
    expect(w.findAll('button').some((b) => b.text() === 'Disconnect')).toBe(false);
    await btn(w, 'Clear Disconnected').trigger('click');
    expect(h.purge).toHaveBeenCalled();
    h.s.sessions = [sess()];
    expect((await mountView()).text()).not.toContain('Clear Disconnected');
  });

  it('refreshes sessions and discovered agents together', async () => {
    const w = await mountView();
    fetchMock.mockClear();
    await btn(w, 'Refresh').trigger('click');
    expect(h.refresh).toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledWith('http://d/api/ai-sessions/discovered');
  });

  it('re-polls discovered agents every 30s and stops on unmount', async () => {
    const w = await mountView();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    w.unmount();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(120_000);
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

describe('AgentsView — discovered agents', () => {
  it('shows details and nests sub-agents under their parent', async () => {
    const parent = agent({
      id: 'p',
      name: 'Parent',
      modelId: 'claude-opus',
      workingDir: '/work/app',
      mcpEndpoint: 'http://m/mcp',
    });
    const child = agent({
      id: 'c',
      name: 'Child',
      parentAgentId: 'p',
      modelId: 'claude-haiku',
      workingDir: '/work/sub',
      isRunning: false,
    });
    const orphan = agent({ id: 'o', name: 'Orphan', parentAgentId: 'missing' });
    const w = await mountView({ agents: [parent, child, orphan] });
    const t = w.text();
    expect(t).toContain('Parent');
    expect(t).toContain('opus');
    expect(t).not.toContain('claude-opus');
    expect(t).toContain('work/app');
    expect(t).toContain('http://m/mcp');
    expect(t).toContain('Running');
    expect(t).toContain('sub-agent');
    expect(t).toContain('haiku');
    expect(t).toContain('Orphan'); // unknown parent → shown as a root
    expect(w.findAll('.ml-6')).toHaveLength(1);
  });

  it('sorts roots running-first, most recent, or by name', async () => {
    const list = [
      agent({ name: 'B', isRunning: false, lastSeen: iso(1000) }),
      agent({ name: 'A', isRunning: false, lastSeen: iso(9000) }),
      agent({ name: 'C', isRunning: true, lastSeen: iso(5000) }),
    ];
    const w = await mountView({ agents: list });
    const names = () =>
      w.findAll('section:last-of-type .font-semibold.text-sm').map((x) => x.text());
    expect(names()).toEqual(['C', 'B', 'A']);
    await w.find('header select').setValue('recent');
    expect(names()).toEqual(['B', 'C', 'A']);
    await w.find('header select').setValue('name');
    expect(names()).toEqual(['A', 'B', 'C']);
  });

  it('searches name, kind and working directory', async () => {
    const w = await mountView({
      agents: [
        agent({ name: 'One', kind: 'cursor' }),
        agent({ name: 'Two', workingDir: '/x/needle' }),
      ],
    });
    await search(w).setValue('cursor');
    expect(w.text()).toContain('One');
    expect(w.text()).not.toContain('Two');
    await search(w).setValue('needle');
    expect(w.text()).toContain('Two');
  });

  it('nests several sub-agents under one parent', async () => {
    const parent = agent({ id: 'p', name: 'Parent' });
    const w = await mountView({
      agents: [
        parent,
        agent({ id: 'c1', name: 'One', parentAgentId: 'p' }),
        agent({ id: 'c2', name: 'Two', parentAgentId: 'p' }),
      ],
    });
    expect(w.findAll('.ml-6')).toHaveLength(1);
    expect(w.text()).toContain('One');
    expect(w.text()).toContain('Two');
  });

  it('keeps unknown-status sessions in a stable order after the known ones', async () => {
    h.s.sessions = [
      sess({ agentName: 'X', status: 'odd' as AISession['status'] }),
      sess({ agentName: 'Y', status: 'weird' as AISession['status'] }),
      sess({ agentName: 'Z', status: 'active' }),
    ];
    const w = await mountView();
    expect(cards(w)[0].find('.font-semibold').text()).toBe('Z');
  });
});

describe('AgentsView — activity tab', () => {
  async function open(activities: AIActivity[] = []) {
    h.s.activities = activities;
    const w = await mountView();
    await btn(w, 'Activity').trigger('click');
    return w;
  }

  it('shows a spinner while loading, then an empty state', async () => {
    h.s.actLoading = true;
    const loading = await open();
    expect(loading.find('.animate-spin').exists()).toBe(true);
    h.s.actLoading = false;
    expect((await open()).text()).toContain('No AI activity detected');
  });

  it('lists activity with colours per agent and badges per type', async () => {
    const w = await open([
      act({ agentName: 'Claude', activityType: 'tool_use', projectPath: '/work/app' }),
      act({ agentName: 'GitHub Copilot', activityType: 'file_edit' }),
      act({ agentName: 'Cursor', activityType: 'thinking' }),
      act({ agentName: 'Continue', activityType: 'generation' }),
      act({ agentName: 'Other', activityType: 'weird' as AIActivity['activityType'] }),
    ]);
    const t = w.text();
    for (const x of ['tool_use', 'file_edit', 'thinking', 'generation', 'weird', 'work/app'])
      expect(t).toContain(x);
    const classes = w
      .findAll('span.text-xs.font-semibold.truncate')
      .map((e) => e.classes().join(' '));
    expect(classes[0]).toContain('violet');
    expect(classes[1]).toContain('blue');
    expect(classes[2]).toContain('cyan');
    expect(classes[3]).toContain('emerald');
    expect(classes[4]).toContain('slate');
  });

  it('filters by agent and project', async () => {
    const w = await open([
      act({ agentName: 'claude', projectPath: '/work/a', summary: 'A1' }),
      act({ agentName: 'copilot', projectPath: '/work/b', summary: 'B1' }),
    ]);
    const [agentSel, projectSel] = w.findAll('select').slice(1);
    await agentSel.setValue('copilot');
    expect(w.text()).toContain('B1');
    expect(w.text()).not.toContain('A1');
    await agentSel.setValue('');
    await projectSel.setValue('/work/a');
    expect(w.text()).toContain('A1');
    expect(w.text()).not.toContain('B1');
  });

  it('ranks a session with an unknown status after the known ones', async () => {
    h.s.sessions = [
      sess({ agentName: 'Odd', status: 'mystery' as AISession['status'] }),
      sess({ agentName: 'Fine', status: 'active' }),
    ];
    const w = await mountView();
    expect(cards(w).map((c) => c.find('.font-semibold').text())).toEqual(['Fine', 'Odd']);
  });

  it('shows a running sub-agent that has no model or working directory', async () => {
    const parent = agent({ id: 'p', name: 'Parent' });
    const child = agent({
      id: 'c',
      name: 'Helper',
      parentAgentId: 'p',
      isRunning: true,
      modelId: undefined,
      workingDir: undefined,
    });
    const w = await mountView({ agents: [parent, child] });
    expect(w.text()).toContain('Helper');
    expect(w.find('.ml-6 .bg-emerald-400').exists()).toBe(true);
  });
});
