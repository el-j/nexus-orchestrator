import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import TaskDetailDrawer from './TaskDetailDrawer.vue';
import type { Task } from '../types/domain';

const api = vi.hoisted(() => ({ cancelTask: vi.fn(), toastAdd: vi.fn() }));
vi.mock('../types/wails', () => ({ cancelTask: api.cancelTask }));
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: api.toastAdd }) }));

const DrawerStub = {
  props: ['visible'],
  emits: ['update:visible'],
  template: '<div data-testid="drawer"><slot name="header" /><slot /></div>',
};
const ButtonStub = {
  props: ['label', 'loading'],
  emits: ['click'],
  template: '<button :data-loading="loading" @click="$emit(\'click\')">{{ label }}</button>',
};

const task = (o: Partial<Task> = {}): Task =>
  ({
    id: 't-1',
    instruction: 'Write docs',
    projectPath: '/p',
    targetFile: 'README.md',
    status: 'QUEUED',
    command: 'write',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...o,
  }) as Task;

const mountDrawer = (t: Task | null) =>
  mount(TaskDetailDrawer, {
    props: { task: t, modelValue: true },
    global: { stubs: { Drawer: DrawerStub, Button: ButtonStub } },
  });

beforeEach(() => {
  api.cancelTask.mockReset();
  api.toastAdd.mockReset();
});

describe('TaskDetailDrawer', () => {
  it('renders nothing but the header chrome without a task', () => {
    const w = mountDrawer(null);
    expect(w.text()).toContain('Task Details');
    expect(w.text()).not.toContain('Instruction');
  });

  it('shows the task fields', () => {
    const w = mountDrawer(task({ providerHint: 'ollama', modelId: 'llama3' }));
    const t = w.text();
    expect(t).toContain('t-1');
    expect(t).toContain('write');
    expect(t).toContain('Write docs');
    expect(t).toContain('/p');
    expect(t).toContain('README.md');
    expect(t).toContain('ollama');
    expect(t).toContain('llama3');
  });

  it('falls back to placeholders for optional fields', () => {
    const w = mountDrawer(task({ command: undefined, projectPath: '', targetFile: '' }));
    expect(w.text()).toContain('auto');
    expect(w.text()).toContain('Auto'); // provider + model
    expect(w.text()).toContain('—');
  });

  it('shows the output only when there are logs', () => {
    expect(mountDrawer(task()).text()).not.toContain('Output');
    expect(mountDrawer(task({ logs: 'line 1' })).text()).toContain('line 1');
  });

  it('offers Cancel for queued and Interrupt for processing tasks only', () => {
    expect(mountDrawer(task({ status: 'QUEUED' })).text()).toContain('Cancel Task');
    const processing = mountDrawer(task({ status: 'PROCESSING' })).text();
    expect(processing).toContain('Interrupt');
    expect(processing).not.toContain('Cancel Task');
    const done = mountDrawer(task({ status: 'COMPLETED' })).text();
    expect(done).not.toContain('Cancel Task');
    expect(done).not.toContain('Interrupt');
  });

  it('cancels, toasts, and tells the parent to close', async () => {
    api.cancelTask.mockResolvedValue(undefined);
    const w = mountDrawer(task());
    await w.find('button').trigger('click');
    await flushPromises();
    expect(api.cancelTask).toHaveBeenCalledWith('t-1');
    expect(api.toastAdd).toHaveBeenCalledWith(expect.objectContaining({ severity: 'success' }));
    expect(w.emitted('cancelled')![0]).toEqual(['t-1']);
    expect(w.emitted('close')).toHaveLength(1);
    expect(w.emitted('update:modelValue')![0]).toEqual([false]);
  });

  it('reports a failed cancel and keeps the drawer open', async () => {
    api.cancelTask.mockRejectedValue(new Error('too late'));
    const w = mountDrawer(task({ status: 'PROCESSING' }));
    await w.find('button').trigger('click');
    await flushPromises();
    expect(w.text()).toContain('too late');
    expect(api.toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error', detail: 'too late' }),
    );
    expect(w.emitted('cancelled')).toBeUndefined();
    expect(w.emitted('update:modelValue')).toBeUndefined();
    expect(w.find('button').attributes('data-loading')).not.toBe('true');
  });

  it('stringifies a non-Error failure', async () => {
    api.cancelTask.mockRejectedValue('nope');
    const w = mountDrawer(task({ status: 'PROCESSING' }));
    await w.find('button').trigger('click');
    await flushPromises();
    expect(w.text()).toContain('nope');
  });

  it('propagates visibility changes from the drawer', () => {
    const w = mountDrawer(task());
    w.findComponent(DrawerStub).vm.$emit('update:visible', false);
    expect(w.emitted('update:modelValue')![0]).toEqual([false]);
  });
});
