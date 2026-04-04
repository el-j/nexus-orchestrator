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

/** Builds a new NexusClient from current settings, optionally running port sweep. */
async function buildClient(): Promise<NexusClient> {
  const url = daemonUrl();
  const port = mcpPort();
  const sweep =
    vscode.workspace.getConfiguration('nexus').get<boolean>('enableMCPPortSweep') ?? true;
  const c = new NexusClient(url, port);
  if (sweep) {
    const candidates = [...new Set([port, 63988, 63987, 63989, 63990, 63986])];
    try {
      await c.tryConnect(candidates);
    } catch {
      // no port responded — proceed with configured url
    }
  }
  return c;
}

/** Returns the shared NexusStatusBar instance (available after activation). */
export function getStatusBar(): NexusStatusBar {
  if (!statusBar) {
    throw new Error('nexus: extension not yet activated');
  }
  return statusBar;
}

export function activate(context: vscode.ExtensionContext): void {
  client = new NexusClient(daemonUrl(), mcpPort());
  // Fire-and-forget port sweep on activation
  void buildClient()
    .then((c) => {
      client = c;
      statusBar?.dispose();
      statusBar = new NexusStatusBar(client);
      context.subscriptions.push(statusBar.startPolling(30000));
    })
    .catch(() => {
      /* ignore — client already set above */
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

  // ── Status bar ──────────────────────────────────────────────────────────────
  statusBar = new NexusStatusBar(client);
  context.subscriptions.push(statusBar.startPolling(30000)); // matches backend health cache TTL (30 s)

  // Re-create the client and refresh status bar whenever the daemon URL or MCP port changes.
  // Dispose the old status bar first to prevent accumulating pollers.
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (
        e.affectsConfiguration('nexus.daemonUrl') ||
        e.affectsConfiguration('nexus.mcpPort') ||
        e.affectsConfiguration('nexus.enableMCPPortSweep')
      ) {
        void buildClient().then((c) => {
          client = c;
          statusBar?.dispose();
          statusBar = new NexusStatusBar(client);
          context.subscriptions.push(statusBar.startPolling(30000));
        });
      }
    }),
  );

  // ── nexus.reconnect ─────────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('nexus.reconnect', async () => {
      try {
        client = await buildClient();
        statusBar?.dispose();
        statusBar = new NexusStatusBar(client);
        context.subscriptions.push(statusBar.startPolling(30000));
        vscode.window.showInformationMessage('Nexus: reconnected to daemon');
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
}

export function deactivate(): Promise<void> | void {
  return monitor?.stop();
}
