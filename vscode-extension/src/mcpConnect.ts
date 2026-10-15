/**
 * mcpConnect.ts: makes the agent of the editor this extension runs in able to call Nexus tools.
 *
 * An agent that cannot reach the Nexus MCP server can only work around it, for example by editing
 * the database by hand, and then Nexus shows state it never ran. So:
 *  - VS Code (1.101+): the extension registers the server with VS Code's own MCP support, no files.
 *  - Antigravity: its agent reads ~/.gemini/.../mcp_config.json. The extension merges a `nexus`
 *    server into that file (asking first, once) and keeps the entry pointing at the bridge it ships
 *    (dist/mcpProxy.js), so it survives extension updates.
 */
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import * as vscode from 'vscode';

/** The editors this module knows how to connect. */
export type Host = 'vscode' | 'antigravity' | 'other';

/** Tells the editor from `vscode.env.appName`. */
export function detectHost(appName: string): Host {
  if (/antigravity/i.test(appName)) return 'antigravity';
  if (/visual studio code|vscode/i.test(appName)) return 'vscode';
  return 'other';
}

/** The MCP server entry written for stdio agents. */
export interface StdioEntry {
  command: string;
  args: string[];
  env: Record<string, string>;
}

/** Builds the entry that runs the bundled bridge with the extension host's own Node. */
export function stdioEntry(proxyPath: string, mcpUrl: string, token: string): StdioEntry {
  return {
    command: process.execPath,
    args: [proxyPath],
    env: {
      ELECTRON_RUN_AS_NODE: '1',
      NEXUS_MCP_URL: mcpUrl,
      ...(token ? { NEXUS_MCP_TOKEN: token } : {}),
    },
  };
}

/** Antigravity's MCP config: the file in ~/.gemini/antigravity (often a symlink), else ~/.gemini/config. */
export function antigravityConfigPath(home: string): string | undefined {
  for (const [file, dir] of [
    [
      path.join(home, '.gemini', 'antigravity', 'mcp_config.json'),
      path.join(home, '.gemini', 'antigravity'),
    ],
    [path.join(home, '.gemini', 'config', 'mcp_config.json'), path.join(home, '.gemini', 'config')],
  ] as const) {
    if (fs.existsSync(file) || fs.existsSync(dir)) return file;
  }
  return undefined;
}

/** What `connectAntigravity` found or did. */
export type ConnectState =
  'connected' | 'updated' | 'already' | 'absent' | 'invalid' | 'not-installed';

function readConfig(file: string): { doc: Record<string, unknown>; raw: string } | undefined {
  let raw = '';
  try {
    raw = fs.readFileSync(file, 'utf8');
  } catch {
    // a missing file is an empty config
  }
  if (raw.trim() === '') return { doc: {}, raw };
  try {
    const parsed: unknown = JSON.parse(raw);
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) return undefined;
    return { doc: parsed as Record<string, unknown>, raw };
  } catch {
    return undefined;
  }
}

/**
 * Looks at Antigravity's MCP config and, when `write` is set, adds or refreshes the nexus entry
 * (other servers and keys stay; the previous file is kept as <file>.nexus-backup; the file is
 * replaced atomically and a symlink is written through). Without `write` it only reports:
 * "already" (up to date), "updated" is only returned after a write, "absent" (no nexus entry),
 * "connected" after adding one, "invalid" for a file that is not a JSON object, "not-installed"
 * when Antigravity has no config folder.
 */
export function connectAntigravity(home: string, entry: StdioEntry, write: boolean): ConnectState {
  const shown = antigravityConfigPath(home);
  if (!shown) return 'not-installed';
  let file = shown;
  try {
    file = fs.realpathSync(shown);
  } catch {
    // the file does not exist yet: create it where it would be
  }
  const cfg = readConfig(file);
  if (!cfg) return 'invalid';
  const servers = (cfg.doc['mcpServers'] as Record<string, unknown> | undefined) ?? {};
  const current = servers['nexus'];
  if (JSON.stringify(current) === JSON.stringify(entry)) return 'already';
  if (!write) return 'absent';
  const next = { ...cfg.doc, mcpServers: { ...servers, nexus: entry } };
  if (cfg.raw !== '') fs.writeFileSync(`${file}.nexus-backup`, cfg.raw, { mode: 0o600 });
  fs.mkdirSync(path.dirname(file), { recursive: true });
  const tmp = `${file}.nexus-tmp`;
  fs.writeFileSync(tmp, JSON.stringify(next, null, 2) + '\n', { mode: 0o600 });
  fs.renameSync(tmp, file);
  return current === undefined ? 'connected' : 'updated';
}

/** True when the nexus entry exists but differs from `entry` (an older extension version wrote it). */
export function hasStaleEntry(home: string, entry: StdioEntry): boolean {
  const file = antigravityConfigPath(home);
  const cfg = file ? readConfig(fs.existsSync(file) ? fs.realpathSync(file) : file) : undefined;
  const current = (cfg?.doc['mcpServers'] as Record<string, unknown> | undefined)?.['nexus'];
  return current !== undefined && JSON.stringify(current) !== JSON.stringify(entry);
}

