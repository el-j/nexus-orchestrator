/**
 * planFiles.ts — Reads plans and tasks from the markdown files an agent writes
 * (.claude/plans/PLAN-*.md, .claude/tasks/TASK-*.md, and the same under .nexus), so the
 * sidebar shows the orchestration of a repository whose agent never produced an
 * orchestrator.json. The parser is lenient on purpose: front matter when there is any,
 * otherwise the first heading, a "Status:" line and the first PLAN-n mentioned.
 */

import type { OrchestratorPlan, OrchestratorTask } from './workspaceScanner';

/** One plan or task file, reduced to what the tree shows. */
export interface PlanFileEntry {
  id: string;
  title: string;
  status: string;
  role: string;
  /** The plan a task belongs to; empty for plans and for tasks that name none. */
  planId: string;
  /** The task ids a plan lists itself. */
  tasks: string[];
}

/** The slice of `fs` the reader needs; tests pass a fake. */
export interface PlanFs {
  readdirSync(dir: string): string[];
  readFileSync(path: string): string;
  statSync(path: string): { mtimeMs: number };
}

const ID_IN_NAME = /^((?:PLAN|TASK)-\d+)/i;
const DONE = /^(done|completed?|closed|merged|shipped|cancell?ed)$/i;
const ACTIVE = /^(active|in[-_ ]?progress|doing|started|wip)$/i;

