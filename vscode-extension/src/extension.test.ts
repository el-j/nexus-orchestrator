import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const h = vi.hoisted(() => {
  const commands = new Map<string, (...a: unknown[]) => unknown>();
  return {
    commands,
    treeProviders: new Map<string, unknown>(),
    configListener: undefined as
      undefined | ((e: { affectsConfiguration(k: string): boolean }) => void),
    config: {} as Record<string, unknown>,
    folders: [{ uri: { fsPath: '/workspace' } }] as Array<{ uri: { fsPath: string } }> | undefined,
    ui: {
      showQuickPick: vi.fn(),
      showInformationMessage: vi.fn(),
      showErrorMessage: vi.fn(),
      showInputBox: vi.fn(),
      showOpenDialog: vi.fn(),
      executeCommand: vi.fn(),
    },
    instances: {
      client: undefined as unknown,
      statusBar: undefined as unknown,
      monitor: undefined as unknown,
      queue: undefined as unknown,
      orch: undefined as unknown,
      sessions: undefined as unknown,
      detector: undefined as unknown,
      scanner: undefined as unknown,
    },
    fns: {
      sendCurrentContext: vi.fn(),
      submitTask: vi.fn(),
      selectProvider: vi.fn(),
      viewQueue: vi.fn(),
      showProviders: vi.fn(),
      delegate: vi.fn(),
      showLog: vi.fn(),
    },
    env: process.env['NEXUS_API_TOKEN'],
    failStatusUpdate: false,
  };
});

vi.mock('vscode', () => ({
  Uri: { file: (p: string) => ({ fsPath: p }) },
  window: {
    registerTreeDataProvider: (id: string, p: unknown) => (
      h.treeProviders.set(id, p),
      { dispose: vi.fn() }
    ),
    showQuickPick: h.ui.showQuickPick,
    showInformationMessage: h.ui.showInformationMessage,
    showErrorMessage: h.ui.showErrorMessage,
    showInputBox: h.ui.showInputBox,
    showOpenDialog: h.ui.showOpenDialog,
  },
  commands: {
    registerCommand: (id: string, fn: (...a: unknown[]) => unknown) => (
      h.commands.set(id, fn),
      { dispose: vi.fn() }
    ),
    executeCommand: h.ui.executeCommand,
  },
  workspace: {
    get workspaceFolders() {
      return h.folders;
    },
    getConfiguration: () => ({ get: <T>(k: string) => h.config[k] as T | undefined }),
    onDidChangeConfiguration: (l: typeof h.configListener) => (
      (h.configListener = l),
      { dispose: vi.fn() }
    ),
  },
}));

