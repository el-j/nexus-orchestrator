import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { NexusClient, NexusClientError } from './nexusClient';

const BASE = 'http://127.0.0.1:63987';

type FetchInit = RequestInit & { headers?: Record<string, string> };
const fetchMock = vi.fn<(url: string, init?: FetchInit) => Promise<Response>>();

const json = (body: unknown, status = 200) =>
  Promise.resolve(
    new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }),
  );
const text = (body: string, status: number) => Promise.resolve(new Response(body, { status }));
const empty = (status = 204) => Promise.resolve(new Response(null, { status }));

const task = (o: Record<string, unknown> = {}) => ({
  id: 't1',
  projectPath: '/p',
  targetFile: 'a.go',
  instruction: 'do it',
  contextFiles: null,
  status: 'QUEUED',
  createdAt: 'c',
  updatedAt: 'u',
  ...o,
});
const session = (o: Record<string, unknown> = {}) => ({
  id: 's1',
  agentName: 'Copilot',
  source: 'vscode',
  status: 'active',
  lastActivity: 'now',
  ...o,
});

const last = () => {
  const [url, init] = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
  return { url, init: init ?? {}, body: init?.body ? JSON.parse(init.body as string) : undefined };
};

let client: NexusClient;
beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
  client = new NexusClient(BASE, 63988);
});
afterEach(() => vi.unstubAllGlobals());

describe('configuration', () => {
  it('exposes and updates its connection in place', () => {
    expect(client.getMcpPort()).toBe(63988);
    expect(client.getBaseUrl()).toBe(BASE);
    client.configure('http://10.0.0.5:7000', 7001, 'tok');
    expect(client.getBaseUrl()).toBe('http://10.0.0.5:7000');
    expect(client.getMcpPort()).toBe(7001);
  });

  it('uses the new URL and token after configure()', async () => {
    client.configure('http://10.0.0.5:7000', 7001, 'tok');
    fetchMock.mockImplementation(() => json([]));
    await client.getTasks();
    expect(last().url).toBe('http://10.0.0.5:7000/api/tasks');
    expect(last().init.headers).toMatchObject({ Authorization: 'Bearer tok' });
  });

  it('clears the token when configured without one', async () => {
    client.configure(BASE, 63988, 'tok');
    client.configure(BASE, 63988);
    fetchMock.mockImplementation(() => json([]));
    await client.getTasks();
    expect(last().init.headers).not.toHaveProperty('Authorization');
  });
});

describe('requests', () => {
  it('send the bearer token on every call when one is set', async () => {
    const authed = new NexusClient(BASE, 63988, 'secret');
    fetchMock.mockImplementation(() => json({ status: 'ok' }));
    await authed.health();
    expect(last().init.headers).toMatchObject({ Authorization: 'Bearer secret' });
    fetchMock.mockImplementation(() => empty());
    await authed.cancelTask('t1');
    expect(last().init.headers).toMatchObject({ Authorization: 'Bearer secret' });
  });

  it('send no Authorization header without a token', async () => {
    fetchMock.mockImplementation(() => json([]));
    await client.getTasks();
    expect(last().init.headers).not.toHaveProperty('Authorization');
  });

  it('carry a timeout signal', async () => {
    fetchMock.mockImplementation(() => json([]));
    await client.getTasks();
    expect(last().init.signal).toBeInstanceOf(AbortSignal);
  });

  it('turn a timeout into a readable error', async () => {
    const err = new Error('timed out');
    err.name = 'TimeoutError';
    fetchMock.mockRejectedValue(err);
    await expect(client.getTasks()).rejects.toThrow(/no response from .*\/api\/tasks within 15s/);
    await expect(client.getTasks()).rejects.toBeInstanceOf(NexusClientError);
    const abort = new Error('aborted');
    abort.name = 'AbortError';
    fetchMock.mockRejectedValue(abort);
    await expect(client.getTasks()).rejects.toBeInstanceOf(NexusClientError);
  });

  it('let other network errors through untouched', async () => {
    fetchMock.mockRejectedValue(new TypeError('fetch failed'));
    await expect(client.getTasks()).rejects.toThrow('fetch failed');
  });

  it('report the daemon’s error message on a failure', async () => {
    fetchMock.mockImplementation(() => json({ error: 'task queue is full' }, 429));
    await expect(client.submitTask({ instruction: 'x', projectPath: '/p' })).rejects.toThrow(
      'nexus: POST /api/tasks: HTTP 429 — task queue is full',
    );
  });

  it('fall back to raw text, or nothing, when the error body is not the usual JSON', async () => {
    fetchMock.mockImplementation(() => text('Bad Gateway', 502));
    await expect(client.getTasks()).rejects.toThrow(
      'nexus: GET /api/tasks: HTTP 502 — Bad Gateway',
    );
    fetchMock.mockImplementation(() => text('', 500));
    await expect(client.getTasks()).rejects.toThrow(/HTTP 500$/);
    fetchMock.mockImplementation(() => json({ error: 5 }, 400));
    await expect(client.getTasks()).rejects.toThrow('{"error":5}');
  });
});

