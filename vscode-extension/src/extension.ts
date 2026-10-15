/**
 * extension.ts — VS Code extension entry point for Nexus Orchestrator.
 *
 * Activation: registers all commands and exposes getClient() for other modules.
 */

import * as vscode from 'vscode';
import { NexusClient } from './nexusClient';
import {
  sendCurrentContextCommand,
  submitTaskCommand,
  selectProviderCommand,
  viewQueueCommand,
  showProvidersCommand,
} from './commands';
import { NexusStatusBar } from './statusBar';
import { TaskItem, TaskQueueProvider } from './taskQueueProvider';
import { SessionMonitor } from './sessionMonitor';
import { WorkspaceScanner } from './workspaceScanner';
import { WorkspaceOrchViewProvider } from './workspaceOrchView';
import { getNexusActivityChannel, showNexusActivityLog } from './activityLog';
import { AgentDetector } from './agentDetector';
import { AISessionsTreeProvider, AISessionItem } from './aiSessionsTreeProvider';
import { delegateToNexusCommand } from './commands/delegateToNexus';
import { LmShare } from './lmShare';
import { LiveOutput } from './liveOutput';
import { registerPlanFolders } from './planFolders';
import { McpConnect } from './mcpConnect';

let client: NexusClient | undefined;
let statusBar: NexusStatusBar | undefined;
let monitor: SessionMonitor | undefined;

/** Returns the shared NexusClient instance (created during activation). */
export function getClient(): NexusClient {
  if (!client) {
    throw new Error('nexus: extension not yet activated');
  }
  return client;
}

/** Reads the current daemon URL from workspace/user settings. */
function daemonUrl(): string {
  return (
    vscode.workspace.getConfiguration('nexus').get<string>('daemonUrl') ?? 'http://127.0.0.1:63987'
  );
}

/** Reads the configured MCP port. */
function mcpPort(): number {
  return vscode.workspace.getConfiguration('nexus').get<number>('mcpPort') ?? 63988;
}

/** Bearer token for a daemon started with NEXUS_API_TOKEN: the setting, else the environment. */
function apiToken(): string {
  return (
    vscode.workspace.getConfiguration('nexus').get<string>('apiToken') ||
    process.env['NEXUS_API_TOKEN'] ||
    ''
  );
}

/**
 * Points the shared client at the configured daemon. When that daemon does not
 * answer and the port sweep is enabled, probes the usual REST ports and adopts the
 * first that does. The client is updated in place so every component sees it.
 */
export async function reconfigureClient(c: NexusClient): Promise<boolean> {
  const url = daemonUrl();
  const mcp = mcpPort();
  const token = apiToken();
  c.configure(url, mcp, token);
  if (await c.health()) return true;

  const sweep =
    vscode.workspace.getConfiguration('nexus').get<boolean>('enableMCPPortSweep') ?? true;
  if (!sweep) return false;
  try {
    const parsed = new URL(url);
    const configured = Number(parsed.port) || 63987;
    const found = await c.tryConnect([...new Set([configured, 63987, 63986, 63989, 63990])]);
    if (found !== configured) {
      parsed.port = String(found);
      c.configure(parsed.origin, mcp, token);
    }
    return true;
  } catch {
    return false; // nothing answered — keep the configured URL
  }
}

/** Returns the shared NexusStatusBar instance (available after activation). */
export function getStatusBar(): NexusStatusBar {
  if (!statusBar) {
    throw new Error('nexus: extension not yet activated');
  }
  return statusBar;
}

