import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mockGetServerAddr } = vi.hoisted(() => ({
  mockGetServerAddr: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  getServerAddr: mockGetServerAddr,
}));

describe('useServerUrl / resolveServerUrl', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('calls getServerAddr and returns the resolved URL', async () => {
    mockGetServerAddr.mockResolvedValue('http://127.0.0.1:63987');
    const { resolveServerUrl } = await import('./useServerUrl');

    const url = await resolveServerUrl();

    expect(mockGetServerAddr).toHaveBeenCalledTimes(1);
    expect(url).toBe('http://127.0.0.1:63987');
  });

  it('caches the URL — only calls getServerAddr once across multiple calls', async () => {
    mockGetServerAddr.mockResolvedValue('http://127.0.0.1:63987');
    const { resolveServerUrl } = await import('./useServerUrl');

    const url1 = await resolveServerUrl();
    const url2 = await resolveServerUrl();
    const url3 = await resolveServerUrl();

    expect(mockGetServerAddr).toHaveBeenCalledTimes(1);
    expect(url1).toBe(url2);
    expect(url2).toBe(url3);
  });

  it('fresh module has no cached value — calls getServerAddr again', async () => {
    // Second test module reset ensures no bleed between tests
    vi.resetModules();
    mockGetServerAddr.mockResolvedValue('http://127.0.0.1:9999');
    const { resolveServerUrl } = await import('./useServerUrl');

    const url = await resolveServerUrl();

    expect(url).toBe('http://127.0.0.1:9999');
    expect(mockGetServerAddr).toHaveBeenCalledTimes(1);
  });
});