describe('connection probing', () => {
  it('finds the first responding REST port using /api/health (not /health)', async () => {
    fetchMock.mockImplementation((url) =>
      url.includes(':63989') ? json({ status: 'ok' }) : Promise.reject(new Error('refused')),
    );
    expect(await client.tryConnect([63987, 63987, 63988, 63989])).toBe(63989);
    const urls = fetchMock.mock.calls.map((c) => c[0]);
    expect(urls).toEqual([
      'http://127.0.0.1:63987/api/health',
      'http://127.0.0.1:63988/api/health',
      'http://127.0.0.1:63989/api/health',
    ]);
  });

  it('skips ports that answer with an error status', async () => {
    fetchMock.mockImplementation((url) => (url.includes(':2') ? json({}) : text('nope', 404)));
    expect(await client.tryConnect([1, 2])).toBe(2);
  });

  it('fails when nothing answers', async () => {
    fetchMock.mockRejectedValue(new Error('refused'));
    await expect(client.tryConnect([1, 2])).rejects.toThrow('no responding daemon');
  });

  it('health() is true only for an ok status', async () => {
    fetchMock.mockImplementation(() => json({ status: 'ok' }));
    expect(await client.health()).toBe(true);
    fetchMock.mockImplementation(() => json({ status: 'degraded' }));
    expect(await client.health()).toBe(false);
    fetchMock.mockImplementation(() => json({}, 503));
    expect(await client.health()).toBe(false);
    fetchMock.mockRejectedValue(new Error('down'));
    expect(await client.health()).toBe(false);
  });
});

describe('tasks', () => {
  it('submits a task and returns the full task fetched by id', async () => {
    fetchMock
      .mockImplementationOnce(() => json({ task_id: 't 9', status: 'QUEUED' }, 201))
      .mockImplementationOnce(() => json(task({ id: 't 9' })));
    const t = await client.submitTask({ instruction: 'do it', projectPath: '/p', command: 'plan' });
    expect(fetchMock.mock.calls[0][0]).toBe(`${BASE}/api/tasks`);
    expect(fetchMock.mock.calls[0][1]?.method).toBe('POST');
    expect(JSON.parse(fetchMock.mock.calls[0][1]?.body as string)).toMatchObject({
      command: 'plan',
    });
    expect(fetchMock.mock.calls[1][0]).toBe(`${BASE}/api/tasks/t%209`);
    expect(t.id).toBe('t 9');
    expect(t.contextFiles).toEqual([]); // null from Go becomes []
  });

  it('lists queued and all tasks, and fetches one', async () => {
    fetchMock.mockImplementation(() => json([task()]));
    expect(await client.getTasks()).toHaveLength(1);
    expect(last().url).toBe(`${BASE}/api/tasks`);
    await client.getAllTasks();
    expect(last().url).toBe(`${BASE}/api/tasks/all`);
    fetchMock.mockImplementation(() => json(task()));
    expect((await client.getTask('x')).id).toBe('t1');
  });

  it('rejects a malformed task with a validation error', async () => {
    fetchMock.mockImplementation(() => json([{ id: 1 }]));
    await expect(client.getTasks()).rejects.toThrow(/response validation failed/);
    fetchMock.mockImplementation(() => json(task({ status: 'WEIRD' })));
    await expect(client.getTask('t1')).rejects.toBeInstanceOf(NexusClientError);
  });

  it('cancels a task and reports failures', async () => {
    fetchMock.mockImplementation(() => empty());
    await client.cancelTask('t/1');
    expect(last().url).toBe(`${BASE}/api/tasks/t%2F1`);
    expect(last().init.method).toBe('DELETE');
    fetchMock.mockImplementation(() => json({ error: 'not found' }, 404));
    await expect(client.cancelTask('t1')).rejects.toThrow(
      'nexus: cancel task t1: HTTP 404 — not found',
    );
  });

  it('claims a task for a session and updates its status', async () => {
    fetchMock.mockImplementation(() => json(task({ status: 'PROCESSING' })));
    await client.claimTask('t1', 's1');
    expect(last().url).toBe(`${BASE}/api/tasks/t1/claim`);
    expect(last().body).toEqual({ sessionId: 's1' });
    fetchMock.mockImplementation(() => json(task({ status: 'COMPLETED' })));
    await client.updateTaskStatus('t1', 's1', 'COMPLETED', 'done');
    expect(last().url).toBe(`${BASE}/api/tasks/t1/status`);
    expect(last().init.method).toBe('PUT');
    expect(last().body).toEqual({ sessionId: 's1', status: 'COMPLETED', logs: 'done' });
    fetchMock.mockImplementation(() => json({ error: 'wrong session' }, 403));
    await expect(client.updateTaskStatus('t1', 'x', 'FAILED')).rejects.toThrow(
      'PUT /api/tasks/t1/status: HTTP 403 — wrong session',
    );
  });
});