vi.mock('./nexusClient', () => {
  class NexusClient {
    configure = vi.fn();
    getBaseUrl = vi.fn(() => 'http://127.0.0.1:63987');
    health = vi.fn().mockResolvedValue(true);
    tryConnect = vi.fn().mockRejectedValue(new Error('none'));
    getTasks = vi.fn().mockResolvedValue([]);
    cancelTask = vi.fn().mockResolvedValue(undefined);
    listAISessions = vi.fn().mockResolvedValue([]);
    ingestKnowledge = vi.fn().mockResolvedValue(3);
    getBrainStatus = vi.fn().mockResolvedValue({ entryCount: 5, totalTokens: 99 });
    initProject = vi.fn().mockResolvedValue({ entryCount: 7 });
    searchKnowledge = vi.fn().mockResolvedValue([]);
    constructor(
      public url: string,
      public port: number,
      public token?: string,
    ) {
      h.instances.client = this;
    }
  }
  return { NexusClient };
});
vi.mock('./statusBar', () => ({
  NexusStatusBar: class {
    update = vi.fn(() =>
      h.failStatusUpdate ? Promise.reject(new Error('status bar failed')) : Promise.resolve(),
    );
    dispose = vi.fn();
    startPolling = vi.fn(() => ({ dispose: vi.fn() }));
    constructor() {
      h.instances.statusBar = this;
    }
  },
}));
vi.mock('./sessionMonitor', () => ({
  SessionMonitor: class {
    start = vi.fn().mockResolvedValue(undefined);
    stop = vi.fn().mockResolvedValue(undefined);
    constructor() {
      h.instances.monitor = this;
    }
  },
}));
vi.mock('./taskQueueProvider', () => {
  class TaskItem {
    constructor(public task: { id: string }) {}
  }
  class TaskQueueProvider {
    refresh = vi.fn();
    getScopeToWorkspace = vi.fn(() => true);
    setScopeToWorkspace = vi.fn();
    startPolling = vi.fn(() => ({ dispose: vi.fn() }));
    constructor() {
      h.instances.queue = this;
    }
  }
  return { TaskItem, TaskQueueProvider };
});
vi.mock('./workspaceScanner', () => ({
  WorkspaceScanner: class {
    start = vi.fn();
    dispose = vi.fn();
    constructor() {
      h.instances.scanner = this;
    }
  },
}));
vi.mock('./workspaceOrchView', () => ({
  WorkspaceOrchViewProvider: class {
    refresh = vi.fn();
    constructor() {
      h.instances.orch = this;
    }
  },
}));
vi.mock('./aiSessionsTreeProvider', () => ({
  AISessionItem: class {},
  AISessionsTreeProvider: class {
    refresh = vi.fn();
    startPolling = vi.fn(() => ({ dispose: vi.fn() }));
    constructor() {
      h.instances.sessions = this;
    }
  },
}));
vi.mock('./agentDetector', () => ({
  AgentDetector: class {
    start = vi.fn();
    onDidChange = vi.fn();
    detectAll = vi.fn().mockResolvedValue([]);
    dispose = vi.fn();
    constructor() {
      h.instances.detector = this;
    }
  },
}));
vi.mock('./activityLog', () => ({
  getNexusActivityChannel: () => ({ dispose: vi.fn() }),
  showNexusActivityLog: h.fns.showLog,
}));
vi.mock('./commands', () => ({
  sendCurrentContextCommand: h.fns.sendCurrentContext,
  submitTaskCommand: h.fns.submitTask,
  selectProviderCommand: h.fns.selectProvider,
  viewQueueCommand: h.fns.viewQueue,
  showProvidersCommand: h.fns.showProviders,
}));
vi.mock('./commands/delegateToNexus', () => ({ delegateToNexusCommand: h.fns.delegate }));

type Ctx = { subscriptions: Array<{ dispose(): void }> };
let ctx: Ctx;
const flush = () => vi.advanceTimersByTimeAsync(0);
const call = (id: string, ...args: unknown[]) => h.commands.get(id)!(...args);
const C = <T>(k: keyof typeof h.instances) => h.instances[k] as T;
type FakeClient = {
  configure: ReturnType<typeof vi.fn>;
  health: ReturnType<typeof vi.fn>;
  tryConnect: ReturnType<typeof vi.fn>;
  getTasks: ReturnType<typeof vi.fn>;
  cancelTask: ReturnType<typeof vi.fn>;
  listAISessions: ReturnType<typeof vi.fn>;
  ingestKnowledge: ReturnType<typeof vi.fn>;
  getBrainStatus: ReturnType<typeof vi.fn>;
  initProject: ReturnType<typeof vi.fn>;
  searchKnowledge: ReturnType<typeof vi.fn>;
  url: string;
  port: number;
  token?: string;
};

async function activateExtension() {
  vi.resetModules();
  const mod = await import('./extension');
  ctx = { subscriptions: [] };
  mod.activate(ctx as unknown as import('vscode').ExtensionContext);
  await flush();
  return mod;
}

beforeEach(() => {
  vi.useFakeTimers();
  h.commands.clear();
  h.treeProviders.clear();
  h.configListener = undefined;
  h.config = {};
  h.folders = [{ uri: { fsPath: '/workspace' } }];
  Object.values(h.ui).forEach((f) => f.mockReset());
  Object.values(h.fns).forEach((f) => f.mockReset());
  h.ui.executeCommand.mockResolvedValue(undefined);
  delete process.env['NEXUS_API_TOKEN'];
});
afterEach(() => {
  vi.useRealTimers();
  if (h.env === undefined) delete process.env['NEXUS_API_TOKEN'];
  else process.env['NEXUS_API_TOKEN'] = h.env;
});

