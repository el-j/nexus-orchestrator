/**
 * selectProvider.ts — Quick-pick command to choose a default provider / model.
 *
 * Saves the selection to workspace config nexus.defaultProvider / nexus.defaultModel.
 */

import * as vscode from "vscode";
import { NexusClient } from "../nexusClient";

interface ProviderPickItem extends vscode.QuickPickItem {
  providerName: string;
  modelName: string;
}

export async function selectProviderCommand(
  client: NexusClient
): Promise<void> {
  let providers;
  try {
    providers = await client.getProviders();
  } catch {
    vscode.window.showErrorMessage(
      "Nexus: Could not reach daemon. Check your daemon URL setting."
    );
    return;
  }

  const active = providers.filter((p) => p.active);

  if (active.length === 0) {
    vscode.window.showErrorMessage(
      "No active providers. Check your Nexus daemon."
    );
    return;
  }

  const items: ProviderPickItem[] = [
    {
      label: "$(zap) Auto (let Nexus choose)",
      description: "Use whatever provider/model is available",
      providerName: "",
      modelName: "",
    },
  ];

  for (const p of active) {
    const models =
      p.models && p.models.length > 0
        ? p.models
        : [p.activeModel ?? "default"];
    for (const m of models) {
      items.push({
        label: `${p.name} / ${m}`,
        description: p.name,
        providerName: p.name,
        modelName: m,
      });
    }
  }

  const chosen = await vscode.window.showQuickPick(items, {
    placeHolder: "Select default provider / model",
    title: "Nexus: Select Provider / Model",
  });

  if (!chosen) {
    return;
  }

  const cfg = vscode.workspace.getConfiguration("nexus");
  await cfg.update(
    "defaultProvider",
    chosen.providerName,
    vscode.ConfigurationTarget.Workspace
  );
  await cfg.update(
    "defaultModel",
    chosen.modelName,
    vscode.ConfigurationTarget.Workspace
  );

  const displayName = chosen.providerName
    ? `${chosen.providerName} / ${chosen.modelName}`
    : "Auto";
  vscode.window.showInformationMessage(
    `Default provider set to: ${displayName}`
  );
}
