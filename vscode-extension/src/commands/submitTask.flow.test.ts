import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient, Provider, Task } from '../nexusClient';

const ui = vi.hoisted(() => ({
  showInputBox: vi.fn(),
  showQuickPick: vi.fn(),
  showInformationMessage: vi.fn(),
  showErrorMessage: vi.fn(),
  showWarningMessage: vi.fn(),
  showTextDocument: vi.fn(),
  executeCommand: vi.fn(),
  withProgress: vi.fn(),
  config: { defaultProvider: '', defaultModel: '' } as Record<string, string | undefined>,
  state: {
    activeEditor: undefined as { document: { uri: { fsPath: string } } } | undefined,
    folders: [{ uri: { fsPath: '/workspace' } }] as Array<{ uri: { fsPath: string } }> | undefined,
    docs: [] as Array<{
      isDirty: boolean;
      isUntitled: boolean;
      uri: { scheme: string; fsPath: string };
    }>,
  },
  log: vi.fn(),
  remember: vi.fn(),
  showLog: vi.fn(),
}));

vi.mock('vscode', () => ({
  ProgressLocation: { Notification: 15 },
  Uri: { file: (p: string) => ({ fsPath: p }) },
  window: {
    get activeTextEditor() {
      return ui.state.activeEditor;
    },
    showInputBox: ui.showInputBox,
    showQuickPick: ui.showQuickPick,
    showInformationMessage: ui.showInformationMessage,
    showErrorMessage: ui.showErrorMessage,
    showWarningMessage: ui.showWarningMessage,
    showTextDocument: ui.showTextDocument,
    withProgress: ui.withProgress,
  },
  workspace: {
    get workspaceFolders() {
      return ui.state.folders;
    },
    get textDocuments() {
      return ui.state.docs;
    },
    getConfiguration: () => ({ get: (k: string) => ui.config[k] }),
  },
  commands: { executeCommand: ui.executeCommand },
}));
vi.mock('../activityLog', () => ({
  logNexusActivity: ui.log,
  rememberQueuedTask: ui.remember,
  showNexusActivityLog: ui.showLog,
}));

import { submitTaskCommand, sendCurrentContextCommand } from './submitTask';

const task = (o: Partial<Task> = {}): Task => ({
  id: 'abcd-1234-ef56',
  projectPath: '/workspace',
  targetFile: 'src/a.ts',
  instruction: 'do it',
  contextFiles: [],
  status: 'COMPLETED',
  createdAt: '',
  updatedAt: '',
  ...o,
});

interface FakeClient {
  getProviders: ReturnType<typeof vi.fn>;
  submitTask: ReturnType<typeof vi.fn>;
  getTask: ReturnType<typeof vi.fn>;
  cancelTask: ReturnType<typeof vi.fn>;
}
let client: FakeClient;
let cancelHandlers: Array<() => void | Promise<void>>;
const progress = { report: vi.fn() };

const asClient = () => client as unknown as NexusClient;
const URL = 'http://127.0.0.1:63987';

/** Drives a full flow with timers; resolves when the command finishes. */
async function run(fn: typeof submitTaskCommand = submitTaskCommand): Promise<void> {
  const p = fn(asClient(), URL);
  await vi.runAllTimersAsync();
  await p;
}

