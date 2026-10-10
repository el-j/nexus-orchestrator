import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useBrain } from './useBrain';
import type { ProjectKnowledge } from '../types/domain';

const api = vi.hoisted(() => ({
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
vi.mock('../types/wails', () => api);

const entry = (id: string) => ({ id }) as ProjectKnowledge;

beforeEach(() => {
  Object.values(api).forEach((f) => f.mockReset());
});

describe('useBrain', () => {
  it('fetchStatus loads status and toggles loading', async () => {
    api.getBrainStatus.mockResolvedValue({ initialized: true });
    const b = useBrain('/p');
    const p = b.fetchStatus();
    expect(b.loading.value).toBe(true);
    await p;
    expect(b.status.value).toEqual({ initialized: true });
    expect(b.loading.value).toBe(false);
    expect(api.getBrainStatus).toHaveBeenCalledWith('/p');
  });

  it('init forwards the CLAUDE.md path (default empty) and stores the status', async () => {
    api.initProject.mockResolvedValue({ initialized: true });
    const b = useBrain('/p');
    await b.init();
    expect(api.initProject).toHaveBeenCalledWith('/p', '');
    await b.init('/p/CLAUDE.md');
    expect(api.initProject).toHaveBeenLastCalledWith('/p', '/p/CLAUDE.md');
    expect(b.status.value).toEqual({ initialized: true });
  });

  it('ingest returns the count and refreshes status; returns 0 and records the error on failure', async () => {
    api.ingestKnowledge.mockResolvedValueOnce(4).mockRejectedValueOnce(new Error('bad file'));
    api.getBrainStatus.mockResolvedValue({ initialized: true });
    const b = useBrain('/p');
    expect(await b.ingest('/p/a.md')).toBe(4);
    expect(api.getBrainStatus).toHaveBeenCalledTimes(1);
    expect(await b.ingest('/p/b.md')).toBe(0);
    expect(b.error.value).toBe('bad file');
  });

  it('listEntries passes the kind filter', async () => {
    api.listKnowledge.mockResolvedValue([entry('a')]);
    const b = useBrain('/p');
    await b.listEntries();
    expect(api.listKnowledge).toHaveBeenCalledWith('/p', '');
    await b.listEntries('decision');
    expect(api.listKnowledge).toHaveBeenLastCalledWith('/p', 'decision');
    expect(b.entries.value).toEqual([entry('a')]);
  });

  it('deleteEntry removes the row locally and refreshes status', async () => {
    api.listKnowledge.mockResolvedValue([entry('a'), entry('b')]);
    api.getBrainStatus.mockResolvedValue({});
    const b = useBrain('/p');
    await b.listEntries();
    await b.deleteEntry('a');
    expect(api.deleteKnowledge).toHaveBeenCalledWith('a');
    expect(b.entries.value.map((e) => e.id)).toEqual(['b']);
    expect(api.getBrainStatus).toHaveBeenCalled();
  });

  it('deleteEntry keeps the row when the delete fails', async () => {
    api.listKnowledge.mockResolvedValue([entry('a')]);
    api.deleteKnowledge.mockRejectedValue(new Error('locked'));
    const b = useBrain('/p');
    await b.listEntries();
    await b.deleteEntry('a');
    expect(b.entries.value).toHaveLength(1);
    expect(b.error.value).toBe('locked');
  });

  it('search uses limit 10 by default', async () => {
    api.searchKnowledge.mockResolvedValue([{ heading: 'h' }]);
    const b = useBrain('/p');
    await b.search('auth');
    expect(api.searchKnowledge).toHaveBeenCalledWith('/p', 'auth', 10);
    await b.search('auth', 3);
    expect(api.searchKnowledge).toHaveBeenLastCalledWith('/p', 'auth', 3);
    expect(b.searchResults.value).toEqual([{ heading: 'h' }]);
  });

  it('fetchFileMap passes the focus area', async () => {
    api.getFileMap.mockResolvedValue(['a.go']);
    const b = useBrain('/p');
    await b.fetchFileMap();
    expect(api.getFileMap).toHaveBeenCalledWith('/p', '');
    await b.fetchFileMap('internal');
    expect(api.getFileMap).toHaveBeenLastCalledWith('/p', 'internal');
    expect(b.fileMap.value).toEqual(['a.go']);
  });

  it('getContext and getFocusedContext return the response, with token defaults', async () => {
    api.getProjectContext.mockResolvedValue({ text: 'ctx' });
    api.getFocusedContext.mockResolvedValue({ text: 'focus' });
    const b = useBrain('/p');
    expect(await b.getContext()).toEqual({ text: 'ctx' });
    expect(api.getProjectContext).toHaveBeenCalledWith('/p', 800);
    expect(await b.getFocusedContext('why?')).toEqual({ text: 'focus' });
    expect(api.getFocusedContext).toHaveBeenCalledWith('/p', 'why?', 400);
    await b.getContext(50);
    expect(api.getProjectContext).toHaveBeenLastCalledWith('/p', 50);
  });

  it.each([
    ['fetchStatus', 'getBrainStatus', (b: ReturnType<typeof useBrain>) => b.fetchStatus()],
    ['init', 'initProject', (b: ReturnType<typeof useBrain>) => b.init()],
    ['listEntries', 'listKnowledge', (b: ReturnType<typeof useBrain>) => b.listEntries()],
    ['search', 'searchKnowledge', (b: ReturnType<typeof useBrain>) => b.search('q')],
    ['fetchFileMap', 'getFileMap', (b: ReturnType<typeof useBrain>) => b.fetchFileMap()],
  ])(
    '%s records an Error message, or stringifies other values, and clears loading',
    async (_n, fn, run) => {
      api[fn as keyof typeof api]
        .mockRejectedValueOnce(new Error('boom'))
        .mockRejectedValueOnce('plain');
      const b = useBrain('/p');
      await run(b);
      expect(b.error.value).toBe('boom');
      expect(b.loading.value).toBe(false);
      await run(b);
      expect(b.error.value).toBe('plain');
    },
  );

  it('context getters return null and record the error on failure', async () => {
    api.getProjectContext.mockRejectedValueOnce(new Error('a')).mockRejectedValueOnce('b');
    api.getFocusedContext.mockRejectedValueOnce(new Error('c')).mockRejectedValueOnce('d');
    const b = useBrain('/p');
    expect(await b.getContext()).toBeNull();
    expect(b.error.value).toBe('a');
    expect(await b.getContext()).toBeNull();
    expect(b.error.value).toBe('b');
    expect(await b.getFocusedContext('q')).toBeNull();
    expect(b.error.value).toBe('c');
    expect(await b.getFocusedContext('q')).toBeNull();
    expect(b.error.value).toBe('d');
    expect(b.loading.value).toBe(false);
  });

  it('clears a previous error when the next call starts', async () => {
    api.getBrainStatus.mockRejectedValueOnce(new Error('x')).mockResolvedValueOnce({});
    const b = useBrain('/p');
    await b.fetchStatus();
    expect(b.error.value).toBe('x');
    await b.fetchStatus();
    expect(b.error.value).toBeNull();
  });
});
