import * as vscode from 'vscode';

let outputChannel: vscode.OutputChannel | undefined;
const queuedTaskIds = new Set<string>();
let lastQueuedTaskId: string | undefined;

function ensureChannel(): vscode.OutputChannel {
  if (!outputChannel) {
    outputChannel = vscode.window.createOutputChannel('Nexus Orchestrator');
  }
  return outputChannel;
}

/** Returns the shared "Nexus Activity" output channel, creating it on first use. */
export function getNexusActivityChannel(): vscode.OutputChannel {
  return ensureChannel();
}

/** Reveals the activity output channel without taking focus. */
export function showNexusActivityLog(preserveFocus = false): void {
  ensureChannel().show(preserveFocus);
}

/** Appends a timestamped line to the activity channel and remembers it for the snapshot. */
export function logNexusActivity(scope: string, message: string): void {
  ensureChannel().appendLine(`[${new Date().toISOString()}] [${scope}] ${message}`);
}

/** Records which surface (command, tree view, delegate...) queued a task, for later log lines. */
export function rememberQueuedTask(taskId: string): void {
  queuedTaskIds.add(taskId);
  lastQueuedTaskId = taskId;
}

/** Returns the surface that queued the task, if this window queued it. */
export function getKnownTaskSource(taskId: string): string | undefined {
  if (queuedTaskIds.has(taskId)) {
    return 'vscode queue';
  }
  return undefined;
}

/** Returns the most recent activity lines (for diagnostics and tests). */
export function getActivitySnapshot(): { lastQueuedTaskId?: string } {
  return { lastQueuedTaskId };
}
