import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import ProjectActivityView from './ProjectActivityView.vue';
import type { AIActivity } from '../types/domain';

const h = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: h.push }) }));
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: () => Promise.resolve('http://d'),
}));

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
const act = (o: Partial<AIActivity> = {}): AIActivity =>
  ({
    id: Math.random().toString(36).slice(2),
    agentName: 'claude',
    activityType: 'message',
    summary: 'did a thing',
    projectPath: '/work/app',
    timestamp: new Date(NOW - 60_000).toISOString(),
    ...o,
  }) as AIActivity;

const fetchMock = vi.fn();
const respond = (body: unknown, status = 200) =>
  fetchMock.mockImplementation(() =>
    Promise.resolve(new Response(JSON.stringify(body), { status })),
  );
const flush = () => vi.advanceTimersByTimeAsync(0);
const mountView = async () => {
  const w = mount(ProjectActivityView);
  await flush();
  return w;
};

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  h.push.mockReset();
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('ProjectActivityView', () => {
  it('requests the last 24h of activity', async () => {
    respond([]);
    await mountView();
    const url = new URL(fetchMock.mock.calls[0][0]);
    expect(url.origin + url.pathname).toBe('http://d/api/activities/timeline');
    expect(url.searchParams.get('limit')).toBe('200');
    expect(url.searchParams.get('since')).toBe(new Date(NOW - 24 * 3600_000).toISOString());
  });

  it('shows the empty state (also for a null body)', async () => {
    respond(null);
    expect((await mountView()).text()).toContain('No project activity yet');
  });

  it('shows the error for a failed request, and recovers on the next poll', async () => {
    respond({}, 503);
    const w = await mountView();
    expect(w.text()).toContain('HTTP 503');
    respond([act()]);
    await vi.advanceTimersByTimeAsync(15_000);
    expect(w.text()).not.toContain('HTTP 503');
    expect(w.text()).toContain('app');
  });

  it('uses a generic message for non-Error failures', async () => {
    fetchMock.mockRejectedValue('weird');
    expect((await mountView()).text()).toContain('Failed to load activities');
  });

  it('groups by project, newest project first, with distinct agents', async () => {
    respond([
      act({
        projectPath: '/work/old',
        timestamp: new Date(NOW - 3600_000).toISOString(),
        agentName: 'copilot',
      }),
      act({ projectPath: '/work/new', agentName: 'claude' }),
      act({ projectPath: '/work/new', agentName: 'claude' }),
      act({ projectPath: '/work/new', agentName: 'cursor' }),
    ]);
    const w = await mountView();
    const cards = w.findAll('.rounded-xl.border');
    expect(cards).toHaveLength(2);
    expect(cards[0].text()).toContain('work/new');
    expect(cards[0].text()).toContain('2 agents');
    expect(cards[1].text()).toContain('work/old');
    expect(cards[1].text()).toContain('1 agent');
    expect(cards[1].text()).not.toContain('1 agents');
  });

  it('shows only the five most recent activities per project, newest first', async () => {
    respond(
      Array.from({ length: 7 }, (_, i) =>
        act({ summary: `event ${i}`, timestamp: new Date(NOW - i * 1000).toISOString() }),
      ),
    );
    const t = (await mountView()).text();
    expect(t).toContain('event 0');
    expect(t).toContain('event 4');
    expect(t).not.toContain('event 5');
  });

  it('collects activity without a project under one card', async () => {
    respond([act({ projectPath: undefined }), act({ projectPath: undefined })]);
    const w = await mountView();
    expect(w.findAll('.rounded-xl.border')).toHaveLength(1);
    expect(w.text()).toContain('(no project)');
  });

  it('shortens Windows and single-segment paths', async () => {
    respond([act({ projectPath: 'C:\\code\\proj' }), act({ projectPath: 'solo' })]);
    const t = (await mountView()).text();
    expect(t).toContain('code/proj');
    expect(t).toContain('solo');
  });

  it('maps activity types to icons, with a fallback', async () => {
    respond([
      act({ activityType: 'tool_use' }),
      act({ activityType: 'file_edit' }),
      act({ activityType: 'weird' as AIActivity['activityType'] }),
    ]);
    const t = (await mountView()).text();
    expect(t).toContain('🔧');
    expect(t).toContain('📝');
    expect(t).toContain('•');
  });

  it('opens the timeline filtered to the project of the card', async () => {
    respond([act({ projectPath: '/work/app' })]);
    const w = await mountView();
    await w
      .findAll('button')
      .find((b) => b.text().includes('View timeline'))!
      .trigger('click');
    expect(h.push).toHaveBeenCalledWith({ name: 'live-activity', query: { project: '/work/app' } });
  });

  it('opens the unfiltered timeline for the "(no project)" card', async () => {
    respond([act({ projectPath: undefined })]);
    const w = await mountView();
    await w
      .findAll('button')
      .find((b) => b.text().includes('View timeline'))!
      .trigger('click');
    expect(h.push).toHaveBeenCalledWith({ name: 'live-activity', query: {} });
  });

  it('polls every 15s and stops on unmount', async () => {
    respond([]);
    const w = await mountView();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    w.unmount();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