beforeEach(() => {
  vi.useFakeTimers();
  Object.values(ui).forEach((v) => typeof v === 'function' && 'mockReset' in v && v.mockReset());
  progress.report.mockReset();
  ui.config = { defaultProvider: '', defaultModel: '' };
  ui.state.activeEditor = { document: { uri: { fsPath: '/workspace/src/a.ts' } } };
  ui.state.folders = [{ uri: { fsPath: '/workspace' } }];
  ui.state.docs = [];
  cancelHandlers = [];
  client = {
    getProviders: vi.fn().mockResolvedValue([]),
    submitTask: vi.fn().mockResolvedValue(task({ status: 'QUEUED' })),
    getTask: vi.fn().mockResolvedValue(task()),
    cancelTask: vi.fn().mockResolvedValue(undefined),
  };
  ui.showInputBox.mockResolvedValue('Refactor it');
  ui.showQuickPick.mockImplementation(async (items: unknown[]) => items[0]);
  ui.showInformationMessage.mockResolvedValue('Queue Task');
  ui.withProgress.mockImplementation(
    async (_o: unknown, work: (p: unknown, t: unknown) => Promise<void>) =>
      work(progress, { onCancellationRequested: (h: () => void) => cancelHandlers.push(h) }),
  );
});
afterEach(() => vi.useRealTimers());

describe('submitTask — prompts', () => {
  it('queues a manual task from the active editor with no context picker', async () => {
    await run();
    expect(client.submitTask).toHaveBeenCalledWith({
      instruction: 'Refactor it',
      projectPath: '/workspace',
      targetFile: 'src/a.ts',
      providerHint: undefined,
      modelId: undefined,
      contextFiles: [],
    });
    expect(ui.remember).toHaveBeenCalledWith('abcd-1234-ef56');
  });

  it('validates the instruction input', async () => {
    await run();
    const { validateInput } = ui.showInputBox.mock.calls[0][0];
    expect(validateInput('   ')).toBe('An instruction is required');
    expect(validateInput('ok')).toBeUndefined();
  });

  it('asks for a target file when there is no editor, or it is outside the workspace', async () => {
    ui.state.activeEditor = undefined;
    ui.showInputBox.mockResolvedValueOnce('Do X').mockResolvedValueOnce('lib/x.ts');
    await run();
    expect(client.submitTask).toHaveBeenCalledWith(
      expect.objectContaining({ targetFile: 'lib/x.ts' }),
    );

    client.submitTask.mockClear();
    // a sibling directory that merely shares the prefix must NOT count as inside the workspace
    ui.state.activeEditor = { document: { uri: { fsPath: '/workspace-other/secret.ts' } } };
    ui.showInputBox.mockReset().mockResolvedValueOnce('Do Y').mockResolvedValueOnce('given.ts');
    await run();
    expect(client.submitTask).toHaveBeenCalledWith(
      expect.objectContaining({ targetFile: 'given.ts' }),
    );
  });

  it.each([
    ['instruction', () => ui.showInputBox.mockResolvedValueOnce(undefined)],
    [
      'target file',
      () => {
        ui.state.activeEditor = undefined;
        ui.showInputBox.mockResolvedValueOnce('Do X').mockResolvedValueOnce(undefined);
      },
    ],
    ['route', () => ui.showQuickPick.mockResolvedValueOnce(undefined)],
    ['confirmation', () => ui.showInformationMessage.mockResolvedValueOnce(undefined)],
  ])('stops quietly when the %s prompt is dismissed', async (_name, dismiss) => {
    dismiss();
    await run();
    expect(client.submitTask).not.toHaveBeenCalled();
    expect(ui.showErrorMessage).not.toHaveBeenCalled();
  });

  it('refuses to run without an open folder', async () => {
    ui.state.folders = undefined;
    await run();
    expect(ui.showErrorMessage).toHaveBeenCalledWith('Open a folder to submit a task');
    ui.state.folders = [];
    await run();
    expect(client.submitTask).not.toHaveBeenCalled();
  });

  it('shows a dash for the target when the task has no target file', async () => {
    ui.state.activeEditor = undefined;
    ui.showInputBox.mockResolvedValueOnce('Plan only').mockResolvedValueOnce('');
    await run();
    expect(ui.showInformationMessage.mock.calls[0][0]).toContain('Target: —;');
    expect(client.submitTask).toHaveBeenCalledWith(expect.objectContaining({ targetFile: '' }));
  });

  it('asks for confirmation with the target, context count and route', async () => {
    await run();
    expect(ui.showInformationMessage.mock.calls[0][0]).toContain(
      'Target: src/a.ts; Context files: 0; Route: Auto',
    );
    expect(ui.showInformationMessage.mock.calls[0][1]).toEqual({ modal: true });
  });
});