describe('providers and sessions', () => {
  it('lists providers', async () => {
    fetchMock.mockImplementation(() => json([{ name: 'Ollama', active: true, models: ['a'] }]));
    expect((await client.getProviders())[0].name).toBe('Ollama');
    fetchMock.mockImplementation(() => json([{ name: 'x' }]));
    await expect(client.getProviders()).rejects.toThrow(/validation failed/);
  });

  it('registers, lists and deregisters sessions', async () => {
    fetchMock.mockImplementation(() => json(session()));
    const s = await client.registerSession({ agentName: 'Copilot', source: 'vscode' });
    expect(s.id).toBe('s1');
    expect(last().url).toBe(`${BASE}/api/ai-sessions`);
    fetchMock.mockImplementation(() => json([session()]));
    expect(await client.getAISessions()).toHaveLength(1);
    expect(await client.listAISessions()).toHaveLength(1);
    fetchMock.mockImplementation(() => empty());
    await client.deregisterSession('s 1');
    expect(last().url).toBe(`${BASE}/api/ai-sessions/s%201`);
    fetchMock.mockImplementation(() => json({ error: 'gone' }, 404));
    await expect(client.deregisterSession('s1')).rejects.toThrow(
      'deregister session s1: HTTP 404 — gone',
    );
  });

  it('heartbeats, tolerating a 404 (session was cleaned up) but not other errors', async () => {
    fetchMock.mockImplementation(() => empty(200));
    await client.heartbeatSession('s1');
    expect(last().url).toBe(`${BASE}/api/ai-sessions/s1/heartbeat`);
    fetchMock.mockImplementation(() => text('', 404));
    await expect(client.heartbeatSession('s1')).resolves.toBeUndefined();
    fetchMock.mockImplementation(() => json({ error: 'boom' }, 500));
    await expect(client.heartbeatSession('s1')).rejects.toThrow(
      'heartbeat session s1: HTTP 500 — boom',
    );
  });

  it('lists a session’s tasks, discovered agents and delegates', async () => {
    fetchMock.mockImplementation(() => json([task()]));
    await client.getSessionTasks('s/1');
    expect(last().url).toBe(`${BASE}/api/ai-sessions/s%2F1/tasks`);
    fetchMock.mockImplementation(() => json([{ id: 'a1' }]));
    expect(await client.getDiscoveredAgents()).toEqual([{ id: 'a1' }]);
    fetchMock.mockImplementation(() => json({ instruction: 'go', sessionId: 's1' }));
    expect(await client.delegateSession('s1')).toEqual({ instruction: 'go', sessionId: 's1' });
    expect(last().url).toBe(`${BASE}/api/ai-sessions/s1/delegate`);
  });
});

