/**
 * sessionMonitor.ts — Detects active GitHub Copilot chat activity and
 * registers it as an AISession with the nexusOrchestrator daemon.
 */

import * as vscode from "vscode";
import { NexusClient } from "./nexusClient";
import { getNexusActivityChannel, logNexusActivity } from "./activityLog";

export class SessionMonitor {
  private sessionId: string | undefined;
  private isReregistering = false;
  private heartbeatTimer: NodeJS.Timeout | undefined;
  private claimTimer: NodeJS.Timeout | undefined;
  private modelChangeListener: vscode.Disposable | undefined;
  private readonly outputChannel: vscode.OutputChannel;

  constructor(
    private readonly client: NexusClient,
    private readonly context: vscode.ExtensionContext
  ) {
    this.outputChannel = getNexusActivityChannel();
  }

  async start(): Promise<void> {
    await this.detectAndRegister();

    // Retry if initial detection failed (Copilot may still be initializing)
    if (!this.sessionId) {
      const retryDelays = [2000, 5000, 10000];
      for (const delay of retryDelays) {
        if (this.sessionId) break;
        await new Promise<void>(resolve => {
          const t = setTimeout(resolve, delay);
          // Allow early exit if extension disposes
          this.context.subscriptions.push({ dispose: () => clearTimeout(t) });
        });
        if (!this.sessionId) {
          await this.detectAndRegister();
        }
      }
    }

    // Listen for model changes (handles Copilot account sign-in/sign-out)
    if (vscode.lm?.onDidChangeChatModels) {
      this.modelChangeListener = vscode.lm.onDidChangeChatModels(() => {
        void this.detectAndRegister();
      });
      this.context.subscriptions.push(this.modelChangeListener);
    }
    // Heartbeat every 60s
    this.heartbeatTimer = setInterval(() => void this.heartbeat(), 60_000);
    // Poll for QUEUED tasks to auto-claim every 10s
    this.claimTimer = setInterval(() => void this.pollAndClaim(), 10_000);
  }

  async stop(): Promise<void> {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
    }
    if (this.claimTimer) {
      clearInterval(this.claimTimer);
    }
    if (this.sessionId) {
      try {
        await this.client.deregisterSession(this.sessionId);
      } catch {
        // silent — deregistration is best-effort
      }
    }
    this.sessionId = undefined;
  }

  private async detectAndRegister(): Promise<void> {
    try {
      // Check if Copilot is available
      const models =
        vscode.lm?.selectChatModels
          ? await vscode.lm.selectChatModels({ vendor: "copilot" })
          : [];
      if (models.length === 0) {
        logNexusActivity('copilot', 'models not available yet; waiting to register session');
        return; // Copilot not available
      }

      const workspacePath =
        vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? "";
      const externalId = `${vscode.env.machineId}:${workspacePath}`;

      const session = await this.client.registerSession({
        agentName: "GitHub Copilot",
        source: "vscode",
        projectPath: workspacePath,
        externalId,
      });
      this.sessionId = session.id;
      logNexusActivity('copilot', `session registered: ${session.id}`);
      await this.context.workspaceState.update("nexus.sessionId", session.id);
    } catch (error) {
      logNexusActivity('copilot', `registration failed: ${error}`);
    }
  }

  private async heartbeat(): Promise<void> {
    if (!this.sessionId) {
      return;
    }
    try {
      await this.client.heartbeatSession(this.sessionId);
    } catch (error) {
      // If the session no longer exists on the server (e.g. cleaned up after
      // being idle), fall back to a full re-registration.
      logNexusActivity('copilot', `heartbeat failed (${error}); re-registering`);
      if (this.isReregistering) {
        return;
      }
      this.isReregistering = true;
      this.sessionId = undefined;
      try {
        await this.detectAndRegister();
      } finally {
        this.isReregistering = false;
      }
    }
  }

  private async pollAndClaim(): Promise<void> {
    if (!this.sessionId) return;
    try {
      const tasks = await this.client.getTasks();
      const queued = tasks.filter(t => t.status === "QUEUED");
      for (const task of queued) {
        try {
          const claimed = await this.client.claimTask(task.id, this.sessionId);
          logNexusActivity('copilot', `claimed task ${claimed.id} (${claimed.instruction.slice(0, 60)})`);
        } catch {
          // Another agent may have claimed it first — skip
        }
      }
    } catch {
      // Daemon may be unreachable — skip silently
    }
  }
}
