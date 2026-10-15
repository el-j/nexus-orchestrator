/**
 * workspaceScanner.ts — Scans open workspace folders for Nexus orchestrator
 * state files and exposes their content for the sidebar tree view.
 */

import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import { assemblePlans, scanPlanFiles, type PlanFilesConfig, type PlanFs } from './planFiles';

/** The real file system, as the plan-file reader sees it. */
const realFs: PlanFs = {
  readdirSync: (dir) =>
    fs.readdirSync(dir, { withFileTypes: true }).map((e) => ({
      name: e.name,
      isDirectory: e.isDirectory(),
    })),
  readFileSync: (p) => fs.readFileSync(p, 'utf8'),
  statSync: (p) => fs.statSync(p),
};

/** The `planFiles` block of the project's .nexus/policy.json, or {} when absent or unreadable. */
export function readPolicyPlanFiles(root: string): PlanFilesConfig {
  try {
    const raw: unknown = JSON.parse(
      fs.readFileSync(path.join(root, '.nexus', 'policy.json'), 'utf8'),
    );
    const block = (raw as { planFiles?: PlanFilesConfig } | null)?.planFiles;
    return block && typeof block === 'object' ? block : {};
  } catch {
    return {};
  }
}

/**
 * Where this folder keeps plans and tasks: the project's policy.json plus the user's
 * `nexus.planFiles.*` settings. The folder lists add up; the search of the whole project is on
 * unless either one turns it off.
 */
export function planFilesConfig(root: string): Required<PlanFilesConfig> {
  const policy = readPolicyPlanFiles(root);
  const settings = vscode.workspace.getConfiguration('nexus.planFiles');
  const list = (a: unknown, b: unknown): string[] => [
    ...new Set([...(Array.isArray(a) ? a : []), ...(Array.isArray(b) ? b : [])].map(String)),
  ];
  return {
    plans: list(policy.plans, settings.get<string[]>('plans', [])),
    tasks: list(policy.tasks, settings.get<string[]>('tasks', [])),
    discover: policy.discover !== false && settings.get<boolean>('discover', true) !== false,
  };
}

/** A single task entry extracted from orchestrator.json */
export interface OrchestratorTask {
  id: string;
  title: string;
  status: string;
  role: string;
}

/** A single plan entry extracted from orchestrator.json */
export interface OrchestratorPlan {
  id: string;
  title: string;
  status: string;
  tasks: OrchestratorTask[];
}

/** The parsed state of one workspace folder's orchestrator.json */
export interface WorkspaceOrchestration {
  /** Absolute path to the workspace folder root */
  folderPath: string;
  /** Human-readable folder name (last path segment) */
  folderName: string;
  /** Current active plan, or null */
  activePlan: OrchestratorPlan | null;
  /** All plans, newest first (by array position in the JSON) */
  allPlans: OrchestratorPlan[];
  /** When the file was last modified */
  lastModified: Date;
  /** Project-relative folders the plan and task files were read from */
  sources: string[];
}

/**
 * Scans workspace folders for Nexus plans: .claude/orchestrator.json when there is one, and the
 * plan and task markdown files (.claude/plans, .claude/tasks, the same under .nexus) that agents
 * write whether or not they maintain the JSON.
 */
export class WorkspaceScanner implements vscode.Disposable {
  private readonly _onDidChange = new vscode.EventEmitter<void>();
  readonly onDidChange = this._onDidChange.event;

  private readonly watchers: vscode.FileSystemWatcher[] = [];
  private orchestrations: WorkspaceOrchestration[] = [];

  constructor(private readonly context: vscode.ExtensionContext) {}

