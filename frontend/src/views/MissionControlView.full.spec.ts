import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import MissionControlView from './MissionControlView.vue';
import type { AISession, ProviderInfo, Task } from '../types/domain';

const h = vi.hoisted(() => ({
  refreshTasks: vi.fn(),
  promoteTask: vi.fn(),
  getAllTasks: vi.fn(),
  toastAdd: vi.fn(),
  s: {
    tasks: [] as unknown[],
    tasksLoading: false,
    providers: [] as unknown[],
    providersLoading: false,
    sessions: [] as unknown[],
    sessionsLoading: false,
  },
}));
vi.mock('../composables/useTasks', () => ({
  useTasks: () => ({
    tasks: ref(h.s.tasks),
    loading: ref(h.s.tasksLoading),
    refresh: h.refreshTasks,
    promoteTask: h.promoteTask,
  }),
}));
vi.mock('../composables/useProviders', () => ({
  useProviders: () => ({ providers: ref(h.s.providers), loading: ref(h.s.providersLoading) }),
}));
vi.mock('../composables/useAISessions', () => ({
  useAISessions: () => ({ sessions: ref(h.s.sessions), loading: ref(h.s.sessionsLoading) }),
}));
vi.mock('../types/wails', () => ({ getAllTasks: h.getAllTasks }));
vi.mock('../composables/useToast', () => ({ useToast: () => ({ add: h.toastAdd }) }));

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
let n = 0;
const task = (o: Partial<Task> = {}): Task =>
  ({
    id: `T-${n++}`,
    instruction: 'Do it',
    status: 'QUEUED',
    updatedAt: new Date(NOW - 60_000).toISOString(),
    ...o,
  }) as Task;
const prov = (o: Partial<ProviderInfo> = {}): ProviderInfo => ({
  name: 'LM Studio',
  active: true,
  activeModel: 'qwen',
  models: [],
  ...o,
});
const session = (o: Partial<AISession> = {}): AISession =>
  ({
    id: `s${n++}`,
    agentName: 'Copilot',
    status: 'active',
    lastActivity: new Date(NOW).toISOString(),
    ...o,
  }) as AISession;

const Drawer = {
  name: 'TaskDetailDrawer',
  props: ['modelValue', 'task'],
  emits: ['cancelled', 'update:modelValue'],
  template: '<div class="drawer" />',
};
const Form = { name: 'TaskSubmitForm', emits: ['submitted'], template: '<div class="form" />' };
const Badge = { props: ['status'], template: '<span class="badge">{{ status }}</span>' };
const Skeleton = { template: '<div class="skeleton" />' };

const flush = () => vi.advanceTimersByTimeAsync(0);
const mounted: Array<{ unmount: () => void }> = [];
async function mountView() {
  const w = mount(MissionControlView, {
    global: {
      stubs: {
        TaskDetailDrawer: Drawer,
        TaskSubmitForm: Form,
        TaskStatusBadge: Badge,
        UiSkeleton: Skeleton,
      },
    },
  });
  mounted.push(w);
  await flush();
  return w;
}
type W = Awaited<ReturnType<typeof mountView>>;
const btn = (w: W, t: string) => w.findAll('button').find((b) => b.text() === t)!;

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  n = 0;
  h.refreshTasks.mockReset();
  h.promoteTask.mockReset().mockResolvedValue(undefined);
  h.getAllTasks.mockReset().mockResolvedValue([]);
  h.toastAdd.mockReset();
  h.s = {
    tasks: [],
    tasksLoading: false,
    providers: [],
    providersLoading: false,
    sessions: [],
    sessionsLoading: false,
  };
});
afterEach(() => {
  mounted.splice(0).forEach((w) => w.unmount());
  vi.useRealTimers();
});

describe('MissionControlView — status bar', () => {
  it('counts agents (active or idle), queued, processing and active providers', async () => {
    h.s.sessions = [session(), session({ status: 'idle' }), session({ status: 'disconnected' })];
    h.s.tasks = [task(), task(), task({ status: 'PROCESSING' }), task({ status: 'COMPLETED' })];
    h.s.providers = [prov(), prov({ name: 'x', active: false })];
    const t = (await mountView()).find('header').text().replace(/\s+/g, ' ');
    expect(t).toContain('2 Agents');
    expect(t).toContain('2 Queued');
    expect(t).toContain('1 Processing');
    expect(t).toContain('1 Providers');
  });
});

