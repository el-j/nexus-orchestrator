import * as vscode from 'vscode';
import * as fs from 'fs/promises';
import * as os from 'os';
import * as path from 'path';
import { NexusClient } from './nexusClient';

export interface DetectedAgent {
  agentName: string;
  source: 'vscode-discovered';
  externalId: string;
  projectPath?: string;
  capabilities: string[];
  detectionMethod: string;
}

const knownAIExtensions: Record<string, { name: string; capabilities: string[] }> = {
  'saoudrizwan.claude-dev':         { name: 'Cline',               capabilities: ['file-write', 'code-execute', 'terminal'] },
  'continue.continue':              { name: 'Continue',            capabilities: ['file-write', 'code-execute', 'chat'] },
  'codeium.codeium':                { name: 'Codeium',             capabilities: ['chat'] },
  'codegpt.codegpt-4':              { name: 'CodeGPT',             capabilities: ['chat'] },
  'anysphere.cursor-always-local':  { name: 'Cursor AI',           capabilities: ['file-write', 'code-execute'] },
  'github.copilot':                 { name: 'GitHub Copilot',      capabilities: ['chat'] },
  'github.copilot-chat':            { name: 'GitHub Copilot Chat', capabilities: ['chat'] },
};

export class AgentDetector implements vscode.Disposable {
  private readonly _onDidChange = new vscode.EventEmitter<void>();
  readonly onDidChange: vscode.Event<void> = this._onDidChange.event;

  private intervalHandle: ReturnType<typeof setInterval> | undefined;
  private sessionMap = new Map<string, string>(); // externalId → sessionId
  private isDisposed = false;
  private machineId: string;

  constructor(
    private readonly client: NexusClient,
    private readonly context: vscode.ExtensionContext,
  ) {
    this.machineId = context.globalState.get<string>('machineId') ??
      `m-${Math.random().toString(36).slice(2)}`;
    void context.globalState.update('machineId', this.machineId);
  }

  start(): void {
    if (this.isDisposed) return;
    void this.tick();
    this.intervalHandle = setInterval(() => void this.tick(), 30_000);
  }

  stop(): void {
    if (this.intervalHandle !== undefined) {
      clearInterval(this.intervalHandle);
      this.intervalHandle = undefined;
    }
  }

  dispose(): void {
    this.isDisposed = true;
    this.stop();
    this._onDidChange.dispose();
  }

  async detectAll(): Promise<DetectedAgent[]> {
    const results = await Promise.allSettled([
      this.detectVSCodeExtensions(),
      this.detectFromFilesystem(),
      this.detectFromTerminals(),
      this.detectActiveLMParticipants(),
    ]);

    const seen = new Map<string, DetectedAgent>();
    for (const r of results) {
      if (r.status === 'fulfilled') {
        for (const agent of r.value) {
          if (!seen.has(agent.externalId)) {
            seen.set(agent.externalId, agent);
          }
        }
      }
    }
    return [...seen.values()];
  }

  private async tick(): Promise<void> {
    if (this.isDisposed) return;
    try {
      const detected = await this.detectAll();
      const currentIds = new Set(detected.map(a => a.externalId));

      // Register new agents
      for (const agent of detected) {
        if (!this.sessionMap.has(agent.externalId)) {
          try {
            const workspacePath = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '';
            const session = await this.client.registerSession({
              agentName: agent.agentName,
              source: 'vscode',
              externalId: agent.externalId,
              projectPath: agent.projectPath ?? workspacePath,
            });
            this.sessionMap.set(agent.externalId, session.id);
          } catch (_) { /* silent */ }
        }
      }

      // Heartbeat existing agents
      for (const [externalId, sessionId] of this.sessionMap) {
        if (currentIds.has(externalId)) {
          try { await this.client.heartbeatSession(sessionId); } catch (_) { /* silent */ }
        }
      }

      // Deregister disappeared agents
      for (const [externalId, sessionId] of this.sessionMap) {
        if (!currentIds.has(externalId)) {
          try { await this.client.deregisterSession(sessionId); } catch (_) { /* silent */ }
          this.sessionMap.delete(externalId);
        }
      }

      this._onDidChange.fire();
    } catch (_) { /* silent */ }
  }