  /** Scans all workspace folders and starts file watchers. Call once at activation. */
  start(): void {
    this.scan();
    this.startWatchers();

    // Re-scan when workspace folders are added/removed or the plan-folder settings change
    const rebuild = (): void => {
      this.disposeWatchers();
      this.scan();
      this.startWatchers();
      this._onDidChange.fire();
    };
    this.context.subscriptions.push(
      vscode.workspace.onDidChangeWorkspaceFolders(rebuild),
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration('nexus.planFiles')) rebuild();
      }),
    );
  }

  /** Returns the most recently scanned orchestrations. */
  getOrchestrations(): WorkspaceOrchestration[] {
    return this.orchestrations;
  }

  /** Re-reads the orchestrator.json and plan/task files of every folder synchronously. */
  scan(): void {
    this.orchestrations = [];
    const folders = vscode.workspace.workspaceFolders ?? [];
    for (const folder of folders) {
      const root = folder.uri.fsPath;
      const filePath = path.join(root, '.claude', 'orchestrator.json');
      let parsed: ReturnType<typeof parseOrchestratorFile> = null;
      let modified = 0;
      if (fs.existsSync(filePath)) {
        try {
          parsed = parseOrchestratorFile(fs.readFileSync(filePath, 'utf8'));
          if (parsed) modified = fs.statSync(filePath).mtime.getTime();
        } catch (e) {
          console.error('WorkspaceScanner: file I/O failed:', e);
        }
      }
      const found = scanPlanFiles(root, realFs, planFilesConfig(root));
      if (!parsed && found.plans.length === 0 && found.orphans.length === 0) continue;
      const { allPlans, activePlan } = assemblePlans(
        found,
        parsed?.activePlanId ?? null,
        parsed?.allPlans ?? [],
      );
      this.orchestrations.push({
        folderPath: root,
        folderName: path.basename(root),
        activePlan,
        allPlans,
        lastModified: new Date(Math.max(modified, found.newest)),
        sources: found.folders,
      });
    }
  }

  private startWatchers(): void {
    const folders = vscode.workspace.workspaceFolders ?? [];
    const refresh = (): void => {
      this.scan();
      this._onDidChange.fire();
    };
    for (const folder of folders) {
      // The JSON and policy, the default plan and task folders, any PLAN-n / TASK-n file, and the
      // folders this project configured.
      const cfg = planFilesConfig(folder.uri.fsPath);
      const patterns = [
        '{.claude,.nexus}/{orchestrator.json,policy.json,plans/**/*.md,tasks/**/*.md}',
        '**/{PLAN,TASK}-*.md',
        ...[...cfg.plans, ...cfg.tasks].map((d) => `${d}/**/*.md`),
      ];
      for (const pattern of patterns) {
        const watcher = vscode.workspace.createFileSystemWatcher(
          new vscode.RelativePattern(folder, pattern),
        );
        watcher.onDidChange(refresh);
        watcher.onDidCreate(refresh);
        watcher.onDidDelete(refresh);
        this.context.subscriptions.push(watcher);
        this.watchers.push(watcher);
      }
    }
  }

  private disposeWatchers(): void {
    for (const w of this.watchers) {
      w.dispose();
    }
    this.watchers.length = 0;
  }

  dispose(): void {
    this.disposeWatchers();
    this._onDidChange.dispose();
  }
}

/**
 * Parses a raw orchestrator.json string.
 * Returns null if the file is not recognized as a valid orchestrator file.
 * This is intentionally lenient — missing fields default gracefully.
 */
function parseOrchestratorFile(raw: string):
  | (Omit<WorkspaceOrchestration, 'folderPath' | 'folderName' | 'lastModified' | 'sources'> & {
      activePlanId: string | null;
    })
  | null {
  let json: Record<string, unknown>;
  try {
    const parsed: unknown = JSON.parse(raw);
    // "null", "42" or a bare array are valid JSON but not an orchestrator file
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) return null;
    json = parsed as Record<string, unknown>;
  } catch {
    return null;
  }
  // Minimal validation: must have a 'plans' or 'tasks' key
  if (!json.plans && !json.tasks) return null;

  const rawPlans = (json.plans as Record<string, RawPlan> | undefined) ?? {};
  const rawTasks = (json.tasks as Record<string, RawTask> | undefined) ?? {};

  const allPlans: OrchestratorPlan[] = Object.values(rawPlans).map((p) => ({
    id: String(p.id ?? ''),
    title: String(p.title ?? ''),
    status: String(p.status ?? 'unknown'),
    tasks: ((p.tasks ?? []) as unknown[]).map((tid) => {
      const key = String(tid);
      const t = rawTasks[key];
      return t
        ? {
            id: String(t.id ?? key),
            title: String(t.title ?? key),
            status: String(t.status ?? 'unknown'),
            role: String(t.role ?? ''),
          }
        : { id: key, title: key, status: 'unknown', role: '' };
    }),
  }));

  const activePlanId = (json.activePlanId as string | null | undefined) ?? null;
  const activePlan = allPlans.find((p) => p.id === activePlanId) ?? null;

  return { activePlan, allPlans, activePlanId };
}

interface RawPlan {
  id?: unknown;
  title?: unknown;
  status?: unknown;
  tasks?: unknown[];
}

interface RawTask {
  id?: unknown;
  title?: unknown;
  status?: unknown;
  role?: unknown;
}
