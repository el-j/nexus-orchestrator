/**
 * statusBar.ts — Status bar item that reflects Nexus daemon health and task state.
 *
 * Text logic:
 *  - Daemon unreachable → $(warning) Nexus: offline
 *  - Reachable, no active tasks → $(zap) Nexus  (tooltip: N providers active)
 *  - QUEUED/PROCESSING tasks → $(sync~spin) Nexus: N tasks
 *  - FAILED tasks in last hour → $(error) Nexus: N failed
 *  - WAITING tasks (every provider is at a limit) are counted as W; the tooltip names the providers that
 *    cool down, the plans that are nearly used up and the tokens of the last 24 hours
 */

import * as vscode from 'vscode';
import { NexusClient, AISession, AIActivity } from './nexusClient';
import { getActivitySnapshot } from './activityLog';
import { coolingNote, quotaNote } from './providerLimits';

function compact(n: number): string {
  return n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n);
}

/** Status bar item showing daemon connectivity and queue depth. */
export class NexusStatusBar {
  private item: vscode.StatusBarItem;
  private pollTimer?: NodeJS.Timeout;

  constructor(private client: NexusClient) {
    this.item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
    this.item.command = 'nexus.statusBarAction';
    this.item.show();
  }

  /**
   * Begin polling. The first update fires immediately; subsequent updates fire
   * every `intervalMs` milliseconds (minimum 10 000 ms enforced by the caller).
   * Returns a Disposable so it can be pushed onto context.subscriptions.
   */
  startPolling(intervalMs = 10000): vscode.Disposable {
    if (this.pollTimer !== undefined) clearInterval(this.pollTimer); // never stack two pollers
    void this.update();
    this.pollTimer = setInterval(() => void this.update(), intervalMs);
    return { dispose: () => this.dispose() };
  }

