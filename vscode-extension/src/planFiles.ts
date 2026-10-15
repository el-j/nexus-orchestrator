/**
 * planFiles.ts — Finds the plans and tasks an agent wrote as markdown, wherever the project
 * keeps them, and links tasks to plans. The rules are the same as in the daemon
 * (internal/core/services/planfiles.go); both are tested against testdata/planfiles/cases.json.
 *
 * Where files are searched, in this order (the first file with an id wins):
 *  1. the folders the project configured (plans and tasks, whatever the files are called),
 *  2. .claude/{plans,tasks} and .nexus/{plans,tasks},
 *  3. unless discover is off: any markdown file whose name is PLAN-n or TASK-n, whose folder is
 *     called plans/plan or tasks/task, or whose front matter says `type: plan` or `type: task`.
 *
 * A task belongs to the plan it names (planId / plan), else to the plan that lists it in its
 * front matter, else to the plan whose name its folder carries (tasks/PLAN-2/TASK-5.md), else
 * to the first plan its text mentions, else to a plan whose text mentions the task.
 */

import * as path from 'path';
import type { OrchestratorPlan, OrchestratorTask } from './workspaceScanner';

/** One plan or task file, reduced to what the tree shows. */
export interface PlanFileEntry {
  /** The front matter id, else PLAN-n / TASK-n from the file name, else the file name. */
  id: string;
  title: string;
  status: string;
  role: string;
  /** Project-relative path with forward slashes. */
  rel: string;
  /** The plan a task names itself (front matter planId / plan / plan_id). */
  planRef: string;
  /** The task ids a plan lists in its front matter. */
  listed: string[];
  /** The first PLAN-n a task's text mentions. */
  bodyPlan: string;
  /** The TASK-n a plan's text mentions. */
  mentioned: string[];
  stem: string;
}

/** A plan with the tasks linked to it. */
export interface PlanNode extends PlanFileEntry {
  tasks: PlanFileEntry[];
}

/** Everything found under one project. */
export interface PlanScan {
  plans: PlanNode[];
  /** Tasks that no plan claims. */
  orphans: PlanFileEntry[];
  /** Project-relative folders the files came from, sorted. */
  folders: string[];
  /** Modification time (ms) of the newest file read, 0 when there was none. */
  newest: number;
}

/** Where plans and tasks live; every part is optional (see the `planFiles` block of .nexus/policy.json). */
export interface PlanFilesConfig {
  plans?: string[];
  tasks?: string[];
  discover?: boolean;
}

/** The slice of `fs` the scan needs; tests pass a fake. */
export interface PlanFs {
  readdirSync(dir: string): Array<{ name: string; isDirectory: boolean }>;
  readFileSync(path: string): string;
  statSync(path: string): { mtimeMs: number; size: number };
}

const MAX_FILE_BYTES = 256 << 10;
const MAX_WALK_DEPTH = 6;
const MAX_WALK_ENTRIES = 20000;
const MAX_FOLDER_DEPTH = 3;
const SKIP_DIRS = new Set([
  'node_modules',
  'vendor',
  'build',
  'dist',
  'target',
  '__pycache__',
  'coverage',
  'out',
  'tmp',
]);
const DONE = /^(done|completed?|closed|merged|shipped|cancell?ed)$/i;
const ACTIVE = /^(active|in[-_ ]?progress|doing|started|wip)$/i;

