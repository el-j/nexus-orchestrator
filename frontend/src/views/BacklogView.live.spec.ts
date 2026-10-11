import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import BacklogView from './BacklogView.vue';
import type { Task } from '../types/domain';

const h = vi.hoisted(() => ({
  getBacklog: vi.fn(),
  on: vi.fn(),
  off: vi.fn(),
  connected: { value: true },
  project: { value: null as string | null },
}));
vi.mock('../types/wails', () => ({ getBacklog: h.getBacklog }));
vi.mock('../composables/useGlobalSSE', async () => {
  const { ref } = await import('vue');
  const connected = ref(true);
  h.connected = connected as unknown as { value: boolean };
  return { useGlobalSSE: () => ({ on: h.on, off: h.off, connected }) };
});
vi.mock('../composables/useProjectState', async () => {
  const { ref } = await import('vue');
  const currentProject = ref<string | null>(null);
  h.project = currentProject as unknown as { value: string | null };
  return { currentProject };
});

const List = {
  name: 'BacklogList',
  props: ['items'],
  emits: ['promoted', 'dismissed'],
  template: '<div class="list" />',
};
const mounted: Array<{ unmount: () => void }> = [];
const flush = () => vi.advanceTimersByTimeAsync(0);
const mountView = async () => {
  const w = mount(BacklogView, {
    global: { stubs: { BacklogList: List, RefreshIndicator: true } },
  });
  mounted.push(w);
  await flush();
  return w;
};
const t = (id: string) => ({ id }) as Task;

beforeEach(() => {
  vi.useFakeTimers();
  h.getBacklog.mockReset().mockResolvedValue([]);
  h.on.mockReset();
  h.off.mockReset();
  h.connected.value = true;
  h.project.value = null;
});
afterEach(() => {
  mounted.splice(0).forEach((w) => {
    try {
      w.unmount();
    } catch {
      /* already unmounted */
    }
  });
  vi.useRealTimers();
});

describe('BacklogView — data', () => {
  it('asks for the whole backlog, or the selected project’s', async () => {
    await mountView();
    expect(h.getBacklog).toHaveBeenLastCalledWith('');
    h.project.value = '/work/app';
    await flush();
    expect(h.getBacklog).toHaveBeenLastCalledWith('/work/app');
  });

  it('counts items with singular/plural wording and passes them to the list', async () => {
    h.getBacklog.mockResolvedValue([t('a')]);
    const one = await mountView();
    expect(one.find('header').text()).toContain('1 item');
    expect(one.find('header').text()).not.toContain('1 items');
    expect(one.findComponent(List).props('items')).toHaveLength(1);

    h.getBacklog.mockResolvedValue([t('a'), t('b')]);
    const two = await mountView();
    expect(two.find('header').text()).toContain('2 items');
  });

  it('treats a null response as empty', async () => {
    h.getBacklog.mockResolvedValue(null);
    expect((await mountView()).findComponent(List).props('items')).toEqual([]);
  });

  it('refreshes when the list reports a promote or a dismiss', async () => {
    const w = await mountView();
    h.getBacklog.mockClear();
    w.findComponent(List).vm.$emit('promoted');
    await flush();
    w.findComponent(List).vm.$emit('dismissed');
    await flush();
    expect(h.getBacklog).toHaveBeenCalledTimes(2);
  });

  it('shows a load error, offers Retry, and recovers', async () => {
    h.getBacklog.mockRejectedValueOnce(new Error('boom'));
    const w = await mountView();
    expect(w.text()).toContain('boom');
    h.getBacklog.mockResolvedValue([t('a')]);
    await w
      .findAll('button')
      .find((b) => b.text() === 'Retry')!
      .trigger('click');
    await flush();
    expect(w.text()).not.toContain('boom');
    expect(w.findComponent(List).props('items')).toHaveLength(1);
  });

  it('uses a generic message for non-Error failures', async () => {
    h.getBacklog.mockRejectedValueOnce('weird');
    expect((await mountView()).text()).toContain('Failed to load backlog');
  });

  it('shows "Retrying…" and ignores overlapping refreshes', async () => {
    let done!: (v: Task[]) => void;
    h.getBacklog.mockReturnValueOnce(Promise.reject(new Error('first')));
    const w = await mountView();
    h.getBacklog.mockReturnValueOnce(new Promise<Task[]>((r) => (done = r)));
    await w
      .findAll('button')
      .find((b) => b.text() === 'Retry')!
      .trigger('click');
    expect(w.text()).toContain('Retrying…');
    h.getBacklog.mockClear();
    w.findComponent(List).vm.$emit('promoted'); // overlaps the in-flight request
    await flush();
    expect(h.getBacklog).not.toHaveBeenCalled();
    done([]);
    await flush();
  });
});

describe('BacklogView — live updates', () => {
  const handler = () =>
    h.on.mock.calls.find((c) => c[0] === '*')![1] as (d: { type: string }) => void;

  it('refreshes on task events only', async () => {
    await mountView();
    h.getBacklog.mockClear();
    for (const type of ['connected', 'log', 'ai_activity_new']) handler()({ type });
    await flush();
    expect(h.getBacklog).not.toHaveBeenCalled();
    handler()({ type: 'task.draft' });
    await flush();
    expect(h.getBacklog).toHaveBeenCalledTimes(1);
  });

  it('polls every 2s only while the stream is down', async () => {
    h.connected.value = false;
    await mountView();
    h.getBacklog.mockClear();
    await vi.advanceTimersByTimeAsync(6000);
    expect(h.getBacklog).toHaveBeenCalledTimes(3);

    h.connected.value = true;
    await flush();
    h.getBacklog.mockClear();
    await vi.advanceTimersByTimeAsync(6000);
    expect(h.getBacklog).not.toHaveBeenCalled();

    h.connected.value = false;
    await flush();
    expect(h.getBacklog).toHaveBeenCalledTimes(1);
  });

  it('does not stack poll timers and stops them on unmount', async () => {
    h.connected.value = false;
    const w = await mountView();
    h.connected.value = true;
    await flush();
    h.connected.value = false;
    await flush();
    h.getBacklog.mockClear();
    await vi.advanceTimersByTimeAsync(2000);
    expect(h.getBacklog).toHaveBeenCalledTimes(1);
    w.unmount();
    expect(h.off).toHaveBeenCalledWith('*', expect.any(Function));
    h.getBacklog.mockClear();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(h.getBacklog).not.toHaveBeenCalled();
  });

  it('unmounting while not polling is harmless', async () => {
    const w = await mountView();
    expect(() => w.unmount()).not.toThrow();
  });
});
