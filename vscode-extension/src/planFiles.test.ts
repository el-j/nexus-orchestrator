import { describe, expect, it } from 'vitest';
import {
  assemblePlans,
  isDone,
  parsePlanFile,
  pickActive,
  scanPlanFiles,
  UNASSIGNED_ID,
  type PlanFileEntry,
  type PlanFs,
  type PlanScan,
} from './planFiles';

const parse = (rel: string, text: string, kind?: 'plan' | 'task') =>
  parsePlanFile(rel, text, kind)!;

describe('parsePlanFile', () => {
  it('reads front matter: id, title, status, role, plan and task lists', () => {
    const { entry, kind } = parse(
      'x/TASK-007.md',
      `---\nid: TASK-007\ntitle: 'Fix the tokens'\nstatus: in-progress\nrole: coder\nplanId: PLAN-002\n---\n# ignored\n`,
    );
    expect(kind).toBe('task');
    expect(entry).toMatchObject({
      id: 'TASK-007',
      title: 'Fix the tokens',
      status: 'in-progress',
      role: 'coder',
      planRef: 'PLAN-002',
    });
    const plan = parse(
      'x/PLAN-002.md',
      `---\nid: PLAN-002\ntasks: [TASK-007, "TASK-008", 'TASK-009']\n---\nsee TASK-1 and task-1 and TASK-2\n`,
    );
    expect(plan.entry.listed).toEqual(['TASK-007', 'TASK-008', 'TASK-009']);
    expect(plan.entry.mentioned).toEqual(['TASK-1', 'TASK-2']);
    expect(plan.entry.status).toBe('unknown');
  });

  it('works without front matter, as an agent that does not know the format writes it', () => {
    const { entry } = parse(
      'docs/TASK-010.md',
      `# TASK-010 — Add focus ring to Button\n\n**Status:** Done\n\nPart of PLAN-003, see also PLAN-009.\n`,
    );
    expect(entry).toMatchObject({
      id: 'TASK-010',
      title: 'Add focus ring to Button',
      status: 'Done',
      bodyPlan: 'PLAN-003',
    });
    expect(parse('a/TASK-011.md', `# TASK-011: Colon title\nStatus: todo\n`).entry).toMatchObject({
      title: 'Colon title',
      status: 'todo',
    });
  });

  it('falls back to the file name for id and title, and tolerates an unterminated header', () => {
    expect(parse('a/quality-audit.md', 'no heading here', 'plan').entry).toMatchObject({
      id: 'quality-audit',
      title: 'quality-audit',
      status: 'unknown',
    });
    expect(parse('a/PLAN-1.md', '---\nid: PLAN-1\nnever closed').entry.id).toBe('PLAN-1');
    expect(parse('a/PLAN-004-launch.md', '# Launch\n').entry).toMatchObject({
      id: 'PLAN-004',
      title: 'Launch',
    });
  });

  it('decides the kind from front matter type, the file name or the folder, else it is not a plan or task', () => {
    expect(parse('w/auth.md', '---\ntype: Task\n---\n').kind).toBe('task');
    expect(parse('w/auth.md', '---\ntype: plan\n---\n').kind).toBe('plan');
    expect(parse('docs/plans/launch.md', '# x').kind).toBe('plan');
    expect(parse('docs/Task/fix.md', '# x').kind).toBe('task');
    expect(parsePlanFile('docs/notes.md', '# x')).toBeNull();
    expect(parsePlanFile('docs/notes.md', '---\ntype: memo\n---\n')).toBeNull();
  });

  it('reads multi-line lists, other key names for the plan, and skips lines that are not key: value', () => {
    const p = parse(
      'a/PLAN-005.md',
      '---\nid: PLAN-005\ntasks:\n  - TASK-1\n  - "TASK-2"\n  just prose\nnote\n---\n',
    );
    expect(p.entry.listed).toEqual(['TASK-1', 'TASK-2']);
    expect(parse('a/TASK-1.md', '---\nplan: PLAN-7\n---\n').entry.planRef).toBe('PLAN-7');
    expect(parse('a/TASK-2.md', '---\nplan_id: PLAN-8\n---\n').entry.planRef).toBe('PLAN-8');
    expect(parse('a/TASK-3.md', '---\n- stray\ntitle: T\n---\n').entry.title).toBe('T');
    expect(parse('a/TASK-4.md', '---\ntitle: x\r\n---\r\n').entry.title).toBe('x');
  });
});

