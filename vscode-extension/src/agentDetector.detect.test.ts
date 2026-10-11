import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient } from './nexusClient';

const h = vi.hoisted(() => ({
  files: new Map<string, string>(),
  dirs: new Set<string>(),
  installed: new Set<string>(),
  terminals: [] as Array<{ name: string; creationOptions: { shellPath?: string } }>,
  lm: undefined as undefined | { selectChatModels: (f: object) => Promise<unknown[]> },
  workspace: [{ uri: { fsPath: '/workspace' } }] as Array<{ uri: { fsPath: string } }> | undefined,
}));

vi.mock('fs/promises', () => ({
  readFile: vi.fn(async (p: string) => {
    if (h.files.has(p)) return h.files.get(p);
    throw new Error('ENOENT');
  }),
  stat: vi.fn(async (p: string) => {
    if (h.dirs.has(p)) return {};
    throw new Error('ENOENT');
  }),
}));
vi.mock('os', () => ({ homedir: () => '/home/u' }));
vi.mock('vscode', () => {
  class EventEmitter {
    listeners: Array<() => void> = [];
    event = (l: () => void) => this.listeners.push(l);
    fire = () => this.listeners.forEach((l) => l());
    dispose = vi.fn();
  }
  return {
    EventEmitter,
    extensions: { getExtension: (id: string) => (h.installed.has(id) ? { id } : undefined) },
    window: {
      get terminals() {
        return h.terminals;
      },
    },
    workspace: {
      get workspaceFolders() {
        return h.workspace;
      },
    },
    get lm() {
      return h.lm;
    },
  };
});

const HOME = '/home/u';
const CLAUDE_SETTINGS = `${HOME}/.claude/settings.json`;
const MAC_DIR = `${HOME}/Library/Application Support/Claude`;
const LINUX_DIR = `${HOME}/.config/claude`;
const MAC_CFG = `${MAC_DIR}/claude_desktop_config.json`;
const LINUX_CFG = `${LINUX_DIR}/claude_desktop_config.json`;

function context() {
  const store = new Map<string, unknown>();
  return {
    globalState: {
      get: (k: string) => store.get(k),
      update: vi.fn(async (k: string, v: unknown) => void store.set(k, v)),
    },
  } as unknown as import('vscode').ExtensionContext;
}
const fakeClient = (o: Record<string, unknown> = {}) =>
  ({
    registerSession: vi.fn().mockResolvedValue({ id: 's1' }),
    heartbeatSession: vi.fn().mockResolvedValue(true),
    deregisterSession: vi.fn().mockResolvedValue(undefined),
    ...o,
  }) as unknown as NexusClient;

async function detector(client = fakeClient(), ctx = context()) {
  const { AgentDetector } = await import('./agentDetector');
  return new AgentDetector(client, ctx);
}
const names = async (d: Awaited<ReturnType<typeof detector>>) =>
  (await d.detectAll()).map((a) => a.agentName).sort();

beforeEach(() => {
  vi.resetModules();
  h.files.clear();
  h.dirs.clear();
  h.installed.clear();
  h.terminals = [];
  h.lm = undefined;
  h.workspace = [{ uri: { fsPath: '/workspace' } }];
});
afterEach(() => vi.useRealTimers());

describe('machine id', () => {
  it('keeps the stored id, or creates and stores one', async () => {
    const stored = context();
    await stored.globalState.update('machineId', 'm-fixed');
    h.installed.add('continue.continue');
    const d = await detector(fakeClient(), stored);
    expect((await d.detectAll())[0].externalId).toBe('discover:m-fixed:ext:continue.continue');

    const fresh = context();
    await detector(fakeClient(), fresh);
    expect(fresh.globalState.update).toHaveBeenCalledWith(
      'machineId',
      expect.stringMatching(/^m-/),
    );
  });
});