describe('activate', () => {
  it('builds the client from settings, with the token from the setting or the environment', async () => {
    h.config = { daemonUrl: 'http://example:1', mcpPort: 9, apiToken: 'from-setting' };
    await activateExtension();
    expect(C<FakeClient>('client')).toMatchObject({
      url: 'http://example:1',
      port: 9,
      token: 'from-setting',
    });

    h.config = {};
    process.env['NEXUS_API_TOKEN'] = 'from-env';
    await activateExtension();
    expect(C<FakeClient>('client')).toMatchObject({
      url: 'http://127.0.0.1:63987',
      port: 63988,
      token: 'from-env',
    });

    delete process.env['NEXUS_API_TOKEN'];
    await activateExtension();
    expect(C<FakeClient>('client').token).toBe('');
  });

  it('registers the tree views, every command, and subscribes everything it starts', async () => {
    await activateExtension();
    expect([...h.treeProviders.keys()].sort()).toEqual([
      'nexus.aiSessions',
      'nexus.taskQueue',
      'nexus.workspaceAgents',
    ]);
    for (const id of [
      'nexus.refreshWorkspaceAgents',
      'nexus.reconnect',
      'nexus.sendCurrentContext',
      'nexus.submitTask',
      'nexus.viewQueue',
      'nexus.showActivityLog',
      'nexus.cancelTask',
      'nexus.selectProvider',
      'nexus.toggleWorkspaceScope',
      'nexus.showProviders',
      'nexus.statusBarAction',
      'nexus.refreshAISessions',
      'nexus.delegateToNexus',
      'nexus.delegateAllSessions',
      'nexus.brain.ingest',
      'nexus.brain.status',
      'nexus.brain.init',
      'nexus.brain.search',
    ]) {
      expect(h.commands.has(id), id).toBe(true);
    }
    expect(ctx.subscriptions.length).toBeGreaterThan(20);
    expect(C<{ start: ReturnType<typeof vi.fn> }>('monitor').start).toHaveBeenCalled();
    expect(C<{ start: ReturnType<typeof vi.fn> }>('scanner').start).toHaveBeenCalled();
    expect(C<{ start: ReturnType<typeof vi.fn> }>('detector').start).toHaveBeenCalled();
  });

  it('getClient/getStatusBar return the shared instances', async () => {
    const mod = await activateExtension();
    expect(mod.getClient()).toBe(C('client'));
    expect(mod.getStatusBar()).toBe(C('statusBar'));
  });

  it('getClient and getStatusBar throw before activation', async () => {
    vi.resetModules();
    const mod = await import('./extension');
    expect(() => mod.getClient()).toThrow('not yet activated');
    expect(() => mod.getStatusBar()).toThrow('not yet activated');
  });

  it('refreshes the status bar once the connection is sorted out, and ignores a failure', async () => {
    await activateExtension();
    expect(C<{ update: ReturnType<typeof vi.fn> }>('statusBar').update).toHaveBeenCalled();
  });

  it('deactivate stops the session monitor', async () => {
    const mod = await activateExtension();
    await mod.deactivate();
    expect(C<{ stop: ReturnType<typeof vi.fn> }>('monitor').stop).toHaveBeenCalled();
    vi.resetModules();
    const fresh = await import('./extension');
    expect(fresh.deactivate()).toBeUndefined(); // never activated
  });
});

describe('activation', () => {
  it('does not fail when the first status-bar refresh after reconfiguring rejects', async () => {
    h.failStatusUpdate = true;
    try {
      await expect(activateExtension()).resolves.toBeDefined();
    } finally {
      h.failStatusUpdate = false;
    }
  });
});

