import { describe, expect, it, vi, beforeEach } from 'vitest';
import { useBrain } from './useBrain';
import * as wails from '../types/wails';

vi.mock('../types/wails', () => ({
  getBrainStatus: vi.fn(),
  initProject: vi.fn(),
  ingestKnowledge: vi.fn(),
  listKnowledge: vi.fn(),
  deleteKnowledge: vi.fn(),
  searchKnowledge: vi.fn(),
  getFileMap: vi.fn(),
  getProjectContext: vi.fn(),
  getFocusedContext: vi.fn(),
}));

describe('useBrain', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetchStatus updates status ref on success', async () => {
    const mockStatus = {
      projectPath: '/test/repo',
      initialized: true,
      entryCount: 10,
      kindCounts: { architecture: 5 },
      totalTokens: 500,
    };
    vi.mocked(wails.getBrainStatus).mockResolvedValue(mockStatus);

    const brain = useBrain('/test/repo');
    await brain.fetchStatus();

    expect(brain.status.value).toEqual(mockStatus);
    expect(brain.loading.value).toBe(false);
    expect(brain.error.value).toBeNull();
  });

  it('search updates searchResults ref', async () => {
    const mockResults = [
      { title: 'Arch', topic: 'Arch', kind: 'architecture', content: 'test', tokens: 10 },
    ];
    vi.mocked(wails.searchKnowledge).mockResolvedValue(mockResults);

    const brain = useBrain('/test/repo');
    await brain.search('architecture', 5);

    expect(brain.searchResults.value).toEqual(mockResults);
    expect(wails.searchKnowledge).toHaveBeenCalledWith('/test/repo', 'architecture', 5);
  });

  it('deleteEntry calls deleteKnowledge and updates entries ref', async () => {
    vi.mocked(wails.deleteKnowledge).mockResolvedValue();
    vi.mocked(wails.getBrainStatus).mockResolvedValue({
      projectPath: '/test/repo',
      initialized: true,
      entryCount: 0,
      kindCounts: {},
      totalTokens: 0,
    });

    const brain = useBrain('/test/repo');
    brain.entries.value = [
      {
        id: 'k1',
        projectPath: '/test/repo',
        kind: 'convention',
        topic: 'style',
        content: 'rules',
        tokenCount: 20,
        relevanceScore: 0.8,
        createdAt: '2026-04-14T02:00:00Z',
        updatedAt: '2026-04-14T02:00:00Z',
      },
    ];

    await brain.deleteEntry('k1');
    expect(wails.deleteKnowledge).toHaveBeenCalledWith('k1');
    expect(brain.entries.value).toHaveLength(0);
  });

  it('getContext calls getProjectContext', async () => {
    const mockResponse = {
      projectPath: '/test/repo',
      sections: [],
      totalTokens: 0,
      truncated: false,
    };
    vi.mocked(wails.getProjectContext).mockResolvedValue(mockResponse);

    const brain = useBrain('/test/repo');
    const res = await brain.getContext(500);

    expect(res).toEqual(mockResponse);
    expect(wails.getProjectContext).toHaveBeenCalledWith('/test/repo', 500);
  });
});