function fakeFs(
  files: Record<string, string>,
  opts: { failRead?: string; failStat?: string; size?: number } = {},
): PlanFs {
  return {
    readdirSync: (dir) => {
      const prefix = dir.replace(/\/$/, '') + '/';
      const kids = new Map<string, boolean>();
      for (const p of Object.keys(files)) {
        if (!p.startsWith(prefix)) continue;
        const rest = p.slice(prefix.length);
        const [head] = rest.split('/');
        kids.set(head!, rest.includes('/'));
      }
      if (kids.size === 0) throw new Error('ENOENT');
      return [...kids].map(([name, isDirectory]) => ({ name, isDirectory }));
    },
    readFileSync: (p) => {
      if (p === opts.failRead) throw new Error('EACCES');
      return files[p]!;
    },
    statSync: (p) => {
      if (p === opts.failStat) throw new Error('EACCES');
      return { mtimeMs: p.endsWith('TASK-002.md') ? 5000 : 1000, size: opts.size ?? 10 };
    },
  };
}

describe('scanPlanFiles', () => {
  it('reports the newest change and skips an unreadable file and a missing folder', () => {
    const files = {
      '/r/.claude/plans/PLAN-001.md': '# One',
      '/r/.claude/tasks/TASK-001.md': '# A',
      '/r/.claude/tasks/TASK-002.md': '# B',
      '/r/.claude/tasks/notes.txt': 'skip',
    };
    const scan = scanPlanFiles('/r', fakeFs(files, { failRead: '/r/.claude/tasks/TASK-001.md' }));
    expect(scan.plans.map((p) => p.id)).toEqual(['PLAN-001']);
    expect(scan.orphans.map((t) => t.id)).toEqual(['TASK-002']);
    expect(scan.newest).toBe(5000);
    expect(
      scanPlanFiles('/r', fakeFs(files, { failStat: '/r/.claude/plans/PLAN-001.md' })).plans,
    ).toEqual([]);
  });

  it('does not read a file that is too large to be a plan or task', () => {
    const files = { '/r/.claude/plans/PLAN-001.md': '# One' };
    expect(scanPlanFiles('/r', fakeFs(files, { size: 300 << 10 })).plans).toEqual([]);
  });

  it('reads configured folders only a few levels deep', () => {
    const files = {
      '/r/w/a/b/c/ok.md': '# ok',
      '/r/w/a/b/c/d/too-deep.md': '# deep',
    };
    const scan = scanPlanFiles('/r', fakeFs(files), { plans: ['w'], discover: false });
    expect(scan.plans.map((p) => p.id)).toEqual(['ok']);
  });

  it('stops searching a project after a bounded number of entries', () => {
    const many: Array<{ name: string; isDirectory: boolean }> = [];
    for (let i = 0; i < 20001; i++) many.push({ name: `f${i}.txt`, isDirectory: false });
    many.push({ name: 'PLAN-1.md', isDirectory: false });
    const fs: PlanFs = {
      readdirSync: (dir) => {
        if (dir !== '/r') throw new Error('ENOENT');
        return many;
      },
      readFileSync: () => '# x',
      statSync: () => ({ mtimeMs: 1, size: 1 }),
    };
    expect(scanPlanFiles('/r', fs).plans).toEqual([]);
  });

  it('cleans configured folders and survives a folder that is a file', () => {
    const files = { '/r/docs/plans/launch.md': '# Launch', '/r/work/not-a-folder': 'x' };
    const scan = scanPlanFiles('/r', fakeFs(files), {
      plans: [' /docs\\plans/ ', '.', ''],
      tasks: ['work/not-a-folder'],
      discover: false,
    });
    expect(scan.plans.map((p) => p.id)).toEqual(['launch']);
  });
});

