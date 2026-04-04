/**
 * submitTask.ts — Implementation of the `nexus.submitTask` command (TASK-133).
 */

import * as path from "path";
import * as vscode from "vscode";
import { NexusClient, Provider, TaskStatus } from "../nexusClient";
import { logNexusActivity, rememberQueuedTask, showNexusActivityLog } from "../activityLog";

const TERMINAL_STATUSES: ReadonlySet<TaskStatus> = new Set<TaskStatus>([
  "COMPLETED",
  "FAILED",
  "CANCELLED",
  "TOO_LARGE",
  "NO_PROVIDER",
]);

const POLL_INTERVAL_MS = 2_000;
const POLL_TIMEOUT_MS = 5 * 60 * 1_000; // 5 minutes

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isDaemonUnreachable(err: unknown): boolean {
  if (err instanceof TypeError) return true;
  const msg = err instanceof Error ? err.message : String(err);
  return (
    msg.includes("ECONNREFUSED") ||
    msg.includes("fetch failed") ||
    msg.includes("Failed to fetch")
  );
}

interface ProviderPickItem extends vscode.QuickPickItem {
  providerHint?: string;
  modelId?: string;
}

interface ContextPickItem extends vscode.QuickPickItem {
  contextFiles: string[];
}

interface SubmissionDraft {
  instruction: string;
  projectPath: string;
  targetFile?: string;
  providerHint?: string;
  modelId?: string;
  contextFiles: string[];
  routeLabel: string;
}

function shortTaskId(taskId: string): string {
  return taskId.replace(/-/g, "").slice(0, 8);
}

function getWorkspaceDefaults(): { providerHint?: string; modelId?: string } {
  const cfg = vscode.workspace.getConfiguration("nexus");
  const providerHint = cfg.get<string>("defaultProvider")?.trim();
  const modelId = cfg.get<string>("defaultModel")?.trim();
  return {
    providerHint: providerHint || undefined,
    modelId: modelId || undefined,
  };
}

function buildProviderItems(
  providers: Provider[],
  defaults: { providerHint?: string; modelId?: string }
): ProviderPickItem[] {
  const items: ProviderPickItem[] = [
    defaults.providerHint || defaults.modelId
      ? {
          label: `Workspace default (${defaults.providerHint ?? "Auto"}${defaults.modelId ? ` / ${defaults.modelId}` : ""})`,
          description: "Use the workspace default route",
          providerHint: defaults.providerHint,
          modelId: defaults.modelId,
        }
      : {
          label: "Auto (let Nexus choose)",
          description: "Use the default provider and model",
        },
  ];

  items.push({
    label: "Auto (let Nexus choose)",
    description: "Use the daemon's current routing decision",
  });

  for (const p of providers) {
    if (p.models && p.models.length > 0) {
      for (const m of p.models) {
        items.push({ label: `${p.name} / ${m}`, providerHint: p.name, modelId: m });
      }
    } else {
      items.push({ label: p.name, providerHint: p.name });
    }
  }

  return items;
}

function workspaceRelativePath(wsRoot: string, filePath: string): string | undefined {
  if (!filePath.startsWith(wsRoot)) {
    return undefined;
  }
  return path.relative(wsRoot, filePath);
}

function collectWorkspacePaths(wsRoot: string, editor: vscode.TextEditor | undefined): ContextPickItem[] {
  const items: ContextPickItem[] = [];
  const activePath = editor ? workspaceRelativePath(wsRoot, editor.document.uri.fsPath) : undefined;

  if (activePath) {
    items.push({
      label: "Active file",
      description: activePath,
      contextFiles: [activePath],
      picked: true,
    });
  }

  const modifiedPaths = Array.from(
    new Set(
      vscode.workspace.textDocuments
        .filter((doc) => doc.isDirty && doc.uri.scheme === "file")
        .map((doc) => workspaceRelativePath(wsRoot, doc.uri.fsPath))
        .filter((value): value is string => Boolean(value))
    )
  ).filter((candidate) => candidate !== activePath);

  if (modifiedPaths.length > 0) {
    items.push({
      label: `Modified workspace files (${modifiedPaths.length})`,
      description: modifiedPaths.join(", "),
      contextFiles: modifiedPaths,
    });
  }

  const openEditorPaths = Array.from(
    new Set(
      vscode.workspace.textDocuments
        .filter((doc) => !doc.isUntitled && doc.uri.scheme === "file")
        .map((doc) => workspaceRelativePath(wsRoot, doc.uri.fsPath))
        .filter((value): value is string => Boolean(value))
    )
  ).filter((candidate) => candidate !== activePath && !modifiedPaths.includes(candidate));

  if (openEditorPaths.length > 0) {
    items.push({
      label: `Open editors (${openEditorPaths.length})`,
      description: openEditorPaths.join(", "),
      contextFiles: openEditorPaths,
    });
  }

  return items;
}