  private async detectVSCodeExtensions(): Promise<DetectedAgent[]> {
    const results: DetectedAgent[] = [];
    for (const [extId, info] of Object.entries(knownAIExtensions)) {
      if (vscode.extensions.getExtension(extId)) {
        results.push({
          agentName: info.name,
          source: 'vscode-discovered',
          externalId: `discover:${this.machineId}:ext:${extId}`,
          capabilities: info.capabilities,
          detectionMethod: 'vscode-extension',
        });
      }
    }
    return results;
  }

  private async detectFromFilesystem(): Promise<DetectedAgent[]> {
    const results: DetectedAgent[] = [];
    const home = os.homedir();

    // Claude CLI
    try {
      const data = JSON.parse(await fs.readFile(path.join(home, '.claude', 'settings.json'), 'utf8')) as unknown;
      if (data && typeof data === 'object' && 'apiKey' in data && typeof (data as Record<string, unknown>).apiKey === 'string') {
        results.push({
          agentName: 'Claude CLI',
          source: 'vscode-discovered',
          externalId: `discover:${this.machineId}:fs:claude-cli`,
          capabilities: ['code-execute', 'terminal'],
          detectionMethod: 'fs-config',
        });
      }
    } catch (_) { /* not found */ }

    // Claude Desktop
    const desktopPaths = [
      path.join(home, 'Library', 'Application Support', 'Claude'),
      path.join(home, '.config', 'claude'),
    ];
    for (const p of desktopPaths) {
      try {
        await fs.stat(p);
        results.push({
          agentName: 'Claude Desktop',
          source: 'vscode-discovered',
          externalId: `discover:${this.machineId}:fs:claude-desktop`,
          capabilities: ['chat', 'mcp-client'],
          detectionMethod: 'fs-config',
        });
        break;
      } catch (_) { /* not found */ }
    }

    // MCP servers from Claude Desktop config
    const mcpConfigPaths = [
      path.join(home, 'Library', 'Application Support', 'Claude', 'claude_desktop_config.json'),
      path.join(home, '.config', 'claude', 'claude_desktop_config.json'),
    ];
    for (const cfgPath of mcpConfigPaths) {
      try {
        const data = JSON.parse(await fs.readFile(cfgPath, 'utf8')) as unknown;
        if (
          data &&
          typeof data === 'object' &&
          'mcpServers' in data &&
          data.mcpServers !== null &&
          typeof data.mcpServers === 'object'
        ) {
          for (const serverName of Object.keys(data.mcpServers as Record<string, unknown>)) {
            results.push({
              agentName: `MCP: ${serverName}`,
              source: 'vscode-discovered',
              externalId: `discover:${this.machineId}:mcp:${serverName}`,
              capabilities: ['mcp-client'],
              detectionMethod: 'fs-mcp-config',
            });
          }
        }
        break;
      } catch (_) { /* not found */ }
    }

    return results;
  }

  private async detectFromTerminals(): Promise<DetectedAgent[]> {
    const pattern = /claude|cline|continue|cursor|copilot/i;
    const results: DetectedAgent[] = [];
    for (const term of vscode.window.terminals) {
      const shellPath = (term.creationOptions as vscode.TerminalOptions).shellPath ?? '';
      if (pattern.test(term.name) || pattern.test(shellPath)) {
        results.push({
          agentName: `${term.name} (terminal)`,
          source: 'vscode-discovered',
          externalId: `discover:${this.machineId}:term:${term.name}`,
          capabilities: ['terminal', 'code-execute'],
          detectionMethod: 'terminal',
        });
      }
    }
    return results;
  }

  private async detectActiveLMParticipants(): Promise<DetectedAgent[]> {
    const results: DetectedAgent[] = [];
    try {
      if (vscode.lm?.selectChatModels) {
        const models = await vscode.lm.selectChatModels({});
        for (const m of models) {
          if (m.vendor !== 'copilot') {
            results.push({
              agentName: `${m.vendor}/${m.family}`,
              source: 'vscode-discovered',
              externalId: `discover:${this.machineId}:lm:${m.vendor}:${m.family}`,
              capabilities: ['chat'],
              detectionMethod: 'lm-participant',
            });
          }
        }
      }
    } catch (_) { /* lm not available */ }
    return results;
  }
}
