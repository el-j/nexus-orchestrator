import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NexusClient } from './nexusClient';

describe('NexusClient.getUsage', () => {
  const originalFetch = globalThis.fetch;
  let client: NexusClient;
  const answer = (status: number, body: unknown) => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: status < 400,
      status,
      json: async () => body,
      text: async () => JSON.stringify(body),
    } as Response);
  };
  beforeEach(() => {
    client = new NexusClient('http://127.0.0.1:63987', 63988);
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it('reads the usage report of the period and fills the optional fields', async () => {
    answer(200, {
      totals: { calls: 2, inputTokens: 10, outputTokens: 5 },
      usage: [{ provider: 'Claude', calls: 2, inputTokens: 10, outputTokens: 5 }],
    });
    const report = await client.getUsage('7d');
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'http://127.0.0.1:63987/api/usage?since=7d',
      expect.anything(),
    );
    expect(report.since).toBe('');
    expect(report.usage[0]).toMatchObject({ provider: 'Claude', model: '', role: '' });
  });

  it('defaults to 24 hours and treats a null list as empty', async () => {
    answer(200, { since: 'x', totals: { calls: 0, inputTokens: 0, outputTokens: 0 }, usage: null });
    expect((await client.getUsage()).usage).toEqual([]);
    expect(String((globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0])).toContain(
      'since=24h',
    );
  });

  it('rejects an answer that is not a usage report and a daemon without the ledger', async () => {
    answer(200, []);
    await expect(client.getUsage()).rejects.toThrow();
    answer(404, { error: 'not found' });
    await expect(client.getUsage()).rejects.toThrow();
  });
});
