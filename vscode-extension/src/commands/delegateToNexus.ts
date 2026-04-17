import * as vscode from 'vscode';
import * as path from 'path';
import { NexusClient, AISession } from '../nexusClient';
import { AISessionItem } from '../aiSessionsTreeProvider';
import { getNexusActivityChannel } from '../activityLog';

function delegationPath(session: AISession): 'cli' | 'mcp' | 'copilot' {
  if (session.agentName === 'GitHub Copilot' || session.agentName === 'GitHub Copilot Chat') {
    return 'copilot';
  }
  if (
    session.source === 'mcp' ||
    session.agentName.includes('Claude Desktop') ||
    session.agentName.includes('Antigravity')
  ) {
    return 'mcp';
  }
  return 'cli';
}

export async function delegateToNexusCommand(
  client: NexusClient,
  sessionOrItem?: AISessionItem | AISession,
): Promise<void> {
  const outputChannel = getNexusActivityChannel();
  let session: AISession | undefined;

  if (sessionOrItem instanceof AISessionItem) {
    session = sessionOrItem.session;
  } else if (sessionOrItem && 'id' in sessionOrItem) {
    session = sessionOrItem as AISession;
  } else {
    // Command palette: pick from active non-delegated sessions
    let sessions: AISession[] = [];
    try {
      sessions = await client.listAISessions();
    } catch {
      sessions = [];
    }

    const active = sessions.filter((s) => s.status === 'active' && !s.delegatedToNexus);
    if (active.length === 0) {
      vscode.window.showInformationMessage('No active, non-delegated AI sessions found.');
      return;
    }
    const picked = await vscode.window.showQuickPick(
      active.map((s) => ({ label: s.agentName, description: s.projectPath ?? '', session: s })),
      { placeHolder: 'Select an AI agent to delegate to Nexus' },
    );
    if (!picked) return;
    session = picked.session;
  }

  if (!session) return;

  let instruction: string;
  try {
    const resp = await client.delegateSession(session.id);
    instruction = resp.instruction;
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    vscode.window.showErrorMessage(`Failed to delegate session: ${msg}`);
    return;
  }

  const projectPath =
    session.projectPath ?? vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '';

  switch (delegationPath(session)) {
    case 'cli': {
      const fileUri = vscode.Uri.file(path.join(projectPath, '.nexus-delegate.md'));
      await vscode.workspace.fs.writeFile(fileUri, Buffer.from(instruction, 'utf8'));
      const terminal = vscode.window.createTerminal({ name: 'Nexus Delegate' });
      terminal.show();
      terminal.sendText(`echo "=== Nexus Delegation Instruction ===" && cat "${fileUri.fsPath}"`);
      try {
        const action = await vscode.window.showInformationMessage(
          'Delegation instruction written to .nexus-delegate.md',
          'Open File',
        );
        if (action === 'Open File') {
          await vscode.window.showTextDocument(fileUri);
        }
      } catch (err) {
        outputChannel.appendLine(
          `[delegateToNexus] cli action error: ${err instanceof Error ? err.message : String(err)}`,
        );
      }
      break;
    }

    case 'mcp': {
      try {
        await client.submitTask({
          instruction,
          projectPath,
          targetFile: '.nexus-delegate.md',
          command: 'auto',
        });
      } catch {
        /* best effort */
      }
      try {
        await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
      } catch (err) {
        outputChannel.appendLine(
          `[delegateToNexus] mcp chat open failed: ${err instanceof Error ? err.message : String(err)}`,
        );
        try {
          const a = await vscode.window.showInformationMessage(
            'Task submitted to Nexus queue.',
            'Copy Instruction',
          );
          if (a === 'Copy Instruction') {
            await vscode.env.clipboard.writeText(instruction);
          }
        } catch (err2) {
          outputChannel.appendLine(
            `[delegateToNexus] mcp fallback error: ${err2 instanceof Error ? err2.message : String(err2)}`,
          );
        }
      }
      break;
    }

    case 'copilot': {
      try {
        await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
      } catch (err) {
        outputChannel.appendLine(
          `[delegateToNexus] copilot chat open failed: ${err instanceof Error ? err.message : String(err)}`,
        );
        try {
          const a = await vscode.window.showInformationMessage(
            'Could not open Copilot Chat. Copy the delegation instruction?',
            'Copy',
          );
          if (a === 'Copy') {
            await vscode.env.clipboard.writeText(instruction);
          }
        } catch (err2) {
          outputChannel.appendLine(
            `[delegateToNexus] copilot fallback error: ${err2 instanceof Error ? err2.message : String(err2)}`,
          );
        }
      }
      break;
    }
  }

  // Refresh tree view so delegated session turns green
  try {
    await vscode.commands.executeCommand('nexus.refreshAISessions');
  } catch (err) {
    outputChannel.appendLine(
      `[delegateToNexus] refresh failed: ${err instanceof Error ? err.message : String(err)}`,
    );
  }
}