async function pickContextFiles(
  wsRoot: string,
  editor: vscode.TextEditor | undefined
): Promise<string[] | undefined> {
  const items = collectWorkspacePaths(wsRoot, editor);
  if (items.length === 0) {
    return [];
  }

  const picked = await vscode.window.showQuickPick(items, {
    canPickMany: true,
    ignoreFocusOut: true,
    title: "Nexus: Choose Context To Queue",
    placeHolder: "Select the workspace files Nexus should use as context",
  });
  if (!picked) {
    return undefined;
  }

  return Array.from(new Set(picked.flatMap((item) => item.contextFiles)));
}

async function promptForTargetFile(
  wsRoot: string,
  editor: vscode.TextEditor | undefined,
  mode: "manual" | "current-context"
): Promise<string | undefined> {
  if (editor) {
    const relative = workspaceRelativePath(wsRoot, editor.document.uri.fsPath);
    if (relative) {
      return relative;
    }
  }

  return vscode.window.showInputBox({
    prompt:
      mode === "current-context"
        ? "Target file path (relative to project root)"
        : "Target file path (relative to project root)",
    ignoreFocusOut: true,
  });
}

async function selectRoute(client: NexusClient): Promise<ProviderPickItem | undefined> {
  let providers: Provider[] = [];
  try {
    providers = await client.getProviders();
  } catch {
    // non-fatal — fall through with workspace defaults and auto route
  }

  return vscode.window.showQuickPick(buildProviderItems(providers, getWorkspaceDefaults()), {
    placeHolder: "Select provider and model",
    ignoreFocusOut: true,
  });
}

async function collectSubmissionDraft(
  client: NexusClient,
  mode: "manual" | "current-context"
): Promise<SubmissionDraft | undefined> {
  const editor = vscode.window.activeTextEditor;
  const instruction = await vscode.window.showInputBox({
    prompt: mode === "current-context" ? "Task instruction" : "Task instruction",
    placeHolder:
      mode === "current-context"
        ? "Describe what Nexus should change; context is selected next"
        : "Describe the task to queue",
    ignoreFocusOut: true,
  });
  if (instruction === undefined) {
    return undefined;
  }

  const workspaceFolders = vscode.workspace.workspaceFolders;
  if (!workspaceFolders || workspaceFolders.length === 0) {
    vscode.window.showErrorMessage("Open a folder to submit a task");
    return undefined;
  }

  const wsRoot = workspaceFolders[0].uri.fsPath;
  const targetFile = await promptForTargetFile(wsRoot, editor, mode);
  if (targetFile === undefined) {
    return undefined;
  }

  const contextFiles =
    mode === "current-context" ? await pickContextFiles(wsRoot, editor) : [];
  if (contextFiles === undefined) {
    return undefined;
  }

  const pickedRoute = await selectRoute(client);
  if (pickedRoute === undefined) {
    return undefined;
  }

  return {
    instruction,
    projectPath: wsRoot,
    targetFile,
    providerHint: pickedRoute.providerHint,
    modelId: pickedRoute.modelId,
    contextFiles,
    routeLabel: pickedRoute.label,
  };
}

async function confirmDraft(draft: SubmissionDraft): Promise<boolean> {
  const action = await vscode.window.showInformationMessage(
    `Queue in Nexus? Target: ${draft.targetFile || "—"}; Context files: ${draft.contextFiles.length}; Route: ${draft.routeLabel}`,
    { modal: true },
    "Queue Task"
  );
  return action === "Queue Task";
}

