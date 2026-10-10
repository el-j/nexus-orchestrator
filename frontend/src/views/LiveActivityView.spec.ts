import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref, reactive, nextTick } from 'vue';
import LiveActivityView from './LiveActivityView.vue';
import type { AIActivity } from '../types/domain';

const h = vi.hoisted(() => ({
  route: { query: {} as Record<string, unknown> },
  rows: { value: [] as unknown[] },
  loading: { value: false },
  scanNow: vi.fn(),
  scanning: { value: false },
}));
vi.mock('vue-router', () => ({ useRoute: () => h.route }));
vi.mock('../composables/useActivities', () => ({
  useActivities: () => ({ activities: ref(h.rows.value), loading: ref(h.loading.value) }),
}));
vi.mock('../composables/useDiscovery', () => ({
  useDiscovery: () => ({ scanning: ref(h.scanning.value), scanNow: h.scanNow }),
}));

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
let n = 0;
const act = (o: Partial<AIActivity> = {}): AIActivity =>
  ({
    id: `a${n++}`,
    agentName: 'claude',
    activityType: 'message',
    summary: 'did a thing',
    projectPath: '/work/app',
    timestamp: new Date(NOW - 10_000).toISOString(),
    ...o,
  }) as AIActivity;

const Card = { props: ['activity'], template: '<div class="card">{{ activity.summary }}</div>' };
const Indicator = { props: ['lastRefreshed'], template: '<i class="indicator" />' };
const mountView = () =>
  mount(LiveActivityView, {
    global: { stubs: { AIActivityCard: Card, RefreshIndicator: Indicator } },
  });
const selects = (w: ReturnType<typeof mountView>) => w.findAll('select');

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  h.route = reactive({ query: {} });
  h.rows.value = [];
  h.loading.value = false;
  h.scanning.value = false;
  h.scanNow.mockReset();
});
afterEach(() => vi.useRealTimers());

describe('LiveActivityView — states', () => {
  it('shows a spinner while the first load is in flight', () => {
    h.loading.value = true;
    expect(mountView().find('.animate-spin').exists()).toBe(true);
  });

  it('shows the empty state with no activity', () => {
    expect(mountView().text()).toContain('No AI activity detected');
  });

  it('counts agents and activities in the header', () => {
    h.rows.value = [act(), act({ agentName: 'copilot' }), act()];
    const t = mountView().find('header').text();
    expect(t).toContain('2 agents active');
    expect(t).toContain('3 activities');
  });

  it('scans from the header and shows progress while scanning', async () => {
    const w = mountView();
    await w.find('header button').trigger('click');
    expect(h.scanNow).toHaveBeenCalledTimes(1);
    h.scanning.value = true;
    const busy = mountView().find('header button');
    expect(busy.text()).toContain('Scanning');
    expect(busy.attributes('disabled')).toBeDefined();
  });
});

describe('LiveActivityView — filters', () => {
  beforeEach(() => {
    h.rows.value = [
      act({ agentName: 'claude', projectPath: '/work/a', summary: 'A1', activityType: 'message' }),
      act({
        agentName: 'copilot',
        projectPath: '/work/b',
        summary: 'B1',
        activityType: 'tool_use',
      }),
    ];
  });

  it('offers each agent and project once, sorted', () => {
    h.rows.value = [
      ...(h.rows.value as AIActivity[]),
      act({ agentName: 'claude', projectPath: undefined }),
    ];
    const w = mountView();
    expect(
      selects(w)[0]
        .findAll('option')
        .map((o) => o.text()),
    ).toEqual(['All agents', 'claude', 'copilot']);
    expect(
      selects(w)[1]
        .findAll('option')
        .map((o) => o.text()),
    ).toEqual(['All projects', 'work/a', 'work/b']);
  });

  it('filters by agent', async () => {
    const w = mountView();
    await selects(w)[0].setValue('copilot');
    expect(w.text()).toContain('B1');
    expect(w.text()).not.toContain('A1');
  });

  it('filters by project', async () => {
    const w = mountView();
    await selects(w)[1].setValue('/work/a');
    expect(w.text()).toContain('A1');
    expect(w.text()).not.toContain('B1');
  });

  it('toggles type filters on and off', async () => {
    const w = mountView();
    const tool = w.find('button[title="tool_use"]');
    await tool.trigger('click');
    expect(w.text()).toContain('B1');
    expect(w.text()).not.toContain('A1');
    await w.find('button[title="message"]').trigger('click'); // second type: union
    expect(w.text()).toContain('A1');
    await tool.trigger('click');
    await w.find('button[title="message"]').trigger('click');
    expect(w.text()).toContain('B1'); // none selected = all
  });

  it('shows the empty state when filters match nothing', async () => {
    const w = mountView();
    await selects(w)[0].setValue('claude');
    await selects(w)[1].setValue('/work/b');
    expect(w.text()).toContain('No AI activity detected');
  });
});

