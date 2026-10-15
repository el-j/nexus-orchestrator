import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AISession, NexusClient } from '../nexusClient';

const v = vi.hoisted(() => ({
  writeFile: vi.fn(),
  createTerminal: vi.fn(),
  executeCommand: vi.fn(),
  showInformationMessage: vi.fn(),
  showErrorMessage: vi.fn(),
  showWarningMessage: vi.fn(),
  showTextDocument: vi.fn(),
  showQuickPick: vi.fn(),
  clipboard: vi.fn(),
  folders: [{ uri: { fsPath: '/workspace' } }] as Array<{ uri: { fsPath: string } }> | undefined,
  terminal: { show: vi.fn(), sendText: vi.fn() },
}));

vi.mock('vscode', () => {
  class Uri {
    constructor(public fsPath: string) {}
    static file(p: string) {
      return new Uri(p);
    }
  }
  class TreeItem {
    constructor(public label: string) {}
  }
  return {
    Uri,
    TreeItem,
    TreeItemCollapsibleState: { None: 0 },
    ThemeIcon: class {},
    ThemeColor: class {},
    MarkdownString: class {},
    workspace: {
      get workspaceFolders() {
        return v.folders;
      },
      fs: { writeFile: v.writeFile },
    },
    window: {
      createTerminal: v.createTerminal,
      showInformationMessage: v.showInformationMessage,
      showErrorMessage: v.showErrorMessage,
      showWarningMessage: v.showWarningMessage,
      showTextDocument: v.showTextDocument,
      showQuickPick: v.showQuickPick,
    },
    commands: { executeCommand: v.executeCommand },
    env: { clipboard: { writeText: v.clipboard } },
  };
});

import { delegateToNexusCommand } from './delegateToNexus';
import { AISessionItem } from '../aiSessionsTreeProvider';

const session = (o: Partial<AISession> = {}): AISession => ({
  id: 's1',
  agentName: 'Cline',
  source: 'vscode',
  status: 'active',
  lastActivity: '',
  projectPath: '/workspace',
  ...o,
});
const fakeClient = (o: Record<string, unknown> = {}) =>
  ({
    delegateSession: vi.fn().mockResolvedValue({ instruction: 'DO THE THING', sessionId: 's1' }),
    submitTask: vi.fn().mockResolvedValue({ id: 't1' }),
    listAISessions: vi.fn().mockResolvedValue([]),
    ...o,
  }) as unknown as NexusClient;
const sentToShell = () => v.terminal.sendText.mock.calls.map((c) => c[0] as string).join('\n');

beforeEach(() => {
  Object.values(v).forEach((f) => typeof f === 'function' && 'mockReset' in f && f.mockReset());
  v.folders = [{ uri: { fsPath: '/workspace' } }];
  v.terminal = { show: vi.fn(), sendText: vi.fn() };
  v.createTerminal.mockReturnValue(v.terminal);
  v.writeFile.mockResolvedValue(undefined);
  v.executeCommand.mockResolvedValue(undefined);
  v.showInformationMessage.mockResolvedValue(undefined);
});

