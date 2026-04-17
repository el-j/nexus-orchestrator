import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useBrain } from './useBrain';

const {
  mockGetBrainStatus,
  mockInitProject,
  mockIngestKnowledge,
  mockListKnowledge,
  mockDeleteKnowledge,
  mockSearchKnowledge,
  mockGetFileMap,
} = vi.hoisted(() => ({
  mockGetBrainStatus: vi.fn(),
  mockInitProject: vi.fn(),
  mockIngestKnowledge: vi.fn(),
  mockListKnowledge: vi.fn(),
  mockDeleteKnowledge: vi.fn(),
  mockSearchKnowledge: vi.fn(),
  mockGetFileMap: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  getBrainStatus: mockGetBrainStatus,
  initProject: mockInitProject,
  ingestKnowledge: mockIngestKnowledge,
  listKnowledge: mockListKnowledge,
  deleteKnowledge: mockDeleteKnowledge,
  searchKnowledge: mockSearchKnowledge,
  getFileMap: mockGetFileMap,
}));

const PROJECT = '/proj/test';

function mountUseBrain() {
  let state: ReturnType<typeof useBrain>;
  const Harness = defineComponent({
    setup() {
      state = useBrain(PROJECT);
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

const stubStatus = {
  projectPath: PROJECT,
  initialized: true,
  entryCount: 3,
  kindCounts: {},
  totalTokens: 500,
};

describe('useBrain', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetBrainStatus.mockResolvedValue(stubStatus);
    mockInitProject.mockResolvedValue(stubStatus);
    mockIngestKnowledge.mockResolvedValue(2);
    mockListKnowledge.mockResolvedValue([]);
    mockDeleteKnowledge.mockResolvedValue(undefined);
    mockSearchKnowledge.mockResolvedValue([]);
    mockGetFileMap.mockResolvedValue([]);
  });

  it('fetchStatus populates status.value and clears loading/error', async () => {
    const { wrapper, get } = mountUseBrain();
    expect(get().loading.value).toBe(false);

    const p = get().fetchStatus();
    expect(get().loading.value).toBe(true);
    await p;
    await flushPromises();

    expect(get().status.value).toEqual(stubStatus);
    expect(get().loading.value).toBe(false);
    expect(get().error.value).toBeNull();
    expect(mockGetBrainStatus).toHaveBeenCalledWith(PROJECT);

    wrapper.unmount();
  });

  it('fetchStatus sets error.value on failure', async () => {
    mockGetBrainStatus.mockRejectedValue(new Error('network error'));
    const { wrapper, get } = mountUseBrain();
    await get().fetchStatus();

    expect(get().error.value).toBe('network error');
    expect(get().loading.value).toBe(false);

    wrapper.unmount();
  });

  it('init calls initProject and updates status', async () => {
    const { wrapper, get } = mountUseBrain();
    await get().init('CLAUDE.md');
    await flushPromises();

    expect(mockInitProject).toHaveBeenCalledWith(PROJECT, 'CLAUDE.md');
    expect(get().status.value).toEqual(stubStatus);

    wrapper.unmount();
  });

  it('ingest calls ingestKnowledge then refreshes status; returns count', async () => {
    const { wrapper, get } = mountUseBrain();
    const count = await get().ingest('/some/file.md');
    await flushPromises();

    expect(mockIngestKnowledge).toHaveBeenCalledWith(PROJECT, '/some/file.md');
    expect(mockGetBrainStatus).toHaveBeenCalledWith(PROJECT);
    expect(count).toBe(2);

    wrapper.unmount();
  });

  it('ingest returns 0 and sets error on failure', async () => {
    mockIngestKnowledge.mockRejectedValue(new Error('ingest failed'));
    const { wrapper, get } = mountUseBrain();
    const count = await get().ingest('/bad.md');

    expect(count).toBe(0);
    expect(get().error.value).toBe('ingest failed');

    wrapper.unmount();
  });

  it('listEntries populates entries.value', async () => {
    const entry = {
      id: 'k-1',
      projectPath: PROJECT,
      kind: 'rule',
      topic: 'T',
      content: 'C',
      tokenCount: 10,
      relevanceScore: 1,
      createdAt: '',
      updatedAt: '',
    };
    mockListKnowledge.mockResolvedValue([entry]);

    const { wrapper, get } = mountUseBrain();
    await get().listEntries('rule');
    await flushPromises();

    expect(mockListKnowledge).toHaveBeenCalledWith(PROJECT, 'rule');
    expect(get().entries.value).toHaveLength(1);
    expect(get().entries.value[0].id).toBe('k-1');

    wrapper.unmount();
  });

  it('deleteEntry removes entry client-side and refreshes status', async () => {
    const entry = {
      id: 'k-del',
      projectPath: PROJECT,
      kind: 'rule',
      topic: 'T',
      content: 'C',
      tokenCount: 5,
      relevanceScore: 1,
      createdAt: '',
      updatedAt: '',
    };
    mockListKnowledge.mockResolvedValue([entry]);

    const { wrapper, get } = mountUseBrain();
    await get().listEntries();
    await flushPromises();
    expect(get().entries.value).toHaveLength(1);

    await get().deleteEntry('k-del');
    await flushPromises();

    expect(mockDeleteKnowledge).toHaveBeenCalledWith('k-del');
    expect(get().entries.value).toHaveLength(0);
    expect(mockGetBrainStatus).toHaveBeenCalled();

    wrapper.unmount();
  });

  it('search populates searchResults.value', async () => {
    const section = {
      title: 'S',
      topic: 'architecture',
      kind: 'rule',
      content: 'text',
      tokens: 30,
      source: 'file',
    };
    mockSearchKnowledge.mockResolvedValue([section]);

    const { wrapper, get } = mountUseBrain();
    await get().search('architecture', 5);
    await flushPromises();

    expect(mockSearchKnowledge).toHaveBeenCalledWith(PROJECT, 'architecture', 5);
    expect(get().searchResults.value).toHaveLength(1);
    expect(get().searchResults.value[0].topic).toBe('architecture');

    wrapper.unmount();
  });

  it('fetchFileMap populates fileMap.value', async () => {
    mockGetFileMap.mockResolvedValue(['src/a.ts', 'src/b.ts']);

    const { wrapper, get } = mountUseBrain();
    await get().fetchFileMap('api');
    await flushPromises();

    expect(mockGetFileMap).toHaveBeenCalledWith(PROJECT, 'api');
    expect(get().fileMap.value).toEqual(['src/a.ts', 'src/b.ts']);

    wrapper.unmount();
  });
});
