/**
 * viewQueue.ts — Implementation of the `nexus.viewQueue` command (TASK-134).
 */

import * as vscode from 'vscode';

/** Command: show the daemon's task queue in a quick pick. */
export function viewQueueCommand(): void {
  vscode.commands.executeCommand('nexus.taskQueue.focus');
}
