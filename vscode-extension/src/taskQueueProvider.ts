/**
 * taskQueueProvider.ts — VS Code TreeDataProvider for the Nexus task queue.
 * Implements TASK-134.
 */

import * as vscode from 'vscode'
import { NexusClient, Task, TaskStatus } from './nexusClient'
import { getKnownTaskSource } from './activityLog'

function statusEmoji(status: TaskStatus): string {
  switch (status) {
    case 'QUEUED':
    case 'PROCESSING':
      return '🕐'
    case 'COMPLETED':
      return '✅'
    case 'FAILED':
      return '❌'
    case 'CANCELLED':
      return '⛔'
    case 'TOO_LARGE':
      return '📏'
    case 'NO_PROVIDER':
      return '🔌'
    default:
      return '❓'
  }
}

function statusIcon(status: TaskStatus): vscode.ThemeIcon {
  switch (status) {
    case 'QUEUED':
      return new vscode.ThemeIcon('clock')
    case 'PROCESSING':
      return new vscode.ThemeIcon('loading~spin')
    case 'COMPLETED':
      return new vscode.ThemeIcon('pass')
    case 'FAILED':
      return new vscode.ThemeIcon('error')
    case 'CANCELLED':
      return new vscode.ThemeIcon('circle-slash')
    case 'TOO_LARGE':
      return new vscode.ThemeIcon('file-binary')
    case 'NO_PROVIDER':
      return new vscode.ThemeIcon('plug')
    default:
      return new vscode.ThemeIcon('question')
  }
}

function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max) + '…' : s
}

export class TaskItem extends vscode.TreeItem {
  constructor(public readonly task: Task) {
    const emoji = statusEmoji(task.status)
    const shortId = task.id.replace(/-/g, '').slice(0, 8)
    super(
      `${emoji} #${shortId} — ${truncate(task.instruction, 25)}`,
      vscode.TreeItemCollapsibleState.None
    )

    const parts: string[] = []
    const source = getKnownTaskSource(task.id)
    if (source) parts.push(source)
    if (task.providerHint) parts.push(task.providerHint)
    if (task.modelId) parts.push(task.modelId)
    this.description = parts.join(' / ')

    const tooltipLines = [
      `**Instruction:** ${task.instruction}`,
      `**ID:** ${task.id}`,
      `**Status:** ${task.status}`,
      `**Source:** ${source ?? 'unknown'}`,
      `**Project:** ${task.projectPath}`,
      `**Target:** ${task.targetFile || '—'}`,
      `**Provider:** ${task.providerHint || '—'}`,
      `**Model:** ${task.modelId || '—'}`,
      `**Created:** ${task.createdAt}`,
      `**Updated:** ${task.updatedAt}`,
    ]
    if (task.logs) {
      const safeLogs = task.logs.slice(0, 500).replace(/[<>&]/g, '') +
        (task.logs.length > 500 ? '...' : '')
      tooltipLines.push(`\n**Logs:**\n\`\`\`\n${safeLogs}\n\`\`\``)
    }
    this.tooltip = new vscode.MarkdownString(tooltipLines.join('\n\n'))

    this.iconPath = statusIcon(task.status)
    this.contextValue = 'nexusTask'
  }
}

export class TaskQueueProvider implements vscode.TreeDataProvider<TaskItem> {
  private readonly _onDidChangeTreeData =
    new vscode.EventEmitter<TaskItem | undefined | void>()
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event

  constructor(private readonly client: NexusClient) {}

  refresh(): void {
    this._onDidChangeTreeData.fire()
  }

  startPolling(intervalMs = 5000): vscode.Disposable {
    const handle = setInterval(() => this.refresh(), intervalMs)
    return new vscode.Disposable(() => clearInterval(handle))
  }

  getTreeItem(element: TaskItem): vscode.TreeItem {
    return element
  }

  async getChildren(): Promise<TaskItem[]> {
    try {
      const tasks = await this.client.getTasks()
      return tasks
        .sort(
          (a, b) =>
            new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
        )
        .map(t => new TaskItem(t))
    } catch {
      return []
    }
  }
}
