/**
 * statusBar.ts — Status bar item that reflects Nexus daemon health and task state.
 *
 * Text logic:
 *  - Daemon unreachable → $(warning) Nexus: offline
 *  - Reachable, no active tasks → $(zap) Nexus  (tooltip: N providers active)
 *  - QUEUED/PROCESSING tasks → $(sync~spin) Nexus: N tasks
 *  - FAILED tasks in last hour → $(error) Nexus: N failed
 */

import * as vscode from "vscode";
import { NexusClient, AISession } from "./nexusClient";
import { getActivitySnapshot } from "./activityLog";

export class NexusStatusBar {
  private item: vscode.StatusBarItem;
  private pollTimer?: NodeJS.Timeout;

  constructor(private client: NexusClient) {
    this.item = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100
    );
    this.item.command = "nexus.statusBarAction";
    this.item.show();
  }

  /**
   * Begin polling. The first update fires immediately; subsequent updates fire
   * every `intervalMs` milliseconds (minimum 10 000 ms enforced by the caller).
   * Returns a Disposable so it can be pushed onto context.subscriptions.
   */
  startPolling(intervalMs = 10000): vscode.Disposable {
    void this.update();
    this.pollTimer = setInterval(() => void this.update(), intervalMs);
    return { dispose: () => this.dispose() };
  }

  /** Fetch daemon state and refresh the status bar item. */
  async update(): Promise<void> {
    try {
      const alive = await this.client.health();
      if (!alive) {
        this.item.text = "$(warning) Nexus: offline";
        this.item.tooltip = "Click to open actions";
        return;
      }

      const [providersResult, tasksResult, aiSessionsResult] = await Promise.allSettled([
        this.client.getProviders(),
        this.client.getTasks(),
        this.client.getAISessions(),
      ]);

      const providers = providersResult.status === 'fulfilled' ? providersResult.value : [];
      const tasks = tasksResult.status === 'fulfilled' ? tasksResult.value : [];
      const aiSessions = aiSessionsResult.status === 'fulfilled' ? aiSessionsResult.value : [];
      const isDegraded = providersResult.status === 'rejected' || tasksResult.status === 'rejected';
      if (isDegraded) {
        const failed = [
          providersResult.status === 'rejected' ? 'providers' : null,
          tasksResult.status === 'rejected' ? 'tasks' : null,
        ].filter(Boolean).join(', ');
        console.warn('statusBar: degraded — failed to fetch:', failed);
      }

      const activeProviders = providers.filter((p) => p.active);
      const activeSessions = aiSessions.filter((s) => s.status === "active");
      const activeSessionCount = activeSessions.length;
      const mcpSessions = activeSessions.filter((s) => s.source === "mcp").length;
      const vscodeSessions = activeSessions.filter((s) => s.source === "vscode").length;
      const httpSessions = activeSessions.filter((s) => s.source === "http").length;
      const lastQueuedTaskId = getActivitySnapshot().lastQueuedTaskId;
      const oneHourAgo = Date.now() - 60 * 60 * 1000;

      const activeTasks = tasks.filter(
        (t) => t.status === "QUEUED" || t.status === "PROCESSING"
      );
      const failedTasks = tasks.filter(
        (t) =>
          t.status === "FAILED" &&
          new Date(t.updatedAt).getTime() >= oneHourAgo
      );

      const tooltipLines = [
        `Providers active: ${activeProviders.length}`,
        `Nexus queue: ${activeTasks.length} active task(s)`,
        `Copilot direct sessions: ${vscodeSessions}`,
        `MCP sessions: ${mcpSessions}`,
        `HTTP sessions: ${httpSessions}`,
        `AI sessions total: ${activeSessionCount}`,
      ];
      if (lastQueuedTaskId) {
        tooltipLines.push(`Last queued by extension: #${lastQueuedTaskId.replace(/-/g, "").slice(0, 8)}`);
      }
      if (isDegraded) {
        tooltipLines.push("Some daemon reads failed during the last refresh");
      }
      tooltipLines.push("Click to open Nexus actions");
      const tooltip = tooltipLines.join("\n");

      if (failedTasks.length > 0) {
        this.item.text = `$(error) Nexus Q${activeTasks.length} M${mcpSessions} V${vscodeSessions}`;
        this.item.tooltip = tooltip;
      } else if (activeTasks.length > 0) {
        this.item.text = `$(sync~spin) Nexus Q${activeTasks.length} M${mcpSessions} V${vscodeSessions}${isDegraded ? ' ~' : ''}`;
        this.item.tooltip = tooltip;
      } else if (isDegraded) {
        this.item.text = "$(warning) Nexus: degraded";
        this.item.tooltip = tooltip;
      } else {
        this.item.text = `$(zap) Nexus Q0 M${mcpSessions} V${vscodeSessions}`;
        this.item.tooltip = tooltip;
      }
    } catch (error) {
      console.warn('statusBar: update failed:', error);
      this.item.text = "$(warning) Nexus: offline";
      this.item.tooltip = "Click to open actions";
    }
  }

  dispose(): void {
    if (this.pollTimer !== undefined) {
      clearInterval(this.pollTimer);
      this.pollTimer = undefined;
    }
    this.item.dispose();
  }
}
