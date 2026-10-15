import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync, existsSync } from 'fs';
import { tmpdir } from 'os';
import * as path from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const h = vi.hoisted(() => ({
  folders: [] as Array<{ uri: { fsPath: string } }>,
  picks: [] as unknown[],
  commands: new Map<string, () => Promise<void>>(),
  executed: [] as string[],
  open: undefined as undefined | Array<{ fsPath: string }>,
  info: [] as string[],
  errors: [] as string[],
  shown: [] as string[],
  quickPickItems: [] as Array<Array<{ id?: string; label: string }>>,
  placeHolders: [] as string[],
  folderPick: undefined as undefined | { uri: { fsPath: string } },
}));

vi.mock('vscode', () => ({
  Uri: { file: (p: string) => ({ fsPath: p }) },
  workspace: {
    get workspaceFolders() {
      return h.folders;
    },
    openTextDocument: async (u: { fsPath: string }) => ({ uri: u }),
  },
  commands: {
    registerCommand: (id: string, fn: () => Promise<void>) => {
      h.commands.set(id, fn);
      return { dispose() {} };
    },
    executeCommand: async (id: string) => {
      h.executed.push(id);
    },
  },
  window: {
    showWorkspaceFolderPick: async () => h.folderPick,
    showQuickPick: async (
      items: Array<{ id?: string; label: string }>,
      opts?: { placeHolder?: string },
    ) => {
      h.quickPickItems.push(items);
      h.placeHolders.push(opts?.placeHolder ?? '');
      return h.picks.shift();
    },
    showOpenDialog: async () => h.open,
    showInformationMessage: async (m: string) => void h.info.push(m),
    showErrorMessage: async (m: string) => void h.errors.push(m),
    showTextDocument: async (d: { uri: { fsPath: string } }) => void h.shown.push(d.uri.fsPath),
  },
}));

import { registerPlanFolders } from './planFolders';

let root: string;
const policyPath = () => path.join(root, '.nexus', 'policy.json');
const policy = () => JSON.parse(readFileSync(policyPath(), 'utf8')) as Record<string, unknown>;
const run = () => h.commands.get('nexus.configurePlanFolders')!();
const scanner = (sources: string[] = []) =>
  ({
    getOrchestrations: () => [{ folderPath: root, sources }],
  }) as unknown as Parameters<typeof registerPlanFolders>[1];

beforeEach(() => {
  root = mkdtempSync(path.join(tmpdir(), 'nexus-pf-'));
  Object.assign(h, {
    folders: [{ uri: { fsPath: root } }],
    picks: [],
    executed: [],
    open: undefined,
    info: [],
    errors: [],
    shown: [],
    quickPickItems: [],
    placeHolders: [],
    folderPick: undefined,
  });
  h.commands.clear();
  registerPlanFolders({ subscriptions: [] } as never, scanner(['.claude/plans', 'docs']));
});
afterEach(() => rmSync(root, { recursive: true, force: true }));