describe('installed extensions', () => {
  it('finds every known AI extension', async () => {
    for (const id of [
      'saoudrizwan.claude-dev',
      'continue.continue',
      'codeium.codeium',
      'codegpt.codegpt-4',
      'anysphere.cursor-always-local',
      'github.copilot',
      'github.copilot-chat',
    ]) {
      h.installed.add(id);
    }
    expect(await names(await detector())).toEqual([
      'Cline',
      'CodeGPT',
      'Codeium',
      'Continue',
      'Cursor AI',
      'GitHub Copilot',
      'GitHub Copilot Chat',
    ]);
  });

  it('attaches the Copilot model id to Copilot extensions only', async () => {
    h.installed.add('github.copilot');
    h.installed.add('continue.continue');
    h.lm = {
      selectChatModels: vi
        .fn()
        .mockResolvedValue([{ id: 'gpt-x', vendor: 'copilot', family: 'gpt' }]),
    };
    const found = await (await detector()).detectAll();
    expect(found.find((a) => a.agentName === 'GitHub Copilot')?.modelId).toBe('gpt-x');
    expect(found.find((a) => a.agentName === 'Continue')?.modelId).toBeUndefined();
  });

  it('falls back to the model name, and tolerates no models or a failing lookup', async () => {
    h.installed.add('github.copilot');
    h.lm = {
      selectChatModels: vi.fn().mockResolvedValue([{ name: 'by-name', vendor: 'copilot' }]),
    };
    expect((await (await detector()).detectAll())[0].modelId).toBe('by-name');
    h.lm = { selectChatModels: vi.fn().mockResolvedValue([]) };
    expect((await (await detector()).detectAll())[0].modelId).toBeUndefined();
    h.lm = { selectChatModels: vi.fn().mockRejectedValue(new Error('no')) };
    expect(await names(await detector())).toEqual(['GitHub Copilot']);
  });
});

describe('a failing strategy', () => {
  it('does not hide what the other strategies found', async () => {
    h.installed.add('continue.continue');
    const d = await detector();
    vi.spyOn(
      d as unknown as { detectFromTerminals(): Promise<unknown[]> },
      'detectFromTerminals',
    ).mockRejectedValue(new Error('terminals unavailable'));
    expect(await names(d)).toContain('Continue');
  });
});

describe('filesystem detection', () => {
  it('detects Claude CLI only when settings.json holds an apiKey', async () => {
    h.files.set(CLAUDE_SETTINGS, JSON.stringify({ apiKey: 'sk-secret' }));
    const found = await (await detector()).detectAll();
    expect(found.map((a) => a.agentName)).toEqual(['Claude CLI']);
    expect(JSON.stringify(found)).not.toContain('sk-secret'); // the key itself is never reported

    for (const bad of [
      JSON.stringify({}),
      JSON.stringify({ apiKey: 5 }),
      'null',
      '"str"',
      'not json',
    ]) {
      h.files.set(CLAUDE_SETTINGS, bad);
      expect(await names(await detector())).toEqual([]);
    }
  });

  it('detects Claude Desktop from either config directory, once', async () => {
    h.dirs.add(MAC_DIR);
    h.dirs.add(LINUX_DIR);
    expect(await names(await detector())).toEqual(['Claude Desktop']);
    h.dirs.delete(MAC_DIR);
    expect(await names(await detector())).toEqual(['Claude Desktop']);
  });

  it('lists MCP servers from the first readable Claude Desktop config', async () => {
    h.files.set(MAC_CFG, JSON.stringify({ mcpServers: { github: {}, fs: {} } }));
    h.files.set(LINUX_CFG, JSON.stringify({ mcpServers: { ignored: {} } }));
    expect(await names(await detector())).toEqual(['MCP: fs', 'MCP: github']);
  });

  it('reads the Linux config when the macOS one is absent, and ignores configs without servers', async () => {
    h.files.set(LINUX_CFG, JSON.stringify({ mcpServers: { only: {} } }));
    expect(await names(await detector())).toEqual(['MCP: only']);
    for (const cfg of [{}, { mcpServers: null }, { mcpServers: 'x' }]) {
      h.files.clear();
      h.files.set(MAC_CFG, JSON.stringify(cfg));
      expect(await names(await detector())).toEqual([]);
    }
    h.files.clear();
    h.files.set(MAC_CFG, 'null');
    expect(await names(await detector())).toEqual([]);
  });
});

describe('terminals', () => {
  it('detects AI tools by terminal name or shell path', async () => {
    h.terminals = [
      { name: 'claude', creationOptions: {} },
      { name: 'bash', creationOptions: { shellPath: '/usr/local/bin/cursor-agent' } },
      { name: 'zsh', creationOptions: {} },
      { name: 'x', creationOptions: undefined as unknown as { shellPath?: string } },
    ];
    // creationOptions may be undefined for some terminals; that must not crash detection
    h.terminals[3].creationOptions = {};
    expect(await names(await detector())).toEqual(['bash (terminal)', 'claude (terminal)']);
  });
});

