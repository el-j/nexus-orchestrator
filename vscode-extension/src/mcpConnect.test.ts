import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
  existsSync,
  lstatSync,
} from 'fs';
import { tmpdir } from 'os';
import * as path from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const h = vi.hoisted(() => ({
  appName: 'Antigravity',
  mode: 'ask' as string,
  api: true,
  commands: new Map<string, () => Promise<void>>(),
  defs: [] as unknown[],
  provider: undefined as undefined | { provideMcpServerDefinitions(): unknown[] },
  info: [] as string[],
  errors: [] as string[],
  choice: undefined as string | undefined,
  prompts: 0,
}));

vi.mock('vscode', () => {
  const lm = {
    registerMcpServerDefinitionProvider: (
      _id: string,
      p: { provideMcpServerDefinitions(): unknown[] },
    ) => {
      h.provider = p;
      return { dispose() {} };
    },
  };
  class McpHttpServerDefinition {
    constructor(
      public label: string,
      public uri: { value: string },
      public headers?: Record<string, string>,
    ) {}
  }
  return {
    Uri: { parse: (v: string) => ({ value: v }) },
    env: {
      get appName() {
        return h.appName;
      },
    },
    get lm() {
      return h.api ? lm : undefined;
    },
    get McpHttpServerDefinition() {
      return h.api ? McpHttpServerDefinition : undefined;
    },
    workspace: { getConfiguration: () => ({ get: (_k: string, d: string) => h.mode ?? d }) },
    commands: {
      registerCommand: (id: string, fn: () => Promise<void>) => {
        h.commands.set(id, fn);
        return { dispose() {} };
      },
    },
    window: {
      showInformationMessage: async (m: string, ...buttons: string[]) => {
        h.info.push(m);
        if (buttons.length > 0) h.prompts++;
        return buttons.length > 0 ? h.choice : undefined;
      },
      showErrorMessage: async (m: string) => void h.errors.push(m),
    },
  };
});

import {
  antigravityConfigPath,
  connectAntigravity,
  detectHost,
  hasStaleEntry,
  McpConnect,
  stdioEntry,
} from './mcpConnect';

let home: string;
const ENTRY = stdioEntry('/ext/dist/mcpProxy.js', 'http://127.0.0.1:63988/mcp', '');
const config = () => path.join(home, '.gemini', 'config', 'mcp_config.json');
const put = (p: string, text: string) => {
  mkdirSync(path.dirname(p), { recursive: true });
  writeFileSync(p, text);
};
const read = (p = config()) =>
  JSON.parse(readFileSync(p, 'utf8')) as { mcpServers: Record<string, unknown> };

beforeEach(() => {
  home = mkdtempSync(path.join(tmpdir(), 'nexus-mcp-'));
  Object.assign(h, {
    appName: 'Antigravity',
    mode: 'ask',
    api: true,
    defs: [],
    provider: undefined,
    info: [],
    errors: [],
    choice: undefined,
    prompts: 0,
  });
  h.commands.clear();
});
afterEach(() => rmSync(home, { recursive: true, force: true }));

describe('helpers', () => {
  it('tells the editor from its name', () => {
    expect(detectHost('Antigravity IDE')).toBe('antigravity');
    expect(detectHost('Visual Studio Code - Insiders')).toBe('vscode');
    expect(detectHost('Cursor')).toBe('other');
  });

  it('runs the bundled bridge with the extension host node and passes the token only when there is one', () => {
    expect(ENTRY).toEqual({
      command: process.execPath,
      args: ['/ext/dist/mcpProxy.js'],
      env: { ELECTRON_RUN_AS_NODE: '1', NEXUS_MCP_URL: 'http://127.0.0.1:63988/mcp' },
    });
    expect(stdioEntry('/p.js', 'u', 't').env['NEXUS_MCP_TOKEN']).toBe('t');
  });

  it('finds the Antigravity config, preferring ~/.gemini/antigravity, or nothing', () => {
    expect(antigravityConfigPath(home)).toBeUndefined();
    mkdirSync(path.join(home, '.gemini', 'config'), { recursive: true });
    expect(antigravityConfigPath(home)).toBe(config());
    mkdirSync(path.join(home, '.gemini', 'antigravity'), { recursive: true });
    expect(antigravityConfigPath(home)).toBe(
      path.join(home, '.gemini', 'antigravity', 'mcp_config.json'),
    );
  });
});

