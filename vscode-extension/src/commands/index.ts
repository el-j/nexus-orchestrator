/**
 * commands/index.ts — Barrel export for all Nexus command implementations.
 */

import * as vscode from 'vscode';
import type { NexusClient, Provider } from '../nexusClient';

export { sendCurrentContextCommand, submitTaskCommand } from './submitTask';
export { selectProviderCommand } from './selectProvider';
export { viewQueueCommand } from './viewQueue';
export { delegateToNexusCommand } from './delegateToNexus';

export async function showProvidersCommand(client: NexusClient): Promise<void> {
  let providers: Provider[];
  try {
    providers = await client.getProviders();
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    vscode.window.showErrorMessage(`Nexus: Failed to fetch providers — ${msg}`);
    return;
  }

  if (providers.length === 0) {
    const openSettingsAction = 'Open Settings';
    const picked = await vscode.window.showInformationMessage(
      'Nexus: No providers configured. Start LM Studio or Ollama, or configure provider settings.',
      openSettingsAction,
    );
    if (picked === openSettingsAction) {
      await vscode.commands.executeCommand('workbench.action.openSettings', 'nexus.');
    }
    return;
  }

  interface ProviderPickItem extends vscode.QuickPickItem {
    provider: Provider;
  }

  const items: ProviderPickItem[] = providers.map((p) => ({
    label: p.name,
    description:
      [p.active ? '● active' : '○ inactive', p.kind ? `kind: ${p.kind}` : undefined, p.baseURL]
        .filter(Boolean)
        .join(' · ') || undefined,
    detail:
      [
        p.details,
        p.error ? `Error: ${p.error}` : undefined,
        p.contextLimit ? `Context limit: ${p.contextLimit}` : undefined,
        p.activeModel ? `Model: ${p.activeModel}` : undefined,
        p.models && p.models.length > 0 ? `Available: ${p.models.join(', ')}` : undefined,
      ]
        .filter(Boolean)
        .join(' · ') || undefined,
    provider: p,
  }));

  const picked = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select a provider to view details',
    title: 'Nexus: LLM Providers',
  });

  if (!picked) return;

  const p = picked.provider;
  const lines: string[] = [`${p.name} — ${p.active ? 'Active' : 'Inactive'}`];
  if (p.kind) lines.push(`Kind: ${p.kind}`);
  if (p.baseURL) lines.push(`Base URL: ${p.baseURL}`);
  if (p.activeModel) lines.push(`Current model: ${p.activeModel}`);
  if (p.models && p.models.length > 0)
    lines.push(`Models (${p.models.length}): ${p.models.join(', ')}`);
  if (p.details) lines.push(`Details: ${p.details}`);
  if (p.error) lines.push(`Last error: ${p.error}`);

  vscode.window.showInformationMessage(lines.join('\n'));
}
