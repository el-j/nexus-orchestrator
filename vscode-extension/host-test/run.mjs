// Runs the extension inside a real VS Code (Electron) host against a real
// nexus-daemon. Usage: node host-test/run.mjs   (set NEXUS_DAEMON_BIN to the built daemon)
//   Linux CI needs a display: xvfb-run -a node host-test/run.mjs
import { spawn } from 'node:child_process';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { existsSync } from 'node:fs';
import { downloadAndUnzipVSCode, runTests } from '@vscode/test-electron';

const here = dirname(fileURLToPath(import.meta.url));
const extensionDevelopmentPath = resolve(here, '..');
const daemonBin = resolve(
  process.env.NEXUS_DAEMON_BIN ?? join(here, '../../build/bin/nexus-daemon-e2e'),
);

const work = mkdtempSync(join(tmpdir(), 'nexus-host-'));
const workspace = join(work, 'workspace');
const home = join(work, 'home');
const { mkdirSync } = await import('node:fs');
mkdirSync(workspace, { recursive: true });
mkdirSync(home, { recursive: true });
writeFileSync(
  join(workspace, 'CLAUDE.md'),
  '# Project\n\n## Architecture\nThe walrus scheduler owns retries.\n\n## Conventions\nPrefer small files.\n',
);

const daemon = spawn(daemonBin, [], {
  stdio: 'inherit',
  env: {
    ...process.env,
    HOME: home,
    USERPROFILE: home,
    NEXUS_DB_PATH: join(work, 'nexus.db'),
    NEXUS_LISTEN_ADDR: '127.0.0.1:63987',
    NEXUS_MCP_ADDR: '127.0.0.1:63988',
    NEXUS_LMSTUDIO_URL: 'http://127.0.0.1:1',
    NEXUS_OLLAMA_URL: 'http://127.0.0.1:1',
    NEXUS_SCAN_INTERVAL: '1h',
  },
});

async function waitForDaemon() {
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch('http://127.0.0.1:63987/api/health')).ok) return;
    } catch {
      /* not up yet */
    }
    await new Promise((ok) => setTimeout(ok, 200));
  }
  throw new Error('daemon did not become healthy');
}

// Newer macOS builds ship the executable as "Code"; older ones as "Electron".
function fixExecutable(path) {
  if (existsSync(path)) return path;
  const alt = join(dirname(path), 'Code');
  return existsSync(alt) ? alt : path;
}

let code = 1;
try {
  await waitForDaemon();
  const version = process.env.VSCODE_VERSION ?? 'stable';
  const vscodeExecutablePath = fixExecutable(await downloadAndUnzipVSCode(version));
  await runTests({
    vscodeExecutablePath,
    extensionDevelopmentPath,
    extensionTestsPath: join(here, 'suite/index.cjs'),
    extensionTestsEnv: {
      NEXUS_TEST_WORKSPACE: workspace,
      NEXUS_TEST_API: 'http://127.0.0.1:63987',
    },
    launchArgs: [workspace, '--disable-extensions', '--user-data-dir', join(work, 'user-data')],
  });
  code = 0;
} catch (err) {
  console.error('extension host tests failed:', err);
} finally {
  daemon.kill('SIGTERM');
  rmSync(work, { recursive: true, force: true });
}
process.exit(code);
