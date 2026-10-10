import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import TaskQueue from './TaskQueue.vue';
import type { Task } from '../types/domain';

const ButtonStub = {
  emits: ['click'],
  template: '<button class="cancel-btn" @click="$emit(\'click\', $event)"></button>',
};

function task(overrides: Partial<Task> = {}): Task {
  return {
    id: 't1',
    projectPath: '/work/app',
    targetFile: 'main.go',
    instruction: 'Write tests',
    contextFiles: [],
    modelId: '',
    providerHint: '',
    command: 'execute',
    status: 'QUEUED',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    logs: '',
    ...overrides,
  };
}

const mountQueue = (props: { tasks: Task[]; loading: boolean }) =>
  mount(TaskQueue, {
    props,
    global: {
      stubs: { Skeleton: { template: '<div class="skeleton"></div>' }, Button: ButtonStub },
      directives: { tooltip: {} },
    },
  });

describe('TaskQueue', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-06-15T12:00:00Z'));
  });
  afterEach(() => vi.useRealTimers());

  it('shows three skeleton rows while loading', () => {
    const w = mountQueue({ tasks: [], loading: true });
    expect(w.findAll('.skeleton')).toHaveLength(3);
    expect(w.text()).not.toContain('Queue is empty');
  });

  it('shows the empty state when there are no tasks', () => {
    const w = mountQueue({ tasks: [], loading: false });
    expect(w.text()).toContain('Queue is empty');
  });

  it('renders each task with its instruction, path and status', () => {
    const w = mountQueue({
      tasks: [task(), task({ id: 't2', instruction: 'Second', status: 'PROCESSING' })],
      loading: false,
    });
    expect(w.text()).toContain('Write tests');
    expect(w.text()).toContain('/work/app');
    expect(w.text()).toContain('main.go');
    expect(w.text()).toContain('PROCESSING');
  });

  it('emits select with the task when a row is clicked', async () => {
    const t = task();
    const w = mountQueue({ tasks: [t], loading: false });
    await w.find('.group').trigger('click');
    expect(w.emitted('select')?.[0]).toEqual([t]);
  });

  it('offers cancel only for QUEUED tasks and does not also select the row', async () => {
    const w = mountQueue({
      tasks: [task(), task({ id: 't2', status: 'COMPLETED' })],
      loading: false,
    });
    const buttons = w.findAll('.cancel-btn');
    expect(buttons).toHaveLength(1);
    await buttons[0].trigger('click');
    expect(w.emitted('cancel')?.[0]).toEqual(['t1']);
    expect(w.emitted('select')).toBeUndefined();
  });

  it.each([
    [10_000, 'just now'],
    [5 * 60_000, '5m ago'],
    [3 * 3_600_000, '3h ago'],
  ])('formats creation time %i ms ago as "%s"', (ms, label) => {
    const w = mountQueue({
      tasks: [task({ createdAt: new Date(Date.now() - ms).toISOString() })],
      loading: false,
    });
    expect(w.text()).toContain(label);
  });

  it('falls back to a calendar date for old tasks', () => {
    const iso = new Date(Date.now() - 5 * 86_400_000).toISOString();
    const w = mountQueue({ tasks: [task({ createdAt: iso })], loading: false });
    expect(w.text()).toContain(new Date(iso).toLocaleDateString());
  });

  it('previews logs of finished tasks, truncating long ones', () => {
    const long = 'x'.repeat(100);
    const w = mountQueue({
      tasks: [
        task({ id: 'c', status: 'COMPLETED', logs: 'all good' }),
        task({ id: 'f', status: 'FAILED', logs: long }),
        task({ id: 'q', status: 'QUEUED', logs: 'not shown while queued' }),
      ],
      loading: false,
    });
    expect(w.text()).toContain('all good');
    expect(w.text()).toContain('x'.repeat(80) + '…');
    expect(w.text()).not.toContain('not shown while queued');
  });
});
