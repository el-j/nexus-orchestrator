import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { NexusClient } from './nexusClient';

describe('NexusClient Brain APIs', () => {
  let client: NexusClient;
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    client = new NexusClient('http://127.0.0.1:63987', 63988);
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it('searchKnowledge unwraps results array correctly', async () => {
    const mockResults = [
      { id: '1', kind: 'architecture', title: 'Arch', content: 'Clean Arch', tokenCount: 50 },
    ];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ results: mockResults }),
    } as Response);

    const results = await client.searchKnowledge('/workspace', 'clean architecture', 5);
    expect(results).toEqual(mockResults);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'http://127.0.0.1:63987/api/brain/search?projectPath=%2Fworkspace&query=clean+architecture&limit=5',
    );
  });

  it('initProject sends POST /api/brain/init and returns BrainStatus', async () => {
    const mockStatus = {
      projectPath: '/workspace',
      entryCount: 15,
      totalTokens: 1200,
      lastIngested: '2026-04-14T02:00:00Z',
    };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => mockStatus,
    } as Response);

    const status = await client.initProject('/workspace');
    expect(status).toEqual(mockStatus);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'http://127.0.0.1:63987/api/brain/init',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ projectPath: '/workspace', claudeMDPath: '' }),
      }),
    );
  });

  it('listKnowledge sends GET /api/brain/knowledge and unwraps items', async () => {
    const mockKnowledge = [
      {
        id: 'k1',
        projectPath: '/workspace',
        kind: 'convention',
        title: 'Conventions',
        content: 'Rules',
      },
    ];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => mockKnowledge,
    } as Response);

    const items = await client.listKnowledge('/workspace', 'convention');
    expect(items).toEqual(mockKnowledge);
  });

  it('deleteKnowledge sends DELETE /api/brain/knowledge/:id', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      json: async () => ({}),
    } as Response);

    await client.deleteKnowledge('k123');
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'http://127.0.0.1:63987/api/brain/knowledge/k123',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });

  it('getFileMap sends GET /api/brain/file-map and unwraps filePaths', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ filePaths: ['src/a.ts', 'src/b.ts'] }),
    } as Response);

    const paths = await client.getFileMap('/workspace');
    expect(paths).toEqual(['src/a.ts', 'src/b.ts']);
  });
});
