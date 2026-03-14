import * as vscode from 'vscode';
import * as path from 'path';
import { NexusClient, AISession } from '../nexusClient';
import { AISessionItem } from '../aiSessionsTreeProvider';

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
    } catch { sessions = []; }

    const active = sessions.filter(s => s.status === 'active' && !s.delegatedToNexus);
    if (active.length === 0) {
      vscode.window.showInformationMessage('No active, non-delegated AI sessions found.');
      return;
    }
    const picked = await vscode.window.showQuickPick(
      active.map(s => ({ label: s.agentName, description: s.projectPath ?? '', session: s })),
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
    vscode.window.showErrorMessage(`Failed to delegate session: ${(err as Error).message}`);
    return;
  }

  const projectPath =
    session.projectPath ??
    vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ??
    '';

  switch (delegationPath(session)) {
    case 'cli': {
      const fileUri = vscode.Uri.file(path.join(projectPath, '.nexus-delegate.md'));
      await vscode.workspace.fs.writeFile(fileUri, Buffer.from(instruction, 'utf8'));
      const terminal = vscode.window.createTerminal({ name: 'Nexus Delegate' });
      terminal.show();
      terminal.sendText(`echo "=== Nexus Delegation Instruction ===" && cat "${fileUri.fsPath}"`);
      vscode.window.showInformationMessage(
        'Delegation instruction written to .nexus-delegate.md',
        'Open File',
      ).then(action => {
        if (action === 'Open File') {
          vscode.window.showTextDocument(fileUri);
        }
      });
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
      } catch { /* best effort */ }
      try {
        await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
      } catch {
        vscode.window.showInformationMessage('Task submitted to Nexus queue.', 'Copy Instruction').then(a => {
          if (a === 'Copy Instruction') vscode.env.clipboard.writeText(instruction);
        });
      }
      break;
    }

    case 'copilot': {
      try {
        await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
      } catch {
        vscode.window.showInformationMessage(
          'Could not open Copilot Chat. Copy the delegation instruction?',
          'Copy',
        ).then(a => {
          if (a === 'Copy') vscode.env.clipboard.writeText(instruction);
        });
      }
      break;
    }
  }

  // Refresh tree view so delegated session turns green
  vscode.commands.executeCommand('nexus.refreshAISessions');
}