async function runSubmissionFlow(
  client: NexusClient,
  daemonUrl: string,
  mode: "manual" | "current-context"
): Promise<void> {
  const draft = await collectSubmissionDraft(client, mode);
  if (!draft) {
    return;
  }

  if (!(await confirmDraft(draft))) {
    return;
  }

  logNexusActivity(
    "queue",
    `${mode === "current-context" ? "explicit queue" : "manual task"} requested: target=${draft.targetFile || "—"}, context=${draft.contextFiles.length}, route=${draft.routeLabel}`
  );

  // 6. Submit the task
  let taskId: string;
  try {
    const task = await client.submitTask({
      instruction: draft.instruction,
      projectPath: draft.projectPath,
      targetFile: draft.targetFile,
      providerHint: draft.providerHint,
      modelId: draft.modelId,
      contextFiles: draft.contextFiles,
    });
    taskId = task.id;
    rememberQueuedTask(taskId);
    logNexusActivity("queue", `submitted from vscode command: #${shortTaskId(taskId)}`);
  } catch (err) {
    if (isDaemonUnreachable(err)) {
      vscode.window.showErrorMessage(
        `Nexus daemon is not running at ${daemonUrl}. Start it with 'nexus-daemon' or the desktop app.`
      );
      logNexusActivity("queue", `submission failed: daemon unreachable at ${daemonUrl}`);
    } else {
      const msg = err instanceof Error ? err.message : String(err);
      vscode.window.showErrorMessage(`Nexus: failed to submit task — ${msg}`);
      logNexusActivity("queue", `submission failed: ${msg}`);
    }
    return;
  }

  // 7–11. Progress notification with polling loop
  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: "Nexus: Task submitted…",
      cancellable: true,
    },
    async (progress, token) => {
      let userCancelled = false;
      let lastStatus: TaskStatus | undefined;

      token.onCancellationRequested(async () => {
        userCancelled = true;
        try {
          await client.cancelTask(taskId);
          logNexusActivity("queue", `task #${shortTaskId(taskId)} cancelled from VS Code`);
        } catch {
          // best-effort cancel — ignore errors
        }
      });

      const start = Date.now();

      while (true) {
        // 8. Timeout guard
        if (Date.now() - start > POLL_TIMEOUT_MS) {
          vscode.window.showWarningMessage(
            "Nexus: task polling timed out after 5 minutes"
          );
          return;
        }

        if (userCancelled) {
          vscode.window.showInformationMessage("Nexus: Task cancelled");
          return;
        }

        await sleep(POLL_INTERVAL_MS);

        // Re-check after sleeping (cancellation may have fired during sleep)
        if (userCancelled) {
          vscode.window.showInformationMessage("Nexus: Task cancelled");
          return;
        }

        let task;
        try {
          task = await client.getTask(taskId);
        } catch {
          // transient network error — keep polling
          continue;
        }

        progress.report({ message: task.status });
        if (task.status !== lastStatus) {
          lastStatus = task.status;
          logNexusActivity("queue", `task #${shortTaskId(taskId)} -> ${task.status}`);
        }

        if (!TERMINAL_STATUSES.has(task.status)) {
          continue;
        }

        // Terminal state reached
        switch (task.status) {
          case "COMPLETED": {
            // 9. Show info + "Open File" action
            const absTarget = path.isAbsolute(task.targetFile)
              ? task.targetFile
              : path.join(task.projectPath, task.targetFile);
            const action = await vscode.window.showInformationMessage(
              `Queued in Nexus: #${shortTaskId(task.id)}`,
              "Open Queue",
              "Open File",
              "Show Activity Log"
            );
            if (action === "Open Queue") {
              await vscode.commands.executeCommand("nexus.viewQueue");
            }
            if (action === "Open File") {
              await vscode.window.showTextDocument(vscode.Uri.file(absTarget));
            }
            if (action === "Show Activity Log") {
              showNexusActivityLog();
            }
            break;
          }
          case "FAILED":
            // 10. Error notification
            vscode.window.showErrorMessage(
              `Nexus: task failed — ${task.logs ?? "no details available"}`
            );
            logNexusActivity("queue", `task #${shortTaskId(task.id)} failed: ${task.logs ?? "no details available"}`);
            break;
          case "TOO_LARGE":
            vscode.window.showErrorMessage(
              "Nexus: task rejected — input too large for the selected model"
            );
            logNexusActivity("queue", `task #${shortTaskId(task.id)} rejected as TOO_LARGE`);
            break;
          case "NO_PROVIDER":
            vscode.window.showErrorMessage(
              "Nexus: task failed — no LLM provider is currently available"
            );
            logNexusActivity("queue", `task #${shortTaskId(task.id)} failed: no provider`);
            break;
          case "CANCELLED":
            // 11. Cancelled externally (e.g. from daemon/CLI)
            vscode.window.showInformationMessage("Nexus: Task cancelled");
            logNexusActivity("queue", `task #${shortTaskId(task.id)} cancelled externally`);
            break;
        }
        return;
      }
    }
  );
}

export async function sendCurrentContextCommand(
  client: NexusClient,
  daemonUrl: string
): Promise<void> {
  await runSubmissionFlow(client, daemonUrl, "current-context");
}

export async function submitTaskCommand(
  client: NexusClient,
  daemonUrl: string
): Promise<void> {
  await runSubmissionFlow(client, daemonUrl, "manual");
}