describe('Nexus: Choose Plan and Task Folders', () => {
  it('tells the user to open a folder first', async () => {
    h.folders = [];
    await run();
    expect(h.info[0]).toMatch(/Open a folder/);
  });

  it('copes with no workspace folders at all, and a folder the scanner has not seen', async () => {
    h.folders = undefined as never;
    await run();
    expect(h.info[0]).toMatch(/Open a folder/);
    h.folders = [{ uri: { fsPath: root } }];
    registerPlanFolders(
      { subscriptions: [] } as never,
      { getOrchestrations: () => [] } as unknown as Parameters<typeof registerPlanFolders>[1],
    );
    h.picks = [undefined];
    await run();
    expect(h.placeHolders.at(-1)).toContain('nothing found yet');
  });

  it('removes from a policy that lists only plans, or only tasks', async () => {
    mkdirSync(path.join(root, '.nexus'));
    writeFileSync(policyPath(), JSON.stringify({ planFiles: { plans: ['a'] } }));
    h.picks = [{ id: 'remove' }, [{ id: 'plans:a' }]];
    await run();
    expect(policy().planFiles).toEqual({ plans: [], tasks: [] });
    writeFileSync(policyPath(), JSON.stringify({ planFiles: { tasks: ['t'] } }));
    h.picks = [{ id: 'remove' }, [{ id: 'tasks:t' }]];
    await run();
    expect(policy().planFiles).toEqual({ plans: [], tasks: [] });
  });

  it('shows where plans are read from and does nothing when the picker is dismissed', async () => {
    h.picks = [undefined];
    await run();
    expect(h.placeHolders[0]).toContain('.claude/plans, docs');
    expect(existsSync(policyPath())).toBe(false);
  });

  it('says so when nothing was found yet', async () => {
    registerPlanFolders({ subscriptions: [] } as never, scanner([]));
    h.picks = [undefined];
    await run();
    expect(h.placeHolders[0]).toContain('nothing found yet');
  });

  it('asks which workspace folder when there are several', async () => {
    h.folders = [{ uri: { fsPath: root } }, { uri: { fsPath: '/other' } }];
    h.folderPick = undefined;
    await run();
    expect(h.quickPickItems).toEqual([]);
    h.folderPick = { uri: { fsPath: root } };
    h.picks = [undefined];
    await run();
    expect(h.quickPickItems).toHaveLength(1);
  });

  it('adds a plan folder and a task folder to .nexus/policy.json, as paths inside the project', async () => {
    mkdirSync(path.join(root, 'design', 'specs'), { recursive: true });
    h.picks = [{ id: 'addPlans' }];
    h.open = [{ fsPath: path.join(root, 'design', 'specs') }];
    await run();
    h.picks = [{ id: 'addTasks' }];
    h.open = [{ fsPath: path.join(root, 'design', 'work') }];
    await run();
    expect(policy()).toEqual({ planFiles: { plans: ['design/specs'], tasks: ['design/work'] } });
    expect(h.executed.filter((c) => c === 'nexus.refreshWorkspaceAgents')).toHaveLength(2);

    // the same folder twice is not added twice
    h.picks = [{ id: 'addPlans' }];
    h.open = [{ fsPath: path.join(root, 'design', 'specs') }];
    await run();
    expect((policy().planFiles as { plans: string[] }).plans).toEqual(['design/specs']);
  });

  it('keeps everything else in an existing policy.json', async () => {
    mkdirSync(path.join(root, '.nexus'));
    writeFileSync(policyPath(), JSON.stringify({ version: 1, verify: 'go test', rules: ['a'] }));
    h.picks = [{ id: 'addPlans' }];
    h.open = [{ fsPath: path.join(root, 'plans') }];
    await run();
    expect(policy()).toMatchObject({
      version: 1,
      verify: 'go test',
      rules: ['a'],
      planFiles: { plans: ['plans'] },
    });
  });

  it('refuses a folder outside the project, the project itself, and a dismissed dialog', async () => {
    for (const open of [
      [{ fsPath: path.join(path.dirname(root), 'elsewhere') }],
      [{ fsPath: root }],
      undefined,
      [],
    ]) {
      h.picks = [{ id: 'addTasks' }];
      h.open = open;
      await run();
    }
    expect(h.errors).toHaveLength(2);
    expect(existsSync(policyPath())).toBe(false);
  });

  it('refuses to touch a policy.json that is not valid JSON', async () => {
    mkdirSync(path.join(root, '.nexus'));
    writeFileSync(policyPath(), '{broken');
    h.picks = [{ id: 'addPlans' }];
    h.open = [{ fsPath: path.join(root, 'plans') }];
    await run();
    expect(h.errors[0]).toMatch(/not valid JSON/);
    expect(readFileSync(policyPath(), 'utf8')).toBe('{broken');
    writeFileSync(policyPath(), '[1]');
    h.picks = [{ id: 'addPlans' }];
    await run();
    expect(h.errors).toHaveLength(2);
  });

  it('removes configured folders, and says when there are none', async () => {
    h.picks = [{ id: 'remove' }];
    await run();
    expect(h.info.at(-1)).toMatch(/No folders are configured/);

    mkdirSync(path.join(root, '.nexus'));
    writeFileSync(
      policyPath(),
      JSON.stringify({ planFiles: { plans: ['a', 'b'], tasks: ['t'], discover: false } }),
    );
    h.picks = [{ id: 'remove' }, [{ id: 'plans:a' }, { id: 'tasks:t' }]];
    await run();
    expect(policy().planFiles).toEqual({ plans: ['b'], tasks: [], discover: false });

    h.picks = [{ id: 'remove' }, undefined];
    await run();
    expect((policy().planFiles as { plans: string[] }).plans).toEqual(['b']);
  });

  it('switches the search of the whole project on and off', async () => {
    h.picks = [{ id: 'toggle' }];
    await run();
    expect((policy().planFiles as { discover: boolean }).discover).toBe(false);
    expect(h.quickPickItems[0]!.find((i) => i.id === 'toggle')!.label).toContain('on');
    h.picks = [{ id: 'toggle' }];
    await run();
    expect((policy().planFiles as { discover: boolean }).discover).toBe(true);
    expect(h.quickPickItems[1]!.find((i) => i.id === 'toggle')!.label).toContain('off');
  });

  it('opens policy.json, creating a skeleton with the block to fill in when there is none', async () => {
    h.picks = [{ id: 'open' }];
    await run();
    expect(h.shown).toEqual([policyPath()]);
    expect(policy()).toEqual({ planFiles: { plans: [], tasks: [] } });
    h.picks = [{ id: 'open' }];
    await run();
    expect(policy()).toEqual({ planFiles: { plans: [], tasks: [] } }); // an existing file is left alone
    expect(h.shown).toHaveLength(2);
  });
});
