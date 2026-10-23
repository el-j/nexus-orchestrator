import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import HistoryView from './HistoryView.vue';
import type { Task } from '../types/domain';

const h = vi.hoisted(() => ({
  on: vi.fn(),
  off: vi.fn(),
  connected: { value: true },
  project: { value: null as string | null },
}));
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: () => Promise.resolve('http://d'),
}));
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

const Drawer = {
  name: 'TaskDetailDrawer',
  props: ['modelValue', 'task'],
  emits: ['update:modelValue', 'cancelled'],
  template: '<div data-testid="drawer" />',
};
const Badge = { props: ['status'], template: '<span class="badge">{{ status }}</span>' };
const Indicator = { props: ['lastRefreshed'], template: '<i class="indicator" />' };

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
let n = 0;
const task = (o: Partial<Task> = {}): Task =>
  ({
    id: `TASK-${n++}`,
    instruction: 'Do something',
    projectPath: '/work/app',
    targetFile: 'src/a.go',
    status: 'COMPLETED',
    updatedAt: new Date(NOW - 60_000).toISOString(),
    ...o,
  }) as Task;

// The mocked stream state is shared by every test, so mounted views must be torn down
// or earlier tests' watchers would react to later connection flips.
const mounted: Array<{ unmount: () => void }> = [];
const mountTracked: typeof mount = ((...args: Parameters<typeof mount>) => {
  const w = mount(...args);
  mounted.push(w);
  return w;
}) as typeof mount;

const fetchMock = vi.fn();
const respond = (body: unknown, status = 200) =>
  fetchMock.mockImplementation(() =>
    Promise.resolve(new Response(JSON.stringify(body), { status })),
  );
