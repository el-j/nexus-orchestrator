/**
 * Integration tests for the GitHub Action run() flow.
 *
 * Uses the real NexusClient (not mocked) with a factory-based HttpClient mock
 * so the full submit → poll → output path is exercised end-to-end.
 *
 * Why factory mock, not jest.spyOn?
 * jest.resetModules() inside runWithTimers() clears the module cache.  The
 * next require('@actions/http-client') creates a FRESH HttpClient class with a
 * new prototype, so spies installed on the old prototype before the reset have
 * no effect.  A factory mock closes over the module-level `mockPostJson` /
 * `mockGetJson` stubs, which survive the reset and can be pre-configured per
 * test.
 */

// ── Module-level stubs (survive jest.resetModules; prefixed "mock" for
//    Jest's hoisting rules that allow references in factory closures) ──────────
const mockPostJson = jest.fn();
const mockGetJson = jest.fn();

const mockInstallDaemon = jest.fn<Promise<string>, [string]>().mockResolvedValue('/bin/nexus');
const mockStartDaemon = jest
  .fn()
  .mockResolvedValue({ stop: jest.fn(), logFile: '/tmp/nexus.log', pid: 1 });
const mockPrintDaemonLog = jest.fn();
const mockResolveAgents = jest.fn().mockResolvedValue([]);
const mockResolveCategory = jest.fn().mockResolvedValue([]);
const mockBuildSwarmPrompt = jest.fn().mockReturnValue('');

// Factory mock: re-executed after jest.resetModules() but closes over the
// module-scope stubs above, so the same configuration is visible to the fresh
// HttpClient class used by the reloaded submit.ts.
jest.mock('@actions/http-client', () => ({
  HttpClient: jest.fn().mockImplementation(() => ({
    postJson: mockPostJson,
    getJson: mockGetJson,
  })),
}));

jest.mock('@actions/core');
jest.mock('../src/installer.js', () => ({ installDaemon: mockInstallDaemon }));
jest.mock('../src/daemon.js', () => ({
  startDaemon: mockStartDaemon,
  printDaemonLog: mockPrintDaemonLog,
}));
jest.mock('../src/agents.js', () => ({
  resolveAgents: mockResolveAgents,
  resolveCategory: mockResolveCategory,
  buildSwarmPrompt: mockBuildSwarmPrompt,
}));
// NOTE: Do NOT mock '../src/submit.js' — exercises the real NexusClient

type CoreMock = {
  _outputs: Record<string, string>;
  _failures: string[];
  _infos: string[];
};

// ── Helpers ───────────────────────────────────────────────────────────────────

function setInputs(overrides: Record<string, string> = {}): void {
  for (const key of Object.keys(process.env)) {
    if (key.startsWith('INPUT_')) delete process.env[key];
  }
  const defaults: Record<string, string> = {
    INPUT_INSTRUCTION: 'Integration test instruction',
    INPUT_PROJECT_PATH: '/workspace',
    INPUT_START_DAEMON: 'false',
    INPUT_DAEMON_URL: 'http://127.0.0.1:63987',
    INPUT_TIMEOUT_SECONDS: '30',
    INPUT_NEXUS_VERSION: 'latest',
    INPUT_AGENT_REF: 'main',
    INPUT_TASK_FILE: '',
    INPUT_TARGET_FILE: '',
    INPUT_CONTEXT_FILES: '',
    INPUT_COMMAND: '',
    INPUT_MODEL: '',
    INPUT_PROVIDER: '',
    INPUT_AGENT: '',
    INPUT_AGENTS: '',
    INPUT_AGENT_CATEGORY: '',
    INPUT_SYSTEM_PROMPT: '',
    INPUT_OPENAI_API_KEY: '',
    INPUT_OPENAI_MODEL: '',
    INPUT_ANTHROPIC_API_KEY: '',
    INPUT_ANTHROPIC_MODEL: '',
    INPUT_GITHUB_COPILOT_TOKEN: '',
    INPUT_GITHUB_COPILOT_MODEL: '',
  };
  Object.assign(
    process.env,
    defaults,
    Object.fromEntries(
      Object.entries(overrides).map(([k, v]) => [`INPUT_${k.toUpperCase().replace(/ /g, '_')}`, v]),
    ),
  );
}