describe('assemblePlans', () => {
  const entry = (
    id: string,
    status = 'unknown',
    extra: Partial<PlanFileEntry> = {},
  ): PlanFileEntry => ({
    id,
    title: id,
    status,
    role: '',
    rel: `${id}.md`,
    planRef: '',
    listed: [],
    bodyPlan: '',
    mentioned: [],
    stem: id,
    ...extra,
  });
  const scanOf = (
    plans: Array<[PlanFileEntry, PlanFileEntry[]]>,
    orphans: PlanFileEntry[] = [],
  ): PlanScan => ({
    plans: plans.map(([p, tasks]) => ({ ...p, tasks })),
    orphans,
    folders: [],
    newest: 0,
  });

  it('picks the active plan: the one the JSON names, else one marked active, else the newest unfinished', () => {
    const plans = [
      entry('PLAN-001', 'done'),
      entry('PLAN-002', 'In Progress'),
      entry('PLAN-003', 'todo'),
    ];
    const scan = scanOf(plans.map((p) => [p, []]));
    expect(assemblePlans(scan, 'PLAN-001', []).activePlan?.id).toBe('PLAN-001');
    expect(assemblePlans(scan, 'missing', []).activePlan?.id).toBe('PLAN-002');
    expect(
      assemblePlans(
        scanOf([
          [entry('PLAN-1', 'completed'), []],
          [entry('PLAN-2'), []],
          [entry('PLAN-3'), []],
        ]),
        null,
        [],
      ).activePlan?.id,
    ).toBe('PLAN-3');
    expect(assemblePlans(scanOf([[entry('PLAN-1', 'done'), []]]), null, []).activePlan).toBeNull();
    expect(assemblePlans(scanOf([]), null, []).activePlan).toBeNull();
  });

  it('lets orchestrator.json keep what it says and adds only what the files know beyond it', () => {
    const jsonPlans = [
      {
        id: 'PLAN-001',
        title: 'From json',
        status: 'done',
        tasks: [{ id: 'TASK-001', title: 'J', status: 'done', role: 'x' }],
      },
    ];
    const scan = scanOf([
      [entry('PLAN-001', 'active'), [entry('TASK-001', 'todo'), entry('TASK-005', 'todo')]],
      [entry('PLAN-002'), []],
    ]);
    const { allPlans } = assemblePlans(scan, null, jsonPlans);
    expect(allPlans.map((p) => [p.id, p.title, p.status])).toEqual([
      ['PLAN-001', 'From json', 'done'],
      ['PLAN-002', 'PLAN-002', 'unknown'],
    ]);
    expect(allPlans[0]!.tasks.map((t) => [t.id, t.status])).toEqual([
      ['TASK-001', 'done'],
      ['TASK-005', 'todo'],
    ]);
  });

  it('links a task no file plan claims to a plan only orchestrator.json knows', () => {
    const jsonPlans = [
      {
        id: 'PLAN-J',
        title: 'J',
        status: 'active',
        tasks: [{ id: 'TASK-5', title: 'five', status: 'done', role: '' }],
      },
    ];
    const byRef = entry('TASK-6', 'todo', { planRef: 'PLAN-J' });
    const byFolder = entry('TASK-7', 'todo', { rel: 'tasks/PLAN-J/TASK-7.md' });
    const byMention = entry('TASK-8', 'todo', { bodyPlan: 'PLAN-J' });
    const listed = entry('TASK-5', 'todo');
    const stray = entry('TASK-9', 'todo', { planRef: 'PLAN-NOPE' });
    const { allPlans } = assemblePlans(
      scanOf([], [byRef, byFolder, byMention, listed, stray]),
      null,
      jsonPlans,
    );
    expect(allPlans[0]!.tasks.map((t) => t.id)).toEqual(['TASK-5', 'TASK-6', 'TASK-7', 'TASK-8']);
    expect(allPlans.at(-1)!.tasks.map((t) => t.id)).toEqual(['TASK-9']);
  });

  it('collects tasks no plan claims in a last pseudo plan that is never the active one', () => {
    const { allPlans, activePlan } = assemblePlans(
      scanOf([[entry('PLAN-1', 'done'), []]], [entry('TASK-9')]),
      null,
      [],
    );
    expect(allPlans.at(-1)).toMatchObject({
      id: UNASSIGNED_ID,
      title: 'Unassigned tasks',
      tasks: [{ id: 'TASK-9' }],
    });
    expect(activePlan).toBeNull();
  });
});

describe('status helpers', () => {
  it('knows finished and active statuses', () => {
    expect(isDone(' Completed ')).toBe(true);
    expect(isDone('todo')).toBe(false);
    expect(pickActive([{ status: 'todo' }, { status: 'WIP' }])?.status).toBe('WIP');
  });
});
