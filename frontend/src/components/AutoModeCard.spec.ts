import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AutoModeCard from './AutoModeCard.vue';
import type { AutoModeRun } from '../types/domain';

const h = vi.hoisted(() => ({
  project: { value: null as string | null },
  startAutoMode: vi.fn(),
  listAutoModeRuns: vi.fn(),
  stopAutoMode: vi.fn(),
}));

vi.mock('../composables/useProjectState', async () => {
  const { ref } = await import('vue');
  const currentProject = ref<string | null>(null);
  h.project = currentProject as unknown as { value: string | null };
  return { currentProject };
});

vi.mock('../types/wails', () => ({
  startAutoMode: h.startAutoMode,
  listAutoModeRuns: h.listAutoModeRuns,
  stopAutoMode: h.stopAutoMode,
}));

const makeRun = (o: Partial<AutoModeRun> = {}): AutoModeRun => ({
  id: 'run-1',
  projectPath: '/work/demo',
  goal: 'Build feature',
  phase: 'executing',
  taskIds: ['t1', 't2'],
  completedTasks: 1,
  failedTasks: 0,
  startedAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
  ...o,
});

describe('AutoModeCard', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    h.project.value = null;
    h.startAutoMode.mockReset();
    h.listAutoModeRuns.mockReset().mockResolvedValue([]);
    h.stopAutoMode.mockReset().mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('asks user to select a project when none is chosen', async () => {
    const w = mount(AutoModeCard);
    await flushPromises();
    expect(w.find('[data-testid="automode-no-project"]').exists()).toBe(true);
    expect(h.listAutoModeRuns).not.toHaveBeenCalled();
    w.unmount();
  });

  it('starts and stops an Auto-Mode run and polls while active', async () => {
    h.project.value = '/work/demo';
    const running = makeRun({ phase: 'executing', repairAttempts: 1, failedTasks: 1 });
    h.startAutoMode.mockResolvedValue(running);
    const w = mount(AutoModeCard);
    await flushPromises();

    await w.find('[data-testid="automode-goal-input"]').setValue('Ship it');
    await w.find('[data-testid="automode-start-btn"]').trigger('click');
    await flushPromises();

    expect(h.startAutoMode).toHaveBeenCalledWith({ projectPath: '/work/demo', goal: 'Ship it' });
    expect(w.find('[data-testid="automode-phase"]').text()).toBe('executing');
    expect(w.text()).toContain('repair #1');

    // Polling fires while active
    h.listAutoModeRuns.mockResolvedValue([makeRun({ phase: 'completed', summary: 'All green' })]);
    await vi.advanceTimersByTimeAsync(3000);
    await flushPromises();
    expect(w.find('[data-testid="automode-summary"]').text()).toBe('All green');

    // Start again and stop
    h.startAutoMode.mockResolvedValue(makeRun({ id: 'run-2', goal: '', phase: 'planning' }));
    await w.find('[data-testid="automode-start-btn"]').trigger('click');
    await flushPromises();
    h.listAutoModeRuns.mockResolvedValue([
      makeRun({ id: 'run-2', phase: 'stopped', summary: 'Stopped' }),
    ]);
    await w.find('[data-testid="automode-stop-btn"]').trigger('click');
    await flushPromises();
    expect(h.stopAutoMode).toHaveBeenCalledWith('run-2');
    expect(w.find('[data-testid="automode-phase"]').text()).toBe('stopped');
    w.unmount();
  });

  it('surfaces errors from list, start, and stop', async () => {
    h.project.value = '/work/demo';
    h.listAutoModeRuns.mockRejectedValueOnce(new Error('list down'));
    const w = mount(AutoModeCard);
    await flushPromises();
    expect(w.find('[data-testid="automode-error"]').text()).toBe('list down');

    h.startAutoMode.mockRejectedValueOnce('start boom');
    await w.find('[data-testid="automode-start-btn"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="automode-error"]').text()).toBe('Failed to start Auto-Mode');

    h.startAutoMode.mockResolvedValueOnce(makeRun({ phase: 'verifying' }));
    await w.find('[data-testid="automode-start-btn"]').trigger('click');
    await flushPromises();

    h.stopAutoMode.mockRejectedValueOnce(new Error('stop boom'));
    await w.find('[data-testid="automode-stop-btn"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="automode-error"]').text()).toBe('stop boom');

    // Failed summary styling + non-Error list rejection + idle timer tick
    h.listAutoModeRuns.mockRejectedValueOnce('raw list err');
    h.project.value = '/work/err';
    await flushPromises();
    expect(w.find('[data-testid="automode-error"]').text()).toBe('Failed to load Auto-Mode status');

    h.listAutoModeRuns.mockResolvedValue([makeRun({ phase: 'failed', error: 'verify failed' })]);
    h.project.value = '/work/other';
    await flushPromises();
    expect(w.find('[data-testid="automode-summary"]').text()).toBe('verify failed');
    await vi.advanceTimersByTimeAsync(3000); // idle tick does not poll

    // Error instance on start, non-Error on stop, and re-starting same run ID
    h.startAutoMode.mockRejectedValueOnce(new Error('start err'));
    await w.find('[data-testid="automode-start-btn"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="automode-error"]').text()).toBe('start err');

    h.startAutoMode.mockResolvedValueOnce(makeRun({ id: 'run-1', phase: 'syncing' }));
    await w.find('[data-testid="automode-start-btn"]').trigger('click');
    await flushPromises();

    h.stopAutoMode.mockRejectedValueOnce('raw stop err');
    await w.find('[data-testid="automode-stop-btn"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="automode-error"]').text()).toBe('Failed to stop Auto-Mode');

    // Busy guard on start via keydown.enter while a start request is in flight
    h.listAutoModeRuns.mockResolvedValue([makeRun({ phase: 'stopped' })]);
    h.project.value = '/work/busy';
    await flushPromises();
    let finishStart!: (r: AutoModeRun) => void;
    h.startAutoMode.mockReturnValueOnce(new Promise<AutoModeRun>((r) => (finishStart = r)));
    const input = w.find('[data-testid="automode-goal-input"]');
    await input.trigger('keydown.enter');
    await input.trigger('keydown.enter'); // ignored while busy
    finishStart(makeRun({ id: 'run-busy', phase: 'executing' }));
    await flushPromises();
    expect(h.startAutoMode).toHaveBeenCalledTimes(5);
    w.unmount();
  });
});