describe('activities', () => {
  it('builds the query from the options that are set', async () => {
    fetchMock.mockImplementation(() => json([{ id: 'x' }]));
    await client.getActivities();
    expect(last().url).toBe(`${BASE}/api/activities`);
    await client.getActivities({
      agentName: 'claude',
      projectPath: '/p q',
      type: 'message',
      limit: 5,
    });
    expect(last().url).toBe(
      `${BASE}/api/activities?agent=claude&project=%2Fp+q&type=message&limit=5`,
    );
  });

  it('returns an empty list when the daemon refuses', async () => {
    fetchMock.mockImplementation(() => json({}, 500));
    expect(await client.getActivities()).toEqual([]);
  });
});

describe('brain', () => {
  it('searches with the q parameter the daemon expects', async () => {
    fetchMock.mockImplementation(() => json({ results: [{ topic: 't' }] }));
    expect(await client.searchKnowledge('/p', 'auth flow')).toEqual([{ topic: 't' }]);
    const url = new URL(last().url);
    expect(url.searchParams.get('q')).toBe('auth flow');
    expect(url.searchParams.get('projectPath')).toBe('/p');
    expect(url.searchParams.get('limit')).toBe('5');
    expect(url.searchParams.has('query')).toBe(false);
  });

  it('returns no results when the envelope has none', async () => {
    fetchMock.mockImplementation(() => json({}));
    expect(await client.searchKnowledge('/p', 'x')).toEqual([]);
  });

  it('ingests, reports status and builds context', async () => {
    fetchMock.mockImplementation(() => json({ ingestedSections: 4 }));
    expect(await client.ingestKnowledge('/p', '/p/CLAUDE.md')).toBe(4);
    fetchMock.mockImplementation(() => json({ initialized: true }));
    await client.getBrainStatus('/p q');
    expect(last().url).toBe(`${BASE}/api/brain/status?projectPath=%2Fp%20q`);
    fetchMock.mockImplementation(() => json({ sections: [] }));
    await client.getProjectContext('/p', 500);
    expect(last().body).toEqual({ projectPath: '/p', maxTokens: 500 });
    await client.getFocusedContext('/p', 'why?', 200);
    expect(last().url).toBe(`${BASE}/api/brain/focused-context`);
    expect(last().body).toEqual({ projectPath: '/p', question: 'why?', maxTokens: 200 });
  });

  it('initialises with and without a CLAUDE.md path', async () => {
    fetchMock.mockImplementation(() => json({}));
    await client.initProject('/p');
    expect(last().body).toEqual({ projectPath: '/p', claudeMDPath: '' });
    await client.initProject('/p', '/p/CLAUDE.md');
    expect(last().body).toEqual({ projectPath: '/p', claudeMDPath: '/p/CLAUDE.md' });
  });

  it('lists knowledge with an optional kind, treating null as empty', async () => {
    fetchMock.mockImplementation(() => json(null));
    expect(await client.listKnowledge('/p')).toEqual([]);
    expect(new URL(last().url).searchParams.has('kind')).toBe(false);
    await client.listKnowledge('/p', 'learning');
    expect(new URL(last().url).searchParams.get('kind')).toBe('learning');
  });

  it('deletes knowledge and reports a failure', async () => {
    fetchMock.mockImplementation(() => empty());
    await client.deleteKnowledge('k/1');
    expect(last().url).toBe(`${BASE}/api/brain/knowledge/k%2F1`);
    fetchMock.mockImplementation(() => json({ error: 'nope' }, 404));
    await expect(client.deleteKnowledge('k1')).rejects.toThrow(
      'DELETE /api/brain/knowledge/k1: HTTP 404 — nope',
    );
  });

  it('reads the file map with an optional focus area, tolerating a missing list', async () => {
    fetchMock.mockImplementation(() => json({ filePaths: ['a.ts'] }));
    expect(await client.getFileMap('/p', 'src')).toEqual(['a.ts']);
    expect(new URL(last().url).searchParams.get('focusArea')).toBe('src');
    fetchMock.mockImplementation(() => json({}));
    expect(await client.getFileMap('/p')).toEqual([]);
    expect(new URL(last().url).searchParams.has('focusArea')).toBe(false);
  });
});
