/**
 * Tests for index.ts action entry point.
 *
 * index.ts calls run() at module load time.  We use jest.resetModules() per
 * test so that each require('../src/index.js') triggers a fresh execution.
 *
 * Factory mocks use `mock*`-prefixed variables (Jest hoisting exception), so
 * the same mock references work across module reloads.
 */

export {};

// ── Shared mock function references (survive jest.resetModules) ────────────

const mockInstallDaemon = jest.fn<Promise<string>, [string]>();
const mockStartDaemon = jest.fn();
const mockPrintDaemonLog = jest.fn();
const mockSubmitTask = jest.fn();
const mockWaitForTask = jest.fn();
const mockResolveAgents = jest.fn().mockResolvedValue([]);
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
  _warnings: string[];
  _infos: string[];
  _secrets: string[];
};

// ── Helpers ────────────────────────────────────────────────────────────────

function setInputs(overrides: Record<string, string> = {}): void {
  // Clear any existing INPUT_* vars
  for (const key of Object.keys(process.env)) {
    if (key.startsWith('INPUT_')) delete process.env[key];
  }
  const defaults: Record<string, string> = {
    INPUT_INSTRUCTION: 'Refactor the handler',
    INPUT_PROJECT_PATH: '/workspace',
    INPUT_START_DAEMON: 'false',
    INPUT_DAEMON_URL: 'http://127.0.0.1:63987',
    INPUT_TIMEOUT_SECONDS: '10',
    INPUT_NEXUS_VERSION: 'latest',
    INPUT_AGENT_REF: 'main',
    // Optional fields default to empty
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
    mockWaitForTask.mockResolvedValue({
      id: 't1',
      status: 'COMPLETED',
      logs: 'all done',
      projectPath: '/',
      instruction: '',
      targetFile: '',
    });
    const core = await runIndex();
    expect(core._failures).toHaveLength(0);
    expect(core._outputs['status']).toBe('COMPLETED');
    expect(core._outputs['task_id']).toBe('t1');
  });

  it('calls setFailed on FAILED task', async () => {
    setInputs({ instruction: 'bad thing', start_daemon: 'false' });
    mockSubmitTask.mockResolvedValue({ task_id: 't2', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({
      id: 't2',
      status: 'FAILED',
      logs: 'oops',
      projectPath: '/',
      instruction: '',
      targetFile: '',
    });
    const core = await runIndex();
    expect(core._failures.length).toBeGreaterThan(0);
    expect(core._failures[0]).toContain('failed');
  });

  it('calls setFailed on TOO_LARGE with context window message', async () => {
    setInputs({ instruction: 'huge prompt', start_daemon: 'false' });
    mockSubmitTask.mockResolvedValue({ task_id: 't3', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({
      id: 't3',
      status: 'TOO_LARGE',
      logs: '',
      projectPath: '/',
      instruction: '',
      targetFile: '',
    });
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
    mockWaitForTask.mockResolvedValue({
      id: 't1',
      status: 'COMPLETED',
      logs: '',
      projectPath: '/',
      instruction: '',
      targetFile: '',
    });
    await runIndex();
    expect(mockInstallDaemon).not.toHaveBeenCalled();
    expect(mockStartDaemon).not.toHaveBeenCalled();
  });

  it('calls installDaemon and startDaemon when start_daemon=true', async () => {
    setInputs({ instruction: 'work', start_daemon: 'true', nexus_version: 'v1.0.0' });
    mockInstallDaemon.mockResolvedValue('/usr/bin/nexus-daemon');
    mockStartDaemon.mockResolvedValue(makeHandle());
    mockSubmitTask.mockResolvedValue({ task_id: 't1', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({
      id: 't1',
      status: 'COMPLETED',
      logs: '',
      projectPath: '/',
      instruction: '',
      targetFile: '',
    });
    await runIndex();
    expect(mockInstallDaemon).toHaveBeenCalledWith('v1.0.0');
    expect(mockStartDaemon).toHaveBeenCalledWith(
      expect.objectContaining({ binPath: '/usr/bin/nexus-daemon' }),
    );
  });

  it('stops daemon in finally block even when task submission fails', async () => {
    setInputs({ instruction: 'work', start_daemon: 'true' });
    const handle = makeHandle();
    mockInstallDaemon.mockResolvedValue('/usr/bin/nexus-daemon');
    mockStartDaemon.mockResolvedValue(handle);
    mockSubmitTask.mockRejectedValue(new Error('submit failed'));
    await runIndex();
    expect(handle.stop as jest.Mock).toHaveBeenCalled();
  });
});

// ── Added coverage ─────────────────────────────────────────────────────────

const completed = (id = 't1', status = 'COMPLETED', logs = '') => ({
  id,
  status,
  logs,
  projectPath: '/',
  instruction: '',
  targetFile: '',
});

function primeSuccess(status = 'COMPLETED'): void {
  mockSubmitTask.mockResolvedValue({ task_id: 't1', status: 'QUEUED' });
  mockWaitForTask.mockResolvedValue(completed('t1', status));
}

const submittedRequest = (): Record<string, unknown> => mockSubmitTask.mock.calls[0][0];

describe('index: task request assembly', () => {
  it('sends only the optional fields that were provided', async () => {
    setInputs({ instruction: 'work', command: 'plan' });
    primeSuccess();
    await runIndex();
    const req = submittedRequest();
    expect(req).toMatchObject({ projectPath: '/workspace', instruction: 'work', command: 'plan' });
    for (const k of ['targetFile', 'contextFiles', 'modelId', 'providerHint']) {
      expect(req).not.toHaveProperty(k);
    }
  });

  it('sends target file, context files, model and provider when given', async () => {
    setInputs({
      instruction: 'work',
      target_file: 'a.go',
      context_files: ' x.go, ,y.go ',
      model: 'qwen',
      provider: 'ollama',
    });
    primeSuccess();
    await runIndex();
    expect(submittedRequest()).toMatchObject({
      targetFile: 'a.go',
      contextFiles: ['x.go', 'y.go'],
      modelId: 'qwen',
      providerHint: 'ollama',
      command: 'auto',
    });
  });

  it('falls back to GITHUB_WORKSPACE, then the working directory, for the project path', async () => {
    setInputs({ instruction: 'work', project_path: '' });
    process.env['GITHUB_WORKSPACE'] = '/ws';
    primeSuccess();
    await runIndex();
    expect(submittedRequest()['projectPath']).toBe('/ws');

    mockSubmitTask.mockClear();
    setInputs({ instruction: 'work', project_path: '' });
    delete process.env['GITHUB_WORKSPACE'];
    primeSuccess();
    await runIndex();
    expect(submittedRequest()['projectPath']).toBe(process.cwd());
  });

  it('reads the instruction from task_file, which wins over the instruction input', async () => {
    const fs = require('fs') as typeof import('fs');
    const os = require('os') as typeof import('os');
    const file = require('path').join(os.tmpdir(), `nexus-task-${process.pid}.md`);
    fs.writeFileSync(file, '  from the file \n');
    setInputs({ instruction: 'ignored', task_file: file });
    primeSuccess();
    await runIndex();
    fs.unlinkSync(file);
    expect(submittedRequest()['instruction']).toBe('from the file');
  });

  it('points at the dashboard of the daemon that was used', async () => {
    setInputs({ instruction: 'work', daemon_url: 'http://nexus.local:9' });
    primeSuccess();
    const core = await runIndex();
    expect(core._infos.some((i) => i.includes('http://nexus.local:9/ui'))).toBe(true);
  });
});

describe('index: agent identity', () => {
  const agent = (name: string) => ({ name, category: 'engineering', systemPrompt: `be ${name}` });

  it('wraps the instruction in a manual system prompt, which beats any agent input', async () => {
    setInputs({ instruction: 'work', system_prompt: 'You are terse.', agent: 'x' });
    primeSuccess();
    await runIndex();
    expect(submittedRequest()['instruction']).toBe('<system>\nYou are terse.\n</system>\n\nwork');
    expect(mockResolveAgents).not.toHaveBeenCalled();
  });

  it('loads a single agent and uses its system prompt', async () => {
    mockResolveAgents.mockResolvedValueOnce([agent('Backend')]);
    setInputs({ instruction: 'work', agent: 'backend', agent_ref: 'v2' });
    primeSuccess();
    await runIndex();
    expect(mockResolveAgents).toHaveBeenCalledWith(['backend'], 'v2');
    expect(submittedRequest()['instruction']).toContain('be Backend');
  });

  it('fails when a single agent cannot be loaded', async () => {
    mockResolveAgents.mockResolvedValueOnce([]);
    setInputs({ instruction: 'work', agent: 'ghost' });
    const core = await runIndex();
    expect(core._failures[0]).toContain('Agent "ghost" not found');
    expect(mockSubmitTask).not.toHaveBeenCalled();
  });

  it('builds a swarm prompt from named agents', async () => {
    mockResolveAgents.mockResolvedValueOnce([agent('A'), agent('B')]);
    mockBuildSwarmPrompt.mockReturnValueOnce('SWARM');
    setInputs({ instruction: 'work', agents: 'a, ,b' });
    primeSuccess();
    await runIndex();
    expect(mockResolveAgents).toHaveBeenCalledWith(['a', 'b'], 'main');
    expect(mockBuildSwarmPrompt).toHaveBeenCalledWith(expect.any(Array), 'Agency Swarm');
    expect(submittedRequest()['instruction']).toContain('SWARM');
  });

  it('builds a swarm prompt from a whole category', async () => {
    mockResolveCategory.mockResolvedValueOnce([agent('A')]);
    mockBuildSwarmPrompt.mockReturnValueOnce('CAT');
    setInputs({ instruction: 'work', agent_category: 'design' });
    primeSuccess();
    await runIndex();
    expect(mockResolveCategory).toHaveBeenCalledWith('design', 'main');
    expect(mockBuildSwarmPrompt).toHaveBeenCalledWith(expect.any(Array), 'design');
    expect(submittedRequest()['instruction']).toContain('CAT');
  });

  it('reports a resolution failure and still stops a started daemon', async () => {
    mockResolveCategory.mockRejectedValueOnce(new Error('Unknown category'));
    const handle = makeHandle();
    mockInstallDaemon.mockResolvedValue('/bin/nexus-daemon');
    mockStartDaemon.mockResolvedValue(handle);
    setInputs({ instruction: 'work', agent_category: 'nope', start_daemon: 'true' });
    const core = await runIndex();
    expect(core._failures[0]).toContain('Unknown category');
    expect(handle.stop as jest.Mock).toHaveBeenCalled();
  });
});

describe('index: input validation', () => {
  it.each(['EXECUTE', 'run', 'AUTO'])(
    'rejects command "%s" before doing any work',
    async (command) => {
      setInputs({ instruction: 'work', command, start_daemon: 'true' });
      const core = await runIndex();
      expect(core._failures[0]).toContain(`Invalid command "${command}"`);
      expect(mockInstallDaemon).not.toHaveBeenCalled();
      expect(mockSubmitTask).not.toHaveBeenCalled();
    },
  );

  it.each(['plan', 'execute', 'auto'])(
    'passes the "%s" command through to the daemon',
    async (command) => {
      setInputs({ instruction: 'work', command });
      primeSuccess();
      await runIndex();
      expect(submittedRequest()).toMatchObject({ command });
    },
  );

  it.each(['abc', '0', '-5', '1.5', '300s'])('rejects timeout_seconds "%s"', async (timeout) => {
    setInputs({ instruction: 'work', timeout_seconds: timeout });
    const core = await runIndex();
    expect(core._failures[0]).toContain('Invalid timeout_seconds');
    expect(mockSubmitTask).not.toHaveBeenCalled();
  });

  it('defaults the timeout to 300 seconds and passes it on in milliseconds', async () => {
    setInputs({ instruction: 'work', timeout_seconds: '' });
    primeSuccess();
    await runIndex();
    expect(mockWaitForTask).toHaveBeenCalledWith('t1', 300_000);
  });

  it('masks provider credentials so they cannot leak into logs', async () => {
    setInputs({
      instruction: 'work',
      openai_api_key: 'sk-openai',
      anthropic_api_key: 'sk-anthropic',
      github_copilot_token: 'ghu_token',
    });
    primeSuccess();
    const core = await runIndex();
    expect(core._secrets).toEqual(['sk-openai', 'sk-anthropic', 'ghu_token']);
  });

  it('masks nothing when no credentials are given', async () => {
    setInputs({ instruction: 'work' });
    primeSuccess();
    const core = await runIndex();
    expect(core._secrets).toEqual([]);
  });

  it('passes credentials and models to the daemon it starts', async () => {
    setInputs({
      instruction: 'work',
      start_daemon: 'true',
      openai_api_key: 'k1',
      openai_model: 'gpt',
      anthropic_api_key: 'k2',
      anthropic_model: 'opus',
      github_copilot_token: 'k3',
      github_copilot_model: 'cp',
    });
    mockInstallDaemon.mockResolvedValue('/bin/d');
    mockStartDaemon.mockResolvedValue(makeHandle());
    primeSuccess();
    await runIndex();
    expect(mockStartDaemon).toHaveBeenCalledWith(
      expect.objectContaining({
        openaiApiKey: 'k1',
        openaiModel: 'gpt',
        anthropicApiKey: 'k2',
        anthropicModel: 'opus',
        githubCopilotToken: 'k3',
        githubCopilotModel: 'cp',
      }),
    );
  });
});

describe('index: terminal statuses', () => {
  it('only warns for a CANCELLED task', async () => {
    setInputs({ instruction: 'work' });
    primeSuccess('CANCELLED');
    const core = await runIndex();
    expect(core._failures).toHaveLength(0);
    expect(core._warnings[0]).toContain('was cancelled');
  });

  it('fails on a status it does not recognise', async () => {
    setInputs({ instruction: 'work' });
    primeSuccess('QUEUED');
    const core = await runIndex();
    expect(core._failures[0]).toContain('Unexpected terminal status: QUEUED');
  });

  it('stringifies a thrown non-Error and reports an empty log', async () => {
    setInputs({ instruction: 'work' });
    mockSubmitTask.mockRejectedValue('plain string');
    const core = await runIndex();
    expect(core._failures[0]).toBe('plain string');
    expect(core._outputs['logs']).toBe('');
    expect(core._outputs['status']).toBe('FAILED');
  });

  it('prints the daemon log after the run', async () => {
    const handle = makeHandle('/tmp/x.log');
    mockInstallDaemon.mockResolvedValue('/bin/d');
    mockStartDaemon.mockResolvedValue(handle);
    setInputs({ instruction: 'work', start_daemon: 'true' });
    primeSuccess();
    await runIndex();
    expect(mockPrintDaemonLog).toHaveBeenCalledWith('/tmp/x.log');
  });
});

describe('index: defaults for blank inputs', () => {
  it('falls back to the public defaults when optional inputs are blank', async () => {
    setInputs({
      instruction: 'x',
      agent_ref: '',
      daemon_url: '',
      nexus_version: '',
      start_daemon: 'true',
      agents: 'backend',
    });
    const handle = makeHandle();
    mockInstallDaemon.mockResolvedValue('/bin/nexus');
    mockStartDaemon.mockResolvedValue(handle);
    mockSubmitTask.mockResolvedValue({ task_id: 't1', status: 'QUEUED' });
    mockWaitForTask.mockResolvedValue({ id: 't1', status: 'COMPLETED', logs: '' });
    await runIndex();
    expect(mockInstallDaemon).toHaveBeenCalledWith('latest');
    expect(mockResolveAgents).toHaveBeenCalledWith(expect.anything(), 'main');
    const { NexusClient } = require('../src/submit.js') as { NexusClient: jest.Mock };
    expect(NexusClient).toHaveBeenCalledWith('http://127.0.0.1:63987');
  });
});
