/**
 * Tests for index.ts action entry point.
 *
 * index.ts calls run() at module load time.  We use jest.resetModules() per
 * test so that each require('../src/index.js') triggers a fresh execution.
 *
 * Factory mocks use `mock*`-prefixed variables (Jest hoisting exception), so
 * the same mock references work across module reloads.
 */

// ── Shared mock function references (survive jest.resetModules) ────────────

const mockInstallDaemon   = jest.fn<Promise<string>, [string]>();
const mockStartDaemon     = jest.fn();
const mockPrintDaemonLog  = jest.fn();
const mockSubmitTask      = jest.fn();
const mockWaitForTask     = jest.fn();
const mockResolveAgents   = jest.fn().mockResolvedValue([]);
const mockResolveCategory = jest.fn().mockResolvedValue([]);
const mockBuildSwarmPrompt = jest.fn().mockReturnValue('');

jest.mock('@actions/core');
jest.mock('../src/installer.js', () => ({
  installDaemon: mockInstallDaemon,
}));
jest.mock('../src/daemon.js', () => ({
  startDaemon: mockStartDaemon,
  printDaemonLog: mockPrintDaemonLog,
}));
jest.mock('../src/submit.js', () => ({
  NexusClient: jest.fn().mockImplementation(() => ({
    submitTask: mockSubmitTask,
    waitForTask: mockWaitForTask,
  })),
}));
jest.mock('../src/agents.js', () => ({
  resolveAgents: mockResolveAgents,
  resolveCategory: mockResolveCategory,
  buildSwarmPrompt: mockBuildSwarmPrompt,
}));

type CoreMock = {
  _outputs: Record<string, string>;
  _failures: string[];
  _infos: string[];
};

// ── Helpers ────────────────────────────────────────────────────────────────

function setInputs(overrides: Record<string, string> = {}): void {
  // Clear any existing INPUT_* vars
  for (const key of Object.keys(process.env)) {
    if (key.startsWith('INPUT_')) delete process.env[key];
  }
  const defaults: Record<string, string> = {
    INPUT_INSTRUCTION:     'Refactor the handler',
    INPUT_PROJECT_PATH:    '/workspace',
    INPUT_START_DAEMON:    'false',
    INPUT_DAEMON_URL:      'http://127.0.0.1:63987',
    INPUT_TIMEOUT_SECONDS: '10',
    INPUT_NEXUS_VERSION:   'latest',
    INPUT_AGENT_REF:       'main',
    // Optional fields default to empty
    INPUT_TASK_FILE:       '',
    INPUT_TARGET_FILE:     '',
    INPUT_CONTEXT_FILES:   '',
    INPUT_COMMAND:         '',
    INPUT_MODEL:           '',
    INPUT_PROVIDER:        '',
    INPUT_AGENT:           '',
    INPUT_AGENTS:          '',
    INPUT_AGENT_CATEGORY:  '',
    INPUT_SYSTEM_PROMPT:   '',
    INPUT_OPENAI_API_KEY:  '',
    INPUT_OPENAI_MODEL:    '',
    INPUT_ANTHROPIC_API_KEY: '',
    INPUT_ANTHROPIC_MODEL: '',
    INPUT_GITHUB_COPILOT_TOKEN: '',
    INPUT_GITHUB_COPILOT_MODEL: '',
  };
  Object.assign(process.env, defaults, Object.fromEntries(
    Object.entries(overrides).map(([k, v]) => [
      `INPUT_${k.toUpperCase().replace(/ /g, '_')}`, v,
    ])
  ));
}

function makeHandle(logFile = '/tmp/daemon.log') {
  return { stop: jest.fn(), logFile, pid: 1 };
}

/**
 * Reset module registry, require both core and index.js in the same registry,
 * wait for run() to settle, then return the fresh core mock for assertions.
 */
async function runIndex(): Promise<CoreMock> {
  jest.resetModules();
  // Require core FIRST — index.ts will pick up the same cached instance.
  const freshCore = require('@actions/core') as CoreMock;
  require('../src/index.js');
  // run() is async; give it time to settle.
  await new Promise((resolve) => setTimeout(resolve, 300));
  return freshCore;
}

