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
import { NexusClient } from "./nexusClient";

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

      const [providers, tasks] = await Promise.all([
        this.client.getProviders(),
        this.client.getTasks(),
      ]);

      const activeProviders = providers.filter((p) => p.active);
      const oneHourAgo = Date.now() - 60 * 60 * 1000;

      const activeTasks = tasks.filter(
        (t) => t.status === "QUEUED" || t.status === "PROCESSING"
      );
      const failedTasks = tasks.filter(
        (t) =>
          t.status === "FAILED" &&
          new Date(t.updatedAt).getTime() >= oneHourAgo
      );

      if (failedTasks.length > 0) {
        this.item.text = `$(error) Nexus: ${failedTasks.length} failed`;
        this.item.tooltip = "Click to open actions";
      } else if (activeTasks.length > 0) {
        this.item.text = `$(sync~spin) Nexus: ${activeTasks.length} tasks`;
        this.item.tooltip = `${activeProviders.length} provider(s) active`;
      } else {
        this.item.text = "$(zap) Nexus";
        this.item.tooltip = `${activeProviders.length} provider(s) active`;
      }
    } catch {
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
