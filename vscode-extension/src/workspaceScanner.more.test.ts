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
  configChange: undefined as
    undefined | ((e: { affectsConfiguration: (s: string) => boolean }) => void),
  settings: {} as Record<string, unknown>,
}));

vi.mock('fs', () => ({
  existsSync: (p: string) => h.files.has(p),
  readdirSync: (dir: string) => {
    const kids = new Map<string, boolean>();
    for (const p of h.files.keys()) {
      if (!p.startsWith(dir + '/')) continue;
      const rest = p.slice(dir.length + 1);
      kids.set(rest.split('/')[0]!, rest.includes('/'));
    }
    if (kids.size === 0) throw new Error('ENOENT');
    return [...kids].map(([name, isDir]) => ({ name, isDirectory: () => isDir }));
  },
  readFileSync: (p: string) => {
    if (h.throwOn === p) throw new Error('EACCES');
    return h.files.get(p);
  },
  statSync: () => ({ mtime: h.mtime, mtimeMs: h.mtime.getTime(), size: 10 }),
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
      getConfiguration: () => ({
        get: (key: string, fallback: unknown) => (key in h.settings ? h.settings[key] : fallback),
      }),
      onDidChangeConfiguration: (f: typeof h.configChange) => {
        h.configChange = f;
        return { dispose: vi.fn() };
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
  h.configChange = undefined;
  h.settings = {};
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

  it('defaults missing plan fields and has no active plan when the id does not match a plan left to do', () => {
    const [o] = scan(doc({ activePlanId: 'nope', plans: { a: { status: 'done' } } }));
    expect(o.activePlan).toBeNull();
    expect(o.allPlans[0]).toEqual({ id: '', title: '', status: 'done', tasks: [] });
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

describe('plans written as markdown files, with or without orchestrator.json', () => {
  const md = (root: string, kind: 'plans' | 'tasks', name: string, text: string) =>
    h.files.set(`${root}/.claude/${kind}/${name}`, text);

  it('shows a repository that only has plan and task files, which is what other agents write', () => {
    md(
      '/repo-one',
      'plans',
      'PLAN-001.md',
      '---\nid: PLAN-001\ntitle: Audit\nstatus: active\n---\n',
    );
    md('/repo-one', 'tasks', 'TASK-001.md', '# TASK-001 — Fix tokens\nStatus: todo\nPLAN-001\n');
    const s = new WorkspaceScanner(ctx());
    s.scan();
    const [o] = s.getOrchestrations();
    expect(o.folderName).toBe('repo-one');
    expect(o.activePlan?.id).toBe('PLAN-001');
    expect(o.allPlans[0].tasks).toEqual([
      { id: 'TASK-001', title: 'Fix tokens', status: 'todo', role: '' },
    ]);
    expect(o.lastModified).toEqual(h.mtime);
  });

  it('adds the files to an orchestrator.json that does not list them', () => {
    h.files.set(
      file('/repo-one'),
      doc({
        activePlanId: 'PLAN-001',
        plans: { a: { id: 'PLAN-001', title: 'J', status: 'active', tasks: [] } },
      }),
    );
    md('/repo-one', 'tasks', 'TASK-009.md', '# TASK-009 — New\nStatus: todo\nPLAN-001\n');
    md('/repo-one', 'plans', 'PLAN-002.md', '# Later\n');
    const s = new WorkspaceScanner(ctx());
    s.scan();
    const [o] = s.getOrchestrations();
    expect(o.allPlans.map((p) => p.id)).toEqual(['PLAN-001', 'PLAN-002']);
    expect(o.allPlans[0].tasks.map((t) => t.id)).toEqual(['TASK-009']);
    expect(o.activePlan?.id).toBe('PLAN-001');
  });

  it('shows nothing for a folder without any plan, task or orchestrator.json', () => {
    const s = new WorkspaceScanner(ctx());
    s.scan();
    expect(s.getOrchestrations()).toEqual([]);
  });
});

describe('configuring where plans and tasks live', () => {
  const policy = (o: object) => h.files.set('/repo-one/.nexus/policy.json', JSON.stringify(o));
  const plansOf = () => {
    const s = new WorkspaceScanner(ctx());
    s.scan();
    return s.getOrchestrations()[0]?.allPlans.map((p) => p.id) ?? [];
  };

  it('reads folders from .nexus/policy.json', () => {
    policy({ planFiles: { plans: ['design/specs'] } });
    h.files.set('/repo-one/design/specs/launch.md', '# Launch\n');
    expect(plansOf()).toEqual(['launch']);
  });

  it('adds the folders of the nexus.planFiles settings to the policy ones, without duplicates', () => {
    policy({ planFiles: { plans: ['a', 'b'] } });
    h.settings = { plans: ['b', 'c'], tasks: ['t'] };
    h.files.set('/repo-one/c/one.md', '# C\n');
    h.files.set('/repo-one/a/two.md', '# A\n');
    expect(plansOf()).toEqual(['one', 'two']);
  });

  it('searches the whole project unless the policy or a setting turns that off', () => {
    h.files.set('/repo-one/docs/PLAN-1.md', '# Elsewhere\n');
    expect(plansOf()).toEqual(['PLAN-1']);
    policy({ planFiles: { discover: false, plans: ['x'] } });
    expect(plansOf()).toEqual([]);
    h.files.delete('/repo-one/.nexus/policy.json');
    h.settings = { discover: false, plans: ['x'] };
    expect(plansOf()).toEqual([]);
  });

  it.each([
    ['not json'],
    ['null'],
    [JSON.stringify({ planFiles: 'x' })],
    [JSON.stringify({ planFiles: null })],
  ])('ignores a policy.json with %s in it and keeps the defaults', (raw) => {
    h.files.set('/repo-one/.nexus/policy.json', raw);
    h.files.set('/repo-one/.claude/plans/PLAN-1.md', '# One\n');
    expect(plansOf()).toEqual(['PLAN-1']);
  });

  it('tolerates settings that are not lists', () => {
    h.settings = { plans: 'docs', tasks: { a: 1 } };
    h.files.set('/repo-one/.claude/plans/PLAN-1.md', '# One\n');
    expect(plansOf()).toEqual(['PLAN-1']);
  });

  it('watches the configured folders and rebuilds when the settings change', () => {
    policy({ planFiles: { plans: ['design/specs'], tasks: ['design/work'] } });
    const s = new WorkspaceScanner(ctx());
    const changed = vi.fn();
    s.onDidChange(changed);
    s.start();
    const patterns = h.watchers.map((w) => (w.pattern as { pattern: string }).pattern);
    expect(patterns).toContain('design/specs/**/*.md');
    expect(patterns).toContain('design/work/**/*.md');
    const before = h.watchers[0];

    h.configChange!({ affectsConfiguration: () => false });
    expect(changed).not.toHaveBeenCalled();
    h.configChange!({ affectsConfiguration: (k) => k === 'nexus.planFiles' });
    expect(before.dispose).toHaveBeenCalled();
    expect(changed).toHaveBeenCalledTimes(1);
  });
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
    const WATCHED = '{.claude,.nexus}/{orchestrator.json,policy.json,plans/**/*.md,tasks/**/*.md}';
    const ANYWHERE = '**/{PLAN,TASK}-*.md';
    expect(h.watchers.map((w) => (w.pattern as { pattern: string }).pattern)).toEqual([
      WATCHED,
      ANYWHERE,
      WATCHED,
      ANYWHERE,
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
    expect(h.watchers).toHaveLength(6); // two for the old folder, four for the two new ones
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
