import { beforeEach, describe, expect, it, vi } from 'vitest';

// ── Shared vscode mock state ──────────────────────────────────────────────────

const registeredCommands: string[] = [];
const registeredTreeProviders: string[] = [];
const subscriptions: unknown[] = [];

const registerCommandMock = vi.fn((cmd: string, _fn: unknown) => {
  registeredCommands.push(cmd);
  return { dispose: vi.fn() };
});
const registerTreeDataProviderMock = vi.fn((id: string, _provider: unknown) => {
  registeredTreeProviders.push(id);
  return { dispose: vi.fn() };
});

vi.mock('vscode', () => ({
  workspace: {
    getConfiguration: vi.fn(() => ({
      get: vi.fn((key: string, def: unknown) => def),
    })),
    workspaceFolders: [{ uri: { fsPath: '/workspace' } }],
    onDidChangeConfiguration: vi.fn(() => ({ dispose: vi.fn() })),
  },
  window: {
    registerTreeDataProvider: registerTreeDataProviderMock,
    createStatusBarItem: vi.fn(() => ({
      text: '',
      tooltip: '',
      command: '',
      show: vi.fn(),
      hide: vi.fn(),
      dispose: vi.fn(),
    })),
    showInformationMessage: vi.fn(),
    showErrorMessage: vi.fn(),
    showQuickPick: vi.fn(),
    showInputBox: vi.fn(),
    activeTextEditor: undefined,
  },
  commands: {
    registerCommand: registerCommandMock,
    executeCommand: vi.fn(),
  },
  StatusBarAlignment: { Right: 2 },
  ThemeIcon: class {
    constructor(
      public id: string,
      public color?: unknown,
    ) {}
  },
  ThemeColor: class {
    constructor(public id: string) {}
  },
  MarkdownString: class {
    constructor(public value: string) {}
  },
  TreeItem: class {
    constructor(
      public label: string,
      public state?: number,
    ) {}
  },
  TreeItemCollapsibleState: { None: 0 },
  EventEmitter: class {
    event = vi.fn();
    fire = vi.fn();
    dispose = vi.fn();
  },
  Disposable: class {
    static from(...d: Array<{ dispose(): void }>) {
      return { dispose: () => d.forEach((x) => x.dispose()) };
    }
    constructor(private fn: () => void) {}
    dispose() {
      this.fn();
    }
  },
  env: { machineId: 'machine-1' },
  lm: {
    selectChatModels: vi.fn().mockResolvedValue([]),
    onDidChangeChatModels: vi.fn(() => ({ dispose: vi.fn() })),
  },
  Uri: { parse: vi.fn((s: string) => ({ toString: () => s })) },
}));

// ── Mock heavy dependencies ───────────────────────────────────────────────────

vi.mock('./commands', () => ({
  sendCurrentContextCommand: vi.fn(),
  submitTaskCommand: vi.fn(),
  selectProviderCommand: vi.fn(),
  viewQueueCommand: vi.fn(),
  showProvidersCommand: vi.fn(),
}));

vi.mock('./commands/delegateToNexus', () => ({
  delegateToNexusCommand: vi.fn(),
}));

vi.mock('./statusBar', () => ({
  NexusStatusBar: vi.fn().mockImplementation(function () {
    return {
      startPolling: vi.fn(() => ({ dispose: vi.fn() })),
      update: vi.fn().mockResolvedValue(undefined),
      dispose: vi.fn(),
    };
  }),
}));

vi.mock('./taskQueueProvider', () => ({
  TaskQueueProvider: vi.fn().mockImplementation(function () {
    return {
      startPolling: vi.fn(() => ({ dispose: vi.fn() })),
      refresh: vi.fn(),
      getTreeItem: vi.fn(),
      getChildren: vi.fn().mockResolvedValue([]),
      onDidChangeTreeData: vi.fn(),
      getScopeToWorkspace: vi.fn().mockReturnValue(false),
      setScopeToWorkspace: vi.fn(),
    };
  }),
  TaskItem: class {},
}));

