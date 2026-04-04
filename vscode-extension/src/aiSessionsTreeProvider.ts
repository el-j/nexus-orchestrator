import * as vscode from 'vscode';
import * as path from 'path';
import { NexusClient, AISession } from './nexusClient';

export class AISessionItem extends vscode.TreeItem {
  constructor(public readonly session: AISession) {
    super(session.agentName, vscode.TreeItemCollapsibleState.None);
    this.contextValue = 'aiSession';
    this.description = buildDescription(session);
    this.iconPath = buildIcon(session);
    this.tooltip = buildTooltip(session);
  }
}

function buildDescription(s: AISession): string {
  let desc = `[${s.source}]`;
  if (s.projectPath) {
    const parts = s.projectPath.split(path.sep).filter(Boolean);
    const short = parts.slice(-2).join(path.sep);
    desc += ` ${short}`;
  }
  return desc;
}

function buildIcon(s: AISession): vscode.ThemeIcon {
  if (s.status === 'active' && s.delegatedToNexus) {
    return new vscode.ThemeIcon('robot', new vscode.ThemeColor('charts.green'));
  }
  if (s.status === 'active') {
    return new vscode.ThemeIcon('robot', new vscode.ThemeColor('charts.yellow'));
  }
  if (s.status === 'idle') {
    return new vscode.ThemeIcon('robot', new vscode.ThemeColor('charts.orange'));
  }
  return new vscode.ThemeIcon('robot', new vscode.ThemeColor('disabledForeground'));
}

function buildTooltip(s: AISession): vscode.MarkdownString {
  const lines = [
    `**${s.agentName}**`,
    `ID: \`${s.id}\``,
    `Status: ${s.status}`,
    `Source: ${s.source}`,
  ];
  if (s.projectPath) lines.push(`Project: ${s.projectPath}`);
  if (s.lastActivity) lines.push(`Last activity: ${s.lastActivity}`);
  if (s.agentCapabilities?.length) lines.push(`Capabilities: ${s.agentCapabilities.join(', ')}`);
  if (s.delegatedToNexus) lines.push(`✓ Delegated to Nexus`);
  return new vscode.MarkdownString(lines.join('\n\n'));
}

export class AISessionsTreeProvider implements vscode.TreeDataProvider<AISessionItem>, vscode.Disposable {
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<void>();
  readonly onDidChangeTreeData: vscode.Event<void> = this._onDidChangeTreeData.event;

  private sessions: AISession[] = [];
  private pollingTimer: NodeJS.Timeout | undefined;

  constructor(private readonly client: NexusClient) {}

  refresh(): void {
    this._onDidChangeTreeData.fire();
  }

  startPolling(ms: number): vscode.Disposable {
    this.pollingTimer = setInterval(() => this.refresh(), ms);
    return new vscode.Disposable(() => {
      if (this.pollingTimer) {
        clearInterval(this.pollingTimer);
        this.pollingTimer = undefined;
      }
    });
  }

  dispose(): void {
    if (this.pollingTimer) {
      clearInterval(this.pollingTimer);
    }
    this._onDidChangeTreeData.dispose();
  }

  getTreeItem(element: AISessionItem): vscode.TreeItem {
    return element;
  }

  async getChildren(): Promise<AISessionItem[]> {
    try {
      this.sessions = await this.client.getAISessions();
    } catch {
      this.sessions = [];
    }

    const sorted = [...this.sessions].sort((a, b) => {
      const order = (s: AISession) =>
        s.status === 'active' ? 0 : s.status === 'idle' ? 1 : 2;
      const ord = order(a) - order(b);
      if (ord !== 0) return ord;
      // within same status, most recent first
      return (b.lastActivity ?? '').localeCompare(a.lastActivity ?? '');
    });

    return sorted.map(s => new AISessionItem(s));
  }
}