describe('reconfigureClient', () => {
  const fake = () => {
    const c = {
      configure: vi.fn(),
      health: vi.fn().mockResolvedValue(true),
      tryConnect: vi.fn(),
    };
    return c as unknown as import('./nexusClient').NexusClient & typeof c;
  };

  it('applies the settings and stops when the daemon answers', async () => {
    h.config = { daemonUrl: 'http://h:1', mcpPort: 7, apiToken: 't' };
    const mod = await activateExtension();
    const c = fake();
    expect(await mod.reconfigureClient(c)).toBe(true);
    expect(c.configure).toHaveBeenCalledWith('http://h:1', 7, 't');
    expect(c.tryConnect).not.toHaveBeenCalled();
  });

  it('does not sweep ports when the sweep is disabled', async () => {
    h.config = { enableMCPPortSweep: false };
    const mod = await activateExtension();
    const c = fake();
    c.health.mockResolvedValue(false);
    expect(await mod.reconfigureClient(c)).toBe(false);
    expect(c.tryConnect).not.toHaveBeenCalled();
  });

  it('adopts the port of a daemon found by the sweep', async () => {
    h.config = { daemonUrl: 'http://127.0.0.1:63987', mcpPort: 63988 };
    const mod = await activateExtension();
    const c = fake();
    c.health.mockResolvedValue(false);
    c.tryConnect.mockResolvedValue(63989);
    expect(await mod.reconfigureClient(c)).toBe(true);
    expect(c.tryConnect).toHaveBeenCalledWith([63987, 63986, 63989, 63990]);
    expect(c.configure).toHaveBeenLastCalledWith('http://127.0.0.1:63989', 63988, '');
  });

  it('keeps the configured URL when the sweep finds the configured port, or nothing', async () => {
    const mod = await activateExtension();
    const c = fake();
    c.health.mockResolvedValue(false);
    c.tryConnect.mockResolvedValue(63987);
    expect(await mod.reconfigureClient(c)).toBe(true);
    expect(c.configure).toHaveBeenCalledTimes(1);

    const d = fake();
    d.health.mockResolvedValue(false);
    d.tryConnect.mockRejectedValue(new Error('none'));
    expect(await mod.reconfigureClient(d)).toBe(false);
  });

  it('probes the port from the configured URL first, and survives an invalid URL', async () => {
    h.config = { daemonUrl: 'http://127.0.0.1:7000' };
    const mod = await activateExtension();
    const c = fake();
    c.health.mockResolvedValue(false);
    c.tryConnect.mockRejectedValue(new Error('none'));
    await mod.reconfigureClient(c);
    expect(c.tryConnect).toHaveBeenCalledWith([7000, 63987, 63986, 63989, 63990]);

    h.config = { daemonUrl: 'not a url' };
    const bad = fake();
    bad.health.mockResolvedValue(false);
    expect(await mod.reconfigureClient(bad)).toBe(false);
  });

  it('assumes port 63987 for a URL without a port', async () => {
    h.config = { daemonUrl: 'http://localhost' };
    const mod = await activateExtension();
    const c = fake();
    c.health.mockResolvedValue(false);
    c.tryConnect.mockRejectedValue(new Error('none'));
    await mod.reconfigureClient(c);
    expect(c.tryConnect.mock.calls[0][0][0]).toBe(63987);
  });
});

describe('settings changes and reconnect', () => {
  const affects = (...keys: string[]) => ({
    affectsConfiguration: (k: string) => keys.includes(k),
  });

  it.each(['nexus.daemonUrl', 'nexus.mcpPort', 'nexus.apiToken', 'nexus.enableMCPPortSweep'])(
    're-points the shared client when %s changes',
    async (key) => {
      await activateExtension();
      const client = C<FakeClient>('client');
      client.configure.mockClear();
      h.configListener!(affects(key));
      await flush();
      expect(client.configure).toHaveBeenCalled();
      expect(C<{ update: ReturnType<typeof vi.fn> }>('statusBar').update).toHaveBeenCalled();
    },
  );

  it('ignores unrelated settings', async () => {
    await activateExtension();
    const client = C<FakeClient>('client');
    client.configure.mockClear();
    h.configListener!(affects('editor.fontSize'));
    await flush();
    expect(client.configure).not.toHaveBeenCalled();
  });

  it('reconnect reports success with the URL, or an error when no daemon answers', async () => {
    await activateExtension();
    await call('nexus.reconnect');
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith(
      'Nexus: connected to http://127.0.0.1:63987',
    );

    C<FakeClient>('client').health.mockResolvedValue(false);
    await call('nexus.reconnect');
    expect(h.ui.showErrorMessage).toHaveBeenCalledWith(
      expect.stringContaining('no daemon answered at http://127.0.0.1:63987'),
    );
  });

  it('reconnect reports an unexpected failure (Error and non-Error)', async () => {
    await activateExtension();
    C<FakeClient>('client').health.mockRejectedValueOnce(new Error('boom'));
    await call('nexus.reconnect');
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus: reconnect failed — boom');
    C<FakeClient>('client').health.mockRejectedValueOnce('weird');
    await call('nexus.reconnect');
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus: reconnect failed — weird');
  });
});