describe('delegate: CLI agents write a file and show it in a terminal', () => {
  it('writes the instruction and prints it with a constant command', async () => {
    await delegateToNexusCommand(fakeClient(), session());
    const [uri, data] = v.writeFile.mock.calls[0];
    expect(uri.fsPath).toBe('/workspace/.nexus-delegate.md');
    expect(Buffer.from(data).toString()).toBe('DO THE THING');
    expect(v.createTerminal).toHaveBeenCalledWith({ name: 'Nexus Delegate', cwd: '/workspace' });
    expect(sentToShell()).toBe(
      'echo "=== Nexus Delegation Instruction ==="; cat .nexus-delegate.md',
    );
    expect(v.executeCommand).toHaveBeenLastCalledWith('nexus.refreshAISessions');
  });

  it('never puts a hostile project path into shell input (regression: command injection)', async () => {
    const hostile = '/workspace/x"; touch /tmp/pwned; echo "$(id)';
    await delegateToNexusCommand(fakeClient(), session({ projectPath: hostile }));
    expect(sentToShell()).not.toContain('pwned');
    expect(sentToShell()).not.toContain('$(id)');
    expect(sentToShell()).not.toContain(hostile);
    expect(v.createTerminal.mock.calls[0][0].cwd).toBe(hostile); // structured option, not shell text
  });

  it.each(['/etc', '/workspace-other', '/workspace/../etc', '/'])(
    'will not write outside the open folders (session path %s)',
    async (requested) => {
      await delegateToNexusCommand(fakeClient(), session({ projectPath: requested }));
      expect(v.writeFile.mock.calls[0][0].fsPath).toBe('/workspace/.nexus-delegate.md');
    },
  );

  it('accepts a subfolder of the workspace and any of several folders', async () => {
    v.folders = [{ uri: { fsPath: '/a' } }, { uri: { fsPath: '/b' } }];
    await delegateToNexusCommand(fakeClient(), session({ projectPath: '/b/pkg' }));
    expect(v.writeFile.mock.calls[0][0].fsPath).toBe('/b/pkg/.nexus-delegate.md');
    await delegateToNexusCommand(fakeClient(), session({ projectPath: undefined }));
    expect(v.writeFile.mock.calls[1][0].fsPath).toBe('/a/.nexus-delegate.md');
  });

  it('refuses without an open folder', async () => {
    v.folders = undefined;
    await delegateToNexusCommand(fakeClient(), session());
    expect(v.showErrorMessage).toHaveBeenCalledWith(expect.stringContaining('open a folder'));
    expect(v.writeFile).not.toHaveBeenCalled();
  });

  it('opens the file when the toast action is chosen', async () => {
    v.showInformationMessage.mockResolvedValueOnce('Open File');
    await delegateToNexusCommand(fakeClient(), session());
    await Promise.resolve();
    expect(v.showTextDocument).toHaveBeenCalledWith(
      expect.objectContaining({ fsPath: '/workspace/.nexus-delegate.md' }),
    );
  });
});

describe('delegate: MCP agents queue a task and open chat', () => {
  const mcp = () => session({ source: 'mcp', agentName: 'Some Agent' });

  it.each([
    session({ agentName: 'Claude Desktop' }),
    session({ agentName: 'Antigravity IDE' }),
    mcp(),
  ])('routes %o to MCP handling', async (s) => {
    const client = fakeClient();
    await delegateToNexusCommand(client, s);
    expect(client.submitTask).toHaveBeenCalledWith(
      expect.objectContaining({
        instruction: 'DO THE THING',
        targetFile: '.nexus-delegate.md',
        command: 'auto',
      }),
    );
    expect(v.executeCommand).toHaveBeenCalledWith('workbench.action.chat.open', {
      query: 'DO THE THING',
    });
    expect(v.writeFile).not.toHaveBeenCalled();
  });

  it('uses the workspace folder, or an empty path, when the session has none', async () => {
    const client = fakeClient();
    await delegateToNexusCommand(client, session({ source: 'mcp', projectPath: undefined }));
    expect(client.submitTask).toHaveBeenLastCalledWith(
      expect.objectContaining({ projectPath: '/workspace' }),
    );
    v.folders = undefined;
    await delegateToNexusCommand(client, session({ source: 'mcp', projectPath: undefined }));
    expect(client.submitTask).toHaveBeenLastCalledWith(
      expect.objectContaining({ projectPath: '' }),
    );
  });

  it('warns when the task could not be queued, but still opens the chat', async () => {
    const client = fakeClient({ submitTask: vi.fn().mockRejectedValue(new Error('queue full')) });
    await delegateToNexusCommand(client, mcp());
    expect(v.showWarningMessage).toHaveBeenCalledWith(
      'Nexus: could not queue the delegated task — queue full',
    );
    expect(v.executeCommand).toHaveBeenCalledWith('workbench.action.chat.open', expect.anything());
    const plain = fakeClient({ submitTask: vi.fn().mockRejectedValue('plain') });
    await delegateToNexusCommand(plain, mcp());
    expect(v.showWarningMessage).toHaveBeenLastCalledWith(
      'Nexus: could not queue the delegated task — plain',
    );
  });

  it('offers to copy the instruction when chat cannot be opened', async () => {
    v.executeCommand.mockRejectedValueOnce(new Error('no chat'));
    v.showInformationMessage.mockResolvedValueOnce('Copy Instruction');
    await delegateToNexusCommand(fakeClient(), mcp());
    await Promise.resolve();
    expect(v.clipboard).toHaveBeenCalledWith('DO THE THING');
    v.clipboard.mockClear();
    v.executeCommand.mockRejectedValueOnce(new Error('no chat'));
    v.showInformationMessage.mockResolvedValueOnce(undefined);
    await delegateToNexusCommand(fakeClient(), mcp());
    await Promise.resolve();
    expect(v.clipboard).not.toHaveBeenCalled();
  });
});