describe('language-model participants', () => {
  it('reports non-Copilot models, preferring id over name', async () => {
    h.lm = {
      selectChatModels: vi.fn().mockResolvedValue([
        { vendor: 'copilot', family: 'gpt', id: 'c1' },
        { vendor: 'anthropic', family: 'claude', id: 'a1' },
        { vendor: 'ollama', family: 'llama', name: 'llama3' },
      ]),
    };
    const found = await (await detector()).detectAll();
    const lm = found.filter((a) => a.detectionMethod === 'lm-participant');
    expect(lm.map((a) => [a.agentName, a.modelId])).toEqual([
      ['anthropic/claude', 'a1'],
      ['ollama/llama', 'llama3'],
    ]);
  });

  it('survives an LM API that throws or is missing', async () => {
    h.lm = { selectChatModels: vi.fn().mockRejectedValue(new Error('x')) };
    expect(await names(await detector())).toEqual([]);
    h.lm = {} as never;
    expect(await names(await detector())).toEqual([]);
  });
});

describe('deduplication and lifecycle', () => {
  it('keeps one entry per external id', async () => {
    h.terminals = [
      { name: 'claude', creationOptions: {} },
      { name: 'claude', creationOptions: {} },
    ];
    expect(await names(await detector())).toEqual(['claude (terminal)']);
  });

  it('registers with the workspace path (or empty), heartbeats known agents and deregisters vanished ones', async () => {
    const client = fakeClient();
    h.installed.add('continue.continue');
    const d = await detector(client);
    const tick = () => (d as unknown as { tick(): Promise<void> }).tick();
    const changed = vi.fn();
    d.onDidChange(changed);

    await tick();
    expect(client.registerSession).toHaveBeenCalledWith(
      expect.objectContaining({
        agentName: 'Continue',
        source: 'vscode',
        projectPath: '/workspace',
      }),
    );
    expect(changed).toHaveBeenCalled();
    await tick();
    expect(client.heartbeatSession).toHaveBeenCalledWith('s1');

    h.installed.clear();
    await tick();
    expect(client.deregisterSession).toHaveBeenCalledWith('s1');

    h.workspace = undefined;
    h.installed.add('continue.continue');
    await tick();
    expect(client.registerSession).toHaveBeenLastCalledWith(
      expect.objectContaining({ projectPath: '' }),
    );
  });

  it('keeps going when registration, heartbeat or deregistration fail', async () => {
    const client = fakeClient({
      registerSession: vi
        .fn()
        .mockRejectedValueOnce(new Error('x'))
        .mockResolvedValue({ id: 's1' }),
      heartbeatSession: vi.fn().mockRejectedValue(new Error('y')),
      deregisterSession: vi.fn().mockRejectedValue(new Error('z')),
    });
    h.installed.add('continue.continue');
    const d = await detector(client);
    const tick = () => (d as unknown as { tick(): Promise<void> }).tick();
    await tick(); // register fails
    await tick(); // register ok
    await tick(); // heartbeat fails
    h.installed.clear();
    await expect(tick()).resolves.toBeUndefined(); // deregister fails
  });

  it('swallows an unexpected detection failure', async () => {
    const d = await detector();
    vi.spyOn(d, 'detectAll').mockRejectedValue(new Error('boom'));
    await expect((d as unknown as { tick(): Promise<void> }).tick()).resolves.toBeUndefined();
  });

  it('polls every 30s after starting, stops, and refuses to start once disposed', async () => {
    vi.useFakeTimers();
    const client = fakeClient();
    h.installed.add('continue.continue');
    const d = await detector(client);
    d.start();
    await vi.advanceTimersByTimeAsync(0);
    expect(client.registerSession).toHaveBeenCalledTimes(1);
    // a tick registers new agents and then heartbeats every known one, new ones included
    expect(client.heartbeatSession).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(30_000);
    expect(client.heartbeatSession).toHaveBeenCalledTimes(2);
    d.stop();
    d.stop(); // idempotent
    await vi.advanceTimersByTimeAsync(120_000);
    expect(client.heartbeatSession).toHaveBeenCalledTimes(2);

    d.dispose();
    d.start();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(client.heartbeatSession).toHaveBeenCalledTimes(2);
    await (d as unknown as { tick(): Promise<void> }).tick();
    expect(client.heartbeatSession).toHaveBeenCalledTimes(2);
  });
});
