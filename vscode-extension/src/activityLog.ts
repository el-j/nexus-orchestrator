import * as vscode from "vscode";

let outputChannel: vscode.OutputChannel | undefined;
const queuedTaskIds = new Set<string>();
let lastQueuedTaskId: string | undefined;

function ensureChannel(): vscode.OutputChannel {
  if (!outputChannel) {
    outputChannel = vscode.window.createOutputChannel("Nexus Orchestrator");
  }
  return outputChannel;
}

export function getNexusActivityChannel(): vscode.OutputChannel {
  return ensureChannel();
}

export function showNexusActivityLog(preserveFocus = false): void {
  ensureChannel().show(preserveFocus);
}

export function logNexusActivity(scope: string, message: string): void {
  ensureChannel().appendLine(`[${new Date().toISOString()}] [${scope}] ${message}`);
}

export function rememberQueuedTask(taskId: string): void {
  queuedTaskIds.add(taskId);
  lastQueuedTaskId = taskId;
}

export function getKnownTaskSource(taskId: string): string | undefined {
  if (queuedTaskIds.has(taskId)) {
    return "vscode queue";
  }
  return undefined;
}

export function getActivitySnapshot(): { lastQueuedTaskId?: string } {
  return { lastQueuedTaskId };
}