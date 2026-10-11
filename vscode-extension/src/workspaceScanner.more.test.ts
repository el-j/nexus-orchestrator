import { beforeEach, describe, expect, it, vi } from 'vitest';

const h = vi.hoisted(() => ({
  folders: [] as Array<{ uri: { fsPath: string } }>,
  files: new Map<string, string>(),
  mtime: new Date('2026-04-04T00:00:00Z'),
  throwOn: undefined as string | undefined,
  watchers: [] as Array<{
    pattern: unknown;
    handlers: Record<string, () => void>;
    dispose: ReturnType<typeof vi.fn>;
  }>,
  folderChange: undefined as undefined | (() => void),
}));

vi.mock('fs', () => ({
  existsSync: (p: string) => h.files.has(p),
  readFileSync: (p: string) => {
    if (h.throwOn === p) throw new Error('EACCES');
    return h.files.get(p);
  },
  statSync: () => ({ mtime: h.mtime }),
}));
vi.mock('vscode', () => {
  class EventEmitter {
    private ls: Array<() => void> = [];
    event = (l: () => void) => (this.ls.push(l), { dispose: () => {} });
    fire = () => this.ls.forEach((l) => l());
    dispose = vi.fn();
  }
  class RelativePattern {
    constructor(
      public folder: unknown,
      public pattern: string,
    ) {}
  }
  return {
    EventEmitter,
    RelativePattern,
    workspace: {
      get workspaceFolders() {
        return h.folders;
      },
      createFileSystemWatcher: (pattern: unknown) => {
        const w = { pattern, handlers: {} as Record<string, () => void>, dispose: vi.fn() };
        h.watchers.push(w);
        return {
          onDidChange: (f: () => void) => (w.handlers['change'] = f),
          onDidCreate: (f: () => void) => (w.handlers['create'] = f),
          onDidDelete: (f: () => void) => (w.handlers['delete'] = f),
          dispose: w.dispose,
        };
      },
      onDidChangeWorkspaceFolders: (f: () => void) => {
        h.folderChange = f;
        return { dispose: vi.fn() };
      },
    },
  };
});

import { WorkspaceScanner } from './workspaceScanner';

const file = (root: string) => `${root}/.claude/orchestrator.json`;
const ctx = () =>
  ({
    subscriptions: [] as Array<{ dispose(): void }>,
  }) as unknown as import('vscode').ExtensionContext;
const doc = (o: object) => JSON.stringify(o);

beforeEach(() => {
  h.folders = [{ uri: { fsPath: '/repo-one' } }];
  h.files.clear();
  h.throwOn = undefined;
  h.watchers.length = 0;
  h.folderChange = undefined;
  vi.spyOn(console, 'error').mockImplementation(() => {});
});

describe('parsing orchestrator.json', () => {
  const scan = (raw: string) => {
    h.files.set(file('/repo-one'), raw);
    const s = new WorkspaceScanner(ctx());
    s.scan();
    return s.getOrchestrations();
  };

  it('resolves plans with their tasks and picks the active plan', () => {
    const [o] = scan(
      doc({
        activePlanId: 'p2',
        plans: {
          a: { id: 'p1', title: 'First', status: 'done', tasks: ['t1'] },
          b: { id: 'p2', title: 'Second', status: 'in-progress', tasks: ['t2', 'ghost'] },
        },
        tasks: {
          t1: { id: 't1', title: 'One', status: 'done', role: 'dev' },
          t2: { title: 'Two' },
        },
      }),
    );
    expect(o.folderName).toBe('repo-one');
    expect(o.lastModified).toEqual(h.mtime);
    expect(o.allPlans.map((p) => p.id)).toEqual(['p1', 'p2']);
    expect(o.activePlan?.id).toBe('p2');
    expect(o.allPlans[0].tasks[0]).toEqual({ id: 't1', title: 'One', status: 'done', role: 'dev' });
    expect(o.allPlans[1].tasks).toEqual([
      { id: 't2', title: 'Two', status: 'unknown', role: '' },
      { id: 'ghost', title: 'ghost', status: 'unknown', role: '' }, // unknown task keeps its id
    ]);
  });

  it('defaults missing plan fields and has no active plan when the id does not match', () => {
    const [o] = scan(doc({ activePlanId: 'nope', plans: { a: {} } }));
    expect(o.activePlan).toBeNull();
    expect(o.allPlans[0]).toEqual({ id: '', title: '', status: 'unknown', tasks: [] });
  });

  it('defaults a referenced task that has no status', () => {
    const [o] = scan(doc({ plans: { p: { id: 'p', tasks: ['t'] } }, tasks: { t: { id: 't' } } }));
    expect(o.allPlans[0].tasks).toEqual([{ id: 't', title: 't', status: 'unknown', role: '' }]);
  });

  it('accepts a file that only has tasks', () => {
    const [o] = scan(doc({ tasks: { t: { id: 't' } } }));
    expect(o.allPlans).toEqual([]);
  });

  it.each([['not json'], ['null'], ['42'], ['"text"'], ['[]'], [doc({})], [doc({ other: 1 })]])(
    'ignores %s',
    (raw) => {
      expect(scan(raw)).toEqual([]);
    },
  );
});