interface McpApi {
  lm?: {
    registerMcpServerDefinitionProvider?: (id: string, provider: unknown) => vscode.Disposable;
  };
  McpHttpServerDefinition?: new (
    label: string,
    uri: vscode.Uri,
    headers?: Record<string, string>,
  ) => unknown;
}

const DISMISSED_KEY = 'nexus.mcpConnectDismissed';

/** Connects the editor's agent to the daemon's MCP server. */
export class McpConnect {
  constructor(
    private readonly context: vscode.ExtensionContext,
    private readonly endpoint: () => { url: string; token: string },
    private readonly home: () => string = os.homedir,
  ) {}

  /** Registers the command and does what the editor needs at startup. */
  register(): void {
    this.context.subscriptions.push(
      vscode.commands.registerCommand('nexus.connectAgent', () => this.connectNow()),
    );
    const host = detectHost(vscode.env.appName);
    if (host === 'vscode') this.registerWithVSCode();
    if (host === 'antigravity') void this.antigravityAtStartup();
  }

  private mode(): 'ask' | 'always' | 'never' {
    const v = vscode.workspace.getConfiguration('nexus.mcp').get<string>('autoConnect', 'ask');
    return v === 'always' || v === 'never' ? v : 'ask';
  }

  private entry(): StdioEntry {
    const { url, token } = this.endpoint();
    return stdioEntry(this.context.asAbsolutePath(path.join('dist', 'mcpProxy.js')), url, token);
  }

  /** VS Code (1.101+) has MCP support of its own: offer the daemon's server through it. */
  private registerWithVSCode(): void {
    const api = vscode as unknown as McpApi;
    if (!api.lm?.registerMcpServerDefinitionProvider || !api.McpHttpServerDefinition) return;
    const Def = api.McpHttpServerDefinition;
    this.context.subscriptions.push(
      api.lm.registerMcpServerDefinitionProvider('nexus.mcp', {
        provideMcpServerDefinitions: () => {
          const { url, token } = this.endpoint();
          return [
            new Def(
              'Nexus Orchestrator',
              vscode.Uri.parse(url),
              token ? { Authorization: `Bearer ${token}` } : {},
            ),
          ];
        },
      }),
    );
  }

  private async antigravityAtStartup(): Promise<void> {
    if (this.mode() === 'never') return;
    const entry = this.entry();
    const state = connectAntigravity(this.home(), entry, false);
    if (state === 'already' || state === 'not-installed' || state === 'invalid') return;
    // An entry an older version of the extension wrote points at a path that no longer exists: refresh it.
    if (hasStaleEntry(this.home(), entry) || this.mode() === 'always') {
      connectAntigravity(this.home(), entry, true);
      return;
    }
    if (this.context.globalState.get<boolean>(DISMISSED_KEY)) return;
    const pick = await vscode.window.showInformationMessage(
      "Antigravity's agent cannot call Nexus yet, so its work is invisible to the orchestrator and its brain. Connect it?",
      'Connect',
      'Not now',
      "Don't ask again",
    );
    if (pick === 'Connect') await this.connectNow();
    else if (pick === "Don't ask again") await this.context.globalState.update(DISMISSED_KEY, true);
  }

  /** The command "Nexus: Connect This Editor's Agent": connects now, whatever the setting says. */
  async connectNow(): Promise<void> {
    const host = detectHost(vscode.env.appName);
    if (host === 'antigravity') {
      const state = connectAntigravity(this.home(), this.entry(), true);
      const text: Record<ConnectState, string> = {
        connected:
          'Nexus added to the Antigravity MCP config. Reload the window (or restart Antigravity) so its agent picks it up.',
        updated:
          'The Nexus entry in the Antigravity MCP config was refreshed. Reload the window so its agent picks it up.',
        already: "Antigravity's agent is already connected to Nexus.",
        absent: 'Nexus could not be added.',
        invalid:
          "Antigravity's MCP config is not valid JSON; fix or empty ~/.gemini/config/mcp_config.json first.",
        'not-installed': 'No Antigravity MCP config folder (~/.gemini) was found.',
      };
      const show =
        state === 'invalid' || state === 'not-installed'
          ? vscode.window.showErrorMessage
          : vscode.window.showInformationMessage;
      void show(`Nexus: ${text[state]}`);
      return;
    }
    if (host === 'vscode') {
      void vscode.window.showInformationMessage(
        'Nexus: VS Code gets the Nexus MCP server from this extension (Chat → Configure Tools, or "MCP: List Servers"). It needs VS Code 1.101 or newer.',
      );
      return;
    }
    void vscode.window.showInformationMessage(
      'Nexus: for this editor run `nexus-cli connect --write` in a terminal (it knows Antigravity, Cursor and Claude Code).',
    );
  }
}
