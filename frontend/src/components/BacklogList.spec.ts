import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import BacklogList from './BacklogList.vue';
import type { Task } from '../types/domain';

const { promoteTask, cancelTask, toastAdd } = vi.hoisted(() => ({
  promoteTask: vi.fn(),
  cancelTask: vi.fn(),
  toastAdd: vi.fn(),
}));
vi.mock('../composables/useTasks', () => ({ useTasks: () => ({ promoteTask, cancelTask }) }));
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: toastAdd }) }));

const ButtonStub = {
  props: ['label', 'loading', 'disabled'],
  emits: ['click'],
  template:
    '<button :data-loading="loading" :disabled="disabled" @click="$emit(\'click\')">{{ label }}</button>',
};

const item = (o: Partial<Task> = {}): Task =>
  ({
    id: 'b1',
    instruction: 'Draft something',
    projectPath: '/p',
    status: 'DRAFT',
    priority: 2,
    ...o,
  }) as Task;

const mountList = (items: Task[]) =>
  mount(BacklogList, { props: { items }, global: { stubs: { Button: ButtonStub } } });

describe('BacklogList', () => {
  beforeEach(() => {
    [promoteTask, cancelTask, toastAdd].forEach((m) => m.mockReset());
    promoteTask.mockResolvedValue(undefined);
    cancelTask.mockResolvedValue(undefined);
  });

  it('shows an empty state', () => {
    expect(mountList([]).text()).toContain('Backlog is empty');
  });

  it.each([
    [1, 'High', 'text-red-400'],
    [2, 'Medium', 'text-amber-400'],
    [undefined, 'Medium', 'text-amber-400'],
    [3, 'Low', 'text-slate-500'],
  ])('labels priority %s as %s', (priority, label, colour) => {
    const w = mountList([item({ priority })]);
    expect(w.text()).toContain(label);
    expect(w.find(`.${colour}`).exists()).toBe(true);
  });

  it('shows the provider, "Auto" when none, tags, and truncates long instructions', () => {
    const long = 'y'.repeat(200);
    const w = mountList([
      item({ providerName: 'ollama', tags: ['urgent'], instruction: long }),
      item({ id: 'b2', providerName: undefined }),
    ]);
    expect(w.text()).toContain('ollama');
    expect(w.text()).toContain('Auto');
    expect(w.text()).toContain('urgent');
    expect(w.text()).toContain('y'.repeat(120) + '…');
    expect(w.text()).not.toContain('y'.repeat(121));
  });

  it('promotes a task and emits "promoted"', async () => {
    const w = mountList([item()]);
    await w.findAll('button')[0].trigger('click');
    await flushPromises();
    expect(promoteTask).toHaveBeenCalledWith('b1');
    expect(w.emitted('promoted')?.[0]).toEqual(['b1']);
  });

  it('dismisses a task and emits "dismissed"', async () => {
    const w = mountList([item()]);
    await w.findAll('button')[1].trigger('click');
    await flushPromises();
    expect(cancelTask).toHaveBeenCalledWith('b1');
    expect(w.emitted('dismissed')?.[0]).toEqual(['b1']);
  });

  it('reports failures with a toast and does not emit', async () => {
    promoteTask.mockRejectedValue(new Error('nope'));
    cancelTask.mockRejectedValue(new Error('denied'));
    const w = mountList([item()]);
    await w.findAll('button')[0].trigger('click');
    await flushPromises();
    await w.findAll('button')[1].trigger('click');
    await flushPromises();
    expect(toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error', summary: 'Promote Failed' }),
    );
    expect(toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error', summary: 'Dismiss Failed' }),
    );
    expect(w.emitted('promoted')).toBeUndefined();
    expect(w.emitted('dismissed')).toBeUndefined();
  });

  it('ignores a second click while an action is in flight', async () => {
    let release!: () => void;
    promoteTask.mockReturnValue(new Promise<void>((r) => (release = r)));
    const w = mountList([item()]);
    const promote = w.findAll('button')[0];
    await promote.trigger('click');
    await promote.trigger('click');
    expect(promoteTask).toHaveBeenCalledTimes(1);
    expect(w.findAll('button')[0].attributes('data-loading')).toBe('true');
    expect(w.findAll('button')[1].attributes('disabled')).toBeDefined();
    release();
    await flushPromises();

    let releaseDismiss!: () => void;
    cancelTask.mockReturnValue(new Promise<void>((r) => (releaseDismiss = r)));
    const dismiss = w.findAll('button')[1];
    await dismiss.trigger('click');
    await dismiss.trigger('click');
    expect(cancelTask).toHaveBeenCalledTimes(1);
    releaseDismiss();
    await flushPromises();
  });
});
