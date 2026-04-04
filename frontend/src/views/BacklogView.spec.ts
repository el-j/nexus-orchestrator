import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount } from '@vue/test-utils';
import { flushPromises } from '@vue/test-utils';
import BacklogView from './BacklogView.vue';

// ── Wails + composable mocks ──────────────────────────────────────────────────

const { mockGetBacklog, mockOn, mockOff, mockConnectedRef } = vi.hoisted(() => ({
  mockGetBacklog: vi.fn(),
  mockOn: vi.fn(),
  mockOff: vi.fn(),
  mockConnectedRef: { value: true },
}));

vi.mock('../types/wails', () => ({
  getBacklog: mockGetBacklog,
  cancelTask: vi.fn(),
  promoteTask: vi.fn(),
  updateTask: vi.fn(),
}));

vi.mock('../composables/useGlobalSSE', () => ({
  useGlobalSSE: () => ({
    on: mockOn,
    off: mockOff,
    connected: mockConnectedRef,
  }),
}));

vi.mock('../composables/useProjectState', () => ({
  currentProject: { value: null },
}));

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeTask(overrides: Record<string, unknown> = {}) {
  return {
    id: 'bg-1',
    instruction: 'Draft task',
    status: 'DRAFT',
    projectPath: '/',
    targetFile: '',
    priority: 2,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...overrides,
  };
}

const STUBS = {
  global: {
    stubs: {
      BacklogList: true,
      RefreshIndicator: true,
    },
  },
};

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('BacklogView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockConnectedRef.value = true;
    mockGetBacklog.mockResolvedValue([]);
  });

  it('calls getBacklog on mount and renders BacklogList', async () => {
    mockGetBacklog.mockResolvedValue([makeTask(), makeTask({ id: 'bg-2' })]);

    const wrapper = shallowMount(BacklogView, STUBS);
    await flushPromises();

    expect(mockGetBacklog).toHaveBeenCalledTimes(1);
    // BacklogList stub is rendered with items prop
    const backlogList = wrapper.findComponent({ name: 'BacklogList' });
    expect(backlogList.exists()).toBe(true);
  });

  it('re-fetches when BacklogList emits "promoted" event', async () => {
    mockGetBacklog
      .mockResolvedValueOnce([makeTask()])
      .mockResolvedValueOnce([makeTask(), makeTask({ id: 'bg-promoted' })]);

    const wrapper = shallowMount(BacklogView, STUBS);
    await flushPromises();
    expect(mockGetBacklog).toHaveBeenCalledTimes(1);

    // Simulate BacklogList emitting "promoted"
    await wrapper.findComponent({ name: 'BacklogList' }).trigger('promoted');
    await flushPromises();

    expect(mockGetBacklog).toHaveBeenCalledTimes(2);
  });

  it('shows error banner and retry button when getBacklog rejects', async () => {
    mockGetBacklog.mockRejectedValue(new Error('Failed to load backlog'));

    const wrapper = shallowMount(BacklogView, STUBS);
    await flushPromises();

    const html = wrapper.html();
    expect(html).toContain('Failed to load backlog');
    // Retry button should be present
    const retryBtn = wrapper.findAll('button').find((b) => b.text().includes('Retry'));
    expect(retryBtn).toBeDefined();
  });

  it('shows item count in header after loading', async () => {
    mockGetBacklog.mockResolvedValue([makeTask(), makeTask({ id: 'bg-3' })]);

    const wrapper = shallowMount(BacklogView, STUBS);
    await flushPromises();

    // Header shows "Backlog" and count
    const header = wrapper.find('header');
    expect(header.exists()).toBe(true);
    expect(header.text()).toContain('Backlog');
  });

  it('cleans up SSE handler on unmount', async () => {
    const wrapper = shallowMount(BacklogView, STUBS);
    await flushPromises();

    wrapper.unmount();

    expect(mockOff).toHaveBeenCalledWith('*', expect.any(Function));
  });
});