describe('connectAntigravity', () => {
  it('reports without writing, then connects an empty config and keeps what was there', () => {
    expect(connectAntigravity(home, ENTRY, true)).toBe('not-installed');
    put(config(), '');
    expect(connectAntigravity(home, ENTRY, false)).toBe('absent');
    expect(readFileSync(config(), 'utf8')).toBe('');
    expect(connectAntigravity(home, ENTRY, true)).toBe('connected');
    expect(read().mcpServers['nexus']).toEqual(ENTRY);

    put(config(), JSON.stringify({ keep: true, mcpServers: { other: { command: 'x' } } }));
    expect(connectAntigravity(home, ENTRY, true)).toBe('connected');
    const doc = read() as { keep: boolean; mcpServers: Record<string, unknown> };
    expect(doc.keep).toBe(true);
    expect(doc.mcpServers['other']).toEqual({ command: 'x' });
    expect(JSON.parse(readFileSync(`${config()}.nexus-backup`, 'utf8')).keep).toBe(true);
    expect(connectAntigravity(home, ENTRY, true)).toBe('already');
  });

  it('refreshes an entry an older extension version wrote', () => {
    put(
      config(),
      JSON.stringify({
        mcpServers: { nexus: { command: 'node', args: ['/old/ext/dist/mcpProxy.js'] } },
      }),
    );
    expect(hasStaleEntry(home, ENTRY)).toBe(true);
    expect(connectAntigravity(home, ENTRY, true)).toBe('updated');
    expect(hasStaleEntry(home, ENTRY)).toBe(false);
  });

  it('writes through a symlink and creates a missing file in an existing folder', () => {
    put(config(), '');
    const link = path.join(home, '.gemini', 'antigravity', 'mcp_config.json');
    mkdirSync(path.dirname(link), { recursive: true });
    symlinkSync(config(), link);
    expect(connectAntigravity(home, ENTRY, true)).toBe('connected');
    expect(lstatSync(link).isSymbolicLink()).toBe(true);
    expect(read().mcpServers['nexus']).toEqual(ENTRY);

    const fresh = mkdtempSync(path.join(tmpdir(), 'nexus-mcp2-'));
    mkdirSync(path.join(fresh, '.gemini', 'antigravity'), { recursive: true });
    expect(connectAntigravity(fresh, ENTRY, true)).toBe('connected');
    expect(existsSync(path.join(fresh, '.gemini', 'antigravity', 'mcp_config.json'))).toBe(true);
    expect(hasStaleEntry(fresh, ENTRY)).toBe(false);
    rmSync(fresh, { recursive: true, force: true });
  });

  it('never touches a config that is not a JSON object', () => {
    for (const bad of ['{nope', '[1]', 'null', '"x"']) {
      put(config(), bad);
      expect(connectAntigravity(home, ENTRY, true)).toBe('invalid');
      expect(readFileSync(config(), 'utf8')).toBe(bad);
      expect(hasStaleEntry(home, ENTRY)).toBe(false);
    }
  });

  it('has no stale entry where there is no config at all', () => {
    expect(hasStaleEntry(home, ENTRY)).toBe(false);
    mkdirSync(path.join(home, '.gemini', 'config'), { recursive: true });
    expect(hasStaleEntry(home, ENTRY)).toBe(false);
  });
});

function connector(): McpConnect {
  const ctx = {
    subscriptions: [] as unknown[],
    asAbsolutePath: (p: string) => path.join('/ext', p),
    globalState: {
      store: new Map<string, unknown>(),
      get(k: string) {
        return this.store.get(k);
      },
      async update(k: string, v: unknown) {
        this.store.set(k, v);
      },
    },
  };
  const c = new McpConnect(
    ctx as never,
    () => ({ url: 'http://127.0.0.1:63988/mcp', token: '' }),
    () => home,
  );
  (c as unknown as { ctx: typeof ctx }).ctx = ctx;
  return c;
}
const flush = () => new Promise((r) => setTimeout(r, 0));

describe('McpConnect in VS Code', () => {
  it('offers the daemon through VS Code MCP support, with the token when set', async () => {
    h.appName = 'Visual Studio Code';
    const c = new McpConnect(
      { subscriptions: [], asAbsolutePath: (p: string) => p } as never,
      () => ({ url: 'http://h:1/mcp', token: 'tok' }),
      () => home,
    );
    c.register();
    const [def] = h.provider!.provideMcpServerDefinitions() as Array<{
      label: string;
      uri: { value: string };
      headers: Record<string, string>;
    }>;
    expect(def).toMatchObject({
      label: 'Nexus Orchestrator',
      uri: { value: 'http://h:1/mcp' },
      headers: { Authorization: 'Bearer tok' },
    });
    h.provider = undefined;
    const c2 = new McpConnect(
      { subscriptions: [], asAbsolutePath: (p: string) => p } as never,
      () => ({ url: 'http://h:1/mcp', token: '' }),
      () => home,
    );
    c2.register();
    expect((h.provider!.provideMcpServerDefinitions()[0] as { headers: object }).headers).toEqual(
      {},
    );
  });

  it('does nothing on a VS Code without MCP support, and the command explains', async () => {
    h.appName = 'Visual Studio Code';
    h.api = false;
    const c = connector();
    c.register();
    expect(h.provider).toBeUndefined();
    await h.commands.get('nexus.connectAgent')!();
    expect(h.info.at(-1)).toContain('1.101');
  });
});