describe('MissionControlView — active work', () => {
  it('shows skeletons while loading, an empty state, then the rows', async () => {
    h.s.tasksLoading = true;
    expect((await mountView()).findAll('.skeleton')).toHaveLength(3);
    h.s.tasksLoading = false;
    expect((await mountView()).text()).toContain('No active tasks');
    h.s.tasks = [task({ instruction: 'Write docs', providerName: 'Ollama', durationMs: 2500 })];
    const t = (await mountView()).text();
    expect(t).toContain('Write docs');
    expect(t).toContain('Ollama');
    expect(t).toContain('2.5s');
  });

  it('formats durations and fills placeholders', async () => {
    h.s.tasks = [
      task({ durationMs: 450 }),
      task({ durationMs: undefined, providerName: undefined }),
    ];
    const t = (await mountView()).text();
    expect(t).toContain('450ms');
    expect(t).toContain('—');
  });

  it('sorts by status priority, merging recent finished tasks not already queued', async () => {
    h.s.tasks = [task({ id: 'q', status: 'QUEUED' }), task({ id: 'p', status: 'PROCESSING' })];
    h.getAllTasks.mockResolvedValue([
      task({ id: 'q', status: 'QUEUED' }), // duplicate of a queued task
      task({ id: 'c', status: 'COMPLETED' }),
      task({ id: 'f', status: 'FAILED' }),
      task({
        id: 'old',
        status: 'COMPLETED',
        updatedAt: new Date(NOW - 2 * 3600_000).toISOString(),
      }),
    ]);
    const w = await mountView();
    const ids = w.findAll('tbody tr').map((r) => r.findAll('td')[1].text());
    expect(ids).toEqual(['p', 'q', 'f', 'c']);
  });

  it('filters Active and Drafts/Backlog with matching empty states', async () => {
    h.s.tasks = [
      task({ id: 'q' }),
      task({ id: 'd', status: 'DRAFT' }),
      task({ id: 'b', status: 'BACKLOG' }),
    ];
    const w = await mountView();
    await btn(w, 'Active').trigger('click');
    expect(w.findAll('tbody tr')).toHaveLength(1);
    await btn(w, 'Drafts/Backlog').trigger('click');
    expect(w.findAll('tbody tr')).toHaveLength(2);
    await btn(w, 'All').trigger('click');
    expect(w.findAll('tbody tr')).toHaveLength(3);

    h.s.tasks = [task({ id: 'q' })];
    const only = await mountView();
    await only
      .findAll('button')
      .find((b) => b.text() === 'Drafts/Backlog')!
      .trigger('click');
    expect(only.text()).toContain('No draft/backlog tasks');
  });

  it('offers Promote for drafts and backlog only, refreshing history afterwards', async () => {
    h.s.tasks = [task({ id: 'd', status: 'DRAFT' }), task({ id: 'q' })];
    const w = await mountView();
    expect(w.findAll('button').filter((b) => b.text().includes('Promote'))).toHaveLength(1);
    h.getAllTasks.mockClear();
    await btn(w, 'Promote ↑').trigger('click');
    await flush();
    expect(h.promoteTask).toHaveBeenCalledWith('d');
    expect(h.getAllTasks).toHaveBeenCalled();
  });

  it('does not open the drawer when Promote is clicked, and ignores a second promote in flight', async () => {
    let done!: () => void;
    h.promoteTask.mockReturnValue(new Promise<void>((r) => (done = r)));
    h.s.tasks = [task({ id: 'd1', status: 'DRAFT' }), task({ id: 'd2', status: 'DRAFT' })];
    const w = await mountView();
    const buttons = () =>
      w.findAll('button').filter((b) => b.text().includes('Promote') || b.text() === '…');
    await buttons()[0].trigger('click');
    await buttons()[1].trigger('click');
    expect(h.promoteTask).toHaveBeenCalledTimes(1);
    expect(w.findComponent(Drawer).props('modelValue')).toBe(false);
    done();
    await flush();
  });

  it('toasts a failed promote', async () => {
    h.promoteTask.mockRejectedValue(new Error('queue full'));
    h.s.tasks = [task({ id: 'd', status: 'DRAFT' })];
    const w = await mountView();
    await btn(w, 'Promote ↑').trigger('click');
    await flush();
    expect(h.toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error', summary: 'Promote Failed' }),
    );
    expect(btn(w, 'Promote ↑').attributes('disabled')).toBeUndefined();
  });

  it('opens the detail drawer for a row, and refreshes both lists after a cancel', async () => {
    h.s.tasks = [task({ id: 'q' })];
    const w = await mountView();
    await w.find('tbody tr').trigger('click');
    expect(w.findComponent(Drawer).props('modelValue')).toBe(true);
    expect(w.findComponent(Drawer).props('task')).toMatchObject({ id: 'q' });
    h.getAllTasks.mockClear();
    w.findComponent(Drawer).vm.$emit('cancelled');
    await flush();
    expect(h.refreshTasks).toHaveBeenCalled();
    expect(h.getAllTasks).toHaveBeenCalled();
  });

  it('refreshes the queue when the submit form reports a new task', async () => {
    const w = await mountView();
    w.findComponent(Form).vm.$emit('submitted', 'T-9');
    expect(h.refreshTasks).toHaveBeenCalled();
  });
});

