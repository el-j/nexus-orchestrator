/**
 * lmProviders.ts: keeps the daemon's provider list in step with the models this editor offers.
 *
 * Each model that `vscode.lm` exposes (Copilot, and any other extension that registers one) is
 * registered with the daemon as its own OpenAI-compatible provider pointing at the local bridge.
 * One provider per model, because the daemon sends the provider's configured model; this also lets
 * tasks and roles pick a model by provider name.
 */
import * as vscode from 'vscode';
import type { NexusClient } from './nexusClient';
import type { RunningBridge } from './lmBridge';

/** Prefix of every provider this module owns; others are never touched. */
export const BRIDGE_PROVIDER_PREFIX = 'vscode-';

/** Provider name for a model: lowercase letters, digits, dot, dash and underscore only. */
export function providerNameFor(vendor: string, modelId: string): string {
  const clean = (s: string): string => s.toLowerCase().replace(/[^a-z0-9._-]+/g, '-');
  return `${BRIDGE_PROVIDER_PREFIX}${clean(vendor)}-${clean(modelId)}`;
}

/** The editor's models that may be shared, as provider name → model id. */
async function sharedModels(vendors: string[]): Promise<Map<string, string>> {
  const all = await vscode.lm.selectChatModels({});
  const shared = all.filter((m) => vendors.length === 0 || vendors.includes(m.vendor));
  return new Map(shared.map((m) => [providerNameFor(m.vendor, m.id), m.id]));
}

/** True when the daemon already lists exactly the wanted providers, pointing at this bridge. */
export async function isInSync(
  client: NexusClient,
  bridge: RunningBridge,
  vendors: string[],
): Promise<boolean> {
  const wanted = await sharedModels(vendors);
  const ours = (await client.getProviders()).filter((p) =>
    p.name.startsWith(BRIDGE_PROVIDER_PREFIX),
  );
  return (
    ours.length === wanted.size &&
    ours.every((p) => wanted.has(p.name) && p.baseURL === bridge.baseUrl)
  );
}

/**
 * Registers the editor's models with the daemon and drops providers left over from an earlier
 * bridge. `vendors` limits what is shared; empty means every vendor. Returns the names registered.
 */
export async function syncLmProviders(
  client: NexusClient,
  bridge: RunningBridge,
  vendors: string[],
): Promise<string[]> {
  const wanted = await sharedModels(vendors);
  await dropLmProviders(client);
  for (const [name, model] of wanted) {
    await client.registerProvider({
      name,
      kind: 'openaicompat',
      baseUrl: bridge.baseUrl,
      apiKey: bridge.token,
      model,
    });
  }
  return [...wanted.keys()];
}

/** Removes every provider this module registered; used when sharing is switched off. */
export async function dropLmProviders(client: NexusClient): Promise<void> {
  for (const p of await client.getProviders()) {
    if (p.name.startsWith(BRIDGE_PROVIDER_PREFIX)) await client.removeProvider(p.name);
  }
}