describe('McpConnect in Antigravity', () => {
  it('asks once; Connect writes the config', async () => {
    put(config(), '');
    h.choice = 'Connect';
    connector().register();
    await flush();
    expect(h.prompts).toBe(1);
    expect((read().mcpServers['nexus'] as { args: string[] }).args[0]).toBe(
      path.join('/ext', 'dist', 'mcpProxy.js'),
    );
    expect(h.info.at(-1)).toContain('Reload the window');
  });

  it('remembers "Don\'t ask again", and Not now asks again next time', async () => {
    put(config(), '');
    h.choice = "Don't ask again";
    const c = connector();
    c.register();
    await flush();
    c.register();
    await flush();
    expect(h.prompts).toBe(1);
    expect(readFileSync(config(), 'utf8')).toBe('');

    h.prompts = 0;
    h.choice = 'Not now';
    const c2 = connector();
    c2.register();
    await flush();
    c2.register();
    await flush();
    expect(h.prompts).toBe(2);
    h.choice = undefined; // dismissed with Escape
    connector().register();
    await flush();
    expect(readFileSync(config(), 'utf8')).toBe('');
  });

  it('connects without asking when the setting says always, and not at all when it says never', async () => {
    put(config(), '');
    h.mode = 'never';
    connector().register();
    await flush();
    expect(readFileSync(config(), 'utf8')).toBe('');
    h.mode = 'always';
    connector().register();
    await flush();
    expect(h.prompts).toBe(0);
    expect(read().mcpServers['nexus']).toBeDefined();
    h.mode = 'surprise'; // an unknown value is read as ask
    put(config(), '');
    h.choice = 'Not now';
    connector().register();
    await flush();
    expect(h.prompts).toBe(1);
  });

  it('refreshes an old entry silently and leaves a current, missing or broken config alone', async () => {
    put(config(), JSON.stringify({ mcpServers: { nexus: { command: 'node', args: ['/old'] } } }));
    connector().register();
    await flush();
    expect(h.prompts).toBe(0);
    expect((read().mcpServers['nexus'] as { args: string[] }).args[0]).toContain('mcpProxy.js');

    const before = readFileSync(config(), 'utf8');
    connector().register();
    await flush();
    expect(readFileSync(config(), 'utf8')).toBe(before);

    put(config(), '{broken');
    connector().register();
    await flush();
    expect(readFileSync(config(), 'utf8')).toBe('{broken');
    rmSync(path.join(home, '.gemini'), { recursive: true, force: true });
    connector().register();
    await flush();
    expect(h.prompts).toBe(0);
  });
});

describe('Nexus: Connect This Editor’s Agent', () => {
  const run = async () => {
    connector().register();
    await flush();
    h.info.length = 0;
    await h.commands.get('nexus.connectAgent')!();
  };

  it('says what happened in Antigravity', async () => {
    h.mode = 'never';
    put(config(), '');
    await run();
    expect(h.info.at(-1)).toContain('added');
    await run();
    expect(h.info.at(-1)).toContain('already connected');
    put(config(), JSON.stringify({ mcpServers: { nexus: { command: 'old' } } }));
    await run();
    expect(h.info.at(-1)).toContain('refreshed');
    put(config(), '{bad');
    await run();
    expect(h.errors.at(-1)).toContain('not valid JSON');
    rmSync(path.join(home, '.gemini'), { recursive: true, force: true });
    await run();
    expect(h.errors.at(-1)).toContain('No Antigravity');
  });

  it('points other editors to the command line', async () => {
    h.appName = 'Cursor';
    await run();
    expect(h.info.at(-1)).toContain('nexus-cli connect --write');
  });

  it('explains VS Code', async () => {
    h.appName = 'Visual Studio Code';
    await run();
    expect(h.info.at(-1)).toContain('MCP: List Servers');
  });
});
