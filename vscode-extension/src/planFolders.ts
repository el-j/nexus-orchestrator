/**
 * planFolders.ts: the command "Nexus: Choose Plan and Task Folders". Plans and tasks are found by
 * themselves; this lets the user say where they live when the project has another layout. The
 * answer is written to the `planFiles` block of the project's .nexus/policy.json, so the daemon
 * and every other tool read the same folders. (The `nexus.planFiles.*` settings do the same for one
 * user.)
 */
import * as fs from 'fs';
import * as path from 'path';
import * as vscode from 'vscode';
import type { PlanFilesConfig } from './planFiles';
import { readPolicyPlanFiles, type WorkspaceScanner } from './workspaceScanner';

type Action = 'addPlans' | 'addTasks' | 'remove' | 'toggle' | 'open';

/** Applies `change` to the planFiles block of <root>/.nexus/policy.json; false when that file is not usable JSON. */
function updatePolicy(root: string, change: (cfg: PlanFilesConfig) => PlanFilesConfig): boolean {
  const file = path.join(root, '.nexus', 'policy.json');
  let policy: Record<string, unknown> = {};
  if (fs.existsSync(file)) {
    try {
      const parsed: unknown = JSON.parse(fs.readFileSync(file, 'utf8'));
      if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error();
      policy = parsed as Record<string, unknown>;
    } catch {
      void vscode.window.showErrorMessage('.nexus/policy.json is not valid JSON; fix it first.');
      return false;
    }
  }
  policy['planFiles'] = change(readPolicyPlanFiles(root));
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(policy, null, 2) + '\n');
  return true;
}

async function addFolder(
  root: string,
  folder: vscode.WorkspaceFolder,
  kind: 'plans' | 'tasks',
): Promise<boolean> {
  const picked = await vscode.window.showOpenDialog({
    canSelectFolders: true,
    canSelectFiles: false,
    canSelectMany: false,
    defaultUri: folder.uri,
    openLabel: kind === 'plans' ? 'Use as plan folder' : 'Use as task folder',
  });
  if (!picked?.[0]) return false;
  const rel = path.relative(root, picked[0].fsPath).split(path.sep).join('/');
  if (rel === '' || rel.startsWith('..') || path.isAbsolute(rel)) {
    void vscode.window.showErrorMessage(
      'Choose a folder inside the project (not the project folder itself).',
    );
    return false;
  }
  return updatePolicy(root, (cfg) => ({
    ...cfg,
    [kind]: [...new Set([...(cfg[kind] ?? []), rel])],
  }));
}

async function run(scanner: WorkspaceScanner): Promise<void> {
  const folders = vscode.workspace.workspaceFolders ?? [];
  if (folders.length === 0) {
    void vscode.window.showInformationMessage(
      'Open a folder first: plan folders are set per project.',
    );
    return;
  }
  const folder = folders.length === 1 ? folders[0]! : await vscode.window.showWorkspaceFolderPick();
  if (!folder) return;
  const root = folder.uri.fsPath;
  const cfg = readPolicyPlanFiles(root);
  const found = scanner.getOrchestrations().find((o) => o.folderPath === root)?.sources ?? [];
  const configured = (cfg.plans?.length ?? 0) + (cfg.tasks?.length ?? 0);
  const whole = cfg.discover !== false;

  const pick = await vscode.window.showQuickPick<vscode.QuickPickItem & { id: Action }>(
    [
      { id: 'addPlans', label: '$(add) Add a plan folder…' },
      { id: 'addTasks', label: '$(add) Add a task folder…' },
      {
        id: 'remove',
        label: '$(trash) Remove a configured folder…',
        description: `${configured} configured`,
      },
      {
        id: 'toggle',
        label: `$(search) Search the whole project for PLAN-n / TASK-n files: ${whole ? 'on' : 'off'}`,
        description: whole ? 'select to turn off' : 'select to turn on',
      },
      { id: 'open', label: '$(json) Open .nexus/policy.json' },
    ],
    {
      placeHolder: `Plans and tasks are read from: ${found.join(', ') || 'nothing found yet'}`,
    },
  );
  if (!pick) return;

  let changed = false;
  if (pick.id === 'addPlans' || pick.id === 'addTasks') {
    changed = await addFolder(root, folder, pick.id === 'addPlans' ? 'plans' : 'tasks');
  } else if (pick.id === 'remove') {
    changed = await removeFolders(root, cfg);
  } else if (pick.id === 'toggle') {
    changed = updatePolicy(root, (c) => ({ ...c, discover: !whole }));
  } else {
    await openPolicy(root);
  }
  if (changed) void vscode.commands.executeCommand('nexus.refreshWorkspaceAgents');
}

async function removeFolders(root: string, cfg: PlanFilesConfig): Promise<boolean> {
  const items = [
    ...(cfg.plans ?? []).map((d) => ({ id: `plans:${d}`, label: d, description: 'plans' })),
    ...(cfg.tasks ?? []).map((d) => ({ id: `tasks:${d}`, label: d, description: 'tasks' })),
  ];
  if (items.length === 0) {
    void vscode.window.showInformationMessage(
      'No folders are configured; plans and tasks are found by themselves.',
    );
    return false;
  }
  const picked = await vscode.window.showQuickPick(items, {
    canPickMany: true,
    placeHolder: 'Folders to stop reading',
  });
  if (!picked) return false;
  const drop = new Set(picked.map((i) => i.id));
  return updatePolicy(root, (c) => ({
    ...c,
    plans: (c.plans ?? []).filter((d) => !drop.has(`plans:${d}`)),
    tasks: (c.tasks ?? []).filter((d) => !drop.has(`tasks:${d}`)),
  }));
}

async function openPolicy(root: string): Promise<void> {
  const file = path.join(root, '.nexus', 'policy.json');
  if (!fs.existsSync(file)) {
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, JSON.stringify({ planFiles: { plans: [], tasks: [] } }, null, 2) + '\n');
  }
  await vscode.window.showTextDocument(
    await vscode.workspace.openTextDocument(vscode.Uri.file(file)),
  );
}

/** Registers `nexus.configurePlanFolders`. */
export function registerPlanFolders(
  context: vscode.ExtensionContext,
  scanner: WorkspaceScanner,
): void {
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.configurePlanFolders', () => run(scanner)),
  );
}
