/**
 * submitTask.ts — Implementation of the `nexus.submitTask` command (TASK-133).
 */

import * as path from "path";
import * as vscode from "vscode";
import { NexusClient, Provider, TaskStatus } from "../nexusClient";

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

function buildProviderItems(providers: Provider[]): ProviderPickItem[] {
  const items: ProviderPickItem[] = [
    {
      label: "Auto (let Nexus choose)",
      description: "Use the default provider and model",
    },
  ];

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

export async function submitTaskCommand(
  client: NexusClient,
  daemonUrl: string
): Promise<void> {
  // 1. Pre-fill instruction from selected text in the active editor
  const editor = vscode.window.activeTextEditor;
  const selectedText = editor?.document.getText(editor.selection).trim() ?? "";

  // 2. Instruction input box
  const instruction = await vscode.window.showInputBox({
    prompt: "Task instruction",
    value: selectedText,
    ignoreFocusOut: true,
  });
  if (instruction === undefined) {
    return; // user cancelled
  }

  // 3. Require an open workspace folder for projectPath
  const workspaceFolders = vscode.workspace.workspaceFolders;
  if (!workspaceFolders || workspaceFolders.length === 0) {
    vscode.window.showErrorMessage("Open a folder to submit a task");
    return;
  }
  const wsRoot = workspaceFolders[0].uri.fsPath;
  const projectPath = wsRoot;

  // 4. Target file — relative path from active editor, or ask
  let targetFile: string | undefined;
  if (editor) {
    const filePath = editor.document.uri.fsPath;
    targetFile = filePath.startsWith(wsRoot)
      ? path.relative(wsRoot, filePath)
      : filePath;
  } else {
    targetFile = await vscode.window.showInputBox({
      prompt: "Target file path (relative to project root)",
      ignoreFocusOut: true,
    });
    if (targetFile === undefined) {
      return; // user cancelled
    }
  }

  // 5. Provider / model quick-pick
  let providers: Provider[] = [];
  try {
    providers = await client.getProviders();
  } catch {
    // non-fatal — fall through with empty list so "Auto" is still offered
  }

  const picked = await vscode.window.showQuickPick(buildProviderItems(providers), {
    placeHolder: "Select provider and model",
    ignoreFocusOut: true,
  });
  if (picked === undefined) {
    return; // user cancelled
  }

  // 6. Submit the task
  let taskId: string;
  try {
    const task = await client.submitTask({
      instruction,
      projectPath,
      targetFile,
      providerHint: (picked as ProviderPickItem).providerHint,
      modelId: (picked as ProviderPickItem).modelId,
    });
    taskId = task.id;
  } catch (err) {
    if (isDaemonUnreachable(err)) {
      vscode.window.showErrorMessage(
        `Nexus daemon is not running at ${daemonUrl}. Start it with 'nexus-daemon' or the desktop app.`
      );
    } else {
      const msg = err instanceof Error ? err.message : String(err);
      vscode.window.showErrorMessage(`Nexus: failed to submit task — ${msg}`);
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

      token.onCancellationRequested(async () => {
        userCancelled = true;
        try {
          await client.cancelTask(taskId);
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
              "✓ Nexus task completed",
              "Open File"
            );
            if (action === "Open File") {
              await vscode.window.showTextDocument(vscode.Uri.file(absTarget));
            }
            break;
          }
          case "FAILED":
            // 10. Error notification
            vscode.window.showErrorMessage(
              `Nexus: task failed — ${task.logs ?? "no details available"}`
            );
            break;
          case "TOO_LARGE":
            vscode.window.showErrorMessage(
              "Nexus: task rejected — input too large for the selected model"
            );
            break;
          case "NO_PROVIDER":
            vscode.window.showErrorMessage(
              "Nexus: task failed — no LLM provider is currently available"
            );
            break;
          case "CANCELLED":
            // 11. Cancelled externally (e.g. from daemon/CLI)
            vscode.window.showInformationMessage("Nexus: Task cancelled");
            break;
        }
        return;
      }
    }
  );
}