  /** Fetch daemon state and refresh the status bar item. */
  async update(): Promise<void> {
    try {
      const alive = await this.client.health();
      if (!alive) {
        this.item.text = '$(warning) Nexus: offline';
        this.item.tooltip = 'Click to open actions';
        return;
      }

      const [providersResult, tasksResult, aiSessionsResult, activitiesResult, usageResult] =
        await Promise.allSettled([
          this.client.getProviders(),
          this.client.getTasks(),
          this.client.getAISessions(),
          this.client.getActivities({ limit: 50 }),
          // An older daemon has no usage ledger: that is not a degraded state.
          Promise.resolve().then(() => this.client.getUsage('24h')),
        ]);

      const providers = providersResult.status === 'fulfilled' ? providersResult.value : [];
      const tasks = tasksResult.status === 'fulfilled' ? tasksResult.value : [];
      const aiSessions = aiSessionsResult.status === 'fulfilled' ? aiSessionsResult.value : [];
      const activities: AIActivity[] =
        activitiesResult.status === 'fulfilled' ? activitiesResult.value : [];
      const isDegraded = providersResult.status === 'rejected' || tasksResult.status === 'rejected';
      if (isDegraded) {
        const failed = [
          providersResult.status === 'rejected' ? 'providers' : null,
          tasksResult.status === 'rejected' ? 'tasks' : null,
        ]
          .filter(Boolean)
          .join(', ');
        console.warn('statusBar: degraded — failed to fetch:', failed);
      }

      // Count unique agents active in the last 5 minutes
      const fiveMinAgo = Date.now() - 5 * 60 * 1000;
      const recentActivities = activities.filter(
        (a) => new Date(a.timestamp).getTime() >= fiveMinAgo,
      );
      const activeAgentNames = new Set(recentActivities.map((a) => a.agentName));
      const activeAgentCount = activeAgentNames.size;
      const generatingCount = recentActivities.filter(
        (a) => a.activityType === 'generation',
      ).length;

      const activeProviders = providers.filter((p) => p.active);
      const activeSessions = aiSessions.filter((s) => s.status === 'active');
      const activeSessionCount = activeSessions.length;
      const mcpSessions = activeSessions.filter((s) => s.source === 'mcp').length;
      const vscodeSessions = activeSessions.filter((s) => s.source === 'vscode').length;
      const httpSessions = activeSessions.filter((s) => s.source === 'http').length;
      const lastQueuedTaskId = getActivitySnapshot().lastQueuedTaskId;
      const oneHourAgo = Date.now() - 60 * 60 * 1000;

      const activeTasks = tasks.filter((t) => t.status === 'QUEUED' || t.status === 'PROCESSING');
      const waitingCount = tasks.filter((t) => t.status === 'WAITING').length;
      const waitTag = waitingCount > 0 ? ` W${waitingCount}` : '';
      const cooling = activeProviders.filter((p) => p.rateLimited && p.rateLimitedUntil);
      const everyProviderCools =
        activeProviders.length > 0 && cooling.length === activeProviders.length;
      const failedTasks = tasks.filter(
        (t) => t.status === 'FAILED' && new Date(t.updatedAt).getTime() >= oneHourAgo,
      );

      const tooltipLines = [
        `Providers active: ${activeProviders.length}`,
        `Nexus queue: ${activeTasks.length} active task(s)`,
        `${activeAgentCount} AI agents active in last 5m`,
        `Copilot direct sessions: ${vscodeSessions}`,
        `MCP sessions: ${mcpSessions}`,
        `HTTP sessions: ${httpSessions}`,
        `AI sessions total: ${activeSessionCount}`,
      ];
      if (waitingCount > 0) {
        tooltipLines.push(`${waitingCount} task(s) waiting for a provider limit to reset`);
      }
      for (const p of activeProviders) {
        const note = coolingNote(p);
        if (note) tooltipLines.push(`${p.name}: ${note}`);
      }
      for (const p of activeProviders) {
        const note = quotaNote(p);
        if (note) tooltipLines.push(`${p.name}: ${note}`);
      }
      const totals = usageResult.status === 'fulfilled' ? usageResult.value.totals : undefined;
      if (totals && totals.calls > 0) {
        tooltipLines.push(
          `Tokens (24h): ${compact(totals.inputTokens)} in · ${compact(totals.outputTokens)} out`,
        );
      }
      if (lastQueuedTaskId) {
        tooltipLines.push(
          `Last queued by extension: #${lastQueuedTaskId.replace(/-/g, '').slice(0, 8)}`,
        );
      }
      if (isDegraded) {
        tooltipLines.push('Some daemon reads failed during the last refresh');
      }
      tooltipLines.push('Click to open Nexus actions');
      const tooltip = tooltipLines.join('\n');

      if (everyProviderCools) {
        this.item.text = '$(watch) Nexus: every provider is cooling down';
        this.item.tooltip = tooltip;
      } else if (failedTasks.length > 0) {
        this.item.text = `$(error) Nexus Q${activeTasks.length}${waitTag} M${mcpSessions} V${vscodeSessions}`;
        this.item.tooltip = tooltip;
      } else if (activeAgentCount > 0) {
        this.item.text = `🤖 ${activeAgentCount} agent(s) · ${generatingCount} generating`;
        this.item.tooltip = tooltip;
      } else if (activeTasks.length > 0) {
        this.item.text = `$(sync~spin) Nexus Q${activeTasks.length}${waitTag} M${mcpSessions} V${vscodeSessions}${isDegraded ? ' ~' : ''}`;
        this.item.tooltip = tooltip;
      } else if (isDegraded) {
        this.item.text = '$(warning) Nexus: degraded';
        this.item.tooltip = tooltip;
      } else if (waitingCount > 0) {
        this.item.text = `$(watch) Nexus Q0${waitTag} M${mcpSessions} V${vscodeSessions}`;
        this.item.tooltip = tooltip;
      } else {
        this.item.text = `$(zap) Nexus Q0 M${mcpSessions} V${vscodeSessions}`;
        this.item.tooltip = tooltip;
      }
    } catch (error) {
      console.warn('statusBar: update failed:', error);
      this.item.text = '$(warning) Nexus: offline';
      this.item.tooltip = 'Click to open actions';
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
