import * as core from '@actions/core';
import * as fs from 'fs';
import * as http from 'http';

export interface DaemonOptions {
  readonly binPath: string;
  readonly listenAddr: string;
  readonly mcpAddr: string;
  readonly dbPath: string;
  readonly openaiApiKey: string;
  readonly openaiModel: string;
  readonly anthropicApiKey: string;
  readonly anthropicModel: string;
  readonly githubCopilotToken: string;
  readonly githubCopilotModel: string;
  readonly logFile: string;
}

export interface DaemonHandle {
  readonly pid: number;
  readonly logFile: string;
  stop(): void;
}

/** Spawn nexus-daemon and wait for its health endpoint to respond */
export async function startDaemon(opts: DaemonOptions): Promise<DaemonHandle> {
  const env: Record<string, string> = {
    ...(process.env as Record<string, string>),
    NEXUS_LISTEN_ADDR: opts.listenAddr,
    NEXUS_MCP_ADDR: opts.mcpAddr,
    NEXUS_DB_PATH: opts.dbPath,
  };

  if (opts.openaiApiKey.length > 0) {
    env['NEXUS_OPENAI_API_KEY'] = opts.openaiApiKey;
    if (opts.openaiModel.length > 0) env['NEXUS_OPENAI_MODEL'] = opts.openaiModel;
  }
  if (opts.anthropicApiKey.length > 0) {
    env['NEXUS_ANTHROPIC_API_KEY'] = opts.anthropicApiKey;
    if (opts.anthropicModel.length > 0) env['NEXUS_ANTHROPIC_MODEL'] = opts.anthropicModel;
  }
  if (opts.githubCopilotToken.length > 0) {
    env['NEXUS_GITHUBCOPILOT_TOKEN'] = opts.githubCopilotToken;
    if (opts.githubCopilotModel.length > 0) env['NEXUS_GITHUBCOPILOT_MODEL'] = opts.githubCopilotModel;
  }

  const logFd = fs.openSync(opts.logFile, 'w');
  const spawnOpts = {
    env,
    detached: true,
    stdio: ['ignore', logFd, logFd] as ['ignore', number, number],
    silent: true,
  };

  // @actions/exec.exec doesn't expose PID — use child_process directly
  const { spawn } = await import('child_process');
  const child = spawn(opts.binPath, [], spawnOpts);
  child.unref();

  if (child.pid == null) {
    fs.closeSync(logFd);
    throw new Error('Failed to spawn nexus-daemon: no PID assigned');
  }
  const pid = child.pid;
  fs.closeSync(logFd);

  // Wait up to 20s for health check
  const healthUrl = `http://${opts.listenAddr}/api/health`;
  await waitForHealth(healthUrl, 20_000);
  core.info(`nexus-daemon started (pid=${pid})`);

  return {
    pid,
    logFile: opts.logFile,
    stop(): void {
      try {
        process.kill(pid, 'SIGTERM');
        core.info(`nexus-daemon stopped (pid=${pid})`);
      } catch {
        // Already dead — ignore
      }
    },
  };
}

/** Poll the health URL until it responds 200 or timeout elapses */
export async function waitForHealth(
  url: string,
  timeoutMs: number
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      await httpGet(url);
      return;
    } catch {
      await sleep(500);
    }
  }
  throw new Error(`nexus-daemon did not become healthy within ${timeoutMs}ms`);
}

/** Print daemon log to GitHub Actions log group */
export function printDaemonLog(logFile: string): void {
  if (!fs.existsSync(logFile)) return;
  core.startGroup('nexus-daemon startup log');
  process.stdout.write(fs.readFileSync(logFile, 'utf-8'));
  core.endGroup();
}

function httpGet(url: string): Promise<void> {
  return new Promise((resolve, reject) => {
    http
      .get(url, { timeout: 2_000 }, (res) => {
        res.resume();
        if (res.statusCode === 200) resolve();
        else reject(new Error(`HTTP ${res.statusCode ?? '?'}`));
      })
      .on('error', reject)
      .on('timeout', () => reject(new Error('timeout')));
  });
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
