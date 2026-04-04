import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useTasks } from './useTasks';

// ── Hoisted mock variables ────────────────────────────────────────────────────

const {
  mockGetQueue,
  mockCancelTask,
  mockPromoteTask,
  mockCreateDraft,
  mockUpdateTask,
  mockOn,
  mockOff,
} = vi.hoisted(() => ({
  mockGetQueue: vi.fn(),
  mockCancelTask: vi.fn(),
  mockPromoteTask: vi.fn(),
  mockCreateDraft: vi.fn(),
  mockUpdateTask: vi.fn(),
  mockOn: vi.fn(),
  mockOff: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  getQueue: mockGetQueue,
  cancelTask: mockCancelTask,
  promoteTask: mockPromoteTask,
  createDraft: mockCreateDraft,
  updateTask: mockUpdateTask,
}));

vi.mock('./useGlobalSSE', () => ({
  useGlobalSSE: () => ({ on: mockOn, off: mockOff, connected: { value: true } }),
}));

vi.mock('./useProjectState', () => ({
  currentProject: { value: null },
}));

// ── Helper ────────────────────────────────────────────────────────────────────

function makeTask(overrides: Record<string, unknown> = {}) {
  return {
    id: 'T-1',
    instruction: 'Test task',
    status: 'QUEUED',
    projectPath: '/',
    targetFile: 'test.go',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...overrides,
  };
}

function mountUseTasks() {
  let state: ReturnType<typeof useTasks>;
  const Harness = defineComponent({
    setup() {
      state = useTasks();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('useTasks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetQueue.mockResolvedValue([]);
    mockCancelTask.mockResolvedValue(undefined);
    mockPromoteTask.mockResolvedValue(undefined);
    mockCreateDraft.mockResolvedValue('new-id');
    mockUpdateTask.mockResolvedValue(makeTask());
  });

  it('populates tasks on mount and resets loading to false', async () => {
    mockGetQueue.mockResolvedValue([makeTask({ id: 'T-1' }), makeTask({ id: 'T-2' })]);

    const { wrapper, get } = mountUseTasks();
    await flushPromises();

    expect(get().tasks.value).toHaveLength(2);
    expect(get().tasks.value[0].id).toBe('T-1');
    expect(get().loading.value).toBe(false);
    expect(mockGetQueue).toHaveBeenCalledTimes(1);

    wrapper.unmount();
  });

  it('SSE task event triggers a refresh of the task list', async () => {
    mockGetQueue
      .mockResolvedValueOnce([makeTask({ id: 'T-initial' })])
      .mockResolvedValueOnce([makeTask({ id: 'T-updated', status: 'PROCESSING' })]);

    const { wrapper, get } = mountUseTasks();
    await flushPromises();

    expect(get().tasks.value[0].id).toBe('T-initial');
    expect(mockGetQueue).toHaveBeenCalledTimes(1);

    // Capture the wildcard SSE handler registered via on('*', fn)
    const sseCalls = mockOn.mock.calls.filter(([type]) => type === '*');
    expect(sseCalls.length).toBeGreaterThan(0);
    const sseHandler = sseCalls[0][1] as (d: Record<string, unknown>) => void;

    // Fire a non-'connected' SSE event to trigger refresh
    sseHandler({ type: 'task_updated', id: 'T-updated' });
    await flushPromises();

    expect(mockGetQueue).toHaveBeenCalledTimes(2);
    expect(get().tasks.value[0].id).toBe('T-updated');

    wrapper.unmount();
  });

  it('cancelTask calls wailsCancelTask then removes task from list on success', async () => {
    mockGetQueue.mockResolvedValueOnce([makeTask({ id: 'T-1' })]).mockResolvedValueOnce([]); // after cancel, list is empty

    const { wrapper, get } = mountUseTasks();
    await flushPromises();
    expect(get().tasks.value).toHaveLength(1);

    await get().cancelTask('T-1');
    await flushPromises();

    expect(mockCancelTask).toHaveBeenCalledWith('T-1');
    expect(get().tasks.value).toHaveLength(0);

    wrapper.unmount();
  });

  it('promoteTask calls wailsPromoteTask and re-fetches', async () => {
    mockGetQueue
      .mockResolvedValueOnce([makeTask({ id: 'D-1', status: 'DRAFT' })])
      .mockResolvedValueOnce([makeTask({ id: 'D-1', status: 'QUEUED' })]);

    const { wrapper, get } = mountUseTasks();
    await flushPromises();

    await get().promoteTask('D-1');
    await flushPromises();

    expect(mockPromoteTask).toHaveBeenCalledWith('D-1');
    expect(mockGetQueue).toHaveBeenCalledTimes(2);
    expect(get().tasks.value[0].status).toBe('QUEUED');

    wrapper.unmount();
  });

  it('sets error.value when getQueue rejects on initial fetch', async () => {
    mockGetQueue.mockRejectedValue(new Error('Network error'));

    const { wrapper, get } = mountUseTasks();
    await flushPromises();

    expect(get().error.value).toBe('Network error');
    expect(get().tasks.value).toHaveLength(0);
    expect(get().loading.value).toBe(false);

    wrapper.unmount();
  });

  it('sets error.value when refresh fails after promoteTask', async () => {
    mockGetQueue
      .mockResolvedValueOnce([makeTask({ id: 'D-2', status: 'DRAFT' })])
      .mockRejectedValueOnce(new Error('queue fetch error'));

    const { wrapper, get } = mountUseTasks();
    await flushPromises();

    await get().promoteTask('D-2');
    await flushPromises();

    expect(get().error.value).toBe('queue fetch error');

    wrapper.unmount();
  });

  it('cleans up SSE handler and interval on unmount', async () => {
    const { wrapper } = mountUseTasks();
    await flushPromises();

    wrapper.unmount();

    expect(mockOff).toHaveBeenCalledWith('*', expect.any(Function));
  });
});
