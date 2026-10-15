const assert = require('node:assert');
const vscode = require('vscode');

const API = process.env.NEXUS_TEST_API;
const WORKSPACE = process.env.NEXUS_TEST_WORKSPACE;
const manifest = require('../../package.json');
const EXTENSION_ID = `${manifest.publisher}.${manifest.name}`;

async function until(check, what, timeoutMs = 20000) {
  const deadline = Date.now() + timeoutMs;
  let last;
  while (Date.now() < deadline) {
    last = await check();
    if (last) return last;
    await new Promise((ok) => setTimeout(ok, 250));
  }
  assert.fail(`timed out waiting for ${what}`);
}

suite('Nexus Orchestrator in a real VS Code host', () => {
  suiteSetup(async () => {
    const ext = vscode.extensions.getExtension(EXTENSION_ID);
    assert.ok(ext, `extension ${EXTENSION_ID} is not installed in the host`);
    await ext.activate();
  });

  test('activates', () => {
    assert.strictEqual(vscode.extensions.getExtension(EXTENSION_ID).isActive, true);
  });

  test('registers every command the manifest contributes', async () => {
    const registered = new Set(await vscode.commands.getCommands(true));
    const missing = manifest.contributes.commands
      .map((c) => c.command)
      .filter((c) => !registered.has(c));
    assert.deepStrictEqual(missing, []);
  });

  test('contributes its settings with defaults', () => {
    const props =
      manifest.contributes.configuration.properties ??
      manifest.contributes.configuration[0].properties;
    const config = vscode.workspace.getConfiguration();
    for (const key of Object.keys(props)) {
      assert.notStrictEqual(config.inspect(key), undefined, `setting ${key} is not registered`);
    }
  });

  test('reconnect finds the running daemon', async () => {
    await vscode.commands.executeCommand('nexus.reconnect');
    const health = await fetch(`${API}/api/health`);
    assert.strictEqual(health.status, 200);
  });

  test('brain init sends the workspace to the daemon, which ingests CLAUDE.md', async () => {
    await vscode.commands.executeCommand('nexus.brain.init', WORKSPACE);
    const status = await until(async () => {
      const res = await fetch(
        `${API}/api/brain/status?projectPath=${encodeURIComponent(WORKSPACE)}`,
      );
      const body = await res.json();
      return body.entryCount > 0 ? body : undefined;
    }, 'the daemon to hold the ingested project knowledge');
    assert.ok(status.entryCount >= 1);

    const hits = await (
      await fetch(`${API}/api/brain/search?projectPath=${encodeURIComponent(WORKSPACE)}&q=walrus`)
    ).json();
    assert.ok(hits.results.length > 0, 'the ingested text is searchable');
  });

  test('view commands run without throwing', async () => {
    for (const cmd of [
      'nexus.showActivityLog',
      'nexus.refreshAISessions',
      'nexus.refreshWorkspaceAgents',
      'nexus.toggleWorkspaceScope',
      'nexus.toggleWorkspaceScope',
    ]) {
      await vscode.commands.executeCommand(cmd);
    }
  });
});