describe('simple commands', () => {
  it('forward to their implementations and refresh the views', async () => {
    await activateExtension();
    const client = C('client');
    await call('nexus.sendCurrentContext');
    expect(h.fns.sendCurrentContext).toHaveBeenCalledWith(client, 'http://127.0.0.1:63987');
    await call('nexus.submitTask');
    expect(h.fns.submitTask).toHaveBeenCalledWith(client, 'http://127.0.0.1:63987');
    expect(C<{ refresh: ReturnType<typeof vi.fn> }>('queue').refresh).toHaveBeenCalledTimes(2);
    call('nexus.viewQueue');
    expect(h.fns.viewQueue).toHaveBeenCalled();
    call('nexus.showActivityLog');
    expect(h.fns.showLog).toHaveBeenCalled();
    await call('nexus.selectProvider');
    expect(h.fns.selectProvider).toHaveBeenCalledWith(client);
    await call('nexus.showProviders');
    expect(h.fns.showProviders).toHaveBeenCalledWith(client);
    call('nexus.refreshWorkspaceAgents');
    expect(C<{ refresh: ReturnType<typeof vi.fn> }>('orch').refresh).toHaveBeenCalled();
  });

  it('toggles the workspace scope and says which way it went', async () => {
    await activateExtension();
    call('nexus.toggleWorkspaceScope');
    expect(
      C<{ setScopeToWorkspace: ReturnType<typeof vi.fn> }>('queue').setScopeToWorkspace,
    ).toHaveBeenCalledWith(false);
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith('Nexus task scope: all projects');
    C<{ getScopeToWorkspace: ReturnType<typeof vi.fn> }>(
      'queue',
    ).getScopeToWorkspace.mockReturnValue(false);
    call('nexus.toggleWorkspaceScope');
    expect(h.ui.showInformationMessage).toHaveBeenLastCalledWith(
      'Nexus task scope: current workspace only',
    );
  });

  it('refreshes the AI sessions tree and re-runs detection', async () => {
    await activateExtension();
    call('nexus.refreshAISessions');
    expect(C<{ refresh: ReturnType<typeof vi.fn> }>('sessions').refresh).toHaveBeenCalled();
    expect(C<{ detectAll: ReturnType<typeof vi.fn> }>('detector').detectAll).toHaveBeenCalled();
  });

  it('refreshes the sessions tree when the detector reports a change', async () => {
    await activateExtension();
    const onChange = C<{ onDidChange: ReturnType<typeof vi.fn> }>('detector').onDidChange.mock
      .calls[0][0] as () => void;
    onChange();
    expect(C<{ refresh: ReturnType<typeof vi.fn> }>('sessions').refresh).toHaveBeenCalled();
  });
});

describe('nexus.cancelTask', () => {
  it('cancels the task of a tree item directly', async () => {
    await activateExtension();
    const { TaskItem } = await import('./taskQueueProvider');
    await call('nexus.cancelTask', new TaskItem({ id: 'abcdef1234' } as never));
    expect(C<FakeClient>('client').cancelTask).toHaveBeenCalledWith('abcdef1234');
    expect(C<{ refresh: ReturnType<typeof vi.fn> }>('queue').refresh).toHaveBeenCalled();
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith('Nexus: Task abcdef12… cancelled');
  });

  it('reports a failed cancel', async () => {
    await activateExtension();
    C<FakeClient>('client')
      .cancelTask.mockRejectedValueOnce(new Error('too late'))
      .mockRejectedValueOnce('weird');
    const { TaskItem } = await import('./taskQueueProvider');
    await call('nexus.cancelTask', new TaskItem({ id: 'abcdef1234' } as never));
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith(
      'Nexus: Failed to cancel task — too late',
    );
    await call('nexus.cancelTask', new TaskItem({ id: 'abcdef1234' } as never));
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus: Failed to cancel task — weird');
  });

  it('from the palette, offers queued and processing tasks only', async () => {
    await activateExtension();
    const t = (id: string, status: string, instruction = 'do') => ({ id, status, instruction });
    C<FakeClient>('client').getTasks.mockResolvedValue([
      t('aaaa-bbbb-cccc', 'QUEUED'),
      t('dddd', 'PROCESSING', 'y'.repeat(60)),
      t('eeee', 'COMPLETED'),
    ]);
    h.ui.showQuickPick.mockImplementation(async (items: Array<{ taskId: string }>) => items[1]);
    await call('nexus.cancelTask');
    const offered = h.ui.showQuickPick.mock.calls[0][0] as Array<{
      label: string;
      description: string;
    }>;
    expect(offered).toHaveLength(2);
    expect(offered[0].label).toBe('#aaaabbbb — do');
    expect(offered[1].label).toBe(`#dddd — ${'y'.repeat(40)}…`);
    expect(C<FakeClient>('client').cancelTask).toHaveBeenCalledWith('dddd');
  });

  it('says so when nothing can be cancelled, when the picker is dismissed, and when the queue cannot be read', async () => {
    await activateExtension();
    await call('nexus.cancelTask');
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith('Nexus: No cancellable tasks');

    C<FakeClient>('client').getTasks.mockResolvedValue([
      { id: 'a', status: 'QUEUED', instruction: 'x' },
    ]);
    h.ui.showQuickPick.mockResolvedValue(undefined);
    await call('nexus.cancelTask');
    expect(C<FakeClient>('client').cancelTask).not.toHaveBeenCalled();

    C<FakeClient>('client')
      .getTasks.mockRejectedValueOnce(new Error('down'))
      .mockRejectedValueOnce('weird');
    await call('nexus.cancelTask');
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus: Failed to fetch tasks — down');
    await call('nexus.cancelTask');
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus: Failed to fetch tasks — weird');
  });
});

