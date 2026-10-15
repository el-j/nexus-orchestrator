/**
 * lmShare.ts: switches the editor-models bridge on and off and keeps it registered.
 *
 * Off by default (`nexus.lmBridge.enabled`): sharing lets the daemon spend the subscription's
 * requests, so it is the user's decision. While on, a periodic check re-registers the providers
 * after a daemon restart (the daemon keeps them in memory only) or when the model list changes.
 */
import * as vscode from 'vscode';
import type { NexusClient } from './nexusClient';
import { startLmBridge, type RunningBridge } from './lmBridge';
import { dropLmProviders, isInSync, syncLmProviders } from './lmProviders';
import { logNexusActivity } from './activityLog';

/** How often registration is verified, in milliseconds. */
const CHECK_INTERVAL_MS = 30_000;

/** Owns the bridge lifecycle for one editor window. */
export class LmShare implements vscode.Disposable {
  private bridge: RunningBridge | undefined;
  private timer: ReturnType<typeof setInterval> | undefined;
  private busy = false;

  constructor(private readonly client: NexusClient) {}

  /** Starts sharing if enabled and follows setting and model-list changes until disposal. */
  register(context: vscode.ExtensionContext): void {
    context.subscriptions.push(this);
    void this.refresh();
    if (vscode.lm?.onDidChangeChatModels) {
      context.subscriptions.push(vscode.lm.onDidChangeChatModels(() => void this.refresh()));
    }
    context.subscriptions.push(
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration('nexus.lmBridge')) void this.refresh();
      }),
    );
  }

  /** Applies the current settings: starts, refreshes or stops sharing. */
  async refresh(): Promise<void> {
    if (this.busy) return;
    this.busy = true;
    try {
      const cfg = vscode.workspace.getConfiguration('nexus.lmBridge');
      if (!cfg.get<boolean>('enabled', false)) return await this.stop();
      this.timer ??= setInterval(() => void this.refresh(), CHECK_INTERVAL_MS);
      this.bridge ??= await startLmBridge();
      if (!(await this.client.health())) return;
      const vendors = cfg.get<string[]>('vendors', []);
      if (await isInSync(this.client, this.bridge, vendors)) return;
      const names = await syncLmProviders(this.client, this.bridge, vendors);
      logNexusActivity('lm-bridge', `${names.length} editor model(s) registered with the daemon`);
    } catch (err) {
      logNexusActivity('lm-bridge', err instanceof Error ? err.message : String(err));
    } finally {
      this.busy = false;
    }
  }

  /** Stops the bridge and takes its providers out of the daemon. */
  async stop(): Promise<void> {
    if (this.timer) clearInterval(this.timer);
    this.timer = undefined;
    if (!this.bridge) return;
    const bridge = this.bridge;
    this.bridge = undefined;
    await bridge.close();
    await dropLmProviders(this.client).catch(() => undefined);
  }

  /** Disposes the share: stops the timer and the bridge. */
  dispose(): void {
    void this.stop();
  }
}
