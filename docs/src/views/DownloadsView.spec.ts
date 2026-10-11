import { describe, it, expect, vi, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { memoryRouter } from '../test/router';

const nav = (platform: string, ua = '', extra: Record<string, unknown> = {}) => {
  Object.defineProperty(window.navigator, 'platform', { value: platform, configurable: true });
  Object.defineProperty(window.navigator, 'userAgent', { value: ua, configurable: true });
  Object.defineProperty(window.navigator, 'userAgentData', {
    value: extra['userAgentData'],
    configurable: true,
  });
};

async function mountDownloads(env: Record<string, string> = {}) {
  for (const [k, v] of Object.entries(env)) vi.stubEnv(k, v);
  vi.resetModules();
  const { default: DownloadsView } = await import('./DownloadsView.vue');
  const router = memoryRouter();
  await router.isReady();
  const w = mount(DownloadsView, { global: { plugins: [router] } });
  await flushPromises();
  return w;
}

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

const hrefs = (w: Awaited<ReturnType<typeof mountDownloads>>) =>
  w.findAll('a').map((a) => a.attributes('href') ?? '');

describe('DownloadsView links (stable channel)', () => {
  it('points every download at /releases/latest/download with the real file names', async () => {
    nav('Linux x86_64');
    const w = await mountDownloads();
    const base = 'https://github.com/el-j/nexus-orchestrator/releases/latest/download';
    const files = hrefs(w).filter((h) => h.startsWith(base));
    for (const f of [
      'nexus-orchestrator-desktop-darwin-arm64.zip',
      'nexus-orchestrator-desktop-darwin-amd64.zip',
      'nexus-orchestrator-desktop-windows-amd64.zip',
      'nexus-orchestrator-desktop-linux-amd64.tar.gz',
      'nexus-orchestrator-darwin-arm64.tar.gz',
      'nexus-orchestrator-linux-arm64.tar.gz',
      'nexus-orchestrator-windows-amd64.zip',
      'nexus-orchestrator.vsix',
    ]) {
      expect(files, f).toContain(`${base}/${f}`);
    }
  });

  it('tells people to install the extension file the release actually publishes', async () => {
    nav('Linux x86_64');
    const w = await mountDownloads();
    expect(w.text()).toContain('code --install-extension nexus-orchestrator.vsix');
    expect(w.text()).not.toContain('nexus-orchestrator-vscode.vsix');
  });

  it('pipes the installer from main', async () => {
    nav('Linux x86_64');
    const w = await mountDownloads();
    expect(w.text()).toContain(
      'raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh',
    );
  });
});

describe('OS detection', () => {
  // The Desktop App heading carries a static "Recommended" pill; only the per-OS card badge reflects detection.
  const recommended = (w: Awaited<ReturnType<typeof mountDownloads>>) =>
    w.findAll('div.mb-2 span').filter((s) => s.text() === 'Recommended');

  it('recognises Windows and Linux (x86-64 and ARM)', async () => {
    nav('Win32');
    let w = await mountDownloads();
    expect(w.text()).toContain('Windows 64-bit');
    expect(recommended(w).length).toBeGreaterThan(0);

    nav('Linux x86_64', 'Mozilla/5.0 (X11; Linux x86_64)');
    w = await mountDownloads();
    expect(w.text()).toContain('Linux 64-bit');

    nav('Linux armv8l', 'Mozilla/5.0 (X11; Linux aarch64)');
    w = await mountDownloads();
    expect(w.text()).toContain('Linux ARM64');
  });

  it('asks the user to choose on an unknown platform', async () => {
    nav('Plan9');
    const w = await mountDownloads();
    expect(w.text()).toContain('Unknown — choose below');
    expect(recommended(w)).toHaveLength(0);
  });

  it('detects Apple Silicon from userAgentData high-entropy hints', async () => {
    nav('MacIntel', '', {
      userAgentData: { getHighEntropyValues: async () => ({ architecture: 'arm' }) },
    });
    const w = await mountDownloads();
    expect(w.text()).toContain('macOS (Apple Silicon)');
    expect(recommended(w).length).toBeGreaterThan(0);
  });

  it('detects Apple Silicon from the legacy architecture field and from the user agent', async () => {
    nav('MacIntel', '', { userAgentData: { architecture: 'arm' } });
    expect((await mountDownloads()).text()).toContain('macOS (Apple Silicon)');
    nav('MacIntel', 'Mozilla/5.0 (Macintosh; ARM64)');
    expect((await mountDownloads()).text()).toContain('macOS (Apple Silicon)');
  });

  it('detects Apple Silicon from the WebGL renderer, and tolerates probes that throw', async () => {
    nav('MacIntel', 'Mozilla/5.0 (Macintosh)', {
      userAgentData: {
        getHighEntropyValues: async () => {
          throw new Error('blocked');
        },
      },
    });
    const gl = {
      getExtension: () => ({ UNMASKED_RENDERER_WEBGL: 1 }),
      getParameter: () => 'Apple M2',
    };
    vi.spyOn(document, 'createElement').mockImplementation(((tag: string) =>
      tag === 'canvas'
        ? ({ getContext: () => gl } as unknown as HTMLCanvasElement)
        : document.createElementNS(
            'http://www.w3.org/1999/xhtml',
            tag,
          )) as typeof document.createElement);
    expect((await mountDownloads()).text()).toContain('macOS (Apple Silicon)');
    vi.restoreAllMocks();

    nav('MacIntel', 'Mozilla/5.0 (Macintosh)');
    vi.spyOn(document, 'createElement').mockImplementation(((tag: string) => {
      if (tag === 'canvas') throw new Error('no canvas');
      return document.createElementNS('http://www.w3.org/1999/xhtml', tag);
    }) as typeof document.createElement);
    expect((await mountDownloads()).text()).toContain('architecture unknown');
    vi.restoreAllMocks();
  });

  it('cannot tell an Intel Mac from one whose renderer is hidden', async () => {
    nav('MacIntel', 'Mozilla/5.0 (Macintosh)');
    const gl = { getExtension: () => null, getParameter: () => '' };
    vi.spyOn(document, 'createElement').mockImplementation(((tag: string) =>
      tag === 'canvas'
        ? ({ getContext: () => gl } as unknown as HTMLCanvasElement)
        : document.createElementNS(
            'http://www.w3.org/1999/xhtml',
            tag,
          )) as typeof document.createElement);
    const w = await mountDownloads();
    expect(w.text()).toContain('architecture unknown');
    vi.restoreAllMocks();
  });
});

describe('beta channel', () => {
  const releases = (list: Array<{ prerelease: boolean; tag_name: string }>) =>
    vi.fn().mockResolvedValue({ ok: true, json: async () => list });

  it('links the newest prerelease tag instead of /latest', async () => {
    nav('Linux x86_64');
    vi.stubGlobal(
      'fetch',
      releases([
        { prerelease: false, tag_name: 'v1.0.0' },
        { prerelease: true, tag_name: 'v1.1.0-beta.3' },
      ]),
    );
    const w = await mountDownloads({ VITE_DOCS_CHANNEL: 'beta' });
    const files = hrefs(w).filter((h) => h.includes('/releases/download/'));
    expect(files.length).toBeGreaterThan(5);
    expect(files.every((h) => h.includes('/releases/download/v1.1.0-beta.3/'))).toBe(true);
    expect(w.text()).toContain(
      'raw.githubusercontent.com/el-j/nexus-orchestrator/v1.1.0-beta.3/scripts/install.sh',
    );
  });

  it('falls back to /latest when there is no prerelease, the API errors, or the request fails', async () => {
    nav('Linux x86_64');
    for (const fetchMock of [
      releases([{ prerelease: false, tag_name: 'v1.0.0' }]),
      vi.fn().mockResolvedValue({ ok: false }),
      vi.fn().mockRejectedValue(new Error('offline')),
    ]) {
      vi.stubGlobal('fetch', fetchMock);
      const w = await mountDownloads({ VITE_DOCS_CHANNEL: 'beta' });
      expect(
        hrefs(w).some((h) => h.includes('/releases/latest/download/nexus-orchestrator-')),
      ).toBe(true);
      expect(w.text()).toContain('/main/scripts/install.sh');
    }
  });

  it('does not call the GitHub API on the stable channel', async () => {
    nav('Linux x86_64');
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    await mountDownloads();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('treats incomplete Mac hints as "architecture unknown"', async () => {
    const noWebGL = () =>
      vi
        .spyOn(document, 'createElement')
        .mockImplementation(((tag: string) =>
          tag === 'canvas'
            ? ({ getContext: () => null } as unknown as HTMLCanvasElement)
            : document.createElementNS(
                'http://www.w3.org/1999/xhtml',
                tag,
              )) as typeof document.createElement);
    for (const userAgentData of [
      { getHighEntropyValues: async () => ({}) },
      { getHighEntropyValues: async () => ({ architecture: 'x86' }) },
      { architecture: 'x86' },
    ]) {
      nav('MacIntel', 'Mozilla/5.0 (Macintosh)', { userAgentData });
      noWebGL();
      expect((await mountDownloads()).text()).toContain('architecture unknown');
      vi.restoreAllMocks();
    }
  });

  it('reads the renderer from the experimental WebGL context and ignores other GPUs', async () => {
    nav('MacIntel', 'Mozilla/5.0 (Macintosh)');
    const gl = {
      getExtension: () => ({ UNMASKED_RENDERER_WEBGL: 1 }),
      getParameter: () => 'NVIDIA GeForce',
    };
    vi.spyOn(document, 'createElement').mockImplementation(((tag: string) =>
      tag === 'canvas'
        ? ({
            getContext: (kind: string) => (kind === 'webgl' ? null : gl),
          } as unknown as HTMLCanvasElement)
        : document.createElementNS(
            'http://www.w3.org/1999/xhtml',
            tag,
          )) as typeof document.createElement);
    expect((await mountDownloads()).text()).toContain('architecture unknown');
    vi.restoreAllMocks();
  });

  it('copes with a browser that reports no platform at all', async () => {
    nav('', '');
    expect((await mountDownloads()).text()).toContain('Unknown');
  });

  it('cannot tell the architecture when the renderer string is empty', async () => {
    nav('MacIntel', 'Mozilla/5.0 (Macintosh)');
    const gl = { getExtension: () => ({ UNMASKED_RENDERER_WEBGL: 1 }), getParameter: () => '' };
    vi.spyOn(document, 'createElement').mockImplementation(((tag: string) =>
      tag === 'canvas'
        ? ({ getContext: () => gl } as unknown as HTMLCanvasElement)
        : document.createElementNS(
            'http://www.w3.org/1999/xhtml',
            tag,
          )) as typeof document.createElement);
    expect((await mountDownloads()).text()).toContain('architecture unknown');
    vi.restoreAllMocks();
  });
});