describe('nexus.statusBarAction', () => {
  it('lists the actions and runs the chosen command', async () => {
    await activateExtension();
    h.ui.showQuickPick.mockImplementation(async (items: Array<{ action: string }>) =>
      items.find((i) => i.action === 'nexus.viewQueue'),
    );
    await call('nexus.statusBarAction');
    expect((h.ui.showQuickPick.mock.calls[0][0] as unknown[]).length).toBe(10);
    expect(h.ui.executeCommand).toHaveBeenCalledWith('nexus.viewQueue');
  });

  it('refreshes the status bar itself for the refresh action, and does nothing when dismissed', async () => {
    await activateExtension();
    const update = C<{ update: ReturnType<typeof vi.fn> }>('statusBar').update;
    update.mockClear();
    h.ui.showQuickPick.mockImplementation(async (items: Array<{ action: string }>) =>
      items.find((i) => i.action === 'nexus.statusBarRefresh'),
    );
    await call('nexus.statusBarAction');
    expect(update).toHaveBeenCalled();
    expect(h.ui.executeCommand).not.toHaveBeenCalled();
    h.ui.showQuickPick.mockResolvedValue(undefined);
    await call('nexus.statusBarAction');
    expect(h.ui.executeCommand).not.toHaveBeenCalled();
  });
});

describe('delegation commands', () => {
  it('delegates the clicked item', async () => {
    await activateExtension();
    const item = {};
    call('nexus.delegateToNexus', item);
    expect(h.fns.delegate).toHaveBeenCalledWith(C('client'), item);
  });

  it('delegateAll delegates every active, not-yet-delegated session', async () => {
    await activateExtension();
    C<FakeClient>('client').listAISessions.mockResolvedValue([
      { id: 'a', status: 'active' },
      { id: 'b', status: 'idle' },
      { id: 'c', status: 'active', delegatedToNexus: true },
      { id: 'd', status: 'active' },
    ]);
    h.fns.delegate.mockResolvedValue(undefined);
    await call('nexus.delegateAllSessions');
    expect(h.fns.delegate.mock.calls.map((c) => (c[1] as { id: string }).id)).toEqual(['a', 'd']);
  });
});