describe('MissionControlView — history', () => {
  const finished = () => [
    task({ id: 'c', status: 'COMPLETED' }),
    task({ id: 'f', status: 'FAILED', updatedAt: new Date(NOW - 30 * 60_000).toISOString() }),
    task({ id: 'x', status: 'CANCELLED', updatedAt: new Date(NOW - 5 * 3600_000).toISOString() }),
    task({
      id: 'old',
      status: 'COMPLETED',
      updatedAt: new Date(NOW - 3 * 86_400_000).toISOString(),
    }),
  ];
  const historyRows = (w: W) => w.findAll('.divide-y > div');

  it('shows an empty state, then only completions and failures of the last hour', async () => {
    expect((await mountView()).text()).toContain('No completed tasks yet');
    h.getAllTasks.mockResolvedValue(finished());
    const w = await mountView();
    expect(historyRows(w)).toHaveLength(2);
    expect(w.text()).toContain('1m ago');
    expect(w.text()).toContain('30m ago');
  });

  it('shows all finished tasks with filters, and goes back', async () => {
    h.getAllTasks.mockResolvedValue(finished());
    const w = await mountView();
    await btn(w, 'Show All History').trigger('click');
    expect(historyRows(w)).toHaveLength(4);
    expect(w.text()).toContain('5h ago');
    expect(w.text()).toContain(new Date(NOW - 3 * 86_400_000).toLocaleDateString());
    await btn(w, 'Cancelled').trigger('click');
    expect(historyRows(w)).toHaveLength(1);
    await btn(w, 'Failed').trigger('click');
    expect(historyRows(w)).toHaveLength(1);
    await btn(w, 'Completed').trigger('click');
    expect(historyRows(w)).toHaveLength(2);
    await btn(w, 'All').trigger('click');
    await btn(w, 'Show Recent').trigger('click');
    expect(historyRows(w)).toHaveLength(2);
  });

  it('caps the recent list at ten', async () => {
    h.getAllTasks.mockResolvedValue(
      Array.from({ length: 14 }, () => task({ status: 'COMPLETED' })),
    );
    expect(historyRows(await mountView())).toHaveLength(10);
  });

  it('opens a history row in the drawer', async () => {
    h.getAllTasks.mockResolvedValue([task({ id: 'c', status: 'COMPLETED' })]);
    const w = await mountView();
    await historyRows(w)[0].trigger('click');
    expect(w.findComponent(Drawer).props('task')).toMatchObject({ id: 'c' });
  });

  it('keeps going when history cannot be loaded, treats null as empty, and polls every 30s', async () => {
    h.getAllTasks.mockRejectedValueOnce(new Error('x'));
    const w = await mountView();
    expect(w.text()).toContain('No completed tasks yet');
    h.getAllTasks.mockResolvedValue(null);
    await vi.advanceTimersByTimeAsync(30_000);
    expect(h.getAllTasks).toHaveBeenCalledTimes(2);
    w.unmount();
    h.getAllTasks.mockClear();
    await vi.advanceTimersByTimeAsync(120_000);
    expect(h.getAllTasks).not.toHaveBeenCalled();
  });
});

