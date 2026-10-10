import { describe, it, expect, vi, beforeEach } from 'vitest';

const { getServerAddr } = vi.hoisted(() => ({ getServerAddr: vi.fn() }));
vi.mock('../types/wails', () => ({ getServerAddr }));

describe('resolveServerUrl', () => {
  beforeEach(() => {
    vi.resetModules(); // the URL is cached at module level
    getServerAddr.mockReset();
  });

  it('asks the Wails binding once and caches the answer for the session', async () => {
    getServerAddr.mockResolvedValue('http://127.0.0.1:63987');
    const { resolveServerUrl } = await import('./useServerUrl');
    expect(await resolveServerUrl()).toBe('http://127.0.0.1:63987');
    expect(await resolveServerUrl()).toBe('http://127.0.0.1:63987');
    expect(getServerAddr).toHaveBeenCalledTimes(1);
  });

  it('does not cache a failure: the next call asks again', async () => {
    getServerAddr.mockRejectedValueOnce(new Error('not ready')).mockResolvedValueOnce('http://h:1');
    const { resolveServerUrl } = await import('./useServerUrl');
    await expect(resolveServerUrl()).rejects.toThrow('not ready');
    expect(await resolveServerUrl()).toBe('http://h:1');
  });
});
