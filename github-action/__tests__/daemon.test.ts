import * as childProcess from 'child_process';
import * as http from 'http';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

jest.mock('@actions/core');
jest.mock('child_process');
jest.mock('http');

import { waitForHealth, printDaemonLog, startDaemon } from '../src/daemon.js';

const mockSpawn = childProcess.spawn as jest.MockedFunction<typeof childProcess.spawn>;
const mockHttpGet = http.get as jest.MockedFunction<typeof http.get>;

/** Helper: configure mockHttpGet to respond with the given status code */
function respondWith(statusCode: number): void {
  mockHttpGet.mockImplementation((_url, _opts, cb) => {
    const res = { statusCode, resume: jest.fn() } as unknown as http.IncomingMessage;
    if (cb) (cb as (r: http.IncomingMessage) => void)(res);
    const req = { on: (_e: string, _h: unknown) => req } as unknown as http.ClientRequest;
    return req;
  });
}

/** Helper: configure mockHttpGet to always return a connection error */
function refuseConnection(): void {
  mockHttpGet.mockImplementation((_url, _opts, _cb) => {
    const req = {
      on: (event: string, handler: (e: Error) => void) => {
        if (event === 'error') setImmediate(() => handler(new Error('ECONNREFUSED')));
        return req;
      },
    } as unknown as http.ClientRequest;
    return req;
  });
}

describe('waitForHealth', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('resolves when server responds 200', async () => {
    respondWith(200);
    await expect(waitForHealth('http://127.0.0.1:9999/api/health', 5000)).resolves.toBeUndefined();
  });

  it('rejects after timeout when server always refuses', async () => {
    refuseConnection();
    await expect(waitForHealth('http://127.0.0.1:9999/api/health', 200)).rejects.toThrow('did not become healthy');
  });

  it('rejects when server returns non-200 status', async () => {
    respondWith(503);
    await expect(waitForHealth('http://127.0.0.1:9999/api/health', 200)).rejects.toThrow('did not become healthy');
  });
});

describe('printDaemonLog', () => {
  it('does nothing if log file does not exist', () => {
    expect(() => printDaemonLog('/nonexistent/path/nexus.log')).not.toThrow();
  });

  it('prints log file content to stdout', () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nexus-test-'));
    const logFile = path.join(tmpDir, 'daemon.log');
    fs.writeFileSync(logFile, 'daemon started\n');

    const writeSpy = jest.spyOn(process.stdout, 'write').mockImplementation(() => true);
    printDaemonLog(logFile);
    expect(writeSpy).toHaveBeenCalledWith(expect.stringContaining('daemon started'));
    writeSpy.mockRestore();

    fs.rmSync(tmpDir, { recursive: true });
  });
});

const baseOpts = {
  binPath: '/usr/local/bin/nexus-daemon',
  listenAddr: '127.0.0.1:19999',
  mcpAddr: '127.0.0.1:19998',
  openaiApiKey: '',
  openaiModel: '',
  anthropicApiKey: '',
  anthropicModel: '',
  githubCopilotToken: '',
  githubCopilotModel: '',
};

function makeTmpDir(): { dir: string; logFile: string } {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'nexus-test-'));
  return { dir, logFile: path.join(dir, 'daemon.log') };
}

describe('startDaemon', () => {
  afterEach(() => {
    jest.clearAllMocks();
    mockSpawn.mockReset();
  });

  it('throws when spawn returns no PID', async () => {
    mockSpawn.mockReturnValue({ pid: undefined, unref: jest.fn() } as unknown as ReturnType<typeof childProcess.spawn>);
    const { dir, logFile } = makeTmpDir();
    await expect(
      startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile })
    ).rejects.toThrow('no PID assigned');
    fs.rmSync(dir, { recursive: true });
  });

  it('returns DaemonHandle with PID and sets LLM env vars', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 12345, unref: jest.fn() } as unknown as ReturnType<typeof childProcess.spawn>);
    const { dir, logFile } = makeTmpDir();
    const handle = await startDaemon({
      ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile,
      openaiApiKey: 'sk-test', openaiModel: 'gpt-4',
    });
    expect(handle.pid).toBe(12345);
    expect(handle.logFile).toBe(logFile);
    const spawnEnv = (mockSpawn.mock.calls[0][2] as { env: Record<string, string> }).env;
    expect(spawnEnv['NEXUS_OPENAI_API_KEY']).toBe('sk-test');
    expect(spawnEnv['NEXUS_OPENAI_MODEL']).toBe('gpt-4');
    expect(spawnEnv['NEXUS_LISTEN_ADDR']).toBe('127.0.0.1:19999');
    const killSpy = jest.spyOn(process, 'kill').mockImplementation(() => true);
    handle.stop();
    expect(killSpy).toHaveBeenCalledWith(12345, 'SIGTERM');
    fs.rmSync(dir, { recursive: true });
  });

  it('stop() does not throw when process is already dead', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 99999, unref: jest.fn() } as unknown as ReturnType<typeof childProcess.spawn>);
    const { dir, logFile } = makeTmpDir();
    const handle = await startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile });
    jest.spyOn(process, 'kill').mockImplementation(() => { throw new Error('ESRCH'); });
    expect(() => handle.stop()).not.toThrow();
    fs.rmSync(dir, { recursive: true });
  });

  it('stop() sends SIGTERM to the daemon process', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 9999, unref: jest.fn() } as unknown as ReturnType<typeof childProcess.spawn>);
    const { dir, logFile } = makeTmpDir();
    const handle = await startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile });
    const killSpy = jest.spyOn(process, 'kill').mockImplementation(() => true);
    handle.stop();
    expect(killSpy).toHaveBeenCalledWith(9999, 'SIGTERM');
    fs.rmSync(dir, { recursive: true });
  });

  it('does not set Anthropic env vars when key is empty', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 100, unref: jest.fn() } as unknown as ReturnType<typeof childProcess.spawn>);
    const { dir, logFile } = makeTmpDir();
    await startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile });
    const spawnEnv = (mockSpawn.mock.calls[0][2] as { env: Record<string, string> }).env;
    expect(spawnEnv['NEXUS_ANTHROPIC_API_KEY']).toBeUndefined();
    fs.rmSync(dir, { recursive: true });
  });
});
