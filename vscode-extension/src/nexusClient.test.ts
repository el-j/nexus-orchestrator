import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NexusClient, NexusClientError } from './nexusClient';

// ── fetch mock helper ─────────────────────────────────────────────────────────

function okJson(body: unknown) {
  return { ok: true, status: 200, json: async () => body, text: async () => JSON.stringify(body) };
}

function failHttp(status: number, body = 'error') {
  return { ok: false, status, json: async () => ({}), text: async () => body };
}

const BASE = 'http://127.0.0.1:63987';

function makeClient() {
  return new NexusClient(BASE, 63988);
}

const stubTask = {
  id: 't-1',
  projectPath: '/proj',
  targetFile: 'main.go',
  instruction: 'write tests',
  contextFiles: [],
  status: 'QUEUED',
  createdAt: '2026-04-01T00:00:00Z',
  updatedAt: '2026-04-01T00:00:00Z',
};

const stubSession = {
  id: 's-1',
  agentName: 'TestAgent',
  source: 'vscode',
  status: 'active',
  lastActivity: '2026-04-01T00:00:00Z',
};

describe('NexusClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // ── getTasks ────────────────────────────────────────────────────────────────

  it('getTasks returns validated Task[]', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(okJson([stubTask])));
    const client = makeClient();

    const tasks = await client.getTasks();

    expect(tasks).toHaveLength(1);
    expect(tasks[0].id).toBe('t-1');
    expect(tasks[0].status).toBe('QUEUED');
  });

  it('getTasks throws NexusClientError on schema mismatch', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(okJson([{ id: 't-bad' }]))); // missing required fields
    const client = makeClient();

    await expect(client.getTasks()).rejects.toThrow();
  });

  it('getTasks throws on HTTP error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(failHttp(500, 'internal error')));
    const client = makeClient();

    await expect(client.getTasks()).rejects.toThrow('HTTP 500');
  });

  // ── getTask ─────────────────────────────────────────────────────────────────

  it('getTask fetches the correct URL', async () => {
    const fetchMock = vi.fn().mockResolvedValue(okJson(stubTask));
    vi.stubGlobal('fetch', fetchMock);
    const client = makeClient();

    const task = await client.getTask('t-1');

    expect(task.id).toBe('t-1');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/tasks/t-1');
  });

  // ── cancelTask ──────────────────────────────────────────────────────────────

  it('cancelTask sends DELETE and resolves on 204', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204, text: async () => '' });
    vi.stubGlobal('fetch', fetchMock);
    const client = makeClient();

    await expect(client.cancelTask('t-del')).resolves.toBeUndefined();
    expect(fetchMock.mock.calls[0][1].method).toBe('DELETE');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/tasks/t-del');
  });

  it('cancelTask throws on 404', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(failHttp(404, 'not found')));
    const client = makeClient();

    await expect(client.cancelTask('missing')).rejects.toThrow('HTTP 404');
  });

  // ── tryConnect ──────────────────────────────────────────────────────────────

  it('tryConnect returns the first port that responds with ok', async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error('ECONNREFUSED'))
      .mockResolvedValueOnce({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    const client = makeClient();

    const port = await client.tryConnect([63986, 63987]);

    expect(port).toBe(63987);
  });

  it('tryConnect throws when no port responds', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')));
    const client = makeClient();

    await expect(client.tryConnect([9001, 9002])).rejects.toThrow('no responding daemon');
  });

  it('tryConnect deduplicates ports', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    const client = makeClient();

    await client.tryConnect([63987, 63987, 63987]);

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  // ── health ──────────────────────────────────────────────────────────────────

  it('health() returns true when daemon reports status:ok', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(okJson({ status: 'ok', service: 'nexus' })));
    const client = makeClient();

    expect(await client.health()).toBe(true);
  });

  it('health() returns false on fetch error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('timeout')));
    const client = makeClient();

    expect(await client.health()).toBe(false);
  });

  it('health() returns false when status is not ok', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(failHttp(503)));
    const client = makeClient();

    expect(await client.health()).toBe(false);
  });

  // ── registerSession ─────────────────────────────────────────────────────────

  it('registerSession posts to /api/ai-sessions and returns validated AISession', async () => {
    const fetchMock = vi.fn().mockResolvedValue(okJson(stubSession));
    vi.stubGlobal('fetch', fetchMock);
    const client = makeClient();

    const session = await client.registerSession({ agentName: 'TestAgent', source: 'vscode' });

    expect(session.id).toBe('s-1');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/ai-sessions');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
  });

  // ── heartbeatSession ────────────────────────────────────────────────────────

  it('heartbeatSession posts to the heartbeat endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    const client = makeClient();

    await expect(client.heartbeatSession('s-1')).resolves.toBeUndefined();
    expect(String(fetchMock.mock.calls[0][0])).toContain('heartbeat');
  });

  it('heartbeatSession silently ignores 404 (session cleaned up)', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(failHttp(404)));
    const client = makeClient();

    await expect(client.heartbeatSession('gone')).resolves.toBeUndefined();
  });

  // ── ingestKnowledge ─────────────────────────────────────────────────────────

  it('ingestKnowledge returns ingestedSections count', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(okJson({ ingestedSections: 7 })));
    const client = makeClient();

    const count = await client.ingestKnowledge('/proj', '/proj/CLAUDE.md');

    expect(count).toBe(7);
  });

  // ── getProviders ─────────────────────────────────────────────────────────────

  it('getProviders returns validated Provider[]', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(okJson([{ name: 'ollama', active: true, kind: 'ollama' }])),
    );
    const client = makeClient();

    const providers = await client.getProviders();

    expect(providers).toHaveLength(1);
    expect(providers[0].name).toBe('ollama');
  });

  // ── updateTaskStatus ─────────────────────────────────────────────────────────

  it('updateTaskStatus sends PUT with sessionId and status', async () => {
    const updated = { ...stubTask, status: 'COMPLETED' };
    const fetchMock = vi.fn().mockResolvedValue(okJson(updated));
    vi.stubGlobal('fetch', fetchMock);
    const client = makeClient();

    const task = await client.updateTaskStatus('t-1', 's-1', 'COMPLETED', 'done');

    expect(task.status).toBe('COMPLETED');
    expect(fetchMock.mock.calls[0][1].method).toBe('PUT');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/status');
  });

  // ── NexusClientError ─────────────────────────────────────────────────────────

  it('NexusClientError has correct name', () => {
    const err = new NexusClientError('test', new Error('cause'));
    expect(err.name).toBe('NexusClientError');
    expect(err.message).toBe('test');
  });

  it('getMcpPort returns the configured port', () => {
    const client = new NexusClient(BASE, 9999);
    expect(client.getMcpPort()).toBe(9999);
  });
});