describe('brain commands', () => {
  it('ingest asks for a file inside the project and reports the count', async () => {
    await activateExtension();
    h.ui.showOpenDialog.mockResolvedValue([{ fsPath: '/workspace/CLAUDE.md' }]);
    await call('nexus.brain.ingest');
    expect(C<FakeClient>('client').ingestKnowledge).toHaveBeenCalledWith(
      '/workspace',
      '/workspace/CLAUDE.md',
    );
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith(
      'Nexus Brain: Ingested 3 sections from /workspace/CLAUDE.md',
    );
    expect(C<{ refresh: ReturnType<typeof vi.fn> }>('orch').refresh).toHaveBeenCalled();
    await call('nexus.brain.ingest', '/explicit');
    expect(C<FakeClient>('client').ingestKnowledge).toHaveBeenLastCalledWith(
      '/explicit',
      '/workspace/CLAUDE.md',
    );
  });

  it('ingest needs a folder, a chosen file, and reports failures', async () => {
    await activateExtension();
    h.folders = undefined;
    await call('nexus.brain.ingest');
    expect(h.ui.showErrorMessage).toHaveBeenCalledWith('Nexus Brain: No workspace folder found.');
    h.folders = [{ uri: { fsPath: '/workspace' } }];
    h.ui.showOpenDialog.mockResolvedValueOnce(undefined).mockResolvedValueOnce([]);
    await call('nexus.brain.ingest');
    await call('nexus.brain.ingest');
    expect(C<FakeClient>('client').ingestKnowledge).not.toHaveBeenCalled();
    h.ui.showOpenDialog.mockResolvedValue([{ fsPath: '/f.md' }]);
    C<FakeClient>('client')
      .ingestKnowledge.mockRejectedValueOnce(new Error('bad'))
      .mockRejectedValueOnce('weird');
    await call('nexus.brain.ingest');
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus Brain: Ingest failed — bad');
    await call('nexus.brain.ingest');
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus Brain: Ingest failed — weird');
  });

  it('status prompts for a path (defaulting to the workspace) and shows the counts', async () => {
    await activateExtension();
    h.ui.showInputBox.mockResolvedValue('/p');
    await call('nexus.brain.status');
    expect(h.ui.showInputBox).toHaveBeenCalledWith({ prompt: 'Project path', value: '/workspace' });
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith('Brain: 5 entries, 99 tokens');
    await call('nexus.brain.status', '/given');
    expect(C<FakeClient>('client').getBrainStatus).toHaveBeenLastCalledWith('/given');
    h.folders = undefined;
    await call('nexus.brain.status');
    expect(h.ui.showInputBox).toHaveBeenLastCalledWith({ prompt: 'Project path', value: '' });
  });

  it('status does nothing without a path and reports failures', async () => {
    await activateExtension();
    h.ui.showInputBox.mockResolvedValue(undefined);
    await call('nexus.brain.status');
    expect(C<FakeClient>('client').getBrainStatus).not.toHaveBeenCalled();
    C<FakeClient>('client').getBrainStatus.mockRejectedValue(new Error('down'));
    await call('nexus.brain.status', '/p');
    expect(h.ui.showErrorMessage).toHaveBeenCalledWith('Brain status failed: down');
  });

  it('init prompts for a path, initialises and reports, or fails cleanly', async () => {
    await activateExtension();
    h.ui.showInputBox.mockResolvedValue('/p');
    await call('nexus.brain.init');
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith(
      'Brain initialized: 7 entries ingested',
    );
    await call('nexus.brain.init', '/given');
    expect(C<FakeClient>('client').initProject).toHaveBeenLastCalledWith('/given');
    h.ui.showInputBox.mockResolvedValue(undefined);
    C<FakeClient>('client').initProject.mockClear();
    await call('nexus.brain.init');
    expect(C<FakeClient>('client').initProject).not.toHaveBeenCalled();
    C<FakeClient>('client').initProject.mockRejectedValue(new Error('nope'));
    await call('nexus.brain.init', '/p');
    expect(h.ui.showErrorMessage).toHaveBeenCalledWith('Brain init failed: nope');
  });

  it('search needs a folder and a query, then lists the results', async () => {
    await activateExtension();
    h.folders = undefined;
    await call('nexus.brain.search');
    expect(h.ui.showErrorMessage).toHaveBeenCalledWith('No workspace folder open');
    h.folders = [{ uri: { fsPath: '/workspace' } }];
    h.ui.showInputBox.mockResolvedValue(undefined);
    await call('nexus.brain.search');
    expect(C<FakeClient>('client').searchKnowledge).not.toHaveBeenCalled();

    h.ui.showInputBox.mockResolvedValue('auth');
    await call('nexus.brain.search');
    expect(h.ui.showInformationMessage).toHaveBeenCalledWith('No results found');
    C<FakeClient>('client').searchKnowledge.mockResolvedValue([
      { topic: 'Auth', content: 'c'.repeat(100) },
    ]);
    await call('nexus.brain.search');
    expect(C<FakeClient>('client').searchKnowledge).toHaveBeenLastCalledWith(
      '/workspace',
      'auth',
      10,
    );
    expect(h.ui.showQuickPick).toHaveBeenCalledWith(
      [{ label: 'Auth', description: 'c'.repeat(80) }],
      { placeHolder: '1 results for "auth"' },
    );
    C<FakeClient>('client').searchKnowledge.mockRejectedValue(new Error('down'));
    await call('nexus.brain.search');
    expect(h.ui.showErrorMessage).toHaveBeenLastCalledWith('Brain search failed: down');
  });
});
