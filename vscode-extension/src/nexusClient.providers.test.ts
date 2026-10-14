import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NexusClient } from './nexusClient';

describe('NexusClient provider registration', () => {
  const originalFetch = globalThis.fetch;
  let client: NexusClient;
  beforeEach(() => {
    client = new NexusClient('http://127.0.0.1:63987', 63988);
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it('registerProvider posts the config as an enabled provider', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({ name: 'vscode-x', kind: 'openaicompat' }),
    } as Response);
    await client.registerProvider({
      name: 'vscode-x',
      kind: 'openaicompat',
      baseUrl: 'http://127.0.0.1:1/v1',
      apiKey: 'k',
      model: 'm',
    });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'http://127.0.0.1:63987/api/providers',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          name: 'vscode-x',
          kind: 'openaicompat',
          baseUrl: 'http://127.0.0.1:1/v1',
          apiKey: 'k',
          model: 'm',
          enabled: true,
        }),
      }),
    );
  });

  it('removeProvider deletes by encoded name', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, status: 204 } as Response);
    await client.removeProvider('vscode a/b');
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'http://127.0.0.1:63987/api/providers/vscode%20a%2Fb',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });

  it('removeProvider treats an already-removed provider as done', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      text: async () => '{"error":"provider not found"}',
    } as Response);
    await expect(client.removeProvider('gone')).resolves.toBeUndefined();
  });

  it('removeProvider surfaces other failures', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      text: async () => '{"error":"boom"}',
    } as Response);
    await expect(client.removeProvider('x')).rejects.toThrow('HTTP 500');
    globalThis.fetch = vi.fn().mockRejectedValue('network down');
    await expect(client.removeProvider('x')).rejects.toBe('network down');
  });
});