describe('submitTask — context picker', () => {
  it('offers active file, modified files and open editors, deduplicated', async () => {
    ui.state.docs = [
      { isDirty: true, isUntitled: false, uri: { scheme: 'file', fsPath: '/workspace/src/a.ts' } }, // active
      { isDirty: true, isUntitled: false, uri: { scheme: 'file', fsPath: '/workspace/src/b.ts' } },
      { isDirty: false, isUntitled: false, uri: { scheme: 'file', fsPath: '/workspace/src/c.ts' } },
      {
        isDirty: false,
        isUntitled: false,
        uri: { scheme: 'file', fsPath: '/workspace-other/d.ts' },
      }, // outside
      { isDirty: false, isUntitled: true, uri: { scheme: 'untitled', fsPath: 'Untitled-1' } },
      { isDirty: true, isUntitled: false, uri: { scheme: 'git', fsPath: '/workspace/src/e.ts' } },
    ];
    ui.showQuickPick.mockImplementationOnce(
      async (items: Array<{ contextFiles: string[] }>) => items,
    );
    await run(sendCurrentContextCommand);
    const items = ui.showQuickPick.mock.calls[0][0] as Array<{
      label: string;
      description: string;
      picked?: boolean;
    }>;
    expect(items.map((i) => i.label)).toEqual([
      'Active file',
      'Modified workspace files (1)',
      'Open editors (1)',
    ]);
    expect(items[0]).toMatchObject({ description: 'src/a.ts', picked: true });
    expect(items[1].description).toBe('src/b.ts');
    expect(items[2].description).toBe('src/c.ts');
    expect(client.submitTask).toHaveBeenCalledWith(
      expect.objectContaining({ contextFiles: ['src/a.ts', 'src/b.ts', 'src/c.ts'] }),
    );
  });

  it('skips the picker when there is nothing to offer', async () => {
    ui.state.activeEditor = undefined;
    ui.showInputBox.mockResolvedValueOnce('Do X').mockResolvedValueOnce('t.ts');
    await run(sendCurrentContextCommand);
    expect(ui.showQuickPick).toHaveBeenCalledTimes(1); // only the route
    expect(client.submitTask).toHaveBeenCalledWith(expect.objectContaining({ contextFiles: [] }));
  });

  it('stops when the context picker is dismissed', async () => {
    ui.showQuickPick.mockResolvedValueOnce(undefined);
    await run(sendCurrentContextCommand);
    expect(client.submitTask).not.toHaveBeenCalled();
  });
});

describe('submitTask — route', () => {
  const labels = () =>
    (ui.showQuickPick.mock.calls[0][0] as Array<{ label: string }>).map((i) => i.label);

  it('offers a single Auto entry when no default is configured', async () => {
    await run();
    expect(labels()).toEqual(['Auto (let Nexus choose)']);
  });

  it('puts the workspace default first, then Auto, then each provider and model', async () => {
    ui.config = { defaultProvider: ' Ollama ', defaultModel: 'llama3' };
    client.getProviders.mockResolvedValue([
      { name: 'Ollama', active: true, models: ['llama3', 'qwen'] },
      { name: 'LM Studio', active: true },
    ] as Provider[]);
    await run();
    expect(labels()).toEqual([
      'Workspace default (Ollama / llama3)',
      'Auto (let Nexus choose)',
      'Ollama / llama3',
      'Ollama / qwen',
      'LM Studio',
    ]);
  });

  it('labels a model-only or provider-only default', async () => {
    ui.config = { defaultProvider: '', defaultModel: 'qwen' };
    await run();
    expect(labels()[0]).toBe('Workspace default (Auto / qwen)');
    ui.showQuickPick.mockClear();
    ui.config = { defaultProvider: 'Ollama', defaultModel: undefined };
    await run();
    expect(labels()[0]).toBe('Workspace default (Ollama)');
  });

  it('sends the chosen provider and model', async () => {
    client.getProviders.mockResolvedValue([
      { name: 'Ollama', active: true, models: ['qwen'] },
    ] as Provider[]);
    ui.showQuickPick.mockImplementationOnce(async (items: Array<{ label: string }>) =>
      items.find((i) => i.label === 'Ollama / qwen'),
    );
    await run();
    expect(client.submitTask).toHaveBeenCalledWith(
      expect.objectContaining({ providerHint: 'Ollama', modelId: 'qwen' }),
    );
  });

  it('still works when the provider list cannot be fetched', async () => {
    client.getProviders.mockRejectedValue(new Error('down'));
    await run();
    expect(labels()).toEqual(['Auto (let Nexus choose)']);
    expect(client.submitTask).toHaveBeenCalled();
  });
});

