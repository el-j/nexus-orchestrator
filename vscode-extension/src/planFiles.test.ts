import { describe, expect, it } from 'vitest';
import { assemblePlans, parsePlanFile, readPlanFiles, type PlanFs } from './planFiles';

describe('parsePlanFile front matter variants', () => {
  it('reads a multi-line list, the plan under other key names, and skips lines that are not key: value', () => {
    const p = parsePlanFile(
      'PLAN-005.md',
      '---\nid: PLAN-005\ntasks:\n  - TASK-1\n  - "TASK-2"\n  just prose\nnote\n---\n',
    );
    expect(p.tasks).toEqual(['TASK-1', 'TASK-2']);
    expect(parsePlanFile('TASK-1.md', '---\nplan: PLAN-7\n---\n').planId).toBe('PLAN-7');
    expect(parsePlanFile('TASK-2.md', '---\nplan_id: PLAN-8\n---\n').planId).toBe('PLAN-8');
    expect(parsePlanFile('TASK-3.md', '---\n- stray\ntitle: T\n---\n').title).toBe('T');
    expect(parsePlanFile('TASK-4.md', '---\ntitle: x\r\n---\r\n').title).toBe('x');
  });
});

describe('parsePlanFile', () => {
  it('reads front matter: id, title, status, role, plan and task lists', () => {
    const e = parsePlanFile(
      'TASK-007.md',
      `---\nid: TASK-007\ntitle: 'Fix the tokens'\nstatus: in-progress\nrole: coder\nplanId: PLAN-002\n---\n# ignored\n`,
    );
    expect(e).toEqual({
      id: 'TASK-007',
      title: 'Fix the tokens',
      status: 'in-progress',
      role: 'coder',
      planId: 'PLAN-002',
      tasks: [],
    });
    const p = parsePlanFile(
      'PLAN-002.md',
      `---\nid: PLAN-002\ntasks: [TASK-007, "TASK-008", 'TASK-009']\n---\n`,
    );
    expect(p.tasks).toEqual(['TASK-007', 'TASK-008', 'TASK-009']);
    expect(p.status).toBe('unknown');
  });

  it('works without front matter, as an agent that does not know the format writes it', () => {
    const e = parsePlanFile(
      'TASK-010.md',
      `# TASK-010 — Add focus ring to Button\n\n**Status:** Done\n\nPart of PLAN-003, see also PLAN-009.\n`,
    );
    expect(e).toMatchObject({
      id: 'TASK-010',
      title: 'Add focus ring to Button',
      status: 'Done',
      planId: 'PLAN-003',
    });
    expect(parsePlanFile('TASK-011.md', `# TASK-011: Colon title\nStatus: todo\n`)).toMatchObject({
      title: 'Colon title',
      status: 'todo',
    });
  });

  it('falls back to the file name for id and title, and tolerates an unterminated header', () => {
    expect(parsePlanFile('quality-audit.md', 'no heading here')).toMatchObject({
      id: 'quality-audit',
      title: 'quality-audit',
      status: 'unknown',
      planId: '',
    });
    expect(parsePlanFile('PLAN-1.md', '---\nid: PLAN-1\nnever closed')).toMatchObject({
      id: 'PLAN-1',
    });
    expect(parsePlanFile('PLAN-004-launch.md', '# Launch\n')).toMatchObject({
      id: 'PLAN-004',
      title: 'Launch',
    });
  });
});

function fakeFs(files: Record<string, string>, failOn?: string): PlanFs {
  return {
    readdirSync: (dir: string) => {
      const names = Object.keys(files)
        .filter((p) => p.startsWith(dir + '/'))
        .map((p) => p.slice(dir.length + 1))
        .filter((n) => !n.includes('/'));
      if (names.length === 0) throw new Error('ENOENT');
      return names;
    },
    readFileSync: (p: string) => {
      if (p === failOn) throw new Error('EACCES');
      return files[p]!;
    },
    statSync: (p: string) => ({ mtimeMs: p.endsWith('TASK-002.md') ? 5000 : 1000 }),
  };
}