function unquote(s: string): string {
  return s
    .trim()
    .replace(/^(['"])(.*)\1$/, '$2')
    .trim();
}

function frontMatter(text: string): { meta: Record<string, string | string[]>; body: string } {
  const lines = text.replace(/\r\n/g, '\n').split('\n');
  if (lines[0]?.trim() !== '---') return { meta: {}, body: text };
  const meta: Record<string, string | string[]> = {};
  let key = '';
  let i = 1;
  for (; i < lines.length && lines[i]!.trim() !== '---'; i++) {
    const line = lines[i]!;
    const item = /^\s+-\s+(.*)$/.exec(line);
    if (item && key) {
      const cur = meta[key];
      meta[key] = [...(Array.isArray(cur) ? cur : []), unquote(item[1]!)];
      continue;
    }
    const kv = /^([A-Za-z_][\w-]*)\s*:\s*(.*)$/.exec(line);
    if (!kv) continue;
    key = kv[1]!;
    const v = kv[2]!.trim();
    if (v.startsWith('[') && v.endsWith(']')) {
      meta[key] = v.slice(1, -1).split(',').map(unquote).filter(Boolean);
    } else {
      meta[key] = unquote(v);
    }
  }
  // An unterminated header (no closing ---) is read as far as it goes; the rest is the body.
  return { meta, body: lines.slice(i + 1).join('\n') };
}

const str = (v: string | string[] | undefined): string => (typeof v === 'string' ? v : '');

/** Parses one plan or task file; `fileName` supplies the id and title when the file has none. */
export function parsePlanFile(fileName: string, text: string): PlanFileEntry {
  const { meta, body } = frontMatter(text);
  const stem = fileName.replace(/\.md$/i, '');
  const id = str(meta['id']) || ID_IN_NAME.exec(stem)?.[1]?.toUpperCase() || stem;
  const heading = /^#\s+(.+?)\s*$/m.exec(body)?.[1] ?? '';
  const title =
    str(meta['title']) || heading.replace(/^(?:PLAN|TASK)-\d+\s*[—–:-]\s*/i, '').trim() || stem;
  const status =
    str(meta['status']) ||
    /^\s*[*_]*status[*_]*\s*:\s*[*_]*\s*([^*_\n][^\n]*?)\s*[*_]*\s*$/im.exec(body)?.[1] ||
    'unknown';
  const isPlan = /^plan/i.test(id);
  const planId =
    str(meta['planId']) ||
    str(meta['plan']) ||
    str(meta['plan_id']) ||
    (isPlan ? '' : (/PLAN-\d+/i.exec(body)?.[0]?.toUpperCase() ?? ''));
  const listed = meta['tasks'];
  return {
    id,
    title,
    status,
    role: str(meta['role']),
    planId,
    tasks: Array.isArray(listed) ? listed : [],
  };
}

/** Everything found under one workspace folder. */
export interface PlanFiles {
  plans: PlanFileEntry[];
  tasks: PlanFileEntry[];
  /** Modification time (ms) of the newest file read, 0 when there was none. */
  newest: number;
}

/** Reads the plan and task markdown files of a folder; unreadable files and folders are skipped. */
export function readPlanFiles(root: string, fs: PlanFs): PlanFiles {
  const out: PlanFiles = { plans: [], tasks: [], newest: 0 };
  for (const kind of ['plans', 'tasks'] as const) {
    const seen = new Set<string>();
    for (const base of ['.claude', '.nexus']) {
      const dir = `${root}/${base}/${kind}`;
      let names: string[];
      try {
        names = fs.readdirSync(dir);
      } catch {
        continue;
      }
      for (const name of names.filter((n) => n.toLowerCase().endsWith('.md')).sort()) {
        try {
          const entry = parsePlanFile(name, fs.readFileSync(`${dir}/${name}`));
          out.newest = Math.max(out.newest, fs.statSync(`${dir}/${name}`).mtimeMs);
          if (!seen.has(entry.id)) {
            seen.add(entry.id);
            out[kind].push(entry);
          }
        } catch {
          // one unreadable file must not hide the others
        }
      }
    }
  }
  return out;
}

const naturalKey = (id: string): [string, number] => {
  const m = /^(.*?)(\d+)\s*$/.exec(id);
  return m ? [m[1]!, Number(m[2])] : [id, 0];
};
const natural = (a: string, b: string): number => {
  const [pa, na] = naturalKey(a);
  const [pb, nb] = naturalKey(b);
  return pa === pb ? na - nb : pa < pb ? -1 : 1;
};

const asTask = (t: PlanFileEntry): OrchestratorTask => ({
  id: t.id,
  title: t.title,
  status: t.status,
  role: t.role,
});

/**
 * Combines plans and tasks from files with the plans of an orchestrator.json (which keep what
 * they say) and picks the active plan: the one the JSON names, else one marked active, else the
 * newest plan that is not finished.
 */
export function assemblePlans(
  filePlans: PlanFileEntry[],
  fileTasks: PlanFileEntry[],
  activePlanId: string | null,
  jsonPlans: OrchestratorPlan[],
): { allPlans: OrchestratorPlan[]; activePlan: OrchestratorPlan | null } {
  const allPlans: OrchestratorPlan[] = jsonPlans.map((p) => ({ ...p, tasks: [...p.tasks] }));
  const listed = new Map<string, string[]>();
  for (const fp of [...filePlans].sort((a, b) => natural(a.id, b.id))) {
    listed.set(fp.id, fp.tasks);
    if (!allPlans.some((p) => p.id === fp.id)) {
      allPlans.push({ id: fp.id, title: fp.title, status: fp.status, tasks: [] });
    }
  }
  const taskById = new Map(fileTasks.map((t) => [t.id, t]));
  for (const plan of allPlans) {
    const wanted = new Set(listed.get(plan.id) ?? []);
    for (const t of fileTasks) if (t.planId === plan.id) wanted.add(t.id);
    const have = new Set(plan.tasks.map((t) => t.id));
    for (const id of [...wanted].filter((id) => !have.has(id)).sort(natural)) {
      const t = taskById.get(id);
      plan.tasks.push(t ? asTask(t) : { id, title: id, status: 'unknown', role: '' });
    }
  }
  const active =
    allPlans.find((p) => p.id === activePlanId) ??
    allPlans.find((p) => ACTIVE.test(p.status.trim())) ??
    [...allPlans].reverse().find((p) => !DONE.test(p.status.trim())) ??
    null;
  return { allPlans, activePlan: active };
}
