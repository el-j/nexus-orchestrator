/**
 * viewQueue.ts — Implementation of the `nexus.viewQueue` command (TASK-134).
 */

import * as vscode from 'vscode'

export function viewQueueCommand(): void {
  vscode.commands.executeCommand('nexus.taskQueue.focus')
}
