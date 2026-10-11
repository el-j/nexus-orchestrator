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
    await expect(waitForHealth('http://127.0.0.1:63987/api/health', 5000)).resolves.toBeUndefined();
  });

  it('rejects after timeout when server always refuses', async () => {
    refuseConnection();
    await expect(waitForHealth('http://127.0.0.1:63987/api/health', 200)).rejects.toThrow(
      'did not become healthy',
    );
  });

  it('rejects when server returns non-200 status', async () => {
    respondWith(503);
    await expect(waitForHealth('http://127.0.0.1:63987/api/health', 200)).rejects.toThrow(
      'did not become healthy',
    );
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
    mockSpawn.mockReturnValue({ pid: undefined, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    await expect(
      startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile }),
    ).rejects.toThrow('no PID assigned');
    fs.rmSync(dir, { recursive: true });
  });

  it('returns DaemonHandle with PID and sets LLM env vars', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 12345, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    const handle = await startDaemon({
      ...baseOpts,
      dbPath: path.join(dir, 'nexus.db'),
      logFile,
      openaiApiKey: 'sk-test',
      openaiModel: 'gpt-4',
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
    mockSpawn.mockReturnValue({ pid: 99999, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    const handle = await startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile });
    jest.spyOn(process, 'kill').mockImplementation(() => {
      throw new Error('ESRCH');
    });
    expect(() => handle.stop()).not.toThrow();
    fs.rmSync(dir, { recursive: true });
  });

  it('stop() sends SIGTERM to the daemon process', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 9999, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    const handle = await startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile });
    const killSpy = jest.spyOn(process, 'kill').mockImplementation(() => true);
    handle.stop();
    expect(killSpy).toHaveBeenCalledWith(9999, 'SIGTERM');
    fs.rmSync(dir, { recursive: true });
  });

  it('does not set Anthropic env vars when key is empty', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 100, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    await startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile });
    const spawnEnv = (mockSpawn.mock.calls[0][2] as { env: Record<string, string> }).env;
    expect(spawnEnv['NEXUS_ANTHROPIC_API_KEY']).toBeUndefined();
    fs.rmSync(dir, { recursive: true });
  });
});

describe('startDaemon: failure handling', () => {
  afterEach(() => {
    jest.restoreAllMocks();
    mockSpawn.mockReset();
  });

  it('terminates the process and prints its log when the daemon never becomes healthy', async () => {
    refuseConnection();
    const { dir, logFile } = makeTmpDir();
    // startDaemon truncates the log when it opens it; the (fake) daemon writes afterwards.
    mockSpawn.mockImplementation(() => {
      fs.writeFileSync(logFile, 'listen tcp: address already in use\n');
      return { pid: 4242, unref: jest.fn() } as unknown as ReturnType<typeof childProcess.spawn>;
    });
    const killSpy = jest.spyOn(process, 'kill').mockImplementation(() => true);
    const writeSpy = jest.spyOn(process.stdout, 'write').mockImplementation(() => true);
    await expect(
      startDaemon({
        ...baseOpts,
        dbPath: path.join(dir, 'nexus.db'),
        logFile,
        healthTimeoutMs: 50,
      }),
    ).rejects.toThrow('did not become healthy within 50ms');
    expect(killSpy).toHaveBeenCalledWith(4242, 'SIGTERM');
    expect(writeSpy).toHaveBeenCalledWith(expect.stringContaining('address already in use'));
    fs.rmSync(dir, { recursive: true });
  });

  it('does not mask the real error when the failed daemon is already gone', async () => {
    refuseConnection();
    mockSpawn.mockReturnValue({ pid: 4243, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    jest.spyOn(process, 'kill').mockImplementation(() => {
      throw new Error('ESRCH');
    });
    const { dir, logFile } = makeTmpDir();
    await expect(
      startDaemon({
        ...baseOpts,
        dbPath: path.join(dir, 'nexus.db'),
        logFile,
        healthTimeoutMs: 50,
      }),
    ).rejects.toThrow('did not become healthy');
    fs.rmSync(dir, { recursive: true });
  });

  it('passes Anthropic and Copilot credentials and models to the daemon', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 77, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    await startDaemon({
      ...baseOpts,
      dbPath: path.join(dir, 'nexus.db'),
      logFile,
      anthropicApiKey: 'ak',
      anthropicModel: 'opus',
      githubCopilotToken: 'ct',
      githubCopilotModel: 'cm',
    });
    const env = (mockSpawn.mock.calls[0][2] as { env: Record<string, string> }).env;
    expect(env['NEXUS_ANTHROPIC_API_KEY']).toBe('ak');
    expect(env['NEXUS_ANTHROPIC_MODEL']).toBe('opus');
    expect(env['NEXUS_GITHUBCOPILOT_TOKEN']).toBe('ct');
    expect(env['NEXUS_GITHUBCOPILOT_MODEL']).toBe('cm');
    fs.rmSync(dir, { recursive: true });
  });

  it('omits a model variable when only the key is given', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 78, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    await startDaemon({
      ...baseOpts,
      dbPath: path.join(dir, 'nexus.db'),
      logFile,
      anthropicApiKey: 'ak',
      githubCopilotToken: 'ct',
      openaiApiKey: 'ok',
    });
    const env = (mockSpawn.mock.calls[0][2] as { env: Record<string, string> }).env;
    expect(env['NEXUS_ANTHROPIC_MODEL']).toBeUndefined();
    expect(env['NEXUS_GITHUBCOPILOT_MODEL']).toBeUndefined();
    expect(env['NEXUS_OPENAI_MODEL']).toBeUndefined();
    fs.rmSync(dir, { recursive: true });
  });

  it('drops env entries whose value is undefined', async () => {
    respondWith(200);
    mockSpawn.mockReturnValue({ pid: 79, unref: jest.fn() } as unknown as ReturnType<
      typeof childProcess.spawn
    >);
    const { dir, logFile } = makeTmpDir();
    await startDaemon({ ...baseOpts, dbPath: path.join(dir, 'nexus.db'), logFile });
    const env = (mockSpawn.mock.calls[0][2] as { env: Record<string, string> }).env;
    expect(Object.values(env).every((v) => typeof v === 'string')).toBe(true);
    fs.rmSync(dir, { recursive: true });
  });
});

describe('waitForHealth: request timeouts', () => {
  it('destroys a health request that times out so sockets do not pile up', async () => {
    const destroy = jest.fn();
    mockHttpGet.mockImplementation((_url, _opts, _cb) => {
      const handlers: Record<string, (e?: Error) => void> = {};
      const req = {
        on: (event: string, h: (e?: Error) => void) => {
          handlers[event] = h;
          if (event === 'timeout') setImmediate(() => h());
          return req;
        },
        destroy: (err: Error) => {
          destroy(err);
          setImmediate(() => handlers['error']?.(err));
        },
      } as unknown as http.ClientRequest;
      return req;
    });
    await expect(waitForHealth('http://127.0.0.1:1/api/health', 30)).rejects.toThrow(
      'did not become healthy',
    );
    expect(destroy).toHaveBeenCalledWith(expect.objectContaining({ message: 'timeout' }));
  });
});