describe('delegate: Copilot', () => {
  it.each(['GitHub Copilot', 'GitHub Copilot Chat'])('opens chat for %s', async (agentName) => {
    await delegateToNexusCommand(fakeClient(), session({ agentName }));
    expect(v.executeCommand).toHaveBeenCalledWith('workbench.action.chat.open', {
      query: 'DO THE THING',
    });
    expect(v.createTerminal).not.toHaveBeenCalled();
  });

  it('offers to copy when chat is unavailable', async () => {
    v.executeCommand.mockRejectedValueOnce(new Error('x'));
    v.showInformationMessage.mockResolvedValueOnce('Copy');
    await delegateToNexusCommand(fakeClient(), session({ agentName: 'GitHub Copilot' }));
    await Promise.resolve();
    expect(v.clipboard).toHaveBeenCalledWith('DO THE THING');
    v.clipboard.mockClear();
    v.executeCommand.mockRejectedValueOnce(new Error('x'));
    v.showInformationMessage.mockResolvedValueOnce(undefined);
    await delegateToNexusCommand(fakeClient(), session({ agentName: 'GitHub Copilot' }));
    await Promise.resolve();
    expect(v.clipboard).not.toHaveBeenCalled();
  });
});

describe('delegate: choosing the session', () => {
  it('accepts a tree item', async () => {
    const client = fakeClient();
    await delegateToNexusCommand(client, new AISessionItem(session({ id: 'from-item' })));
    expect(client.delegateSession).toHaveBeenCalledWith('from-item');
  });

  it('from the palette, offers only active non-delegated sessions', async () => {
    const client = fakeClient({
      listAISessions: vi
        .fn()
        .mockResolvedValue([
          session({ id: 'a' }),
          session({ id: 'b', status: 'idle' }),
          session({ id: 'c', delegatedToNexus: true }),
        ]),
    });
    v.showQuickPick.mockImplementation(async (items: Array<{ session: AISession }>) => items[0]);
    await delegateToNexusCommand(client);
    const offered = v.showQuickPick.mock.calls[0][0] as Array<{
      label: string;
      description: string;
    }>;
    expect(offered).toHaveLength(1);
    expect(client.delegateSession).toHaveBeenCalledWith('a');
  });

  it('shows a description of "" for a session without a project', async () => {
    const client = fakeClient({
      listAISessions: vi.fn().mockResolvedValue([session({ projectPath: undefined })]),
    });
    v.showQuickPick.mockResolvedValue(undefined);
    await delegateToNexusCommand(client);
    expect(
      (v.showQuickPick.mock.calls[0][0] as Array<{ description: string }>)[0].description,
    ).toBe('');
  });

  it('says so when there is nothing to delegate, also when listing fails', async () => {
    await delegateToNexusCommand(fakeClient());
    expect(v.showInformationMessage).toHaveBeenCalledWith(
      'No active, non-delegated AI sessions found.',
    );
    v.showInformationMessage.mockClear();
    await delegateToNexusCommand(
      fakeClient({ listAISessions: vi.fn().mockRejectedValue(new Error('x')) }),
    );
    expect(v.showInformationMessage).toHaveBeenCalledWith(
      'No active, non-delegated AI sessions found.',
    );
  });

  it('stops when the picker is dismissed', async () => {
    const client = fakeClient({ listAISessions: vi.fn().mockResolvedValue([session()]) });
    v.showQuickPick.mockResolvedValue(undefined);
    await delegateToNexusCommand(client);
    expect(client.delegateSession).not.toHaveBeenCalled();
  });

  it('reports a delegation failure (Error and non-Error) and does nothing else', async () => {
    await delegateToNexusCommand(
      fakeClient({ delegateSession: vi.fn().mockRejectedValue(new Error('404')) }),
      session(),
    );
    expect(v.showErrorMessage).toHaveBeenLastCalledWith('Failed to delegate session: 404');
    await delegateToNexusCommand(
      fakeClient({ delegateSession: vi.fn().mockRejectedValue('weird') }),
      session(),
    );
    expect(v.showErrorMessage).toHaveBeenLastCalledWith('Failed to delegate session: weird');
    expect(v.writeFile).not.toHaveBeenCalled();
  });
});