function unquote(s: string): string {
  return s
    .trim()
    .replace(/^(['"])(.*)\1$/, '$2')
    .trim();
}

function frontMatter(text: string): {
  meta: Record<string, string>;
  lists: Record<string, string[]>;
  body: string;
} {
  const meta: Record<string, string> = {};
  const lists: Record<string, string[]> = {};
  const lines = text.replace(/\r\n/g, '\n').split('\n');
  if (lines[0]?.trim() !== '---') return { meta, lists, body: text };
  let key = '';
  let i = 1;
  for (; i < lines.length && lines[i]!.trim() !== '---'; i++) {
    const line = lines[i]!;
    const item = /^[ \t]+-\s+(.*)$/.exec(line);
    if (item && key) {
      (lists[key] ??= []).push(unquote(item[1]!));
      continue;
    }
    const kv = /^([^:\s][^:]*?)\s*:\s*(.*)$/.exec(line);
    if (!kv || /\s/.test(kv[1]!)) continue;
    key = kv[1]!;
    const v = kv[2]!.trim();
    if (v.startsWith('[') && v.endsWith(']')) {
      lists[key] = v.slice(1, -1).split(',').map(unquote).filter(Boolean);
    } else {
      meta[key] = unquote(v);
    }
  }
  // An unterminated header (no closing ---) is read as far as it goes; the rest is the body.
  return { meta, lists, body: lines.slice(i + 1).join('\n') };
}

const isPlanDir = (n: string): boolean => /^(plans?)$/i.test(n);
const isTaskDir = (n: string): boolean => /^(tasks?)$/i.test(n);

/**
 * Parses one file. `kind` is "plan" or "task" when the caller knows it; without it the kind is
 * decided from the front matter type, the file name and the folder name. Returns null when the
 * file is neither.
 */
export function parsePlanFile(
  rel: string,
  text: string,
  kind?: 'plan' | 'task',
): { entry: PlanFileEntry; kind: 'plan' | 'task' } | null {
  const { meta, lists, body } = frontMatter(text);
  const stem = path.posix.basename(rel).replace(/\.[^.]*$/, '');
  const idFromName = /^((?:PLAN|TASK)-\d+)/i.exec(stem)?.[1]?.toUpperCase() ?? '';
  if (!kind) {
    const type = (meta['type'] ?? '').toLowerCase();
    const parent = path.posix.basename(path.posix.dirname(rel));
    if (type === 'plan' || type === 'task') kind = type;
    else if (idFromName.startsWith('PLAN')) kind = 'plan';
    else if (idFromName.startsWith('TASK')) kind = 'task';
    else if (isPlanDir(parent)) kind = 'plan';
    else if (isTaskDir(parent)) kind = 'task';
    else return null;
  }
  const heading = /^#\s+(.+?)\s*$/m.exec(body)?.[1] ?? '';
  const entry: PlanFileEntry = {
    id: meta['id'] || idFromName || stem,
    title: meta['title'] || heading.replace(/^(?:PLAN|TASK)-\d+\s*[—–:-]\s*/i, '').trim() || stem,
    status:
      meta['status'] ||
      /^\s*[*_]*status[*_]*\s*:\s*[*_]*\s*([^*_\n][^\n]*?)\s*[*_]*\s*$/im.exec(body)?.[1] ||
      'unknown',
    role: meta['role'] ?? '',
    rel,
    planRef: '',
    listed: [],
    bodyPlan: '',
    mentioned: [],
    stem,
  };
  if (kind === 'plan') {
    entry.listed = lists['tasks'] ?? [];
    entry.mentioned = [...new Set((body.match(/TASK-\d+/gi) ?? []).map((m) => m.toUpperCase()))];
  } else {
    entry.planRef = meta['planId'] || meta['plan'] || meta['plan_id'] || '';
    entry.bodyPlan = (/PLAN-\d+/i.exec(body)?.[0] ?? '').toUpperCase();
  }
  return { entry, kind };
}

function normalizeFolders(dirs: string[] | undefined): string[] {
  const out: string[] = [];
  for (const d of dirs ?? []) {
    const clean = path.posix.normalize(
      d
        .trim()
        .replace(/\\/g, '/')
        .replace(/^\/+|\/+$/g, ''),
    );
    if (clean !== '.' && clean !== '') out.push(clean);
  }
  return out;
}

function markdownUnder(root: string, dir: string, fs: PlanFs): string[] {
  const out: string[] = [];
  const walk = (rel: string, depth: number): void => {
    let entries: Array<{ name: string; isDirectory: boolean }>;
    try {
      entries = fs.readdirSync(`${root}/${rel}`);
    } catch {
      return;
    }
    for (const e of entries) {
      if (e.isDirectory) {
        if (depth < MAX_FOLDER_DEPTH) walk(`${rel}/${e.name}`, depth + 1);
      } else if (e.name.toLowerCase().endsWith('.md')) {
        out.push(`${rel}/${e.name}`);
      }
    }
  };
  walk(dir, 0);
  return out.sort();
}

function discoverMarkdown(root: string, fs: PlanFs): string[] {
  const out: string[] = [];
  let visited = 0;
  const walk = (rel: string): void => {
    let entries: Array<{ name: string; isDirectory: boolean }>;
    try {
      entries = fs.readdirSync(rel ? `${root}/${rel}` : root);
    } catch {
      return;
    }
    for (const e of entries) {
      if (++visited > MAX_WALK_ENTRIES) return;
      const child = rel ? `${rel}/${e.name}` : e.name;
      if (e.isDirectory) {
        const hidden = e.name.startsWith('.') && !['.claude', '.nexus', '.github'].includes(e.name);
        if (hidden || SKIP_DIRS.has(e.name) || child.split('/').length - 1 >= MAX_WALK_DEPTH) {
          continue;
        }
        walk(child);
      } else {
        const low = e.name.toLowerCase();
        if (low.endsWith('.md') && !['readme.md', 'index.md', 'changelog.md'].includes(low)) {
          out.push(child);
        }
      }
    }
  };
  walk('');
  return out.sort();
}

const order = (id: string): number => {
  const m = /(\d+)\s*$/.exec(id);
  return m ? Number(m[1]) : 1 << 30;
};
const byId = (a: PlanFileEntry, b: PlanFileEntry): number =>
  order(a.id) - order(b.id) || (a.id < b.id ? -1 : 1);
const same = (a: string, b: string): boolean => a.toLowerCase() === b.toLowerCase();

/** Finds the plan and task files of a project and links tasks to plans. */
export function scanPlanFiles(root: string, fs: PlanFs, cfg: PlanFilesConfig = {}): PlanScan {
  const plansDirs = normalizeFolders(cfg.plans);
  const tasksDirs = normalizeFolders(cfg.tasks);
  const discover = cfg.discover !== false;
  const configured = plansDirs.length + tasksDirs.length > 0;

  const plans: PlanFileEntry[] = [];
  const tasks: PlanFileEntry[] = [];
  const seen = new Set<string>();
  const ids = new Set<string>();
  let newest = 0;
  const add = (rel: string, kind?: 'plan' | 'task'): void => {
    if (seen.has(rel)) return;
    seen.add(rel);
    try {
      const st = fs.statSync(`${root}/${rel}`);
      if (st.size > MAX_FILE_BYTES) return;
      const parsed = parsePlanFile(rel, fs.readFileSync(`${root}/${rel}`), kind);
      if (!parsed || ids.has(`${parsed.kind}:${parsed.entry.id}`)) return;
      ids.add(`${parsed.kind}:${parsed.entry.id}`);
      newest = Math.max(newest, st.mtimeMs);
      (parsed.kind === 'plan' ? plans : tasks).push(parsed.entry);
    } catch {
      // one unreadable file must not hide the others
    }
  };
  for (const d of plansDirs) markdownUnder(root, d, fs).forEach((r) => add(r, 'plan'));
  for (const d of tasksDirs) markdownUnder(root, d, fs).forEach((r) => add(r, 'task'));
  if (discover || !configured) {
    for (const base of ['.claude', '.nexus']) {
      markdownUnder(root, `${base}/plans`, fs).forEach((r) => add(r, 'plan'));
      markdownUnder(root, `${base}/tasks`, fs).forEach((r) => add(r, 'task'));
    }
  }
  if (discover) discoverMarkdown(root, fs).forEach((r) => add(r));
  return { ...linkPlanFiles(plans, tasks), newest };
}

function linkPlanFiles(plans: PlanFileEntry[], tasks: PlanFileEntry[]): Omit<PlanScan, 'newest'> {
  plans.sort(byId);
  tasks.sort(byId);
  const nodes: PlanNode[] = plans.map((p) => ({ ...p, tasks: [] }));
  const byRef = (ref: string): number =>
    ref ? nodes.findIndex((p) => same(p.id, ref) || same(p.stem, ref)) : -1;
  const listedBy = (id: string): number =>
    nodes.findIndex((p) => p.listed.some((l) => same(l, id)));
  const mentionedBy = (id: string): number =>
    nodes.findIndex((p) => p.mentioned.some((m) => same(m, id)));
  const orphans: PlanFileEntry[] = [];
  for (const t of tasks) {
    let at = byRef(t.planRef);
    if (at < 0) at = listedBy(t.id);
    if (at < 0) at = byRef(path.posix.basename(path.posix.dirname(t.rel)));
    if (at < 0) at = byRef(t.bodyPlan);
    if (at < 0) at = mentionedBy(t.id);
    if (at < 0) orphans.push(t);
    else nodes[at]!.tasks.push(t);
  }
  // A task a plan lists or mentions but that has no file is still part of the plan.
  for (const n of nodes) {
    const have = new Set(n.tasks.map((t) => t.id.toUpperCase()));
    for (const id of [...n.listed, ...n.mentioned]) {
      if (have.has(id.toUpperCase()) || tasks.some((t) => same(t.id, id))) continue;
      have.add(id.toUpperCase());
      n.tasks.push({
        id,
        title: id,
        status: 'unknown',
        role: '',
        rel: '',
        planRef: '',
        listed: [],
        bodyPlan: '',
        mentioned: [],
        stem: id,
      });
    }
    n.tasks.sort((a, b) => order(a.id) - order(b.id));
  }
  const folders = [...new Set([...plans, ...tasks].map((f) => path.posix.dirname(f.rel)))].sort();
  return { plans: nodes, orphans, folders };
}

/** True when the status marks a finished plan or task. */
export const isDone = (status: string): boolean => DONE.test(status.trim());

/** The plan one is working on: one marked active, else the newest that is not finished. */
export function pickActive<T extends { status: string }>(plans: T[]): T | null {
  return (
    plans.find((p) => ACTIVE.test(p.status.trim())) ??
    [...plans].reverse().find((p) => !DONE.test(p.status.trim())) ??
    null
  );
}

const asTask = (t: PlanFileEntry): OrchestratorTask => ({
  id: t.id,
  title: t.title,
  status: t.status,
  role: t.role,
});

/** The id of the pseudo plan that collects tasks no plan claims. */
export const UNASSIGNED_ID = 'UNASSIGNED';

/**
 * Combines the plans found in files with the plans of an orchestrator.json (which keep what they
 * say; the files add what the JSON does not list) and picks the active plan: the one the JSON
 * names, else one marked active, else the newest that is not finished. Tasks no plan claims are
 * collected in a last pseudo plan "Unassigned tasks", which is never the active one.
 */
export function assemblePlans(
  scan: PlanScan,
  activePlanId: string | null,
  jsonPlans: OrchestratorPlan[],
): { allPlans: OrchestratorPlan[]; activePlan: OrchestratorPlan | null } {
  const allPlans: OrchestratorPlan[] = jsonPlans.map((p) => ({ ...p, tasks: [...p.tasks] }));
  for (const node of scan.plans) {
    let plan = allPlans.find((p) => p.id === node.id);
    if (!plan) {
      plan = { id: node.id, title: node.title, status: node.status, tasks: [] };
      allPlans.push(plan);
    }
    const have = new Set(plan.tasks.map((t) => t.id));
    for (const t of node.tasks) if (!have.has(t.id)) plan.tasks.push(asTask(t));
  }
  // A task no file plan claims may still belong to a plan only the JSON knows.
  const unassigned = scan.orphans.filter((t) => {
    const owner =
      allPlans.find((p) => p.tasks.some((x) => same(x.id, t.id))) ??
      [t.planRef, path.posix.basename(path.posix.dirname(t.rel)), t.bodyPlan]
        .map((ref) => (ref ? allPlans.find((p) => same(p.id, ref)) : undefined))
        .find(Boolean);
    if (owner && !owner.tasks.some((x) => same(x.id, t.id))) owner.tasks.push(asTask(t));
    return !owner;
  });
  const activePlan = allPlans.find((p) => p.id === activePlanId) ?? pickActive(allPlans);
  if (unassigned.length > 0) {
    allPlans.push({
      id: UNASSIGNED_ID,
      title: 'Unassigned tasks',
      status: 'unknown',
      tasks: unassigned.map(asTask),
    });
  }
  return { allPlans, activePlan };
}