describe('LiveActivityView — deep links', () => {
  beforeEach(() => {
    h.rows.value = [
      act({ agentName: 'claude', projectPath: '/work/a', summary: 'A1' }),
      act({ agentName: 'copilot', projectPath: '/work/b', summary: 'B1' }),
    ];
  });

  it('pre-selects the agent from ?agent=', () => {
    h.route.query = { agent: 'copilot' };
    const w = mountView();
    expect((selects(w)[0].element as HTMLSelectElement).value).toBe('copilot');
    expect(w.text()).toContain('B1');
    expect(w.text()).not.toContain('A1');
  });

  it('pre-selects the project from ?project=', () => {
    h.route.query = { project: '/work/a' };
    const w = mountView();
    expect((selects(w)[1].element as HTMLSelectElement).value).toBe('/work/a');
    expect(w.text()).not.toContain('B1');
  });

  it('ignores non-string query values', () => {
    h.route.query = { agent: ['x', 'y'], project: 5 };
    const w = mountView();
    expect(w.text()).toContain('A1');
    expect(w.text()).toContain('B1');
  });

  it('follows the query when navigating within the page, and resets when it is cleared', async () => {
    const w = mountView();
    h.route.query = { agent: 'claude' };
    await nextTick();
    expect(w.text()).not.toContain('B1');
    h.route.query = {};
    await nextTick();
    expect(w.text()).toContain('B1');
  });
});

describe('LiveActivityView — session groups', () => {
  const groups = (w: ReturnType<typeof mountView>) => w.findAll('.mb-4.rounded-xl');

  it('groups by session id, sums tokens, and shows the model', () => {
    h.rows.value = [
      act({ sessionId: 's1', tokensIn: 10, tokensOut: 5, model: 'opus' }),
      act({ sessionId: 's1', tokensIn: 1000, model: undefined }),
      act({ sessionId: 's2' }),
    ];
    const g = groups(mountView());
    expect(g).toHaveLength(2);
    const s1 = g.find((x) => x.text().includes('opus'))!;
    expect(s1.text()).toContain((1015).toLocaleString());
    expect(s1.findAll('.card')).toHaveLength(2);
  });

  it('merges session-less activity of the same agent and project within 60s, splits beyond', () => {
    h.rows.value = [
      act({ timestamp: new Date(NOW - 200_000).toISOString() }),
      act({ timestamp: new Date(NOW - 170_000).toISOString() }),
      act({ timestamp: new Date(NOW - 20_000).toISOString() }),
      act({ timestamp: new Date(NOW - 10_000).toISOString(), agentName: 'copilot' }),
    ];
    expect(groups(mountView())).toHaveLength(3);
  });

  it('orders groups newest first and activities within a group newest first', () => {
    h.rows.value = [
      act({ sessionId: 'old', summary: 'OLD', timestamp: new Date(NOW - 3600_000).toISOString() }),
      act({ sessionId: 'new', summary: 'FIRST', timestamp: new Date(NOW - 20_000).toISOString() }),
      act({ sessionId: 'new', summary: 'SECOND', timestamp: new Date(NOW - 10_000).toISOString() }),
    ];
    const g = groups(mountView());
    expect(g[0].text().indexOf('SECOND')).toBeLessThan(g[0].text().indexOf('FIRST'));
    expect(g[1].text()).toContain('OLD');
  });

  it('marks a session live when it was active in the last minute', () => {
    h.rows.value = [
      act({ sessionId: 'live', timestamp: new Date(NOW - 5_000).toISOString() }),
      act({ sessionId: 'stale', timestamp: new Date(NOW - 600_000).toISOString() }),
    ];
    const g = groups(mountView());
    expect(g[0].find('.bg-emerald-400').exists()).toBe(true);
    expect(g[1].find('.bg-slate-600').exists()).toBe(true);
  });

  it.each([
    ['Claude Code', 'text-violet-400'],
    ['GitHub Copilot', 'text-blue-400'],
    ['Cursor', 'text-cyan-400'],
    ['Continue', 'text-emerald-400'],
    ['LM Studio', 'text-orange-400'],
    ['Ollama', 'text-green-400'],
    ['something else', 'text-slate-300'],
  ])('colours %s', (name, cls) => {
    h.rows.value = [act({ agentName: name })];
    expect(mountView().find('.mb-4.rounded-xl span.font-semibold').classes()).toContain(cls);
  });
});
