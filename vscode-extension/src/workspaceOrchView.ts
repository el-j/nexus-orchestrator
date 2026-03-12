/**
 * workspaceOrchView.ts — Read-only TreeDataProvider showing orchestration
 * state from .claude/orchestrator.json in each open workspace folder.
 */

import * as vscode from 'vscode'
import type { WorkspaceOrchestration, OrchestratorPlan, OrchestratorTask } from './workspaceScanner'
import { WorkspaceScanner } from './workspaceScanner'

// ── Tree node types ──────────────────────────────────────────────────────────

type OrchNode = FolderNode | ActivePlanNode | PlanNode | TaskNode | HistoryNode | EmptyNode

class FolderNode extends vscode.TreeItem {
  readonly kind = 'folder' as const
  constructor(public readonly orch: WorkspaceOrchestration) {
    const _activeSuffix = orch.activePlan ? ` — ${orch.activePlan.title.slice(0, 30)}` : ' — idle'
    super(orch.folderName, vscode.TreeItemCollapsibleState.Expanded)
    this.description = `${orch.allPlans.length} plans`
    this.tooltip = orch.folderPath
    this.iconPath = new vscode.ThemeIcon('folder')
    this.contextValue = 'nexusWorkspaceFolder'
  }
}

class ActivePlanNode extends vscode.TreeItem {
  readonly kind = 'activePlan' as const
  constructor(public readonly plan: OrchestratorPlan) {
    super(`Active: ${plan.title}`, vscode.TreeItemCollapsibleState.Expanded)
    this.description = plan.id
    this.iconPath = planIcon(plan.status)
    this.contextValue = 'nexusActivePlan'
  }
}

class PlanNode extends vscode.TreeItem {
  readonly kind = 'plan' as const
  constructor(public readonly plan: OrchestratorPlan) {
    super(`${plan.id} — ${plan.title.slice(0, 40)}`, vscode.TreeItemCollapsibleState.Collapsed)
    this.description = plan.status
    this.iconPath = planIcon(plan.status)
    this.contextValue = 'nexusPlan'
  }
}

class HistoryNode extends vscode.TreeItem {
  readonly kind = 'history' as const
  constructor(public readonly plans: OrchestratorPlan[]) {
    super('History', vscode.TreeItemCollapsibleState.Collapsed)
    this.description = `${plans.length} plans`
    this.iconPath = new vscode.ThemeIcon('history')
    this.contextValue = 'nexusPlanHistory'
  }
}

class TaskNode extends vscode.TreeItem {
  readonly kind = 'task' as const
  constructor(public readonly task: OrchestratorTask) {
    super(`${task.id} — ${task.title.slice(0, 45)}`, vscode.TreeItemCollapsibleState.None)
    this.description = task.role
    this.iconPath = taskIcon(task.status)
    this.contextValue = 'nexusOrchestratorTask'
  }
}

class EmptyNode extends vscode.TreeItem {
  readonly kind = 'empty' as const
  constructor(msg: string) {
    super(msg, vscode.TreeItemCollapsibleState.None)
    this.iconPath = new vscode.ThemeIcon('info')
    this.contextValue = 'nexusEmpty'
  }
}

function planIcon(status: string): vscode.ThemeIcon {
  switch (status) {
    case 'active':
      return new vscode.ThemeIcon('loading~spin')
    case 'completed':
      return new vscode.ThemeIcon('pass')
    case 'failed':
      return new vscode.ThemeIcon('error')
    default:
      return new vscode.ThemeIcon('circle-outline')
  }
}

function taskIcon(status: string): vscode.ThemeIcon {
  switch (status) {
    case 'done':
      return new vscode.ThemeIcon('pass')
    case 'in-progress':
      return new vscode.ThemeIcon('loading~spin')
    case 'todo':
      return new vscode.ThemeIcon('circle-outline')
    case 'blocked':
      return new vscode.ThemeIcon('error')
    default:
      return new vscode.ThemeIcon('question')
  }
}

// ── TreeDataProvider ─────────────────────────────────────────────────────────

export class WorkspaceOrchViewProvider implements vscode.TreeDataProvider<OrchNode> {
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<OrchNode | undefined>()
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event

  constructor(private readonly scanner: WorkspaceScanner) {
    scanner.onDidChange(() => this._onDidChangeTreeData.fire(undefined))
  }

  refresh(): void {
    this.scanner.scan()
    this._onDidChangeTreeData.fire(undefined)
  }

  getTreeItem(element: OrchNode): vscode.TreeItem {
    return element
  }

  getChildren(element?: OrchNode): OrchNode[] {
    // Root: one FolderNode per workspace folder that has an orchestrator.json
    if (!element) {
      const orchs = this.scanner.getOrchestrations()
      if (orchs.length === 0) {
        return [new EmptyNode('No orchestrator.json found in workspace')]
      }
      return orchs.map(o => new FolderNode(o))
    }

    if (element.kind === 'folder') {
      const { activePlan, allPlans } = element.orch
      const children: OrchNode[] = []
      if (activePlan) {
        children.push(new ActivePlanNode(activePlan))
      }
      const historical = allPlans.filter(p => p.id !== activePlan?.id)
      if (historical.length > 0) {
        children.push(new HistoryNode(historical))
      }
      if (children.length === 0) {
        children.push(new EmptyNode('No plans found'))
      }
      return children
    }

    if (element.kind === 'activePlan') {
      return element.plan.tasks.map(t => new TaskNode(t))
    }

    if (element.kind === 'history') {
      return element.plans.map(p => new PlanNode(p))
    }

    if (element.kind === 'plan') {
      return element.plan.tasks.map(t => new TaskNode(t))
    }

    return []
  }
}