describe('MissionControlView — agents and providers', () => {
  it('shows loading skeletons and empty states', async () => {
    h.s.sessionsLoading = true;
    h.s.providersLoading = true;
    expect((await mountView()).findAll('.skeleton').length).toBeGreaterThanOrEqual(4);
    h.s.sessionsLoading = false;
    h.s.providersLoading = false;
    const t = (await mountView()).text();
    expect(t).toContain('No agents connected');
    expect(t).toContain('No providers available');
  });

  it('describes each session: project, model and status wording', async () => {
    h.s.sessions = [
      session({ agentName: 'A', status: 'active', projectPath: '/work/app', modelId: 'opus' }),
      session({
        agentName: 'B',
        status: 'idle',
        lastActivity: new Date(NOW - 5 * 60_000).toISOString(),
      }),
      session({
        agentName: 'C',
        status: 'idle',
        lastActivity: new Date(NOW - 10_000).toISOString(),
      }),
      session({ agentName: 'D', status: 'disconnected' }),
    ];
    const t = (await mountView()).text();
    expect(t).toContain('app');
    expect(t).toContain('opus');
    expect(t).toContain('idle 5m');
    expect(t).toContain('disconnected');
    expect(t).toContain('—'); // sessions without a project
  });

  it('describes each provider: model, context, URL, failures and last check', async () => {
    h.s.providers = [
      prov({
        name: 'Up',
        contextLimit: 32768,
        baseURL: 'http://127.0.0.1:1234',
        lastChecked: new Date(NOW - 2 * 60_000).toISOString(),
      }),
      prov({ name: 'Down', active: false, consecutiveFails: 3 }),
    ];
    const t = (await mountView()).text();
    expect(t).toContain('32k ctx');
    expect(t).toContain('http://127.0.0.1:1234');
    expect(t).toContain('2m ago');
    expect(t).toContain('3');
  });
});

describe('MissionControlView — edge data', () => {
  it('words one failure in the singular and a missing model as "no model"', async () => {
    h.s.providers = [prov({ activeModel: '', consecutiveFails: 1 })];
    const t = (await mountView()).text().replace(/\s+/g, ' ');
    expect(t).toContain('1 fail');
    expect(t).not.toContain('1 fails');
    expect(t).toContain('no model');
  });

  it('puts a task with an unknown status last and names a root project path', async () => {
    h.s.tasks = [
      task({ status: 'MYSTERY' as Task['status'], instruction: 'odd' }),
      task({ instruction: 'normal' }),
    ];
    h.s.sessions = [session({ projectPath: '/' })];
    const w = await mountView();
    const t = w.text();
    expect(t.indexOf('normal')).toBeLessThan(t.indexOf('odd'));
  });

  it('lists a finished task once even when it is also in the queue', async () => {
    const dup = task({ id: 'same', instruction: 'only once', status: 'COMPLETED' });
    const extra = [
      task({ instruction: 'unknown', status: 'ODD' as Task['status'] }),
      task({ instruction: 'unknown2', status: 'ODD2' as Task['status'] }),
    ];
    h.s.tasks = [dup, ...extra];
    h.getAllTasks.mockResolvedValue([]);
    const without = (await mountView()).text().split('only once').length;
    h.getAllTasks.mockResolvedValue([dup]);
    const withDup = (await mountView()).text().split('only once').length;
    // once in the work list (not twice, despite being in the history too) and once in the history
    expect(without).toBe(2);
    expect(withDup).toBe(3);
  });

  it('closes the detail drawer when it asks to', async () => {
    h.s.tasks = [task()];
    const w = await mountView();
    await w.find('tbody tr').trigger('click');
    const drawer = w.findComponent(Drawer);
    drawer.vm.$emit('update:modelValue', false);
    await w.vm.$nextTick();
    expect(drawer.props('modelValue')).toBe(false);
  });
});