describe('readPlanFiles', () => {
  it('reads .claude and .nexus plans and tasks, only markdown, and reports the newest change', () => {
    const r = readPlanFiles(
      '/r',
      fakeFs({
        '/r/.claude/plans/PLAN-001.md': '# One',
        '/r/.claude/plans/notes.txt': 'skip',
        '/r/.claude/tasks/TASK-001.md': '# A',
        '/r/.claude/tasks/TASK-002.md': '# B',
        '/r/.nexus/plans/PLAN-002.md': '# Two',
      }),
    );
    expect(r.plans.map((p) => p.id)).toEqual(['PLAN-001', 'PLAN-002']);
    expect(r.tasks.map((t) => t.id)).toEqual(['TASK-001', 'TASK-002']);
    expect(r.newest).toBe(5000);
  });

  it('skips an unreadable file and a missing directory without failing', () => {
    const r = readPlanFiles(
      '/r',
      fakeFs(
        { '/r/.claude/tasks/TASK-001.md': '# A', '/r/.claude/tasks/TASK-002.md': '# B' },
        '/r/.claude/tasks/TASK-001.md',
      ),
    );
    expect(r.tasks.map((t) => t.id)).toEqual(['TASK-002']);
    expect(r.plans).toEqual([]);
  });
});

describe('assemblePlans', () => {
  const task = (id: string, planId = '', status = 'todo') => ({
    id,
    title: id,
    status,
    role: '',
    planId,
    tasks: [],
  });
  const plan = (id: string, status = 'unknown', tasks: string[] = []) => ({
    id,
    title: id,
    status,
    role: '',
    planId: '',
    tasks,
  });

  it('attaches tasks by their planId and by the plan’s own list, in natural id order', () => {
    const { allPlans } = assemblePlans(
      [plan('PLAN-010', 'active', ['TASK-002']), plan('PLAN-002', 'done')],
      [task('TASK-010', 'PLAN-010'), task('TASK-002'), task('TASK-001', 'PLAN-002')],
      null,
      [],
    );
    expect(allPlans.map((p) => p.id)).toEqual(['PLAN-002', 'PLAN-010']);
    expect(allPlans[1]!.tasks.map((t) => t.id)).toEqual(['TASK-002', 'TASK-010']);
    expect(allPlans[0]!.tasks.map((t) => t.id)).toEqual(['TASK-001']);
  });

  it('picks the active plan: the file the JSON names, else one marked active, else the newest unfinished', () => {
    const plans = [
      plan('PLAN-001', 'done'),
      plan('PLAN-002', 'In Progress'),
      plan('PLAN-003', 'todo'),
    ];
    expect(assemblePlans(plans, [], 'PLAN-001', []).activePlan?.id).toBe('PLAN-001');
    expect(assemblePlans(plans, [], 'missing', []).activePlan?.id).toBe('PLAN-002');
    expect(
      assemblePlans(
        [plan('PLAN-001', 'completed'), plan('PLAN-002'), plan('PLAN-003')],
        [],
        null,
        [],
      ).activePlan?.id,
    ).toBe('PLAN-003');
    expect(assemblePlans([plan('PLAN-001', 'done')], [], null, []).activePlan).toBeNull();
    expect(assemblePlans([], [], null, []).activePlan).toBeNull();
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
    const { allPlans } = assemblePlans(
      [plan('PLAN-001', 'active'), plan('PLAN-002')],
      [task('TASK-001', 'PLAN-001', 'todo'), task('TASK-005', 'PLAN-001')],
      null,
      jsonPlans,
    );
    expect(allPlans.map((p) => [p.id, p.title, p.status])).toEqual([
      ['PLAN-001', 'From json', 'done'],
      ['PLAN-002', 'PLAN-002', 'unknown'],
    ]);
    expect(allPlans[0]!.tasks.map((t) => [t.id, t.status])).toEqual([
      ['TASK-001', 'done'],
      ['TASK-005', 'todo'],
    ]);
  });

  it('orders ids without a number and ids with different prefixes, and keeps a listed task that has no file', () => {
    const e = (id: string, tasks: string[] = []) => ({
      id,
      title: id,
      status: 'todo',
      role: '',
      planId: '',
      tasks,
    });
    const { allPlans } = assemblePlans(
      [e('quality-audit', ['ghost']), e('PLAN-2'), e('A-1'), e('PLAN-10')],
      [],
      null,
      [],
    );
    expect(allPlans.map((p) => p.id)).toEqual(['A-1', 'PLAN-2', 'PLAN-10', 'quality-audit']);
    expect(allPlans[3]!.tasks).toEqual([
      { id: 'ghost', title: 'ghost', status: 'unknown', role: '' },
    ]);
  });
});