async function runWithTimers(advanceMs = 60_000): Promise<CoreMock> {
  jest.resetModules();
  const freshCore = require('@actions/core') as CoreMock;
  // Require index.js — triggers run() as a side-effect
  require('../src/index.js');
  // Allow initial async work (submitTask) to start resolving
  await Promise.resolve();
  await Promise.resolve();
  // Advance fake timers to flush sleep() calls inside NexusClient.waitForTask
  await jest.advanceTimersByTimeAsync(advanceMs);
  // Flush remaining promise chains
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
  return freshCore;
}

// ── Setup / Teardown ──────────────────────────────────────────────────────────

beforeEach(() => {
  jest.useFakeTimers();
  jest.clearAllMocks();
  mockInstallDaemon.mockResolvedValue('/bin/nexus');
  mockStartDaemon.mockResolvedValue({ stop: jest.fn(), logFile: '/tmp/nexus.log', pid: 1 });
});

afterEach(() => {
  jest.useRealTimers();
  jest.restoreAllMocks();
});

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('index integration: real NexusClient + factory-mocked HttpClient', () => {
  it('success: task completes on 2nd poll — outputs task_id and COMPLETED', async () => {
    setInputs({ timeout_seconds: '30' });

    let pollCount = 0;
    mockPostJson.mockResolvedValue({
      statusCode: 201,
      result: { task_id: 'int-ok-1', status: 'QUEUED' },
      headers: {},
    });
    mockGetJson.mockImplementation(async () => {
      pollCount++;
      return {
        statusCode: 200,
        result: {
          id: 'int-ok-1',
          status: pollCount < 2 ? 'PROCESSING' : 'COMPLETED',
          logs: 'all done',
          projectPath: '/',
          instruction: '',
          targetFile: '',
        },
        headers: {},
      };
    });

    const core = await runWithTimers(60_000);

    expect(core._failures).toHaveLength(0);
    expect(core._outputs['status']).toBe('COMPLETED');
    expect(core._outputs['task_id']).toBe('int-ok-1');
  });

  it('timeout: task never reaches terminal status; action fails with timeout error', async () => {
    // timeout_seconds=0 → waitForTask deadline is already past, throws immediately
    setInputs({ timeout_seconds: '0' });

    mockPostJson.mockResolvedValue({
      statusCode: 201,
      result: { task_id: 'int-timeout-1', status: 'QUEUED' },
      headers: {},
    });
    // getJson should not even be called with 0s timeout
    mockGetJson.mockResolvedValue({
      statusCode: 200,
      result: {
        id: 'int-timeout-1',
        status: 'PROCESSING',
        logs: '',
        projectPath: '/',
        instruction: '',
        targetFile: '',
      },
      headers: {},
    });

    const core = await runWithTimers(10_000);

    expect(core._outputs['status']).toBe('FAILED');
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toMatch(/did not complete/i);
  });

  it('NO_PROVIDER is non-terminal: with 0s timeout, action fails via timeout', async () => {
    // NO_PROVIDER is not in TERMINAL_STATUSES; with 0s timeout waitForTask
    // exits with "did not complete" before ever polling getTask.
    setInputs({ timeout_seconds: '0' });

    mockPostJson.mockResolvedValue({
      statusCode: 201,
      result: { task_id: 'int-noprov-1', status: 'QUEUED' },
      headers: {},
    });
    mockGetJson.mockResolvedValue({
      statusCode: 200,
      result: {
        id: 'int-noprov-1',
        status: 'NO_PROVIDER',
        logs: 'no provider available',
        projectPath: '/',
        instruction: '',
        targetFile: '',
      },
      headers: {},
    });

    const core = await runWithTimers(5_000);

    expect(core._outputs['status']).toBe('FAILED');
    expect(core._failures.length).toBeGreaterThan(0);
  });

  it('network error on submit: action fails with the connection error message', async () => {
    setInputs({ timeout_seconds: '30' });

    mockPostJson.mockRejectedValue(new Error('ECONNREFUSED connect 127.0.0.1:63987'));

    const core = await runWithTimers(5_000);

    expect(core._outputs['status']).toBe('FAILED');
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toContain('ECONNREFUSED');
  });
});