const flush = () => vi.advanceTimersByTimeAsync(0);
async function mountView(tasks: Task[] = []) {
  respond(tasks);
  const w = mountTracked(HistoryView, {
    global: {
      stubs: { TaskDetailDrawer: Drawer, TaskStatusBadge: Badge, RefreshIndicator: Indicator },
    },
  });
  await flush();
  return w;
}
const rows = (w: Awaited<ReturnType<typeof mountView>>) => w.findAll('.cursor-pointer.rounded-xl');
const filterBtn = (w: Awaited<ReturnType<typeof mountView>>, label: string) =>
  w
    .find('[role="group"]')
    .findAll('button')
    .find((b) => b.text() === label)!;

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  n = 0;
  h.on.mockReset();
  h.off.mockReset();
  h.connected.value = true;
  h.project.value = null;
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => {
  mounted.splice(0).forEach((w) => {
    try {
      w.unmount();
    } catch {
      /* already unmounted by the test */
    }
  });
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('HistoryView — loading', () => {
  it('shows a skeleton until the first response', async () => {
    let done!: (r: Response) => void;
    fetchMock.mockReturnValue(new Promise<Response>((r) => (done = r)));
    const w = mountTracked(HistoryView, {
      global: {
        stubs: { TaskDetailDrawer: Drawer, TaskStatusBadge: Badge, RefreshIndicator: Indicator },
      },
    });
    await flush();
    expect(w.findAll('.animate-pulse')).toHaveLength(5);
    done(new Response('[]'));
    await flush();
    expect(w.findAll('.animate-pulse')).toHaveLength(0);
  });

  it('requests all tasks, scoped to the sidebar project when one is chosen', async () => {
    await mountView();
    expect(fetchMock).toHaveBeenLastCalledWith('http://d/api/tasks/all');
    h.project.value = '/work/app';
    await flush();
    expect(fetchMock).toHaveBeenLastCalledWith('http://d/api/tasks/all?projectPath=%2Fwork%2Fapp');
  });

  it('shows an empty state for no finished tasks, also for a null body', async () => {
    respond(null);
    const w = mountTracked(HistoryView, {
      global: {
        stubs: { TaskDetailDrawer: Drawer, TaskStatusBadge: Badge, RefreshIndicator: Indicator },
      },
    });
    await flush();
    expect(w.text()).toContain('No task history yet.');
  });

  it('shows the failure with a Retry that reloads', async () => {
    respond({}, 500);
    const w = mountTracked(HistoryView, {
      global: {
        stubs: { TaskDetailDrawer: Drawer, TaskStatusBadge: Badge, RefreshIndicator: Indicator },
      },
    });
    await flush();
    expect(w.text()).toContain('Failed to load task history: HTTP 500');
    respond([task()]);
    await w
      .findAll('button')
      .find((b) => b.text() === 'Retry')!
      .trigger('click');
    await flush();
    expect(w.text()).not.toContain('Failed to load');
    expect(rows(w)).toHaveLength(1);
  });

  it('stringifies non-Error failures', async () => {
    fetchMock.mockRejectedValue('weird');
    const w = mountTracked(HistoryView, {
      global: {
        stubs: { TaskDetailDrawer: Drawer, TaskStatusBadge: Badge, RefreshIndicator: Indicator },
      },
    });
    await flush();
    expect(w.text()).toContain('weird');
  });
});

describe('HistoryView — list', () => {
  it('lists only finished tasks and counts them with correct pluralisation', async () => {
    const w = await mountView([
      task(),
      task({ status: 'FAILED' }),
      task({ status: 'QUEUED' }),
      task({ status: 'DRAFT' }),
    ]);
    expect(rows(w)).toHaveLength(2);
    expect(w.find('header').text()).toContain('2');
    expect(w.find('header').text()).toContain('tasks');
    const one = await mountView([task()]);
    expect(one.find('header').text()).toContain('1 task');
    expect(one.find('header').text()).not.toContain('1 tasks');
  });

  it('formats id, project, file and finish time', async () => {
    const w = await mountView([
      task({ id: 'abcdef1234567890', projectPath: 'C:\\code\\proj', targetFile: 'a/b/c.go' }),
      task({ id: 'TASK-9', projectPath: '', targetFile: '', updatedAt: '' }),
    ]);
    const t = w.text();
    expect(t).toContain('abcdef12');
    expect(t).not.toContain('abcdef1234567890');
    expect(t).toContain('TASK-9');
    expect(t).toContain('proj');
    expect(t).toContain('c.go');
    expect(t).toContain('1 min ago');
    expect(t).toContain('—');
  });

  it('opens the detail drawer for a row, and reloads after a cancel', async () => {
    const t = task();
    const w = await mountView([t]);
    await rows(w)[0].trigger('click');
    const drawer = w.findComponent(Drawer);
    expect(drawer.props('modelValue')).toBe(true);
    expect(drawer.props('task')).toMatchObject({ id: t.id });
    fetchMock.mockClear();
    drawer.vm.$emit('cancelled');
    await flush();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe('HistoryView — filters', () => {
  const data = () => [
    task({ status: 'COMPLETED' }),
    task({ status: 'COMPLETED' }),
    task({ status: 'FAILED' }),
    task({ status: 'CANCELLED' }),
  ];

  it('filters with the tab buttons', async () => {
    const w = await mountView(data());
    await filterBtn(w, 'Completed').trigger('click');
    expect(rows(w)).toHaveLength(2);
    await filterBtn(w, 'Failed').trigger('click');
    expect(rows(w)).toHaveLength(1);
    await filterBtn(w, 'All').trigger('click');
    expect(rows(w)).toHaveLength(4);
    expect(filterBtn(w, 'All').attributes('aria-pressed')).toBe('true');
  });

  it('summarises every status, including unfinished ones and TOO_LARGE / NO_PROVIDER', async () => {
    const w = await mountView([
      ...data(),
      task({ status: 'QUEUED' }),
      task({ status: 'BACKLOG' }),
      task({ status: 'TOO_LARGE' }),
      task({ status: 'NO_PROVIDER' }),
    ]);
    const badges = w.find('.border-b.border-white\\/5.flex-wrap').findAll('button');
    expect(badges.map((b) => b.text().replace(/\s+/g, ' '))).toEqual(
      expect.arrayContaining([
        'Queued 1',
        'Completed 2',
        'Failed 1',
        'Cancelled 1',
        'Backlog 1',
        'Too Large 1',
        'No Provider 1',
      ]),
    );
  });

  it('toggles the filter from finished-status badges', async () => {
    const w = await mountView(data());
    const badge = () => w.findAll('.flex-wrap button').find((b) => b.text().startsWith('Failed'))!;
    await badge().trigger('click');
    expect(rows(w)).toHaveLength(1);
    expect(badge().attributes('aria-pressed')).toBe('true');
    await badge().trigger('click');
    expect(rows(w)).toHaveLength(4);
  });

  it('does not let unfinished-status badges empty the list', async () => {
    const w = await mountView([...data(), task({ status: 'QUEUED' }), task({ status: 'DRAFT' })]);
    for (const label of ['Queued', 'Draft']) {
      const badge = w.findAll('.flex-wrap button').find((b) => b.text().startsWith(label))!;
      expect(badge.attributes('disabled')).toBeDefined();
      await badge.trigger('click');
      expect(rows(w)).toHaveLength(4);
    }
  });

  it('applies the sidebar project to the list and to the summary', async () => {
    const w = await mountView([
      task({ projectPath: '/a' }),
      task({ projectPath: '/b' }),
      task({ projectPath: '/b', status: 'FAILED' }),
    ]);
    h.project.value = '/b';
    await flush();
    expect(rows(w)).toHaveLength(2);
    expect(
      w
        .findAll('.flex-wrap button')
        .find((b) => b.text().startsWith('Completed'))!
        .text(),
    ).toContain('1');
  });
});

describe('HistoryView — live updates', () => {
  it('refreshes on task events only', async () => {
    await mountView([task()]);
    const handler = h.on.mock.calls.find((c) => c[0] === '*')![1] as (d: { type: string }) => void;
    fetchMock.mockClear();
    for (const type of ['connected', 'log', 'ai_activity_new', 'ai_session_changed'])
      handler({ type });
    await flush();
    expect(fetchMock).not.toHaveBeenCalled();
    handler({ type: 'task.completed' });
    await flush();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('polls every 2s only while the stream is down', async () => {
    h.connected.value = false;
    await mountView([task()]);
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(6000);
    expect(fetchMock).toHaveBeenCalledTimes(3);

    h.connected.value = true;
    await flush();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(6000);
    expect(fetchMock).not.toHaveBeenCalled();

    h.connected.value = false; // stream drops: refresh immediately and resume polling
    await flush();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(2000);
    expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it('does not start a second poll timer', async () => {
    h.connected.value = false;
    await mountView();
    h.connected.value = true;
    await nextTick();
    h.connected.value = false;
    await flush();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(2000);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('unsubscribes and stops polling on unmount', async () => {
    h.connected.value = false;
    const w = await mountView();
    w.unmount();
    expect(h.off).toHaveBeenCalledWith('*', expect.any(Function));
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('unmounting when not polling is harmless', async () => {
    const w = await mountView();
    expect(() => w.unmount()).not.toThrow();
  });

  it('subscribes to nothing when unmounted before the first load finishes', async () => {
    respond([]);
    const w = mount(HistoryView, {
      global: {
        stubs: { TaskDetailDrawer: Drawer, TaskStatusBadge: Badge, RefreshIndicator: Indicator },
      },
    });
    w.unmount();
    await flush();
    expect(h.on).not.toHaveBeenCalled();
  });

  it('falls back to the whole path when it ends with a slash', async () => {
    const w = await mountView([task({ projectPath: '/', targetFile: '/' })]);
    expect(w.text()).toContain('/');
  });

  it('closes the detail drawer when it asks to', async () => {
    const w = await mountView([task()]);
    await rows(w)[0].trigger('click');
    const drawer = w.findComponent(Drawer);
    expect(drawer.props('modelValue')).toBe(true);
    drawer.vm.$emit('update:modelValue', false);
    await nextTick();
    expect(drawer.props('modelValue')).toBe(false);
  });
});
