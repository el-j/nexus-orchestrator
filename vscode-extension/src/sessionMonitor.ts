/**
 * sessionMonitor.ts — Detects active GitHub Copilot chat activity and
 * registers it as an AISession with the nexusOrchestrator daemon.
 */

import * as vscode from "vscode";
import { NexusClient } from "./nexusClient";

export class SessionMonitor {
  private sessionId: string | undefined;
  private heartbeatTimer: NodeJS.Timeout | undefined;
  private modelChangeListener: vscode.Disposable | undefined;

  constructor(
    private readonly client: NexusClient,
    private readonly context: vscode.ExtensionContext
  ) {}

  async start(): Promise<void> {
    await this.detectAndRegister();
    // Listen for model changes (Copilot available/unavailable)
    if (vscode.lm?.onDidChangeChatModels) {
      this.modelChangeListener = vscode.lm.onDidChangeChatModels(() =>
        this.detectAndRegister()
      );
      this.context.subscriptions.push(this.modelChangeListener);
    }
    // Heartbeat every 60s
    this.heartbeatTimer = setInterval(() => this.heartbeat(), 60_000);
  }

  async stop(): Promise<void> {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
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
      await this.context.workspaceState.update("nexus.sessionId", session.id);
    } catch {
      // Silent — session registration is non-critical
    }
  }

  private async heartbeat(): Promise<void> {
    if (this.sessionId) {
      await this.detectAndRegister(); // Re-register updates lastActivity
    }
  }
}