/** Extension entry point: builds the client, status bar, views and commands. */
export function activate(context: vscode.ExtensionContext): void {
  client = new NexusClient(daemonUrl(), mcpPort(), apiToken());
  statusBar = new NexusStatusBar(client);
  context.subscriptions.push(statusBar.startPolling(30000)); // matches backend health cache TTL (30 s)
  // Fire-and-forget: adopt a daemon on another port if the configured one is down.
  void reconfigureClient(client)
    .then(() => statusBar?.update())
    .catch(() => {
      /* the status bar already shows the daemon as unreachable */
    });
  context.subscriptions.push(getNexusActivityChannel());

  // ── Session monitor (GitHub Copilot activity → daemon AISession) ────────────
  monitor = new SessionMonitor(getClient(), context);
  void monitor.start(); // fire-and-forget — non-critical

  // ── Task queue tree view ────────────────────────────────────────────────────
  const provider = new TaskQueueProvider(getClient());
  context.subscriptions.push(vscode.window.registerTreeDataProvider('nexus.taskQueue', provider));
  context.subscriptions.push(provider.startPolling(5000));

  // ── Workspace Agents tree view ──────────────────────────────────────────────
  const workspaceScanner = new WorkspaceScanner(context);
  workspaceScanner.start();
  context.subscriptions.push(workspaceScanner);

  const workspaceOrchProvider = new WorkspaceOrchViewProvider(workspaceScanner, getClient());
  context.subscriptions.push(
    vscode.window.registerTreeDataProvider('nexus.workspaceAgents', workspaceOrchProvider),
  );
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.refreshWorkspaceAgents', () => {
      workspaceOrchProvider.refresh();
    }),
  );

  registerPlanFolders(context, workspaceScanner); // where plans and tasks live, when not the default

  // The editor's own agent must be able to call Nexus tools (VS Code MCP support / Antigravity config).
  new McpConnect(context, () => ({
    url: `http://${new URL(getClient().getBaseUrl()).hostname}:${getClient().getMcpPort()}/mcp`,
    token: apiToken(),
  })).register();

  // ── Editor models → daemon providers (opt-in, see lmShare.ts) ──────────────
  new LmShare(getClient()).register(context);
  new LiveOutput(getClient()).register(context); // task text while the model writes it

  // Re-point the shared client whenever the connection settings change.
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (
        e.affectsConfiguration('nexus.daemonUrl') ||
        e.affectsConfiguration('nexus.mcpPort') ||
        e.affectsConfiguration('nexus.apiToken') ||
        e.affectsConfiguration('nexus.enableMCPPortSweep')
      ) {
        void reconfigureClient(getClient()).then(() => statusBar?.update());
      }
    }),
  );

  // ── nexus.reconnect ─────────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.reconnect', async () => {
      try {
        const ok = await reconfigureClient(getClient());
        await statusBar?.update();
        if (ok) {
          vscode.window.showInformationMessage(`Nexus: connected to ${getClient().getBaseUrl()}`);
        } else {
          vscode.window.showErrorMessage(
            `Nexus: no daemon answered at ${daemonUrl()} — is nexus-daemon running?`,
          );
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Nexus: reconnect failed — ${msg}`);
      }
    }),
  );

  // ── nexus.sendCurrentContext ────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.sendCurrentContext', async () => {
      await sendCurrentContextCommand(getClient(), daemonUrl());
      provider.refresh();
      void getStatusBar().update();
    }),
  );

  // ── nexus.submitTask ────────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.submitTask', async () => {
      await submitTaskCommand(getClient(), daemonUrl());
      provider.refresh();
      void getStatusBar().update();
    }),
  );

  // ── nexus.viewQueue ─────────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.viewQueue', () => {
      viewQueueCommand();
    }),
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.showActivityLog', () => {
      showNexusActivityLog();
    }),
  );

  // ── nexus.cancelTask ────────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.cancelTask', async (item?: TaskItem) => {
      let taskId: string;
      if (item instanceof TaskItem) {
        taskId = item.task.id;
      } else {
        let tasks: import('./nexusClient').Task[];
        try {
          tasks = await getClient().getTasks();
        } catch (err) {
          const msg = err instanceof Error ? err.message : String(err);
          vscode.window.showErrorMessage(`Nexus: Failed to fetch tasks — ${msg}`);
          return;
        }
        const cancellable = tasks.filter((t) => t.status === 'QUEUED' || t.status === 'PROCESSING');
        if (cancellable.length === 0) {
          vscode.window.showInformationMessage('Nexus: No cancellable tasks');
          return;
        }
        const picked = await vscode.window.showQuickPick(
          cancellable.map((t) => ({
            label: `#${t.id.replace(/-/g, '').slice(0, 8)} — ${t.instruction.length > 40 ? t.instruction.slice(0, 40) + '…' : t.instruction}`,
            description: t.status,
            taskId: t.id,
          })),
          { placeHolder: 'Select task to cancel' },
        );
        if (!picked) {
          return;
        }
        taskId = picked.taskId;
      }
      try {
        await getClient().cancelTask(taskId);
        provider.refresh();
        vscode.window.showInformationMessage(`Nexus: Task ${taskId.slice(0, 8)}… cancelled`);
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Nexus: Failed to cancel task — ${msg}`);
      }
    }),
  );

  // ── nexus.selectProvider ────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.selectProvider', async () => {
      await selectProviderCommand(getClient());
    }),
  );

  // ── nexus.toggleWorkspaceScope ──────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.toggleWorkspaceScope', () => {
      const current = provider.getScopeToWorkspace();
      provider.setScopeToWorkspace(!current);
      vscode.window.showInformationMessage(
        `Nexus task scope: ${!current ? 'current workspace only' : 'all projects'}`,
      );
    }),
  );

  // ── nexus.showProviders ─────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.showProviders', async () => {
      await showProvidersCommand(getClient());
    }),
  );

  // ── nexus.statusBarAction ───────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.statusBarAction', async () => {
      interface ActionItem extends vscode.QuickPickItem {
        action: string;
      }
      const items: ActionItem[] = [
        { label: '$(arrow-up) Send Current Context to Queue', action: 'nexus.sendCurrentContext' },
        { label: '$(edit) Compose Manual Task', action: 'nexus.submitTask' },
        { label: '$(list-unordered) View Queue', action: 'nexus.viewQueue' },
        { label: '$(output) Show Activity Log', action: 'nexus.showActivityLog' },
        { label: '$(server) Select Provider / Model', action: 'nexus.selectProvider' },
        { label: '$(database) Get Brain Status', action: 'nexus.brain.status' },
        { label: '$(search) Search Project Brain', action: 'nexus.brain.search' },
        { label: '$(cloud-upload) Ingest Knowledge into Brain', action: 'nexus.brain.ingest' },
        { label: '$(rocket) Initialize Project Brain', action: 'nexus.brain.init' },
        { label: '$(refresh) Refresh Providers', action: 'nexus.statusBarRefresh' },
      ];
      const chosen = await vscode.window.showQuickPick(items, {
        placeHolder: 'Nexus actions',
        title: 'Nexus Orchestrator',
      });
      if (!chosen) {
        return;
      }
      if (chosen.action === 'nexus.statusBarRefresh') {
        await getStatusBar().update();
      } else {
        await vscode.commands.executeCommand(chosen.action);
      }
    }),
  );

  // ── Universal Agent Detector ──────────────────────────────────────────────
  const agentDetector = new AgentDetector(getClient(), context);
  agentDetector.start();
  context.subscriptions.push(agentDetector);

  // ── AI Sessions tree view ─────────────────────────────────────────────────
  const aiSessionsProvider = new AISessionsTreeProvider(getClient());
  context.subscriptions.push(
    vscode.window.registerTreeDataProvider('nexus.aiSessions', aiSessionsProvider),
  );
  agentDetector.onDidChange(() => aiSessionsProvider.refresh());
  context.subscriptions.push(aiSessionsProvider.startPolling(15_000));

  // ── Commands ──────────────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.refreshAISessions', () => {
      aiSessionsProvider.refresh();
      void agentDetector.detectAll();
    }),
    vscode.commands.registerCommand('nexus.delegateToNexus', (item?: AISessionItem) =>
      delegateToNexusCommand(getClient(), item),
    ),
    vscode.commands.registerCommand('nexus.delegateAllSessions', async () => {
      const sessions = await getClient().listAISessions();
      const active = sessions.filter((s) => s.status === 'active' && !s.delegatedToNexus);
      for (const s of active) {
        await delegateToNexusCommand(getClient(), s);
      }
    }),
  );

  // ── nexus.brain.ingest ───────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.brain.ingest', async (projectPath?: string) => {
      const path = projectPath ?? vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
      if (!path) {
        vscode.window.showErrorMessage('Nexus Brain: No workspace folder found.');
        return;
      }
      const fileUris = await vscode.window.showOpenDialog({
        canSelectMany: false,
        canSelectFiles: true,
        canSelectFolders: false,
        filters: { Markdown: ['md'], 'All Files': ['*'] },
        openLabel: 'Ingest into Nexus Brain',
        defaultUri: vscode.Uri.file(path),
      });
      if (!fileUris || fileUris.length === 0) return;
      const filePath = fileUris[0].fsPath;
      try {
        const count = await getClient().ingestKnowledge(path, filePath);
        vscode.window.showInformationMessage(
          `Nexus Brain: Ingested ${count} sections from ${filePath}`,
        );
        workspaceOrchProvider.refresh();
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Nexus Brain: Ingest failed — ${msg}`);
      }
    }),
  );

  // ── nexus.brain.status ───────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.brain.status', async (projectPath?: string) => {
      const path =
        projectPath ??
        (await vscode.window.showInputBox({
          prompt: 'Project path',
          value: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '',
        }));
      if (!path) return;
      try {
        const status = await getClient().getBrainStatus(path);
        vscode.window.showInformationMessage(
          `Brain: ${status.entryCount} entries, ${status.totalTokens} tokens`,
        );
      } catch (e: any) {
        vscode.window.showErrorMessage(`Brain status failed: ${e.message}`);
      }
    }),
  );

  // ── nexus.brain.init ─────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.brain.init', async (projectPath?: string) => {
      const path =
        projectPath ?? (await vscode.window.showInputBox({ prompt: 'Project path to initialize' }));
      if (!path) return;
      try {
        const status = await getClient().initProject(path);
        vscode.window.showInformationMessage(
          `Brain initialized: ${status.entryCount} entries ingested`,
        );
      } catch (e: any) {
        vscode.window.showErrorMessage(`Brain init failed: ${e.message}`);
      }
    }),
  );

  // ── nexus.brain.search ───────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.brain.search', async () => {
      const projectPath = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
      if (!projectPath) {
        vscode.window.showErrorMessage('No workspace folder open');
        return;
      }
      const query = await vscode.window.showInputBox({ prompt: 'Search query' });
      if (!query) return;
      try {
        const results = await getClient().searchKnowledge(projectPath, query, 10);
        if (results.length === 0) {
          vscode.window.showInformationMessage('No results found');
          return;
        }
        const items = results.map((r) => ({
          label: r.topic,
          description: r.content.substring(0, 80),
        }));
        await vscode.window.showQuickPick(items, {
          placeHolder: `${results.length} results for "${query}"`,
        });
      } catch (e: any) {
        vscode.window.showErrorMessage(`Brain search failed: ${e.message}`);
      }
    }),
  );
}

/** Extension exit point: stops pollers and monitors. */
export function deactivate(): Promise<void> | void {
  return monitor?.stop();
}