// ── Tests ──────────────────────────────────────────────────────────────────

beforeEach(() => {
  jest.clearAllMocks();
  mockSubmitTask.mockReset();
  mockWaitForTask.mockReset();
  mockInstallDaemon.mockReset();
  mockStartDaemon.mockReset();
});

describe('index: task completion handling', () => {
  it('sets status output and does not fail on COMPLETED', async () => {
    setInputs({ instruction: 'Build feature', start_daemon: 'false' });
    mockSubmitTask.mockResolvedValue({ task_id: 't1', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({ id: 't1', status: 'COMPLETED', logs: 'all done', projectPath: '/', instruction: '', targetFile: '' });
    const core = await runIndex();
    expect(core._failures).toHaveLength(0);
    expect(core._outputs['status']).toBe('COMPLETED');
    expect(core._outputs['task_id']).toBe('t1');
  });

  it('calls setFailed on FAILED task', async () => {
    setInputs({ instruction: 'bad thing', start_daemon: 'false' });
    mockSubmitTask.mockResolvedValue({ task_id: 't2', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({ id: 't2', status: 'FAILED', logs: 'oops', projectPath: '/', instruction: '', targetFile: '' });
    const core = await runIndex();
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toContain('failed');
  });

  it('calls setFailed on TOO_LARGE with context window message', async () => {
    setInputs({ instruction: 'huge prompt', start_daemon: 'false' });
    mockSubmitTask.mockResolvedValue({ task_id: 't3', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({ id: 't3', status: 'TOO_LARGE', logs: '', projectPath: '/', instruction: '', targetFile: '' });
    const core = await runIndex();
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toContain('context window');
  });

  it('calls setFailed when submitTask throws', async () => {
    setInputs({ instruction: 'work', start_daemon: 'false' });
    mockSubmitTask.mockRejectedValue(new Error('connection refused'));
    const core = await runIndex();
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toContain('connection refused');
  });
});

describe('index: validation errors', () => {
  it('fails when neither instruction nor task_file is provided', async () => {
    setInputs({ instruction: '', task_file: '', start_daemon: 'false' });
    const core = await runIndex();
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toMatch(/instruction|task_file/i);
  });

  it('fails when task_file does not exist', async () => {
    setInputs({ instruction: '', task_file: '/nonexistent/task.md', start_daemon: 'false' });
    const core = await runIndex();
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toContain('task_file not found');
  });
});

describe('index: daemon lifecycle', () => {
  it('does not call installDaemon when start_daemon=false', async () => {
    setInputs({ instruction: 'work', start_daemon: 'false' });
    mockSubmitTask.mockResolvedValue({ task_id: 't1', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({ id: 't1', status: 'COMPLETED', logs: '', projectPath: '/', instruction: '', targetFile: '' });
    await runIndex();
    expect(mockInstallDaemon).not.toHaveBeenCalled();
    expect(mockStartDaemon).not.toHaveBeenCalled();
  });

  it('calls installDaemon and startDaemon when start_daemon=true', async () => {
    setInputs({ instruction: 'work', start_daemon: 'true', nexus_version: 'v1.0.0' });
    mockInstallDaemon.mockResolvedValue('/usr/bin/nexus-daemon');
    mockStartDaemon.mockResolvedValue(makeHandle());
    mockSubmitTask.mockResolvedValue({ task_id: 't1', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({ id: 't1', status: 'COMPLETED', logs: '', projectPath: '/', instruction: '', targetFile: '' });
    await runIndex();
    expect(mockInstallDaemon).toHaveBeenCalledWith('v1.0.0');
    expect(mockStartDaemon).toHaveBeenCalledWith(expect.objectContaining({ binPath: '/usr/bin/nexus-daemon' }));
  });

  it('stops daemon in finally block even when task submission fails', async () => {
    setInputs({ instruction: 'work', start_daemon: 'true' });
    const handle = makeHandle();
    mockInstallDaemon.mockResolvedValue('/usr/bin/nexus-daemon');
    mockStartDaemon.mockResolvedValue(handle);
    mockSubmitTask.mockRejectedValue(new Error('submit failed'));
    await runIndex();
    expect((handle.stop as jest.Mock)).toHaveBeenCalled();
  });
});
