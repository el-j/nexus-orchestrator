/**
 * workspaceScanner.ts — Scans open workspace folders for Nexus orchestrator
 * state files and exposes their content for the sidebar tree view.
 */

import * as vscode from 'vscode'
import * as fs from 'fs'
import * as path from 'path'

/** A single task entry extracted from orchestrator.json */
export interface OrchestratorTask {
  id: string
  title: string
  status: string
  role: string
}

/** A single plan entry extracted from orchestrator.json */
export interface OrchestratorPlan {
  id: string
  title: string
  status: string
  tasks: OrchestratorTask[]
}

/** The parsed state of one workspace folder's orchestrator.json */
export interface WorkspaceOrchestration {
  /** Absolute path to the workspace folder root */
  folderPath: string
  /** Human-readable folder name (last path segment) */
  folderName: string
  /** Current active plan, or null */
  activePlan: OrchestratorPlan | null
  /** All plans, newest first (by array position in the JSON) */
  allPlans: OrchestratorPlan[]
  /** When the file was last modified */
  lastModified: Date
}

export class WorkspaceScanner implements vscode.Disposable {
  private readonly _onDidChange = new vscode.EventEmitter<void>()
  readonly onDidChange = this._onDidChange.event

  private readonly watchers: vscode.FileSystemWatcher[] = []
  private orchestrations: WorkspaceOrchestration[] = []

  constructor(private readonly context: vscode.ExtensionContext) {}

  /** Scans all workspace folders and starts file watchers. Call once at activation. */
  start(): void {
    this.scan()
    this.startWatchers()

    // Re-scan when workspace folders are added/removed
    this.context.subscriptions.push(
      vscode.workspace.onDidChangeWorkspaceFolders(() => {
        this.disposeWatchers()
        this.scan()
        this.startWatchers()
        this._onDidChange.fire()
      })
    )
  }

  /** Returns the most recently scanned orchestrations. */
  getOrchestrations(): WorkspaceOrchestration[] {
    return this.orchestrations
  }

  /** Re-reads all orchestrator.json files synchronously. */
  scan(): void {
    this.orchestrations = []
    const folders = vscode.workspace.workspaceFolders ?? []
    for (const folder of folders) {
      const filePath = path.join(folder.uri.fsPath, '.claude', 'orchestrator.json')
      if (!fs.existsSync(filePath)) continue
      try {
        const raw = fs.readFileSync(filePath, 'utf8')
        const parsed = parseOrchestratorFile(raw)
        if (parsed) {
          const stat = fs.statSync(filePath)
          this.orchestrations.push({
            folderPath: folder.uri.fsPath,
            folderName: path.basename(folder.uri.fsPath),
            ...parsed,
            lastModified: stat.mtime,
          })
        }
      } catch (e) {
        console.error('WorkspaceScanner: file I/O failed:', e)
      }
    }
  }

  private startWatchers(): void {
    const folders = vscode.workspace.workspaceFolders ?? []
    for (const folder of folders) {
      // Use a glob that covers the exact file we care about
      const pattern = new vscode.RelativePattern(folder, '.claude/orchestrator.json')
      const watcher = vscode.workspace.createFileSystemWatcher(pattern)
      const refresh = (): void => {
        this.scan()
        this._onDidChange.fire()
      }
      watcher.onDidChange(refresh)
      watcher.onDidCreate(refresh)
      watcher.onDidDelete(refresh)
      this.context.subscriptions.push(watcher)
      this.watchers.push(watcher)
    }
  }

  private disposeWatchers(): void {
    for (const w of this.watchers) {
      w.dispose()
    }
    this.watchers.length = 0
  }

  dispose(): void {
    this.disposeWatchers()
    this._onDidChange.dispose()
  }
}

/**
 * Parses a raw orchestrator.json string.
 * Returns null if the file is not recognized as a valid orchestrator file.
 * This is intentionally lenient — missing fields default gracefully.
 */
function parseOrchestratorFile(
  raw: string
): Omit<WorkspaceOrchestration, 'folderPath' | 'folderName' | 'lastModified'> | null {
  let json: Record<string, unknown>
  try {
    json = JSON.parse(raw) as Record<string, unknown>
  } catch {
    return null
  }
  // Minimal validation: must have a 'plans' or 'tasks' key
  if (!json.plans && !json.tasks) return null

  const rawPlans = (json.plans as Record<string, RawPlan> | undefined) ?? {}
  const rawTasks = (json.tasks as Record<string, RawTask> | undefined) ?? {}

  const allPlans: OrchestratorPlan[] = Object.values(rawPlans).map(p => ({
    id: String(p.id ?? ''),
    title: String(p.title ?? ''),
    status: String(p.status ?? 'unknown'),
    tasks: ((p.tasks ?? []) as unknown[]).map(tid => {
      const key = String(tid)
      const t = rawTasks[key]
      return t
        ? {
            id: String(t.id ?? key),
            title: String(t.title ?? key),
            status: String(t.status ?? 'unknown'),
            role: String(t.role ?? ''),
          }
        : { id: key, title: key, status: 'unknown', role: '' }
    }),
  }))

  const activePlanId = json.activePlanId as string | null | undefined
  const activePlan = allPlans.find(p => p.id === activePlanId) ?? null

  return { activePlan, allPlans }
}

interface RawPlan {
  id?: unknown
  title?: unknown
  status?: unknown
  tasks?: unknown[]
}

interface RawTask {
  id?: unknown
  title?: unknown
  status?: unknown
  role?: unknown
}