describe('scanning folders', () => {
  it('skips folders without the file, reports unreadable ones, and keeps scanning', () => {
    h.folders = [
      { uri: { fsPath: '/none' } },
      { uri: { fsPath: '/bad' } },
      { uri: { fsPath: '/good' } },
    ];
    h.files.set(file('/bad'), doc({ plans: {} }));
    h.files.set(file('/good'), doc({ plans: {} }));
    h.throwOn = file('/bad');
    const s = new WorkspaceScanner(ctx());
    s.scan();
    expect(s.getOrchestrations().map((o) => o.folderName)).toEqual(['good']);
    expect(console.error).toHaveBeenCalled();
  });

  it('copes with no workspace folders at all', () => {
    (h as { folders: unknown }).folders = undefined;
    const s = new WorkspaceScanner(ctx());
    s.scan();
    s.start();
    expect(s.getOrchestrations()).toEqual([]);
  });

  it('replaces earlier results on every scan', () => {
    h.files.set(file('/repo-one'), doc({ plans: {} }));
    const s = new WorkspaceScanner(ctx());
    s.scan();
    s.scan();
    expect(s.getOrchestrations()).toHaveLength(1);
  });
});

describe('watching', () => {
  it('watches the exact file in every folder and rescans + notifies on create, change and delete', () => {
    h.folders = [{ uri: { fsPath: '/a' } }, { uri: { fsPath: '/b' } }];
    const context = ctx();
    const s = new WorkspaceScanner(context);
    const changed = vi.fn();
    s.onDidChange(changed);
    s.start();
    expect(h.watchers.map((w) => (w.pattern as { pattern: string }).pattern)).toEqual([
      '.claude/orchestrator.json',
      '.claude/orchestrator.json',
    ]);

    h.files.set(file('/a'), doc({ plans: {} }));
    h.watchers[0].handlers['create']();
    expect(s.getOrchestrations()).toHaveLength(1);
    h.watchers[0].handlers['change']();
    h.files.delete(file('/a'));
    h.watchers[0].handlers['delete']();
    expect(s.getOrchestrations()).toEqual([]);
    expect(changed).toHaveBeenCalledTimes(3);
  });

  it('rebuilds its watchers when workspace folders change', () => {
    const s = new WorkspaceScanner(ctx());
    const changed = vi.fn();
    s.onDidChange(changed);
    s.start();
    const first = h.watchers[0];
    h.folders = [{ uri: { fsPath: '/x' } }, { uri: { fsPath: '/y' } }];
    h.folderChange!();
    expect(first.dispose).toHaveBeenCalled();
    expect(h.watchers).toHaveLength(3);
    expect(changed).toHaveBeenCalledTimes(1);
  });

  it('dispose() releases the watchers and the emitter', () => {
    const s = new WorkspaceScanner(ctx());
    s.start();
    s.dispose();
    expect(h.watchers[0].dispose).toHaveBeenCalled();
    s.dispose(); // safe twice
  });
});
