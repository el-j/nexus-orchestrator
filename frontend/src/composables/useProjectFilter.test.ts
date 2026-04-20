import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mockGetAllTasks, mockUseTasks } = vi.hoisted(() => ({
  mockGetAllTasks: vi.fn(),
  mockUseTasks: vi.fn(),
}));

vi.mock('../types/wails', () => ({ getAllTasks: mockGetAllTasks }));
vi.mock('./useTasks', () => ({ useTasks: mockUseTasks }));
vi.mock('./useProjectState', () => {
  const { ref } = require('vue');
  const currentProject = ref<string | null>(null);
  return {
    currentProject,
    setProject: vi.fn((p: string | null) => {
      currentProject.value = p;
    }),
  };
});

function makeTask(projectPath: string) {
  return {
    id: 't-1',
    projectPath,
    status: 'QUEUED',
    instruction: '',
    targetFile: '',
    contextFiles: [],
    modelId: '',
    providerHint: '',
    command: 'auto' as const,
    createdAt: '',
    updatedAt: '',
    logs: '',
  };
}

async function mountUseProjectFilter() {
  vi.resetModules();
  const { useProjectFilter } = await import('./useProjectFilter');
  let state: ReturnType<typeof useProjectFilter>;
  const Harness = defineComponent({
    setup() {
      state = useProjectFilter();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

describe('useProjectFilter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseTasks.mockReturnValue({ tasks: { value: [] } });
    mockGetAllTasks.mockResolvedValue([]);
  });

  it('projectList includes paths from live tasks', async () => {
    mockUseTasks.mockReturnValue({
      tasks: { value: [makeTask('/proj/a'), makeTask('/proj/b')] },
    });
    mockGetAllTasks.mockResolvedValue([]);

    const { wrapper, get } = await mountUseProjectFilter();
    await flushPromises();

    expect(get().projectList.value).toContain('/proj/a');
    expect(get().projectList.value).toContain('/proj/b');

    wrapper.unmount();
  });

  it('projectList includes paths from historical tasks (getAllTasks)', async () => {
    mockUseTasks.mockReturnValue({ tasks: { value: [] } });
    mockGetAllTasks.mockResolvedValue([makeTask('/proj/history')]);

    const { wrapper, get } = await mountUseProjectFilter();
    await flushPromises();

    expect(get().projectList.value).toContain('/proj/history');

    wrapper.unmount();
  });

  it('projectList deduplicates paths from live and history', async () => {
    mockUseTasks.mockReturnValue({ tasks: { value: [makeTask('/proj/shared')] } });
    mockGetAllTasks.mockResolvedValue([makeTask('/proj/shared'), makeTask('/proj/other')]);

    const { wrapper, get } = await mountUseProjectFilter();
    await flushPromises();

    const count = get().projectList.value.filter((p) => p === '/proj/shared').length;
    expect(count).toBe(1);
    expect(get().projectList.value).toContain('/proj/other');

    wrapper.unmount();
  });

  it('projectList is sorted alphabetically', async () => {
    mockUseTasks.mockReturnValue({
      tasks: { value: [makeTask('/proj/z'), makeTask('/proj/a')] },
    });

    const { wrapper, get } = await mountUseProjectFilter();
    await flushPromises();

    const list = get().projectList.value;
    expect(list).toEqual([...list].sort());

    wrapper.unmount();
  });

  it('cleans up the history interval on unmount', async () => {
    const clearSpy = vi.spyOn(globalThis, 'clearInterval');

    const { wrapper } = await mountUseProjectFilter();
    await flushPromises();
    wrapper.unmount();

    expect(clearSpy).toHaveBeenCalled();
  });
});
