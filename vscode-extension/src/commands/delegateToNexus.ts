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

/**
 * Directory the delegation file may be written to. A session's projectPath is
 * supplied by whoever registered it (any HTTP client can), so it is only honoured
 * when it lies inside an open workspace folder; otherwise the first folder is used.
 */
function delegationDir(requested: string | undefined): string | undefined {
  const folders = (vscode.workspace.workspaceFolders ?? []).map((f) => f.uri.fsPath);
  if (requested) {
    for (const folder of folders) {
      const rel = path.relative(folder, requested);
      if (rel === '' || (!rel.startsWith('..') && !path.isAbsolute(rel))) return requested;
    }
  }
  return folders[0];
}

/** Command: hand the current file or selection to Nexus as an instruction file for another agent. */
export async function delegateToNexusCommand(
  client: NexusClient,
  sessionOrItem?: AISessionItem | AISession,
): Promise<void> {
  let session: AISession;

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

  let instruction: string;
  try {
    const resp = await client.delegateSession(session.id);
    instruction = resp.instruction;
  } catch (err) {
    vscode.window.showErrorMessage(
      `Failed to delegate session: ${err instanceof Error ? err.message : String(err)}`,
    );
    return;
  }

  const projectPath =
    session.projectPath ?? vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '';

  switch (delegationPath(session)) {
    case 'cli': {
      const dir = delegationDir(session.projectPath);
      if (!dir) {
        vscode.window.showErrorMessage('Nexus: open a folder before delegating a CLI agent.');
        return;
      }
      const fileUri = vscode.Uri.file(path.join(dir, '.nexus-delegate.md'));
      await vscode.workspace.fs.writeFile(fileUri, Buffer.from(instruction, 'utf8'));
      // The working directory is passed as a structured option and the command is a
      // constant: no path or instruction text is ever interpolated into shell input.
      const terminal = vscode.window.createTerminal({ name: 'Nexus Delegate', cwd: dir });
      terminal.show();
      terminal.sendText('echo "=== Nexus Delegation Instruction ==="; cat .nexus-delegate.md');
      void vscode.window
        .showInformationMessage('Delegation instruction written to .nexus-delegate.md', 'Open File')
        .then((action) => {
          if (action === 'Open File') {
            void vscode.window.showTextDocument(fileUri);
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
      } catch (err) {
        // Still hand the instruction to the chat, but do not hide that nothing was queued.
        vscode.window.showWarningMessage(
          `Nexus: could not queue the delegated task — ${err instanceof Error ? err.message : String(err)}`,
        );
      }
      try {
        await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
      } catch {
        vscode.window
          .showInformationMessage('Task submitted to Nexus queue.', 'Copy Instruction')
          .then((a) => {
            if (a === 'Copy Instruction') vscode.env.clipboard.writeText(instruction);
          });
      }
      break;
    }

    case 'copilot': {
      try {
        await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
      } catch {
        vscode.window
          .showInformationMessage(
            'Could not open Copilot Chat. Copy the delegation instruction?',
            'Copy',
          )
          .then((a) => {
            if (a === 'Copy') vscode.env.clipboard.writeText(instruction);
          });
      }
      break;
    }
  }

  // Refresh tree view so delegated session turns green
  vscode.commands.executeCommand('nexus.refreshAISessions');
}