vi.mock('./sessionMonitor', () => ({
  SessionMonitor: vi.fn().mockImplementation(function () {
    return {
      start: vi.fn().mockResolvedValue(undefined),
      dispose: vi.fn(),
    };
  }),
}));

vi.mock('./workspaceScanner', () => ({
  WorkspaceScanner: vi.fn().mockImplementation(function () {
    return {
      start: vi.fn(),
      dispose: vi.fn(),
      onDidChange: vi.fn(),
      getAgents: vi.fn().mockReturnValue([]),
    };
  }),
}));

vi.mock('./workspaceOrchView', () => ({
  WorkspaceOrchViewProvider: vi.fn().mockImplementation(function () {
    return {
      refresh: vi.fn(),
      getTreeItem: vi.fn(),
      getChildren: vi.fn().mockResolvedValue([]),
      onDidChangeTreeData: vi.fn(),
    };
  }),
}));

vi.mock('./activityLog', () => ({
  getNexusActivityChannel: vi.fn(() => ({ appendLine: vi.fn(), show: vi.fn(), dispose: vi.fn() })),
  showNexusActivityLog: vi.fn(),
  logNexusActivity: vi.fn(),
  getKnownTaskSource: vi.fn(),
}));

vi.mock('./agentDetector', () => ({
  AgentDetector: vi.fn().mockImplementation(function () {
    return {
      start: vi.fn(),
      dispose: vi.fn(),
      onDidChange: vi.fn(),
      detectAll: vi.fn().mockResolvedValue(undefined),
    };
  }),
}));

vi.mock('./aiSessionsTreeProvider', () => ({
  AISessionsTreeProvider: vi.fn().mockImplementation(function () {
    return {
      refresh: vi.fn(),
      getTreeItem: vi.fn(),
      getChildren: vi.fn().mockResolvedValue([]),
      onDidChangeTreeData: vi.fn(),
      startPolling: vi.fn(() => ({ dispose: vi.fn() })),
      dispose: vi.fn(),
    };
  }),
  AISessionItem: class {},
}));

// ── Tests ─────────────────────────────────────────────────────────────────────

function buildContext(): import('vscode').ExtensionContext {
  return {
    subscriptions,
    workspaceState: {
      get: vi.fn(),
      update: vi.fn().mockResolvedValue(undefined),
      keys: vi.fn().mockReturnValue([]),
    },
  } as unknown as import('vscode').ExtensionContext;
}

describe('extension activation', () => {
  beforeEach(() => {
    vi.resetModules();
    registeredCommands.length = 0;
    registeredTreeProviders.length = 0;
    subscriptions.length = 0;
    vi.clearAllMocks();
  });

  it('activate registers core commands', async () => {
    const { activate } = await import('./extension');
    activate(buildContext());

    const cmds = registeredCommands;
    expect(cmds).toContain('nexus.reconnect');
    expect(cmds).toContain('nexus.submitTask');
    expect(cmds).toContain('nexus.viewQueue');
    expect(cmds).toContain('nexus.cancelTask');
  });

  it('activate registers tree data providers', async () => {
    const { activate } = await import('./extension');
    activate(buildContext());

    expect(registeredTreeProviders).toContain('nexus.taskQueue');
    expect(registeredTreeProviders).toContain('nexus.workspaceAgents');
  });

  it('getClient() throws before activation', async () => {
    const { getClient } = await import('./extension');
    // Force re-import so module state is reset
    vi.resetModules();
    const ext = await import('./extension');

    // In a fresh module, client is undefined
    expect(() => ext.getClient()).toThrow('not yet activated');
  });

  it('getClient() returns a NexusClient after activation', async () => {
    const { activate, getClient } = await import('./extension');
    activate(buildContext());

    expect(() => getClient()).not.toThrow();
    expect(getClient()).toBeDefined();
  });
});
