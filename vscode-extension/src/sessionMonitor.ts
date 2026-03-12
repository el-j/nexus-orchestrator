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
  private readonly outputChannel: vscode.OutputChannel;

  constructor(
    private readonly client: NexusClient,
    private readonly context: vscode.ExtensionContext
  ) {
    this.outputChannel = vscode.window.createOutputChannel('Nexus Orchestrator');
    this.context.subscriptions.push(this.outputChannel);
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
        this.outputChannel.appendLine('[SessionMonitor] Copilot models not available yet — will retry');
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
      this.outputChannel.appendLine(`[SessionMonitor] Copilot session registered: ${session.id}`);
      await this.context.workspaceState.update("nexus.sessionId", session.id);
    } catch (error) {
      this.outputChannel.appendLine(`[SessionMonitor] Registration failed: ${error}`);
    }
  }

  private async heartbeat(): Promise<void> {
    if (this.sessionId) {
      await this.detectAndRegister(); // Re-register updates lastActivity
    }
  }
}
