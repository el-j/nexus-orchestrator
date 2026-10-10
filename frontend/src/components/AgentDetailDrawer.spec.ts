import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref, computed } from 'vue';
import AgentDetailDrawer from './AgentDetailDrawer.vue';
import type { AIActivity } from '../types/domain';

const h = vi.hoisted(() => ({
  push: vi.fn(),
  opts: undefined as { agentFilter?: string } | undefined,
  rows: { value: [] as AIActivity[] },
  loading: { value: false },
}));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: h.push }) }));
vi.mock('../composables/useActivities', () => ({
  useActivities: (o: { agentFilter?: string }) => {
    h.opts = o;
    const all = ref(h.rows.value);
    // Mirror the real composable: `filtered` applies the agent filter, `activities` does not.
    return {
      activities: all,
      filtered: computed(() =>
        all.value.filter((a) => !o.agentFilter || a.agentName === o.agentFilter),
      ),
      loading: ref(h.loading.value),
    };
  },
}));

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
const act = (o: Partial<AIActivity> = {}): AIActivity =>
  ({
    id: Math.random().toString(36),
    agentName: 'claude',
    activityType: 'message',
    summary: 'did a thing',
    timestamp: new Date(NOW - 30_000).toISOString(),
    tokensIn: 10,
    tokensOut: 5,
    ...o,
  }) as AIActivity;

const mountDrawer = (visible = true, agentName = 'claude') =>
  mount(AgentDetailDrawer, {
    props: { agentName, modelVisible: visible },
    global: { stubs: { Teleport: true, Transition: false } },
  });

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  h.push.mockReset();
  h.rows.value = [];
  h.loading.value = false;
});
afterEach(() => vi.useRealTimers());

describe('AgentDetailDrawer', () => {
  it('renders nothing while hidden', () => {
    expect(mountDrawer(false).find('aside').exists()).toBe(false);
  });

  it('asks the composable for the named agent only', () => {
    mountDrawer(true, 'copilot');
    expect(h.opts?.agentFilter).toBe('copilot');
  });

  it('does not show other agents’ activity', () => {
    h.rows.value = [
      act({ agentName: 'claude', summary: 'mine' }),
      act({ agentName: 'copilot', summary: 'theirs' }),
    ];
    const w = mountDrawer();
    expect(w.text()).toContain('mine');
    expect(w.text()).not.toContain('theirs');
  });

  it('shows an empty state and a disconnected badge with no activity', () => {
    const w = mountDrawer();
    expect(w.text()).toContain('No recent activity');
    expect(w.text()).toContain('No activity recorded.');
    expect(w.text()).toContain('disconnected');
  });

  it('shows stats: event count, summed tokens and last-active time', () => {
    h.rows.value = [
      act({ tokensIn: 100, tokensOut: 50 }),
      act({ tokensIn: 1, tokensOut: undefined }),
    ];
    const w = mountDrawer();
    expect(w.text()).toContain('151');
    expect(w.text()).toContain('just now');
    expect(w.text()).toContain('Events');
  });

  it.each([
    [30_000, 'active'],
    [5 * 60_000, 'idle'],
    [30 * 60_000, 'disconnected'],
  ])('labels an agent last seen %i ms ago as %s', (ago, label) => {
    h.rows.value = [act({ timestamp: new Date(NOW - ago).toISOString() })];
    const w = mountDrawer();
    expect(w.find('header span').text()).toBe(label);
  });

  it('uses a different colour per status', () => {
    const cls = (ago: number) => {
      h.rows.value = [act({ timestamp: new Date(NOW - ago).toISOString() })];
      return mountDrawer().find('header span').classes().join(' ');
    };
    expect(new Set([cls(1000), cls(5 * 60_000), cls(60 * 60_000)]).size).toBe(3);
  });

  it('lists at most ten recent events with a type icon', () => {
    h.rows.value = Array.from({ length: 12 }, (_, i) =>
      act({ id: `a${i}`, summary: `event ${i}`, activityType: 'tool_use' }),
    );
    const w = mountDrawer();
    expect(w.text()).toContain('event 9');
    expect(w.text()).not.toContain('event 10');
    expect(w.text()).toContain('🔧');
  });

  it.each([
    ['message', '💬'],
    ['thinking', '🧠'],
    ['file_edit', '📝'],
    ['generation', '✨'],
    ['unknown_kind', '•'],
  ])('maps activity type %s to %s', (type, emoji) => {
    h.rows.value = [act({ activityType: type as AIActivity['activityType'] })];
    expect(mountDrawer().text()).toContain(emoji);
  });

  it('closes from the backdrop and the close button', async () => {
    const w = mountDrawer();
    await w.find('.backdrop-blur-sm').trigger('click');
    await w.find('button[aria-label="Close"]').trigger('click');
    expect(w.emitted('update:modelVisible')).toEqual([[false], [false]]);
  });

  it('closes and opens the timeline filtered to this agent', async () => {
    const w = mountDrawer(true, 'copilot');
    await w
      .findAll('button')
      .find((b) => b.text().includes('View in timeline'))!
      .trigger('click');
    expect(w.emitted('update:modelVisible')![0]).toEqual([false]);
    expect(h.push).toHaveBeenCalledWith({ name: 'live-activity', query: { agent: 'copilot' } });
  });
});