describe('submitTask — submission errors', () => {
  it.each([
    new TypeError('x'),
    new Error('connect ECONNREFUSED 127.0.0.1:63987'),
    new Error('fetch failed'),
    new Error('Failed to fetch'),
    new Error('nexus: no response from http://x within 15s'),
  ])('says the daemon is down for %s', async (err) => {
    client.submitTask.mockRejectedValue(err);
    await run();
    expect(ui.showErrorMessage.mock.calls[0][0]).toContain(`not running at ${URL}`);
    expect(client.getTask).not.toHaveBeenCalled();
  });

  it('shows the daemon’s own message for other failures, stringifying non-Errors', async () => {
    client.submitTask.mockRejectedValueOnce(
      new Error('nexus: POST /api/tasks: HTTP 429 — task queue is full'),
    );
    await run();
    expect(ui.showErrorMessage).toHaveBeenLastCalledWith(
      'Nexus: failed to submit task — nexus: POST /api/tasks: HTTP 429 — task queue is full',
    );
    client.submitTask.mockRejectedValueOnce('plain');
    await run();
    expect(ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus: failed to submit task — plain');
  });
});

describe('submitTask — progress and outcome', () => {
  it('reports progress, logs only status changes, and keeps polling until terminal', async () => {
    client.getTask
      .mockResolvedValueOnce(task({ status: 'QUEUED' }))
      .mockResolvedValueOnce(task({ status: 'QUEUED' }))
      .mockResolvedValueOnce(task({ status: 'PROCESSING' }))
      .mockResolvedValueOnce(task({ status: 'COMPLETED' }));
    await run();
    expect(progress.report.mock.calls.map((c) => c[0].message)).toEqual([
      'QUEUED',
      'QUEUED',
      'PROCESSING',
      'COMPLETED',
    ]);
    const transitions = ui.log.mock.calls.map((c) => c[1]).filter((m: string) => m.includes('->'));
    expect(transitions).toHaveLength(3); // QUEUED, PROCESSING, COMPLETED
  });

  it('survives a transient polling error', async () => {
    client.getTask.mockRejectedValueOnce(new Error('blip')).mockResolvedValueOnce(task());
    await run();
    expect(ui.showInformationMessage).toHaveBeenLastCalledWith(
      expect.stringContaining('completed'),
      'Open Queue',
      'Open File',
      'Show Activity Log',
    );
  });

  it('says "completed", not "queued", for a finished task', async () => {
    await run();
    expect(ui.showInformationMessage.mock.calls[1][0]).toBe('Nexus: task #abcd1234 completed');
  });

  it.each([
    ['Open Queue', () => expect(ui.executeCommand).toHaveBeenCalledWith('nexus.viewQueue')],
    ['Show Activity Log', () => expect(ui.showLog).toHaveBeenCalled()],
    [
      'Open File',
      () => expect(ui.showTextDocument).toHaveBeenCalledWith({ fsPath: '/workspace/src/a.ts' }),
    ],
  ])('handles the "%s" action', async (action, check) => {
    ui.showInformationMessage.mockResolvedValueOnce('Queue Task').mockResolvedValueOnce(action);
    await run();
    check();
  });

  it('opens an absolute target file as is', async () => {
    client.getTask.mockResolvedValue(task({ targetFile: '/elsewhere/x.ts' }));
    ui.showInformationMessage
      .mockResolvedValueOnce('Queue Task')
      .mockResolvedValueOnce('Open File');
    await run();
    expect(ui.showTextDocument).toHaveBeenCalledWith({ fsPath: '/elsewhere/x.ts' });
  });

  it('does nothing further when the completion toast is dismissed', async () => {
    ui.showInformationMessage.mockResolvedValueOnce('Queue Task').mockResolvedValueOnce(undefined);
    await run();
    expect(ui.executeCommand).not.toHaveBeenCalled();
    expect(ui.showTextDocument).not.toHaveBeenCalled();
  });

  it('reports a failed task with its logs, or a fallback', async () => {
    client.getTask.mockResolvedValueOnce(task({ status: 'FAILED', logs: 'compile error' }));
    await run();
    expect(ui.showErrorMessage).toHaveBeenLastCalledWith('Nexus: task failed — compile error');
    client.getTask.mockResolvedValueOnce(task({ status: 'FAILED', logs: undefined }));
    await run();
    expect(ui.showErrorMessage).toHaveBeenLastCalledWith(
      'Nexus: task failed — no details available',
    );
  });

  it('explains TOO_LARGE, NO_PROVIDER and an external cancel', async () => {
    client.getTask.mockResolvedValueOnce(task({ status: 'TOO_LARGE' }));
    await run();
    expect(ui.showErrorMessage).toHaveBeenLastCalledWith(expect.stringContaining('too large'));
    client.getTask.mockResolvedValueOnce(task({ status: 'NO_PROVIDER' }));
    await run();
    expect(ui.showErrorMessage).toHaveBeenLastCalledWith(
      expect.stringContaining('no LLM provider'),
    );
    client.getTask.mockResolvedValueOnce(task({ status: 'CANCELLED' }));
    await run();
    expect(ui.showInformationMessage).toHaveBeenLastCalledWith('Nexus: Task cancelled');
  });

  it('gives up after five minutes', async () => {
    client.getTask.mockResolvedValue(task({ status: 'PROCESSING' }));
    await run();
    expect(ui.showWarningMessage).toHaveBeenCalledWith(
      'Nexus: task polling timed out after 5 minutes',
    );
  });

  it('cancels the daemon task when the user cancels the progress notification', async () => {
    client.getTask.mockImplementation(async () => {
      await cancelHandlers[0]();
      return task({ status: 'PROCESSING' });
    });
    await run();
    expect(client.cancelTask).toHaveBeenCalledWith('abcd-1234-ef56');
    expect(ui.showInformationMessage).toHaveBeenLastCalledWith('Nexus: Task cancelled');
  });

  it('notices a cancel that arrives while it is sleeping', async () => {
    ui.withProgress.mockImplementation(
      async (_o: unknown, work: (p: unknown, t: unknown) => Promise<void>) => {
        const p = work(progress, {
          onCancellationRequested: (h: () => void) => cancelHandlers.push(h),
        });
        await cancelHandlers[0](); // the loop is now inside its first sleep
        return p;
      },
    );
    await run();
    expect(client.getTask).not.toHaveBeenCalled();
    expect(ui.showInformationMessage).toHaveBeenLastCalledWith('Nexus: Task cancelled');
  });

  it('ignores a failure of the best-effort cancel', async () => {
    client.cancelTask.mockRejectedValue(new Error('gone'));
    client.getTask.mockImplementation(async () => {
      await cancelHandlers[0]();
      return task({ status: 'PROCESSING' });
    });
    await expect(run()).resolves.toBeUndefined();
  });
});
